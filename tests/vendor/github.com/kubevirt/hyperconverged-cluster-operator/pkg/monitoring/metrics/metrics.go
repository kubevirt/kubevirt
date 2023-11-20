package metrics

import "github.com/machadovilaca/operator-observability/pkg/operatormetrics"

func SetupMetrics() error {
	return operatormetrics.RegisterMetrics(
		operatorMetrics,
		infrastructureMetrics,
	)
}

func ListMetrics() []operatormetrics.Metric {
	return operatormetrics.ListMetrics()
}
