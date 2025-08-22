package operatormetrics

import "github.com/prometheus/client_golang/prometheus"

type HistogramVec struct {
	prometheus.HistogramVec

	metricOpts    MetricOpts
	histogramOpts prometheus.HistogramOpts
}

var _ Metric = &HistogramVec{}

// NewHistogramVec creates a new HistogramVec. The HistogramVec must be
// registered with the Prometheus registry through RegisterMetrics.
func NewHistogramVec(metricOpts MetricOpts, histogramOpts prometheus.HistogramOpts, labels []string) *HistogramVec {
	return &HistogramVec{
		HistogramVec:  *prometheus.NewHistogramVec(makePrometheusHistogramOpts(metricOpts, histogramOpts), labels),
		metricOpts:    metricOpts,
		histogramOpts: histogramOpts,
	}
}

func (c *HistogramVec) GetOpts() MetricOpts {
	return c.metricOpts
}

func (c *HistogramVec) GetHistogramOpts() prometheus.HistogramOpts {
	return c.histogramOpts
}

func (c *HistogramVec) GetType() MetricType {
	return HistogramVecType
}

func (c *HistogramVec) GetBaseType() MetricType {
	return HistogramType
}

func (c *HistogramVec) GetCollector() prometheus.Collector {
	return c.HistogramVec
}
