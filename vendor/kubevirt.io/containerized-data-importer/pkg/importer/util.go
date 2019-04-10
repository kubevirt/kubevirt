package importer

import (
	"io"
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

//MoveFile - moves file
func MoveFile(src, dst string) error {
	klog.Infof("Moving %s to %s", src, dst)
	err := os.Rename(src, dst)
	if err != nil {
		klog.Errorf(err.Error(), "Failed moving %s to %s, are they in the same lun?")
	}
	return err
}

// StreamDataToFile provides a function to stream the specified io.Reader to the specified local file
func StreamDataToFile(dataReader io.Reader, filePath string) error {
	// Attempt to create the file with name filePath.  If it exists, fail.
	outFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.ModePerm)
	defer outFile.Close()
	if err != nil {
		return errors.Wrapf(err, "could not open file %q", filePath)
	}
	klog.V(1).Infof("begin import...\n")
	if _, err = io.Copy(outFile, dataReader); err != nil {
		klog.Errorf("Unable to write file from dataReader: %v\n", err)
		os.Remove(outFile.Name())
		return errors.Wrapf(err, "unable to write to file")
	}
	return nil
}

// CleanDir cleans the contents of a directory including its sub directories, but does NOT remove the
// directory itself.
func CleanDir(dest string) error {
	dir, err := ioutil.ReadDir(dest)
	if err != nil {
		return err
	}
	for _, d := range dir {
		klog.V(3).Infoln("deleting file: " + filepath.Join(dest, d.Name()))
		err = os.RemoveAll(filepath.Join(dest, d.Name()))
		if err != nil {
			return err
		}
	}
	return nil
}
