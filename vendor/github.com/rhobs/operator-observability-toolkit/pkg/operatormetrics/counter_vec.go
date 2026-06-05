package operatormetrics

import "github.com/prometheus/client_golang/prometheus"

type CounterVec struct {
	prometheus.CounterVec

	metricOpts MetricOpts
}

var _ Metric = &CounterVec{}

// NewCounterVec creates a new CounterVec. The CounterVec must be registered
// with the Prometheus registry through RegisterMetrics.
func NewCounterVec(metricOpts MetricOpts, labels []string) *CounterVec {
	metricOpts.labels = labels

	return &CounterVec{
		CounterVec: *prometheus.NewCounterVec(prometheus.CounterOpts(convertOpts(metricOpts)), labels),
		metricOpts: metricOpts,
	}
}

func (c *CounterVec) GetOpts() MetricOpts {
	return c.metricOpts
}

func (c *CounterVec) GetType() MetricType {
	return CounterVecType
}

func (c *CounterVec) GetBaseType() MetricType {
	return CounterType
}

func (c *CounterVec) GetCollector() prometheus.Collector {
	return c.CounterVec
}
