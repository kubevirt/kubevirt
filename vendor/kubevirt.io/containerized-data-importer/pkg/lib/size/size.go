package size

import (
	"encoding/hex"
	"io"
	"strconv"

	"github.com/pkg/errors"
	. "kubevirt.io/containerized-data-importer/pkg/importer"
)

// Return the size in bytes of the provided endpoint. If the endpoint was archived, compressed or converted to
// qcow2 the original image size is returned.
func ImageSize(endpoint, accessKey, secKey string) (int64, error) {
	ds, err := NewDataStream(endpoint, accessKey, secKey)
	if err != nil {
		return -1, errors.Wrapf(err, "unable to create data stream")
	}
	defer ds.Close()
	return Size(ds.Readers, ds.Qemu)
}

// Return the size of the endpoint corresponding to the passed-in reader.
func Size(readers []Reader, qemu bool) (int64, error) {
	r := readers[len(readers)-1] // top-level reader
	if qemu {
		return qemuSize(r.Rdr)
	}
	switch r.RdrType {
	case RdrFile:
		return noSupport()
	case RdrGz:
		return noSupport()
	case RdrHttp:
		return httpSize(r.Rdr)
	case RdrS3:
		return noSupport()
	case RdrTar:
		return noSupport()
	case RdrXz:
		return noSupport()

	default:
		return 0, errors.Errorf("internal error: unsupported reader type %+v", r)
	}
}

func noSupport() (int64, error) {
	return -1, errors.New("unsupported format")
}

func httpSize(r io.Reader) (int64, error) {

	return 0, nil
}

// Return the original (virtual) size of the file represented by the passed-in reader. This reader is assumed
// to be a multi-reader, see importer.constructReaders(). ReadFull is used to read from the reader into a small
// buffer where the qemu header is examined.
// Note: qemu header format defines the size offset:
//   24 - 31:   Virtual disk size in bytes
//   See: https://github.com/zchee/go-qcow2/blob/master/docs/specification.md#header
func qemuSize(r io.Reader) (int64, error) {
	const (
		sizeOffset = 24 // bytes
		fieldSize  = 8  // bytes
		bufSize    = sizeOffset + fieldSize
	)

	buf := make([]byte, bufSize)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return 0, errors.Wrapf(err, "qemu ReadFull error")
	}

	// seek to size field and extract size
	s := hex.EncodeToString(buf[sizeOffset:])
	size, err := strconv.ParseInt(s, 16, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "cannot parse qemu size")
	}
	return size, nil
}
