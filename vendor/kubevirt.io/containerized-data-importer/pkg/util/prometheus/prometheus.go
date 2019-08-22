package prometheus

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"k8s.io/klog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"kubevirt.io/containerized-data-importer/pkg/keys"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

// ProgressReader is a counting reader that reports progress to prometheus.
type ProgressReader struct {
	util.CountingReader
	total    uint64
	progress *prometheus.CounterVec
	ownerUID string
}

// NewProgressReader creates a new instance of a prometheus updating progress reader.
func NewProgressReader(r io.ReadCloser, total uint64, progress *prometheus.CounterVec, ownerUID string) *ProgressReader {
	promReader := &ProgressReader{
		CountingReader: util.CountingReader{
			Reader:  r,
			Current: 0,
		},
		total:    total,
		progress: progress,
		ownerUID: ownerUID,
	}

	return promReader
}

// StartTimedUpdate starts the update timer to automatically update every second.
func (r *ProgressReader) StartTimedUpdate() {
	// Start the progress update thread.
	go r.timedUpdateProgress()
}

func (r *ProgressReader) timedUpdateProgress() {
	cont := true
	for cont {
		// Update every second.
		time.Sleep(time.Second)
		cont = r.updateProgress()
	}
}

func (r *ProgressReader) updateProgress() bool {
	if r.total > 0 {
		currentProgress := 100.0
		if !r.Done && r.Current < r.total {
			currentProgress = float64(r.Current) / float64(r.total) * 100.0
		}
		metric := &dto.Metric{}
		r.progress.WithLabelValues(r.ownerUID).Write(metric)
		if currentProgress > *metric.Counter.Value {
			r.progress.WithLabelValues(r.ownerUID).Add(currentProgress - *metric.Counter.Value)
		}
		klog.V(1).Infoln(fmt.Sprintf("%.2f", currentProgress))
		return !r.Done
	}
	return false
}

// StartPrometheusEndpoint starts an http server providing a prometheus endpoint using the passed
// in directory to store the self signed certificates that will be generated before starting the
// http server.
func StartPrometheusEndpoint(certsDirectory string) {
	keyFile, certFile, err := keys.GenerateSelfSignedCert(certsDirectory, "cloner_target", "pod")
	if err != nil {
		return
	}
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServeTLS(":8443", certFile, keyFile, nil); err != nil {
			return
		}
	}()
}
