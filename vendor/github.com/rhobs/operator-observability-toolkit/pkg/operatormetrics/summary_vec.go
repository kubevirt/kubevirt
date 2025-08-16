package operatormetrics

import "github.com/prometheus/client_golang/prometheus"

type SummaryVec struct {
	prometheus.SummaryVec

	metricOpts  MetricOpts
	summaryOpts prometheus.SummaryOpts
}

var _ Metric = &SummaryVec{}

// NewSummaryVec creates a new SummaryVec. The SummaryVec must be
// registered with the Prometheus registry through RegisterMetrics.
func NewSummaryVec(metricOpts MetricOpts, summaryOpts prometheus.SummaryOpts, labels []string) *SummaryVec {
	return &SummaryVec{
		SummaryVec:  *prometheus.NewSummaryVec(makePrometheusSummaryOpts(metricOpts, summaryOpts), labels),
		metricOpts:  metricOpts,
		summaryOpts: summaryOpts,
	}
}

func (c *SummaryVec) GetOpts() MetricOpts {
	return c.metricOpts
}

func (c *SummaryVec) GetSummaryOpts() prometheus.SummaryOpts {
	return c.summaryOpts
}

func (c *SummaryVec) GetType() MetricType {
	return SummaryVecType
}

func (c *SummaryVec) GetBaseType() MetricType {
	return SummaryType
}

func (c *SummaryVec) GetCollector() prometheus.Collector {
	return c.SummaryVec
}
