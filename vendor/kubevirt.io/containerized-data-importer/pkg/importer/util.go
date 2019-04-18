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
	if endpt == "" {
		// Because we are passing false, we won't decode anything and there is no way to error.
		endpt, _ = util.ParseEnvVar(common.ImporterEndpoint, false)
		if endpt == "" {
			return nil, errors.Errorf("endpoint %q is missing or blank", common.ImporterEndpoint)
		}
	}
	return url.Parse(endpt)
}

// StreamDataToFile provides a function to stream the specified io.Reader to the specified local file
func StreamDataToFile(r io.Reader, fileName string) error {
	var outFile *os.File
	var err error
	if util.GetAvailableSpaceBlock(fileName) < 0 {
		// Attempt to create the file with name filePath.  If it exists, fail.
		outFile, err = os.OpenFile(fileName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.ModePerm)
	} else {
		outFile, err = os.OpenFile(fileName, os.O_EXCL|os.O_WRONLY, os.ModePerm)
	}
	if err != nil {
		return errors.Wrapf(err, "could not open file %q", fileName)
	}
	defer outFile.Close()
	klog.V(1).Infof("Writing data...\n")
	if _, err = io.Copy(outFile, r); err != nil {
		klog.Errorf("Unable to write file from dataReader: %v\n", err)
		os.Remove(outFile.Name())
		return errors.Wrapf(err, "unable to write to file")
	}
	err = outFile.Sync()
	return err
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
