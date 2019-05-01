package importer

import (
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	minio "github.com/minio/minio-go"
	"github.com/pkg/errors"

	"k8s.io/klog"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

// S3Client is the interface to the used S3 client.
type S3Client interface {
	GetObject(bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
}

// may be overridden in tests
var newClientFunc = getS3Client

// S3DataSource is the struct containing the information needed to import from an S3 data source.
// Sequence of phases:
// 1. Info -> Transfer
// 2. Transfer -> Process
// 3. Process -> Convert
type S3DataSource struct {
	// S3 end point
	ep *url.URL
	// User name
	accessKey string
	// Password
	secKey string
	// Reader
	s3Reader io.ReadCloser
	// stack of readers
	readers *FormatReaders
	// The image file in scratch space.
	url *url.URL
}

// NewS3DataSource creates a new instance of the S3DataSource
func NewS3DataSource(endpoint, accessKey, secKey string) (*S3DataSource, error) {
	ep, err := ParseEndpoint(endpoint)
	if err != nil {
		return nil, errors.Wrapf(err, fmt.Sprintf("unable to parse endpoint %q", endpoint))
	}
	s3Reader, err := createS3Reader(ep, accessKey, secKey)
	if err != nil {
		return nil, err
	}
	return &S3DataSource{
		ep:        ep,
		accessKey: accessKey,
		secKey:    secKey,
		s3Reader:  s3Reader,
	}, nil
}

// Info is called to get initial information about the data.
func (sd *S3DataSource) Info() (ProcessingPhase, error) {
	var err error
	sd.readers, err = NewFormatReaders(sd.s3Reader, cdiv1.DataVolumeKubeVirt)
	if err != nil {
		klog.Errorf("Error creating readers: %v", err)
		return ProcessingPhaseError, err
	}
	if !sd.readers.Convert {
		// Downloading a raw file, we can write that directly to the target.
		return ProcessingPhaseTransferDataFile, nil
	}

	return ProcessingPhaseTransferScratch, nil
}

// Transfer is called to transfer the data from the source to a temporary location.
func (sd *S3DataSource) Transfer(path string) (ProcessingPhase, error) {
	if util.GetAvailableSpace(path) <= int64(0) {
		//Path provided is invalid.
		return ProcessingPhaseError, ErrInvalidPath
	}
	file := filepath.Join(path, tempFile)
	err := StreamDataToFile(sd.readers.TopReader(), file)
	if err != nil {
		return ProcessingPhaseError, err
	}
	// If streaming succeeded, then parsing the file into URL will also succeed, no need to check error status
	sd.url, _ = url.Parse(file)
	return ProcessingPhaseProcess, nil
}

// TransferFile is called to transfer the data from the source to the passed in file.
func (sd *S3DataSource) TransferFile(fileName string) (ProcessingPhase, error) {
	err := StreamDataToFile(sd.readers.TopReader(), fileName)
	if err != nil {
		return ProcessingPhaseError, err
	}
	return ProcessingPhaseResize, nil
}

// Process is called to do any special processing before giving the url to the data back to the processor
func (sd *S3DataSource) Process() (ProcessingPhase, error) {
	return ProcessingPhaseConvert, nil
}

// GetURL returns the url that the data processor can use when converting the data.
func (sd *S3DataSource) GetURL() *url.URL {
	return sd.url
}

// Close closes any readers or other open resources.
func (sd *S3DataSource) Close() error {
	var err error
	if sd.readers != nil {
		err = sd.readers.Close()
	}
	return err
}

func createS3Reader(ep *url.URL, accessKey, secKey string) (io.ReadCloser, error) {
	klog.V(3).Infoln("Using S3 client to get data")
	bucket := ep.Host
	object := strings.Trim(ep.Path, "/")
	mc, err := newClientFunc(accessKey, secKey, false)
	if err != nil {
		return nil, errors.Wrapf(err, "could not build minio client for %q", ep.Host)
	}
	klog.V(2).Infof("Attempting to get object %q via S3 client\n", ep.String())
	objectReader, err := mc.GetObject(bucket, object, minio.GetObjectOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "could not get s3 object: \"%s/%s\"", bucket, object)
	}
	return objectReader, nil
}

func getS3Client(accessKey, secKey string, secure bool) (S3Client, error) {
	return minio.NewV4(common.ImporterS3Host, accessKey, secKey, secure)
}
