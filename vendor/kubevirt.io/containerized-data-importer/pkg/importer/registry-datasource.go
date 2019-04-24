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
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"k8s.io/klog"

	"kubevirt.io/containerized-data-importer/pkg/image"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

const (
	//containerDiskImageDir - Expected disk image location in container image as described in
	//https://github.com/kubevirt/kubevirt/blob/master/docs/container-register-disks.md
	containerDiskImageDir = "disk"
)

// RegistryDataSource is the struct containing the information needed to import from a registry data source.
// Sequence of phases:
// 1. Info -> Transfer
// 2. Transfer -> Process
// 3. Process -> Convert (In the process phase the container image layers are extracted, and the url is pointed to the file determined to be the disk image)
type RegistryDataSource struct {
	endpoint    string
	accessKey   string
	secKey      string
	certDir     string
	insecureTLS bool
	imageDir    string
	//The discovered image file in scratch space.
	url *url.URL
}

// NewRegistryDataSource creates a new instance of the Registry Data Source.
func NewRegistryDataSource(endpoint, accessKey, secKey, certDir string, insecureTLS bool) *RegistryDataSource {
	return &RegistryDataSource{
		endpoint:    endpoint,
		accessKey:   accessKey,
		secKey:      secKey,
		certDir:     certDir,
		insecureTLS: insecureTLS,
	}
}

// Info is called to get initial information about the data. No information available for registry currently.
func (rd *RegistryDataSource) Info() (ProcessingPhase, error) {
	return ProcessingPhaseTransferScratch, nil
}

// Transfer is called to transfer the data from the source registry to a temporary location.
func (rd *RegistryDataSource) Transfer(path string) (ProcessingPhase, error) {
	if util.GetAvailableSpace(path) <= int64(0) {
		// Path provided is invalid.
		return ProcessingPhaseError, ErrInvalidPath
	}
	rd.imageDir = filepath.Join(path, containerDiskImageDir)

	klog.V(1).Infof("Copying registry image to scratch space.")
	err := image.CopyRegistryImage(rd.endpoint, path, containerDiskImageDir, rd.accessKey, rd.secKey, rd.certDir, rd.insecureTLS)
	if err != nil {
		return ProcessingPhaseError, errors.Wrapf(err, "Failed to read registry image")
	}

	return ProcessingPhaseProcess, nil
}

// TransferFile is called to transfer the data from the source to the passed in file.
func (rd *RegistryDataSource) TransferFile(fileName string) (ProcessingPhase, error) {
	return ProcessingPhaseError, errors.New("Transferfile should not be called")
}

// Process is called to do any special processing before giving the url to the data back to the processor
func (rd *RegistryDataSource) Process() (ProcessingPhase, error) {
	imageFile, err := getImageFileName(rd.imageDir)
	if err != nil {
		return ProcessingPhaseError, errors.Wrapf(err, "Cannot locate image file")
	}

	// imageFile and rd.imageDir are both valid, thus the Join will be valid, and the parse will work, no need to check for parse errors
	rd.url, _ = url.Parse(filepath.Join(rd.imageDir, imageFile))
	klog.V(3).Infof("Successfully found file. VM disk image filename is %s", rd.url.String())
	return ProcessingPhaseConvert, nil
}

// GetURL returns the url that the data processor can use when converting the data.
func (rd *RegistryDataSource) GetURL() *url.URL {
	return rd.url
}

// Close closes any readers or other open resources.
func (rd *RegistryDataSource) Close() error {
	// No-op, no open readers
	return nil
}

func getImageFileName(dir string) (string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		klog.Errorf("image directory does not exist")
		return "", errors.Errorf("image directory does not exist")
	}

	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		klog.Errorf("Error reading directory")
		return "", errors.Wrapf(err, "image file does not exist in image directory")
	}

	if len(entries) == 0 {
		klog.Errorf("image file does not exist in image directory - directory is empty ")
		return "", errors.New("image file does not exist in image directory - directory is empty")
	}

	fileinfo := entries[len(entries)-1]
	if fileinfo.IsDir() {
		klog.Errorf("image file does not exist in image directory contains another directory ")
		return "", errors.New("image directory contains another directory")
	}

	filename := fileinfo.Name()

	if len(strings.TrimSpace(filename)) == 0 {
		klog.Errorf("image file does not exist in image directory - file has no name ")
		return "", errors.New("image file does has no name")
	}

	klog.V(1).Infof("VM disk image filename is %s", filename)

	return filename, nil
}
