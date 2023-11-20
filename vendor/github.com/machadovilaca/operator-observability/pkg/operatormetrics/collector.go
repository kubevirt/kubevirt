package operatormetrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

// Collector registers a prometheus.Collector with a set of metrics in the
// Prometheus registry. The metrics are collected by calling the CollectCallback
// function.
type Collector struct {
	// Metrics is a list of metrics to be collected by the collector.
	Metrics []Metric

	// CollectCallback is a function that returns a list of CollectionResults.
	// The CollectionResults are used to populate the metrics in the collector.
	CollectCallback func() []CollectorResult
}

type CollectorResult struct {
	Metric Metric
	Labels []string
	Value  float64
}

func (c Collector) Describe(ch chan<- *prometheus.Desc) {
	for _, cm := range c.Metrics {
		cm.getCollector().Describe(ch)
	}
}

func (c Collector) Collect(ch chan<- prometheus.Metric) {
	collectedMetrics := c.CollectCallback()

	for _, cr := range collectedMetrics {
		metric, ok := operatorRegistry.registeredCollectorMetrics[cr.Metric.GetOpts().Name]
		if !ok {
			fmt.Printf("metric %s not found in registry", cr.Metric.GetOpts().Name)
			continue
		}

		if err := collectValue(ch, metric, cr); err != nil {
			fmt.Printf("error collecting metric %s: %v", cr.Metric.GetOpts().Name, err)
		}
	}
}

func collectValue(ch chan<- prometheus.Metric, metric Metric, cr CollectorResult) error {
	switch metric.GetType() {
	case CounterType:
		m := metric.getCollector().(prometheus.Counter)
		m.Add(cr.Value)
		m.Collect(ch)
	case GaugeType:
		m := metric.getCollector().(prometheus.Gauge)
		m.Set(cr.Value)
		m.Collect(ch)
	case HistogramType:
		m := metric.getCollector().(prometheus.Histogram)
		m.Observe(cr.Value)
		m.Collect(ch)
	case SummaryType:
		m := metric.getCollector().(prometheus.Summary)
		m.Observe(cr.Value)
		m.Collect(ch)
	case CounterVecType:
		m := metric.getCollector().(prometheus.CounterVec)
		m.WithLabelValues(cr.Labels...).Add(cr.Value)
		m.Collect(ch)
	case GaugeVecType:
		m := metric.getCollector().(prometheus.GaugeVec)
		m.WithLabelValues(cr.Labels...).Set(cr.Value)
		m.Collect(ch)
	case HistogramVecType:
		m := metric.getCollector().(prometheus.HistogramVec)
		m.WithLabelValues(cr.Labels...).Observe(cr.Value)
		m.Collect(ch)
	case SummaryVecType:
		m := metric.getCollector().(prometheus.SummaryVec)
		m.WithLabelValues(cr.Labels...).Observe(cr.Value)
		m.Collect(ch)
	default:
		return fmt.Errorf("encountered unknown type %v", metric.GetType())
	}

	return nil
}
