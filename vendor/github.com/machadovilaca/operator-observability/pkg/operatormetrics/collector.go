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
	var mType prometheus.ValueType

	switch metric.GetType() {
	case CounterType:
		mType = prometheus.CounterValue
	case GaugeType:
		mType = prometheus.GaugeValue
	case CounterVecType:
		mType = prometheus.CounterValue
	case GaugeVecType:
		mType = prometheus.GaugeValue
	default:
		return fmt.Errorf("encountered unsupported type for collector %v", metric.GetType())
	}

	desc := prometheus.NewDesc(
		metric.GetOpts().Name,
		metric.GetOpts().Help,
		metric.GetOpts().labels,
		metric.GetOpts().ConstLabels,
	)

	cm, err := prometheus.NewConstMetric(desc, mType, cr.Value, cr.Labels...)
	if err != nil {
		return err
	}
	ch <- cm

	return nil
}
