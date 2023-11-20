package operatormetrics

var operatorRegistry = newRegistry()

type operatorRegisterer struct {
	registeredMetrics          map[string]Metric
	registeredCollectorMetrics map[string]Metric
}

func newRegistry() operatorRegisterer {
	return operatorRegisterer{
		registeredMetrics:          map[string]Metric{},
		registeredCollectorMetrics: map[string]Metric{},
	}
}

// RegisterMetrics registers the metrics with the Prometheus registry.
func RegisterMetrics(allMetrics ...[]Metric) error {
	for _, metricList := range allMetrics {
		for _, metric := range metricList {
			err := Register(metric.getCollector())
			if err != nil {
				return err
			}
			operatorRegistry.registeredMetrics[metric.GetOpts().Name] = metric
		}
	}

	return nil
}

// RegisterCollector registers the collector with the Prometheus registry.
func RegisterCollector(collectors ...Collector) error {
	for _, collector := range collectors {
		err := Register(collector)
		if err != nil {
			return err
		}

		for _, cm := range collector.Metrics {
			operatorRegistry.registeredCollectorMetrics[cm.GetOpts().Name] = cm
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

	return result
}
