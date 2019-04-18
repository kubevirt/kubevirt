/*
Copyright 2018 The CDI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package importer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"io"
	"io/ioutil"
	"strconv"

	"github.com/pkg/errors"
	"github.com/ulikunitz/xz"

	"k8s.io/klog"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"kubevirt.io/containerized-data-importer/pkg/image"
)

type reader struct {
	rdrType int
	rdr     io.ReadCloser
}

// FormatReaders contains the stack of readers needed to get information from the input stream (io.ReadCloser)
type FormatReaders struct {
	readers     []reader
	buf         []byte // holds file headers
	Convert     bool
	Archived    bool
	contentType cdiv1.DataVolumeContentType
}

const (
	rdrGz = iota
	rdrMulti
	rdrTar
	rdrXz
	rdrStream
)

// map scheme and format to rdrType
var rdrTypM = map[string]int{
	"gz":     rdrGz,
	"tar":    rdrTar,
	"xz":     rdrXz,
	"stream": rdrStream,
}

// NewFormatReaders creates a new instance of FormatReaders using the input stream and content type passed in.
func NewFormatReaders(stream io.ReadCloser, contentType cdiv1.DataVolumeContentType) (*FormatReaders, error) {
	readers := &FormatReaders{
		buf:         make([]byte, image.MaxExpectedHdrSize),
		contentType: contentType,
	}
	err := readers.constructReaders(stream)
	return readers, err
}

func (fr *FormatReaders) constructReaders(r io.ReadCloser) error {
	fr.appendReader(rdrTypM["stream"], r)
	knownHdrs := image.CopyKnownHdrs() // need local copy since keys are removed
	klog.V(3).Infof("constructReaders: checking compression and archive formats\n")
	var isTarFile bool
	for {
		hdr, err := fr.matchHeader(&knownHdrs)
		if err != nil {
			return errors.WithMessage(err, "could not process image header")
		}
		if hdr == nil {
			break // done processing headers, we have the orig source file
		}
		klog.V(2).Infof("found header of type %q\n", hdr.Format)
		// create format-specific reader and append it to dataStream readers stack
		fr.fileFormatSelector(hdr)
		isTarFile = isTarFile || hdr.Format == "tar"
		// exit loop if hdr is qcow2
		if hdr.Format == "qcow2" {
			break
		}
	}

	if fr.contentType == cdiv1.DataVolumeArchive && !isTarFile {
		return errors.Errorf("cannot process a non tar file as an archive")
	}

	return nil
}

// Append to the receiver's reader stack the passed in reader. If the reader type is multi-reader
// then wrap a multi-reader around the passed in reader. If the reader is not a Closer then wrap a
// nop closer.
func (fr *FormatReaders) appendReader(rType int, x interface{}) {
	if x == nil {
		return
	}
	r, ok := x.(io.Reader)
	if !ok {
		klog.Errorf("internal error: unexpected reader type passed to appendReader()")
		return
	}
	if rType == rdrMulti {
		r = io.MultiReader(r, fr.TopReader())
	}
	if _, ok := r.(io.Closer); !ok {
		r = ioutil.NopCloser(r)
	}
	fr.readers = append(fr.readers, reader{rdrType: rType, rdr: r.(io.ReadCloser)})
}

// TopReader return the top-level io.ReadCloser from the receiver Reader "stack".
func (fr *FormatReaders) TopReader() io.ReadCloser {
	return fr.readers[len(fr.readers)-1].rdr
}

// Based on the passed in header, append the format-specific reader to the readers stack,
// and update the receiver Size field. Note: a bool is set in the receiver for qcow2 files.
func (fr *FormatReaders) fileFormatSelector(hdr *image.Header) {
	var r io.Reader
	var err error
	fFmt := hdr.Format
	switch fFmt {
	case "gz":
		r, err = fr.gzReader()
		if err == nil {
			fr.Archived = true
		}
	case "qcow2":
		r, err = fr.qcow2NopReader(hdr)
		fr.Convert = true
	case "tar":
		r, err = fr.tarReader()
		if err == nil {
			fr.Archived = true
		}
	case "xz":
		r, err = fr.xzReader()
		if err == nil {
			fr.Archived = true
		}
	}
	if err == nil && r != nil {
		fr.appendReader(rdrTypM[fFmt], r)
	}
}

// Return the gz reader and the size of the endpoint "through the eye" of the previous reader.
// Assumes a single file was gzipped.
//NOTE: size in gz is stored in the last 4 bytes of the file. This probably requires the file
//  to be decompressed in order to get its original size. For now 0 is returned.
//TODO: support gz size.
func (fr *FormatReaders) gzReader() (io.ReadCloser, error) {
	gz, err := gzip.NewReader(fr.TopReader())
	if err != nil {
		return nil, errors.Wrap(err, "could not create gzip reader")
	}
	klog.V(2).Infof("gzip: extracting %q\n", gz.Name)
	return gz, nil
}

// Return the size of the endpoint "through the eye" of the previous reader. Note: there is no
// qcow2 reader so nil is returned so that nothing is appended to the reader stack.
// Note: size is stored at offset 24 in the qcow2 header.
func (fr *FormatReaders) qcow2NopReader(h *image.Header) (io.Reader, error) {
	s := hex.EncodeToString(fr.buf[h.SizeOff : h.SizeOff+h.SizeLen])
	_, err := strconv.ParseInt(s, 16, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to determine original qcow2 file size from %+v", s)
	}
	return nil, nil
}

// Return the xz reader and size of the endpoint "through the eye" of the previous reader.
// Assumes a single file was compressed. Note: the xz reader is not a closer so we wrap a
// nop Closer around it.
//NOTE: size is not stored in the xz header. This may require the file to be decompressed in
//  order to get its original size. For now 0 is returned.
//TODO: support gz size.
func (fr *FormatReaders) xzReader() (io.Reader, error) {
	xz, err := xz.NewReader(fr.TopReader())
	if err != nil {
		return nil, errors.Wrap(err, "could not create xz reader")
	}
	return xz, nil
}

// Return the tar reader and size of the endpoint "through the eye" of the previous reader.
// Assumes a single file was archived.
// Note: the size stored in the header is used rather than raw metadata.
func (fr *FormatReaders) tarReader() (io.Reader, error) {
	if fr.contentType == cdiv1.DataVolumeArchive {
		return fr.mulFileTarReader()
	}
	tr := tar.NewReader(fr.TopReader())
	hdr, err := tr.Next() // advance cursor to 1st (and only) file in tarball
	if err != nil {
		return nil, errors.Wrap(err, "could not read tar header")
	}
	klog.V(2).Infof("tar: extracting %q\n", hdr.Name)
	return tr, nil
}

func (fr *FormatReaders) mulFileTarReader() (io.Reader, error) {
	buf, err := ioutil.ReadAll(fr.TopReader())
	if err != nil {
		return nil, err
	}
	tar.NewReader(bytes.NewReader(buf))
	return bytes.NewReader(buf), nil
}

// Return the matching header, if one is found, from the passed-in map of known headers. After a
// successful read append a multi-reader to the receiver's reader stack.
// Note: .iso files are not detected here but rather in the Size() function.
// Note: knownHdrs is passed by reference and modified.
func (fr *FormatReaders) matchHeader(knownHdrs *image.Headers) (*image.Header, error) {
	_, err := fr.read(fr.buf) // read current header
	if err != nil {
		return nil, err
	}
	// append multi-reader so that the header data can be re-read by subsequent readers
	fr.appendReader(rdrMulti, bytes.NewReader(fr.buf))

	// loop through known headers until a match
	for format, kh := range *knownHdrs {
		if kh.Match(fr.buf) {
			// delete this header format key so that it's not processed again
			delete(*knownHdrs, format)
			return &kh, nil
		}
	}
	return nil, nil // no match
}

// Read from top-most reader. Note: ReadFull is needed since there may be intermediate,
// smaller multi-readers in the reader stack, and we need to be able to fill buf.
func (fr *FormatReaders) read(buf []byte) (int, error) {
	return io.ReadFull(fr.TopReader(), buf)
}

// Close Readers in reverse order.
func (fr *FormatReaders) Close() (rtnerr error) {
	var err error
	for i := len(fr.readers) - 1; i >= 0; i-- {
		err = fr.readers[i].rdr.Close()
		if err != nil {
			rtnerr = err // tracking last error
		}
	}
	return rtnerr
}
