package importer

import (
	"io"
	"net/url"
	"os"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

// ParseEndpoint parses the required endpoint and return the url struct.
func ParseEndpoint(endpt string) (*url.URL, error) {
	var err error
	if endpt == "" {
		endpt, err = util.ParseEnvVar(common.ImporterEndpoint, false)
		if err != nil {
			return nil, err
		}
		if endpt == "" {
			return nil, errors.Errorf("endpoint %q is missing or blank", common.ImporterEndpoint)
		}
	}
	return url.Parse(endpt)
}

// StreamDataToFile provides a function to stream the specified io.Reader to the specified local file
func StreamDataToFile(dataReader io.Reader, filePath string) error {
	// Attempt to create the file with name filePath.  If it exists, fail.
	outFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.ModePerm)
	defer outFile.Close()
	if err != nil {
		return errors.Wrapf(err, "could not open file %q", filePath)
	}
	glog.V(1).Infof("begin import...\n")
	if _, err = io.Copy(outFile, dataReader); err != nil {
		os.Remove(outFile.Name())
		return errors.Wrapf(err, "unable to write to file")
	}
	return nil
}
