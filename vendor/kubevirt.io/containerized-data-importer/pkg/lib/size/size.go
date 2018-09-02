package size

import (
	"github.com/pkg/errors"
	. "kubevirt.io/containerized-data-importer/pkg/importer"
)

// Return the size in bytes of the provided endpoint. If the endpoint was archived, compressed or
// converted to qcow2 the original image size is returned.
func Size(endpoint, accessKey, secKey string) (int64, error) {
	ds, err := NewDataStream(endpoint, accessKey, secKey)
	if err != nil {
		return 0, errors.Wrapf(err, "unable to create data stream")
	}
	defer ds.Close()
	return ds.Size, nil
}
