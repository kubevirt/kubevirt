package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/util"
	prometheusutil "kubevirt.io/containerized-data-importer/pkg/util/prometheus"
)

const (
	maxSizeLength = 20
)

var (
	progress = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "clone_progress",
			Help: "The clone progress in percentage",
		},
		[]string{"ownerUID"},
	)
	ownerUID  string
	namedPipe *string
)

func init() {
	namedPipe = flag.String("pipedir", "nopipedir", "The name and directory of the named pipe to read from")
	klog.InitFlags(nil)
	flag.Parse()

	prometheus.MustRegister(progress)
	ownerUID, _ = util.ParseEnvVar(common.OwnerUID, false)
}

func main() {
	defer klog.Flush()
	klog.V(1).Infoln("Starting cloner target")
	certsDirectory, err := ioutil.TempDir("", "certsdir")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(certsDirectory)
	prometheusutil.StartPrometheusEndpoint(certsDirectory)

	if *namedPipe == "nopipedir" {
		klog.Errorf("%+v", fmt.Errorf("Missed named pipe flag"))
		os.Exit(1)
	}

	total, err := collectTotalSize()
	if err != nil {
		klog.Errorf("%+v", err)
		os.Exit(1)
	}
	klog.V(3).Infof("Size read: %d\n", total)

	//re-open pipe with fresh start.
	out, err := os.OpenFile(*namedPipe, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		klog.Errorf("%+v", err)
		os.Exit(1)
	}
	defer out.Close()

	promReader := prometheusutil.NewProgressReader(out, total, progress, ownerUID)
	promReader.StartTimedUpdate()
	volumeMode := v1.PersistentVolumeBlock
	if _, err := os.Stat(common.ImporterWriteBlockPath); os.IsNotExist(err) {
		volumeMode = v1.PersistentVolumeFilesystem
	}
	if volumeMode == v1.PersistentVolumeBlock {
		klog.V(3).Infoln("Writing data to block device")
		err = util.StreamDataToFile(promReader, common.ImporterWriteBlockPath)
	} else {
		klog.V(3).Infoln("Writing data to file system")
		err = util.UnArchiveTar(promReader, ".")
	}

	if err != nil {
		klog.Errorf("%+v", err)
		os.Exit(1)
	}
	klog.V(1).Infoln("clone complete")
}

func collectTotalSize() (uint64, error) {
	klog.V(3).Infoln("Reading total size")
	out, err := os.OpenFile(*namedPipe, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		return uint64(0), err
	}
	defer out.Close()
	return readTotal(out)
}

// read total file size from reader, and return the value as an int64
func readTotal(r io.Reader) (uint64, error) {
	b := make([]byte, 16)

	n, err := r.Read(b)
	if err != nil {
		klog.Errorf("%+v", err)
		return uint64(0), err
	}
	if n != len(b) {
		// Didn't read all 16 bytes..
		return uint64(0), errors.New("Didn't read all bytes for size header")
	}
	return strconv.ParseUint(string(b), 16, 64)
}
