package operatormetrics

import "github.com/prometheus/client_golang/prometheus"

type MetricOpts struct {
	Name        string
	Help        string
	ConstLabels map[string]string
	ExtraFields map[string]string

	labels []string
}

type Metric interface {
	GetOpts() MetricOpts
	GetType() MetricType
	GetBaseType() MetricType
	GetCollector() prometheus.Collector
}

type MetricType string

const (
	CounterType   MetricType = "Counter"
	GaugeType     MetricType = "Gauge"
	HistogramType MetricType = "Histogram"
	SummaryType   MetricType = "Summary"

	CounterVecType   MetricType = "CounterVec"
	GaugeVecType     MetricType = "GaugeVec"
	HistogramVecType MetricType = "HistogramVec"
	SummaryVecType   MetricType = "SummaryVec"
)

func convertOpts(opts MetricOpts) prometheus.Opts {
	return prometheus.Opts{
		Name:        opts.Name,
		Help:        opts.Help,
		ConstLabels: opts.ConstLabels,
	}
}
