package importer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/minio/minio-go"
	"github.com/pkg/errors"
	"github.com/xi2/xz"
	. "kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/image"
)

type DataStreamInterface interface {
	dataStreamSelector() error
	fileFormatSelector(h *image.Header) error
	parseDataPath() (string, string)
	Read(p []byte) (int, error)
	Close() error
}

var _ DataStreamInterface = &dataStream{}

// implements the ReadCloser interface
type dataStream struct {
	Url         *url.URL
	Readers     []Reader
	buf         []byte // holds file headers
	qemu        bool
	Size        int64
	accessKeyId string
	secretKey   string
}

type Reader struct {
	RdrType int
	Rdr     io.ReadCloser
}

const (
	RdrHttp = iota
	RdrS3
	RdrFile
	RdrGz
	RdrMulti
	RdrQcow2
	RdrTar
	RdrXz
)

// map scheme and format to RdrType
var rdrTypM = map[string]int{
	"gz":    RdrGz,
	"http":  RdrHttp,
	"https": RdrHttp,
	"local": RdrFile,
	"qcow2": RdrQcow2,
	"s3":    RdrS3,
	"tar":   RdrTar,
	"xz":    RdrXz,
}

// Return a dataStream object after validating the endpoint and constructing the reader/closer chain.
// Note: the caller must close the `Readers` in reverse order. See Close().
func NewDataStream(endpt, accKey, secKey string) (*dataStream, error) {
	if len(accKey) == 0 || len(secKey) == 0 {
		glog.Warningf("%s and/or %s env variables are empty\n", IMPORTER_ACCESS_KEY_ID, IMPORTER_SECRET_KEY)
	}
	ep, err := ParseEndpoint(endpt)
	if err != nil {
		return nil, errors.Wrapf(err, fmt.Sprintf("unable to parse endpoint %q", endpt))
	}
	ds := &dataStream{
		Url:         ep,
		buf:         make([]byte, image.MaxExpectedHdrSize),
		accessKeyId: accKey,
		secretKey:   secKey,
	}

	// establish readers for each nested format types in the endpoint
	err = ds.constructReaders()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to construct readers")
	}
	if ds.Size > 0 { // done, we determined the size of the original endpoint file
		return ds, nil
	}

	// the endpoint's file size is zero, if it's an iso file then compute its orig size
	ds.Size, err = ds.isoSize()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to calculate iso file size")
	}
	glog.V(Vdebug).Infof("NewDataStream: endpoint %q's computed byte size: %d", ep, ds.Size)
	return ds, nil
}

// Read from top-most reader. Note: ReadFull is needed since there may be intermediate,
// smaller multi-readers in the reader stack, and we need to be able to fill buf.
func (d dataStream) Read(buf []byte) (int, error) {
	return io.ReadFull(d.topReader(), buf)
}

// Close all readers.
func (d dataStream) Close() error {
	return closeReaders(d.Readers)
}

// Based on the endpoint scheme, append the scheme-specific reader to the receiver's
// reader stack.
func (d *dataStream) dataStreamSelector() (err error) {
	var r io.Reader
	scheme := d.Url.Scheme
	switch scheme {
	case "s3":
		r, err = d.s3()
	case "http", "https":
		r, err = d.http()
	case "file":
		r, err = d.local()
	default:
		return errors.Errorf("invalid url scheme: %q", scheme)
	}
	if err == nil {
		d.appendReader(rdrTypM[scheme], r)
	}
	return err
}

func (d dataStream) s3() (io.ReadCloser, error) {
	glog.V(Vdebug).Infoln("Using S3 client to get data")
	bucket := d.Url.Host
	object := strings.Trim(d.Url.Path, "/")
	mc, err := minio.NewV4(IMPORTER_S3_HOST, d.accessKeyId, d.secretKey, false)
	if err != nil {
		return nil, errors.Wrapf(err, "could not build minio client for %q", d.Url.Host)
	}
	glog.V(Vadmin).Infof("Attempting to get object %q via S3 client\n", d.Url.String())
	objectReader, err := mc.GetObject(bucket, object, minio.GetObjectOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "could not get s3 object: \"%s/%s\"", bucket, object)
	}
	return objectReader, nil
}

func (d dataStream) http() (io.ReadCloser, error) {
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.SetBasicAuth(d.accessKeyId, d.secretKey) // Redirects will lose basic auth, so reset them manually
			return nil
		},
	}
	req, err := http.NewRequest("GET", d.Url.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not create HTTP request")
	}
	if len(d.accessKeyId) > 0 && len(d.secretKey) > 0 {
		req.SetBasicAuth(d.accessKeyId, d.secretKey)
	}
	glog.V(Vadmin).Infof("Attempting to get object %q via http client\n", d.Url.String())
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "HTTP request errored")
	}
	if resp.StatusCode != 200 {
		glog.Errorf("http: expected status code 200, got %d", resp.StatusCode)
		return nil, errors.Errorf("expected status code 200, got %d. Status: %s", resp.StatusCode, resp.Status)
	}
	return resp.Body, nil
}

func (d dataStream) local() (io.ReadCloser, error) {
	fn := d.Url.Path
	glog.Warning("the file:/// protocol is intended *only* for debugging and testing. Files outside of the container cannot be accessed.")
	f, err := os.Open(fn)
	if err != nil {
		return nil, errors.Wrapf(err, "could not open file %q", fn)
	}
	//note: if poor perf here consider wrapping this with a buffered i/o Reader
	return f, nil
}

// Copy the source endpoint (vm image) to the provided destination path.
func CopyImage(dest, endpoint, accessKey, secKey string) error {
	glog.V(Vuser).Infof("copying %q to %q...\n", endpoint, dest)
	ds, err := NewDataStream(endpoint, accessKey, secKey)
	if err != nil {
		return errors.Wrapf(err, "unable to create data stream")
	}
	defer ds.Close()
	return ds.copy(dest)
}

// Read the endpoint and determine the file composition (eg. .iso.tar.gz) based on the magic number in
// each known file format header. Set the Reader slice in the receiver and set the Size field to each
// reader's original size. Note: if, when this method returns, the Size is still 0 then another method
// will compute the final size.
// The reader order starts with the lowest level reader, eg. http, used to read file content. The next
// readers are combinations of decompression/archive readers and bytes multi-readers. The multi-readers
// are created so that header data (interpreted by the current reader) is present for the next reader.
// Thus, the last reader in the reader stack is always a multi-reader. Readers are closed in reverse
// order, see the Close method. If a format doesn't natively support Close() a no-op Closer is wrapped
// around the native Reader so that all Readers can be consider ReadClosers.
// Examples:
//   Filename                    Readers (mr == multi-reader)
//   --------                    ----------------------------
//   "https:/foo.tar.gz"         [http, mr, gz, mr, tar, mr]
//   "file:/foo.tar.xz"          [file, mr, xz, mr, tar, mr]
//   "s3://foo-iso"              [s3, mr]
//   "https://foo.qcow2"         [http, mr]		     note: there is no qcow2 reader
//   "https://foo.qcow2.tar.gz"  [http, mr, gz, mr, tar, mr] note: there is no qcow2 reader
// Note: file extensions are ignored.
// Note: readers are not closed here, see dataStream.Close().
// Assumption: a particular header format only appears once in the data stream. Eg. foo.gz.gz is not supported.
func (d *dataStream) constructReaders() error {
	glog.V(Vadmin).Infof("create the initial Reader based on the endpoint's %q scheme", d.Url.Scheme)
	// create the scheme-specific source reader and append it to dataStream readers stack
	err := d.dataStreamSelector()
	if err != nil {
		return errors.WithMessage(err, "could not get data reader")
	}

	// loop through all supported file formats until we do not find a header we recognize
	knownHdrs := image.CopyKnownHdrs() // need local copy since keys are removed
	glog.V(Vdebug).Infof("constructReaders: checking compression and archive formats: %s\n", d.Url.Path)
	for {
		hdr, err := d.matchHeader(&knownHdrs)
		if err != nil {
			return errors.WithMessage(err, "could not process image header")
		}
		// append multi-reader so that the header data is re-read by subsequent readers
		d.appendReader(RdrMulti, bytes.NewReader(d.buf))
		if hdr == nil {
			break // done processing headers, we have the orig source file
		}
		glog.V(Vadmin).Infof("found header of type %q\n", hdr.Format)
		// create format-specific reader and append it to dataStream readers stack
		err = d.fileFormatSelector(hdr)
		if err != nil {
			return errors.WithMessage(err, "could not create compression/unarchive reader")
		}
	}
	if len(d.Readers) <= 2 {
		// 1st rdr is source, 2nd rdr is multi-rdr, >2 means we have additional formats
		glog.V(Vdebug).Infof("constructReaders: no headers found for file %q\n", d.Url.Path)
	}
	glog.V(Vadmin).Infof("done processing %q headers\n", d.Url.Path)
	return nil
}

// Append to the receiver's reader stack the passed in reader. If the reader type is a multi-reader
// then wrap a multi-reader around the passed in reader. If the reader is not a Closer then wrap a
// nop closer.
func (d *dataStream) appendReader(rType int, x interface{}) {
	if x == nil {
		return
	}
	r, ok := x.(io.Reader)
	if !ok {
		glog.Errorf("internal error: unexecected reader type passed to appendReader()")
		return
	}
	if rType == RdrMulti {
		r = io.MultiReader(r, d.topReader())
	}
	if _, ok := r.(io.Closer); !ok {
		r = ioutil.NopCloser(r)
	}
	d.Readers = append(d.Readers, Reader{RdrType: rType, Rdr: r.(io.ReadCloser)})
}

// Return the top-level io.ReadCloser from the receiver Reader "stack".
func (d dataStream) topReader() io.ReadCloser {
	return d.Readers[len(d.Readers)-1].Rdr
}

// Based on the passed in header, append the format-specific reader to the readers stack,
// and update the receiver Size field. Note: a bool is set in the receiver for qcow2 files.
func (d *dataStream) fileFormatSelector(hdr *image.Header) (err error) {
	var r io.Reader
	fFmt := hdr.Format
	switch fFmt {
	case "gz":
		r, d.Size, err = d.gzReader()
	case "qcow2":
		r, d.Size, err = d.qcow2NopReader(hdr)
		d.qemu = true
	case "tar":
		r, d.Size, err = d.tarReader()
	case "xz":
		r, d.Size, err = d.xzReader()
	default:
		return errors.Errorf("mismatch between supported file formats and this header type: %q", fFmt)
	}
	if err == nil {
		d.appendReader(rdrTypM[fFmt], r)
	}
	return err
}

// Return the gz reader and the size of the endpoint "through the eye" of the previous reader.
// Assumes a single file was gzipped.
//NOTE: size in gz is stored in the last 4 bytes of the file. This probably requires the file
//  to be decompressed in order to get its original size. For now 0 is returned.
//TODO: support gz size.
func (d dataStream) gzReader() (io.ReadCloser, int64, error) {
	gz, err := gzip.NewReader(d.topReader())
	if err != nil {
		return nil, 0, errors.Wrap(err, "could not create gzip reader")
	}
	glog.V(Vadmin).Infof("gzip: extracting %q\n", gz.Name)
	size := int64(0) //TODO: implement size
	return gz, size, nil
}

// Return the size of the endpoint "through the eye" of the previous reader. Note: there is no
// qcow2 reader so nil is returned so that nothing is appended to the reader stack.
// Note: size is stored at offset 24 in the qcow2 header.
func (d dataStream) qcow2NopReader(h *image.Header) (io.Reader, int64, error) {
	s := hex.EncodeToString(d.buf[h.SizeOff : h.SizeOff+h.SizeLen])
	size, err := strconv.ParseInt(s, 16, 64)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "unable to determine original qcow2 file size from %+v", s)
	}
	return nil, size, nil
}

// Return the xz reader and size of the endpoint "through the eye" of the previous reader.
// Assumes a single file was compressed. Note: the xz reader is not a closer so we wrap a
// nop Closer around it.
//NOTE: size is not stored in the xz header. This may require the file to be decompressed in
//  order to get its original size. For now 0 is returned.
//TODO: support gz size.
func (d dataStream) xzReader() (io.Reader, int64, error) {
	xz, err := xz.NewReader(d.topReader(), 0) //note: default dict size may be too small
	if err != nil {
		return nil, 0, errors.Wrap(err, "could not create xz reader")
	}
	size := int64(0) //TODO: implement size
	return xz, size, nil
}

// Return the tar reader and size of the endpoint "through the eye" of the previous reader.
// Assumes a single file was archived.
// Note: the size stored in the header is used rather than raw metadata.
func (d dataStream) tarReader() (io.Reader, int64, error) {
	tr := tar.NewReader(d.topReader())
	hdr, err := tr.Next() // advance cursor to 1st (and only) file in tarball
	if err != nil {
		return nil, 0, errors.Wrap(err, "could not read tar header")
	}
	glog.V(Vadmin).Infof("tar: extracting %q\n", hdr.Name)
	return tr, hdr.Size, nil
}

// If the raw endpoint is an ISO file then set the receiver's Size via the iso metadata.
// ISO reference: http://alumnus.caltech.edu/~pje/iso9660.html
func (d *dataStream) isoSize() (int64, error) {
	// iso id values
	const (
		id        = "CD001"
		primaryVD = 1
	)
	// primary volume descriptor sector offset in iso file
	const (
		isoSectorSize        = 2048
		primVolDescriptorOff = 16 * isoSectorSize
	)
	// single volume descriptor layout (independent of location within file)
	// note: offsets are zero-relative and lengths are in bytes
	const (
		vdTypeOff       = 0
		typeLen         = 1
		vdIdOff         = 1
		idLen           = 5
		vdNumSectorsOff = 84
		numSectorsLen   = 4
		vdSectorSizeOff = 130
		sectorSizeLen   = 2
	)
	// primary volume descriptor layout within full iso file (lengths are defined above)
	const (
		typeOff       = vdTypeOff + primVolDescriptorOff
		idOff         = vdIdOff + primVolDescriptorOff
		numSectorsOff = vdNumSectorsOff + primVolDescriptorOff
		sectorSizeOff = vdSectorSizeOff + primVolDescriptorOff // last field we care about
	)
	const bufSize = sectorSizeOff + sectorSizeLen

	buf := make([]byte, bufSize)
	_, err := d.Read(buf) // read primary volume descriptor
	if err != nil {
		return 0, errors.Wrapf(err, "attempting to read ISO primary volume descriptor")
	}

	// append multi-reader so that the iso data can be re-read by subsequent readers
	d.appendReader(RdrMulti, bytes.NewReader(buf))

	// ensure we have an iso file by checking the type and id value
	vdtyp, err := strconv.Atoi(hex.EncodeToString(buf[typeOff : typeOff+typeLen]))
	if err != nil {
		return 0, nil
	}
	if vdtyp != primaryVD && string(buf[idOff:idOff+idLen]) != id {
		glog.V(Vdebug).Infof("isoSize: endpoint %q is not an ISO file", d.Url.Path)
		return 0, nil
	}

	// get the logical block/sector size (expect 2048)
	s := hex.EncodeToString(buf[sectorSizeOff : sectorSizeOff+sectorSizeLen])
	sectSize, err := strconv.ParseInt(s, 16, 64)
	if err != nil {
		return 0, nil
	}
	// get the number sectors
	s = hex.EncodeToString(buf[numSectorsOff : numSectorsOff+numSectorsLen])
	numSects, err := strconv.ParseInt(s, 16, 64)
	if err != nil {
		return 0, nil
	}
	return int64(numSects * sectSize), nil
}

// Return the matching header, if one is found, from the passed-in map of known headers.
// Note: knownHdrs is passed by reference and modified.
func (d dataStream) matchHeader(knownHdrs *image.Headers) (*image.Header, error) {
	_, err := d.Read(d.buf) // read current header
	if err != nil {
		return nil, err
	}
	// loop through known headers until a match
	for format, kh := range *knownHdrs {
		if kh.Match(d.buf) {
			// delete this header format key so that it's not processed again
			delete(*knownHdrs, format)
			return &kh, nil
		}
	}
	return nil, nil // no match
}

// Close the passed-in Readers in reverse order, see constructReaders().
func closeReaders(readers []Reader) (rtnerr error) {
	var err error
	for i := len(readers) - 1; i >= 0; i-- {
		err = readers[i].Rdr.Close()
		if err != nil {
			rtnerr = err // tracking last error
		}
	}
	return rtnerr
}

// Copy endpoint to dest based on passed-in reader.
func (d dataStream) copy(dest string) error {
	return copy(d.topReader(), dest, d.qemu)
}

// Copy the file using its Reader (r) to the passed-in destination (`out`).
func copy(r io.Reader, out string, qemu bool) error {
	out = filepath.Clean(out)
	glog.V(Vadmin).Infof("copying image file to %q", out)
	dest := out
	if qemu {
		// copy to tmp; qemu conversion will write to passed-in destination
		dest = randTmpName(out)
		glog.V(Vdebug).Infof("Copy: temp file for qcow2 conversion: %q", dest)
		defer func(f string) {
			os.Remove(f)
		}(dest)
	}
	// actual copy
	err := StreamDataToFile(r, dest)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("unable to stream data to file %q", dest))
	}
	if qemu {
		glog.V(Vadmin).Infoln("converting qcow2 image")
		err = image.ConvertQcow2ToRaw(dest, out)
		if err != nil {
			return errors.WithMessage(err, "unable to copy image")
		}
	}
	return nil
}

// Return a random temp path with the `src` basename as the prefix and preserving the extension.
// Eg. "/tmp/disk1d729566c74d1003.img".
func randTmpName(src string) string {
	ext := filepath.Ext(src)
	base := filepath.Base(src)
	base = base[:len(base)-len(ext)] // exclude extension
	randName := make([]byte, 8)
	rand.Read(randName)
	return filepath.Join(os.TempDir(), base+hex.EncodeToString(randName)+ext)
}

// parseDataPath only used for debugging
func (d dataStream) parseDataPath() (string, string) {
	pathSlice := strings.Split(strings.Trim(d.Url.EscapedPath(), "/"), "/")
	glog.V(Vdebug).Infof("parseDataPath: url path: %v", pathSlice)
	return pathSlice[0], strings.Join(pathSlice[1:], "/")
}
