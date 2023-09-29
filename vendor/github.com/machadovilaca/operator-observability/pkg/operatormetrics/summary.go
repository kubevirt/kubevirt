package operatormetrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Summary struct {
	prometheus.Summary

	metricOpts  MetricOpts
	summaryOpts SummaryOpts
}

var _ Metric = &Summary{}

type SummaryOpts struct {
	Objectives map[float64]float64
	MaxAge     time.Duration
	AgeBuckets uint32
	BufCap     uint32
}

// NewSummary creates a new Summary. The Summary must be registered with the
// Prometheus registry through RegisterMetrics.
func NewSummary(metricOpts MetricOpts, summaryOpts SummaryOpts) *Summary {
	return &Summary{
		Summary:     prometheus.NewSummary(makePrometheusSummaryOpts(metricOpts, summaryOpts)),
		metricOpts:  metricOpts,
		summaryOpts: summaryOpts,
	}
}

func makePrometheusSummaryOpts(metricOpts MetricOpts, summaryOpts SummaryOpts) prometheus.SummaryOpts {
	return prometheus.SummaryOpts{
		Name:        metricOpts.Name,
		Help:        metricOpts.Help,
		ConstLabels: metricOpts.ConstLabels,
		Objectives:  summaryOpts.Objectives,
		MaxAge:      summaryOpts.MaxAge,
		AgeBuckets:  summaryOpts.AgeBuckets,
		BufCap:      summaryOpts.BufCap,
	}
}

func (c *Summary) GetOpts() MetricOpts {
	return c.metricOpts
}

func (c *Summary) GetSummaryOpts() SummaryOpts {
	return c.summaryOpts
}

func (c *Summary) GetType() MetricType {
	return SummaryType
}

func (c *Summary) getCollector() prometheus.Collector {
	return c.Summary
}
