package operatormetrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Histogram struct {
	prometheus.Histogram

	metricOpts    MetricOpts
	histogramOpts prometheus.HistogramOpts
}

var _ Metric = &Histogram{}

// NewHistogram creates a new Histogram. The Histogram must be registered with the
// Prometheus registry through RegisterMetrics.
func NewHistogram(metricOpts MetricOpts, histogramOpts prometheus.HistogramOpts) *Histogram {
	return &Histogram{
		Histogram:     prometheus.NewHistogram(makePrometheusHistogramOpts(metricOpts, histogramOpts)),
		metricOpts:    metricOpts,
		histogramOpts: histogramOpts,
	}
}

func makePrometheusHistogramOpts(metricOpts MetricOpts, histogramOpts prometheus.HistogramOpts) prometheus.HistogramOpts {
	histogramOpts.Name = metricOpts.Name
	histogramOpts.Help = metricOpts.Help
	histogramOpts.ConstLabels = metricOpts.ConstLabels
	return histogramOpts
}

func (c *Histogram) GetOpts() MetricOpts {
	return c.metricOpts
}

func (c *Histogram) GetHistogramOpts() prometheus.HistogramOpts {
	return c.histogramOpts
}

func (c *Histogram) GetType() MetricType {
	return HistogramType
}

func (c *Histogram) GetBaseType() MetricType {
	return HistogramType
}

func (c *Histogram) GetCollector() prometheus.Collector {
	return c.Histogram
}
