package operatormetrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Summary struct {
	prometheus.Summary

	metricOpts  MetricOpts
	summaryOpts prometheus.SummaryOpts
}

var _ Metric = &Summary{}

// NewSummary creates a new Summary. The Summary must be registered with the
// Prometheus registry through RegisterMetrics.
func NewSummary(metricOpts MetricOpts, summaryOpts prometheus.SummaryOpts) *Summary {
	return &Summary{
		Summary:     prometheus.NewSummary(makePrometheusSummaryOpts(metricOpts, summaryOpts)),
		metricOpts:  metricOpts,
		summaryOpts: summaryOpts,
	}
}

func makePrometheusSummaryOpts(metricOpts MetricOpts, summaryOpts prometheus.SummaryOpts) prometheus.SummaryOpts {
	summaryOpts.Name = metricOpts.Name
	summaryOpts.Help = metricOpts.Help
	summaryOpts.ConstLabels = metricOpts.ConstLabels
	return summaryOpts
}

func (c *Summary) GetOpts() MetricOpts {
	return c.metricOpts
}

func (c *Summary) GetSummaryOpts() prometheus.SummaryOpts {
	return c.summaryOpts
}

func (c *Summary) GetType() MetricType {
	return SummaryType
}

func (c *Summary) GetBaseType() MetricType {
	return SummaryType
}

func (c *Summary) GetCollector() prometheus.Collector {
	return c.Summary
}
