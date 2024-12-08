package operatormetrics

import (
	"cmp"
	"fmt"
	"slices"
)

var operatorRegistry = newRegistry()

type operatorRegisterer struct {
	registeredMetrics map[string]Metric

	registeredCollectors       map[string]Collector
	registeredCollectorMetrics map[string]Metric
}

func newRegistry() operatorRegisterer {
	return operatorRegisterer{
		registeredMetrics:          map[string]Metric{},
		registeredCollectors:       map[string]Collector{},
		registeredCollectorMetrics: map[string]Metric{},
	}
}

// RegisterMetrics registers the metrics with the Prometheus registry.
func RegisterMetrics(allMetrics ...[]Metric) error {
	for _, metricList := range allMetrics {
		for _, metric := range metricList {
			if metricExists(metric) {
				err := unregisterMetric(metric)
				if err != nil {
					return err
				}
			}

			err := registerMetric(metric)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// RegisterCollector registers the collector with the Prometheus registry.
func RegisterCollector(collectors ...Collector) error {
	for _, collector := range collectors {
		if collectorExists(collector) {
			err := unregisterCollector(collector)
			if err != nil {
				return err
			}
		}

		err := registerCollector(collector)
		if err != nil {
			return err
		}
	}

	return nil
}

// UnregisterMetrics unregisters the metrics from the Prometheus registry.
func UnregisterMetrics(allMetrics ...[]Metric) error {
	for _, metricList := range allMetrics {
		for _, metric := range metricList {
			if metricExists(metric) {
				if err := unregisterMetric(metric); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// ListMetrics returns a list of all registered metrics.
func ListMetrics() []Metric {
	var result []Metric

	for _, rm := range operatorRegistry.registeredMetrics {
		result = append(result, rm)
	}

	for _, rc := range operatorRegistry.registeredCollectorMetrics {
		result = append(result, rc)
	}

	slices.SortFunc(result, func(a, b Metric) int {
		return cmp.Compare(a.GetOpts().Name, b.GetOpts().Name)
	})

	return result
}

// CleanRegistry removes all registered metrics.
func CleanRegistry() error {
	for _, metric := range operatorRegistry.registeredMetrics {
		err := unregisterMetric(metric)
		if err != nil {
			return err
		}
	}

	for _, collector := range operatorRegistry.registeredCollectors {
		err := unregisterCollector(collector)
		if err != nil {
			return err
		}
	}

	return nil
}

func metricExists(metric Metric) bool {
	_, ok := operatorRegistry.registeredMetrics[metric.GetOpts().Name]
	return ok
}

func unregisterMetric(metric Metric) error {
	if succeeded := Unregister(metric.GetCollector()); succeeded {
		delete(operatorRegistry.registeredMetrics, metric.GetOpts().Name)
		return nil
	}

	return fmt.Errorf("failed to unregister from Prometheus client metric %s", metric.GetOpts().Name)
}

func registerMetric(metric Metric) error {
	err := Register(metric.GetCollector())
	if err != nil {
		return err
	}
	operatorRegistry.registeredMetrics[metric.GetOpts().Name] = metric

	return nil
}

func collectorExists(collector Collector) bool {
	_, ok := operatorRegistry.registeredCollectors[collector.hash()]
	return ok
}

func unregisterCollector(collector Collector) error {
	if succeeded := Unregister(collector); succeeded {
		delete(operatorRegistry.registeredCollectors, collector.hash())
		for _, metric := range collector.Metrics {
			delete(operatorRegistry.registeredCollectorMetrics, metric.GetOpts().Name)
		}
		return nil
	}

	return fmt.Errorf("failed to unregister from Prometheus client collector with metrics: %s", buildCollectorMetricListString(collector))
}

func registerCollector(collector Collector) error {
	err := Register(collector)
	if err != nil {
		return err
	}

	operatorRegistry.registeredCollectors[collector.hash()] = collector
	for _, cm := range collector.Metrics {
		operatorRegistry.registeredCollectorMetrics[cm.GetOpts().Name] = cm
	}

	return nil
}

func buildCollectorMetricListString(collector Collector) string {
	metricsList := ""
	for _, metric := range collector.Metrics {
		metricsList += metric.GetOpts().Name + ", "
	}
	metricsList = metricsList[:len(metricsList)-2]
	return metricsList
}
