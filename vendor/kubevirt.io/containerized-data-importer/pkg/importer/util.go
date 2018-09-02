package importer

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	. "kubevirt.io/containerized-data-importer/pkg/common"
)

func ParseEnvVar(envVarName string, decode bool) (string, error) {
	value := os.Getenv(envVarName)
	if decode {
		v, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return "", errors.Errorf("error decoding environment variable %q", envVarName)
		}
		value = fmt.Sprintf("%s", v)
	}
	return value, nil
}

// Parse the required endpoint and return the url struct.
func ParseEndpoint(endpt string) (*url.URL, error) {
	var err error
	if endpt == "" {
		endpt, err = ParseEnvVar(IMPORTER_ENDPOINT, false)
		if err != nil {
			return nil, err
		}
		if endpt == "" {
			return nil, errors.Errorf("endpoint %q is missing or blank", IMPORTER_ENDPOINT)
		}
	}
	return url.Parse(endpt)
}

func StreamDataToFile(dataReader io.Reader, filePath string) error {
	// Attempt to create the file with name filePath.  If it exists, fail.
	outFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.ModePerm)
	defer outFile.Close()
	if err != nil {
		return errors.Wrapf(err, "could not open file %q", filePath)
	}
	glog.V(Vuser).Infof("begin import...\n")
	if _, err = io.Copy(outFile, dataReader); err != nil {
		os.Remove(outFile.Name())
		return errors.Wrapf(err, "unable to write to file")
	}
	return nil
}
