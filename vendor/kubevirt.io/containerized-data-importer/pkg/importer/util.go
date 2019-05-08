package importer

import (
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"k8s.io/klog"

	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

// ParseEndpoint parses the required endpoint and return the url struct.
func ParseEndpoint(endpt string) (*url.URL, error) {
	if endpt == "" {
		// Because we are passing false, we won't decode anything and there is no way to error.
		endpt, _ = util.ParseEnvVar(common.ImporterEndpoint, false)
		if endpt == "" {
			return nil, errors.Errorf("endpoint %q is missing or blank", common.ImporterEndpoint)
		}
	}
	return url.Parse(endpt)
}

// CleanDir cleans the contents of a directory including its sub directories, but does NOT remove the
// directory itself.
func CleanDir(dest string) error {
	dir, err := ioutil.ReadDir(dest)
	if err != nil {
		klog.Errorf("Unable read directory to clean: %s, %v", dest, err)
		return err
	}
	for _, d := range dir {
		klog.V(1).Infoln("deleting file: " + filepath.Join(dest, d.Name()))
		err = os.RemoveAll(filepath.Join(dest, d.Name()))
		if err != nil {
			klog.Errorf("Unable to delete file: %s, %v", filepath.Join(dest, d.Name()), err)
			return err
		}
	}
	return nil
}
