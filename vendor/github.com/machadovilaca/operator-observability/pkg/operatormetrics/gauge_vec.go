package operatormetrics

import "github.com/prometheus/client_golang/prometheus"

type GaugeVec struct {
	prometheus.GaugeVec

	metricOpts MetricOpts
}

var _ Metric = &GaugeVec{}

// NewGaugeVec creates a new GaugeVec. The GaugeVec must be registered
// with the Prometheus registry through RegisterMetrics.
func NewGaugeVec(metricOpts MetricOpts, labels []string) *GaugeVec {
	return &GaugeVec{
		GaugeVec:   *prometheus.NewGaugeVec(prometheus.GaugeOpts(convertOpts(metricOpts)), labels),
		metricOpts: metricOpts,
	}
}

func (c *GaugeVec) GetOpts() MetricOpts {
	return c.metricOpts
}

func (c *GaugeVec) GetType() MetricType {
	return GaugeVecType
}

func (c *GaugeVec) getCollector() prometheus.Collector {
	return c.GaugeVec
}
