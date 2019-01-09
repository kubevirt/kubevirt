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

package image

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/system"
	"kubevirt.io/containerized-data-importer/pkg/util"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	networkTimeoutSecs = 3600    //max is 10000
	maxMemory          = 1 << 30 //value from OpenStack Nova
	maxCPUSecs         = 30      //value from OpenStack Nova
	matcherString      = "\\((\\d?\\d\\.\\d\\d)\\/100%\\)"
)

// ImgInfo contains the virtual image information.
type ImgInfo struct {
	// Format contains the format of the image
	Format string `json:"format"`
	// BackingFile is the file name of the backing file
	BackingFile string `json:"backing-filename"`
	// VirtualSize is the disk size of the image which will be read by vm
	VirtualSize int64 `json:"virtual-size"`
	// ActualSize is the size of the qcow2 image
	ActualSize int64 `json:"actual-size"`
}

// QEMUOperations defines the interface for executing qemu subprocesses
type QEMUOperations interface {
	ConvertQcow2ToRaw(string, string) error
	ConvertQcow2ToRawStream(*url.URL, string) error
	Resize(string, resource.Quantity) error
	Info(string) (*ImgInfo, error)
	Validate(string, string) error
	CreateBlankImage(dest string, size resource.Quantity) error
}

type qemuOperations struct{}

var (
	qemuExecFunction = system.ExecWithLimits
	qemuLimits       = &system.ProcessLimitValues{AddressSpaceLimit: maxMemory, CPUTimeLimit: maxCPUSecs}
	qemuIterface     = NewQEMUOperations()
	re               = regexp.MustCompile(matcherString)

	progress = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "import_progress",
			Help: "The import progress in percentage",
		},
		[]string{"ownerUID"},
	)
	ownerUID string
)

func init() {
	prometheus.MustRegister(progress)
	ownerUID, _ = util.ParseEnvVar(common.OwnerUID, false)
}

// NewQEMUOperations returns the default implementation of QEMUOperations
func NewQEMUOperations() QEMUOperations {
	return &qemuOperations{}
}

func (o *qemuOperations) ConvertQcow2ToRaw(src, dest string) error {
	_, err := qemuExecFunction(qemuLimits, nil, "qemu-img", "convert", "-p", "-f", "qcow2", "-O", "raw", src, dest)
	if err != nil {
		os.Remove(dest)
		return errors.Wrap(err, "could not convert local qcow2 image to raw")
	}

	return nil
}

func (o *qemuOperations) ConvertQcow2ToRawStream(url *url.URL, dest string) error {
	jsonArg := fmt.Sprintf("json: {\"file.driver\": \"%s\", \"file.url\": \"%s\", \"file.timeout\": %d}", url.Scheme, url, networkTimeoutSecs)

	_, err := qemuExecFunction(qemuLimits, reportProgress, "qemu-img", "convert", "-p", "-f", "qcow2", "-O", "raw", jsonArg, dest)
	if err != nil {
		os.Remove(dest)
		return errors.Wrap(err, "could not stream/convert qcow2 image to raw")
	}

	return nil
}

// convertQuantityToQemuSize translates a quantity string into a Qemu compatible string.
func convertQuantityToQemuSize(size resource.Quantity) string {
	// size from k8s contains an upper case K, so we need to lower case it before we can pass it to qemu.
	// Below is the message from qemu that explains what it expects.
	// qemu-img: Parameter 'size' expects a non-negative number below 2^64 Optional suffix k, M, G, T, P or E means kilo-, mega-, giga-, tera-, peta-and exabytes, respectively.
	stringSize := strings.Replace(size.String(), "K", "k", -1)
	stringSize = strings.Replace(stringSize, "i", "", -1)
	return stringSize
}

func (o *qemuOperations) Resize(image string, size resource.Quantity) error {
	_, err := qemuExecFunction(qemuLimits, nil, "qemu-img", "resize", "-f", "raw", image, convertQuantityToQemuSize(size))
	if err != nil {
		return errors.Wrapf(err, "Error resizing image %s", image)
	}
	return nil
}

func (o *qemuOperations) Info(image string) (*ImgInfo, error) {
	output, err := qemuExecFunction(qemuLimits, nil, "qemu-img", "info", "--output=json", image)
	if err != nil {
		return nil, errors.Wrapf(err, "Error getting info on image %s", image)
	}
	var info ImgInfo
	err = json.Unmarshal(output, &info)
	if err != nil {
		glog.Errorf("Invalid JSON:\n%s\n", string(output))
		return nil, errors.Wrapf(err, "Invalid json for image %s", image)
	}
	return &info, nil
}

func (o *qemuOperations) Validate(image, format string) error {
	info, err := o.Info(image)
	if err != nil {
		return err
	}

	if info.Format != format {
		return errors.Errorf("Invalid format %s for image %s", info.Format, image)
	}

	if len(info.BackingFile) > 0 {
		return errors.Errorf("Image %s is invalid because it has backing file %s", image, info.BackingFile)
	}

	return nil
}

// ConvertQcow2ToRaw is a wrapper for qemu-img convert which takes a qcow2 file (specified by src) and converts
// it to a raw image (written to the provided dest file)
func ConvertQcow2ToRaw(src, dest string) error {
	return qemuIterface.ConvertQcow2ToRaw(src, dest)
}

// ConvertQcow2ToRawStream converts an http accessible qcow2 image to raw format without locally caching the qcow2 image
func ConvertQcow2ToRawStream(url *url.URL, dest string) error {
	return qemuIterface.ConvertQcow2ToRawStream(url, dest)
}

// Validate does basic validation of a qemu image
func Validate(image, format string) error {
	return qemuIterface.Validate(image, format)
}

func reportProgress(line string) {
	// (45.34/100%)
	matches := re.FindStringSubmatch(line)
	if len(matches) == 2 && ownerUID != "" {
		glog.V(1).Info(matches[1])
		// Don't need to check for an error, the regex made sure its a number we can parse.
		v, _ := strconv.ParseFloat(matches[1], 64)
		progress.WithLabelValues(ownerUID).Set(v)
	}
}

// CreateBlankImage creates empty raw image
func CreateBlankImage(dest string, size resource.Quantity) error {
	glog.V(1).Infof("creating raw image with size %s", size)
	return qemuIterface.CreateBlankImage(dest, size)
}

// CreateBlankImage creates a raw image with a given size
func (o *qemuOperations) CreateBlankImage(dest string, size resource.Quantity) error {
	glog.V(3).Infof("image size is %s", size.String())
	_, err := qemuExecFunction(qemuLimits, nil, "qemu-img", "create", "-f", "raw", dest, convertQuantityToQemuSize(size))
	if err != nil {
		os.Remove(dest)
		return errors.Wrap(err, fmt.Sprintf("could not create raw image with size %s in %s", size.String(), dest))
	}
	return nil
}
