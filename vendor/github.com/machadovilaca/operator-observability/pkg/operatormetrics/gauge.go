package operatormetrics

import "github.com/prometheus/client_golang/prometheus"

type Gauge struct {
	prometheus.Gauge

	metricOpts MetricOpts
}

var _ Metric = &Gauge{}

// NewGauge creates a new Gauge. The Gauge must be registered with the
// Prometheus registry through RegisterMetrics.
func NewGauge(metricOpts MetricOpts) *Gauge {
	return &Gauge{
		Gauge:      prometheus.NewGauge(prometheus.GaugeOpts(convertOpts(metricOpts))),
		metricOpts: metricOpts,
	}
}

func (c *Gauge) GetOpts() MetricOpts {
	return c.metricOpts
}

func (c *Gauge) GetType() MetricType {
	return GaugeType
}

func (c *Gauge) getCollector() prometheus.Collector {
	return c.Gauge
}
