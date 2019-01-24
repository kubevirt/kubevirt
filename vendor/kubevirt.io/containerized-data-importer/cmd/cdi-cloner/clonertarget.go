package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/golang/glog"

	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/util"

	dto "github.com/prometheus/client_model/go"

	"github.com/prometheus/client_golang/prometheus"
)

type prometheusProgressReader struct {
	util.CountingReader
	total int64
}

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
	flag.Parse()
	prometheus.MustRegister(progress)
	ownerUID, _ = util.ParseEnvVar(common.OwnerUID, false)
}

func main() {
	defer glog.Flush()
	glog.V(1).Infoln("Starting cloner target")

	certsDirectory, err := ioutil.TempDir("", "certsdir")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(certsDirectory)
	util.StartPrometheusEndpoint(certsDirectory)

	if *namedPipe == "nopipedir" {
		glog.Errorf("%+v", fmt.Errorf("Missed named pipe flag"))
		os.Exit(1)
	}

	total, err := collectTotalSize()
	if err != nil {
		glog.Errorf("%+v", err)
		os.Exit(1)
	}

	//re-open pipe with fresh start.
	out, err := os.OpenFile(*namedPipe, os.O_RDONLY, 0600)
	if err != nil {
		glog.Errorf("%+v", err)
		os.Exit(1)
	}
	defer out.Close()

	promReader := &prometheusProgressReader{
		CountingReader: util.CountingReader{
			Reader:  out,
			Current: 0,
		},
		total: total,
	}

	// Start the progress update thread.
	go promReader.timedUpdateProgress()

	err = util.UnArchiveTar(promReader, ".")
	if err != nil {
		glog.Errorf("%+v", err)
		os.Exit(1)
	}

	glog.V(1).Infoln("clone complete")
}

func collectTotalSize() (int64, error) {
	glog.V(3).Infoln("Reading total size")
	out, err := os.OpenFile(*namedPipe, os.O_RDONLY, 0600)
	if err != nil {
		return int64(-1), err
	}
	defer out.Close()
	return readTotal(out), nil
}

func (r *prometheusProgressReader) timedUpdateProgress() {
	for true {
		// Update every second.
		time.Sleep(time.Second)
		r.updateProgress()
	}
}

func (r *prometheusProgressReader) updateProgress() {
	if r.total > 0 {
		currentProgress := float64(r.Current) / float64(r.total) * 100.0
		metric := &dto.Metric{}
		progress.WithLabelValues(ownerUID).Write(metric)
		if currentProgress > *metric.Counter.Value {
			progress.WithLabelValues(ownerUID).Add(currentProgress - *metric.Counter.Value)
		}
		glog.V(1).Infoln(fmt.Sprintf("%.2f", currentProgress))
	}
}

// read total file size from reader, and return the value as an int64
func readTotal(r io.Reader) int64 {
	totalScanner := bufio.NewScanner(r)
	if !totalScanner.Scan() {
		glog.Errorf("Unable to determine length of file")
		return -1
	}
	totalText := totalScanner.Text()
	total, err := strconv.ParseInt(totalText, 10, 64)
	if err != nil {
		glog.Errorf("%+v", err)
		return -1
	}
	glog.V(1).Infoln(fmt.Sprintf("total size: %s", totalText))
	return total
}
