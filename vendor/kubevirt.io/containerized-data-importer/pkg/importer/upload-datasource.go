package importer

import (
	"io"
	"net/url"
	"path/filepath"

	"k8s.io/klog"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

// UploadDataSource contains all the information need to upload data into a data volume.
// Sequence of phases:
// 1a. ProcessingPhaseInfo -> ProcessingPhaseTransferScratch (In Info phase the format readers are configured) In case the readers don't contain a raw file.
// 1b. ProcessingPhaseInfo -> ProcessingPhaseTransferDataFile, in the case the readers contain a raw file.
// 2a. ProcessingPhaseTransferScratch -> ProcessingPhaseProcess
// 2b. ProcessingPhaseTransferDataFile -> ProcessingPhaseResize
// 3. ProcessingPhaseProcess -> ProcessingPhaseConvert
type UploadDataSource struct {
	// Data strean
	stream io.ReadCloser
	// stack of readers
	readers *FormatReaders
	// url to a file in scratch space.
	url *url.URL
}

// NewUploadDataSource creates a new instance of an UploadDataSource
func NewUploadDataSource(stream io.ReadCloser) *UploadDataSource {
	return &UploadDataSource{
		stream: stream,
	}
}

// Info is called to get initial information about the data.
func (ud *UploadDataSource) Info() (ProcessingPhase, error) {
	var err error
	// Hardcoded to only accept kubevirt content type.
	ud.readers, err = NewFormatReaders(ud.stream, cdiv1.DataVolumeKubeVirt)
	if err != nil {
		klog.Errorf("Error creating readers: %v", err)
		return ProcessingPhaseError, err
	}
	if !ud.readers.Convert {
		// Uploading a raw file, we can write that directly to the target.
		return ProcessingPhaseTransferDataFile, nil
	}
	return ProcessingPhaseTransferScratch, nil
}

// Transfer is called to transfer the data from the source to the passed in path.
func (ud *UploadDataSource) Transfer(path string) (ProcessingPhase, error) {
	if util.GetAvailableSpace(path) <= int64(0) {
		//Path provided is invalid.
		return ProcessingPhaseError, ErrInvalidPath
	}
	file := filepath.Join(path, tempFile)
	err := StreamDataToFile(ud.readers.TopReader(), file)
	if err != nil {
		return ProcessingPhaseError, err
	}
	// If we successfully wrote to the file, then the parse will succeed.
	ud.url, _ = url.Parse(file)
	return ProcessingPhaseProcess, nil
}

// TransferFile is called to transfer the data from the source to the passed in file.
func (ud *UploadDataSource) TransferFile(fileName string) (ProcessingPhase, error) {
	err := StreamDataToFile(ud.readers.TopReader(), fileName)
	if err != nil {
		return ProcessingPhaseError, err
	}
	return ProcessingPhaseResize, nil
}

// Process is called to do any special processing before giving the url to the data back to the processor
func (ud *UploadDataSource) Process() (ProcessingPhase, error) {
	return ProcessingPhaseConvert, nil
}

// GetURL returns the url that the data processor can use when converting the data.
func (ud *UploadDataSource) GetURL() *url.URL {
	return ud.url
}

// Close closes any readers or other open resources.
func (ud *UploadDataSource) Close() error {
	if ud.stream != nil {
		return ud.stream.Close()
	}
	return nil
}
