package metrics

import "github.com/machadovilaca/operator-observability/pkg/operatormetrics"

var (
	metrics = [][]operatormetrics.Metric{
		operatorMetrics,
	}
)

func SetupMetrics() {
	err := operatormetrics.RegisterMetrics(metrics...)
	if err != nil {
		panic(err)
	}
}

func ListMetrics() []operatormetrics.Metric {
	return operatormetrics.ListMetrics()
}
