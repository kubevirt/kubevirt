package operatormetrics

import "github.com/prometheus/client_golang/prometheus"

type Counter struct {
	prometheus.Counter

	metricOpts MetricOpts
}

var _ Metric = &Counter{}

// NewCounter creates a new Counter. The Counter must be registered with the
// Prometheus registry through RegisterMetrics.
func NewCounter(metricOpts MetricOpts) *Counter {
	return &Counter{
		Counter:    prometheus.NewCounter(prometheus.CounterOpts(convertOpts(metricOpts))),
		metricOpts: metricOpts,
	}
}

func (c *Counter) GetOpts() MetricOpts {
	return c.metricOpts
}

func (c *Counter) GetType() MetricType {
	return CounterType
}

func (c *Counter) GetBaseType() MetricType {
	return CounterType
}

func (c *Counter) GetCollector() prometheus.Collector {
	return c.Counter
}
