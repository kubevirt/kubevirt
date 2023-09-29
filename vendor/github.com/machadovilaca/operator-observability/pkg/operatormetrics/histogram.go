package operatormetrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Histogram struct {
	prometheus.Histogram

	metricOpts    MetricOpts
	histogramOpts HistogramOpts
}

var _ Metric = &Histogram{}

type HistogramOpts struct {
	Buckets                         []float64
	NativeHistogramBucketFactor     float64
	NativeHistogramZeroThreshold    float64
	NativeHistogramMaxBucketNumber  uint32
	NativeHistogramMinResetDuration time.Duration
	NativeHistogramMaxZeroThreshold float64
}

// NewHistogram creates a new Histogram. The Histogram must be registered with the
// Prometheus registry through RegisterMetrics.
func NewHistogram(metricOpts MetricOpts, histogramOpts HistogramOpts) *Histogram {
	return &Histogram{
		Histogram:     prometheus.NewHistogram(makePrometheusHistogramOpts(metricOpts, histogramOpts)),
		metricOpts:    metricOpts,
		histogramOpts: histogramOpts,
	}
}

func makePrometheusHistogramOpts(metricOpts MetricOpts, histogramOpts HistogramOpts) prometheus.HistogramOpts {
	return prometheus.HistogramOpts{
		Name:                            metricOpts.Name,
		Help:                            metricOpts.Help,
		ConstLabels:                     metricOpts.ConstLabels,
		NativeHistogramBucketFactor:     histogramOpts.NativeHistogramBucketFactor,
		NativeHistogramZeroThreshold:    histogramOpts.NativeHistogramZeroThreshold,
		NativeHistogramMaxBucketNumber:  histogramOpts.NativeHistogramMaxBucketNumber,
		NativeHistogramMinResetDuration: histogramOpts.NativeHistogramMinResetDuration,
		NativeHistogramMaxZeroThreshold: histogramOpts.NativeHistogramMaxZeroThreshold,
	}
}

func (c *Histogram) GetOpts() MetricOpts {
	return c.metricOpts
}

func (c *Histogram) GetHistogramOpts() HistogramOpts {
	return c.histogramOpts
}

func (c *Histogram) GetType() MetricType {
	return HistogramType
}

func (c *Histogram) getCollector() prometheus.Collector {
	return c.Histogram
}
