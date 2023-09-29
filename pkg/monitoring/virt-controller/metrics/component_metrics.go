package metrics

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	ioprometheusclient "github.com/prometheus/client_model/go"
)

var (
	operatorMetrics = []operatormetrics.Metric{
		virtControllerLeading,
		virtControllerReady,
	}

	virtControllerLeading = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_virt_controller_leading_status",
			Help: "Indication for an operating virt-controller.",
		},
	)

	virtControllerReady = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_virt_controller_ready_status",
			Help: "Indication for a virt-controller that is ready to take the lead.",
		},
	)
)

func GetVirtControllerMetric() (*ioprometheusclient.Metric, error) {
	dto := &ioprometheusclient.Metric{}
	err := virtControllerLeading.Write(dto)
	return dto, err
}

func SetVirtControllerLeading() {
	virtControllerLeading.Set(1)
}

func SetVirtControllerReady() {
	virtControllerReady.Set(1)
}

func SetVirtControllerNotReady() {
	virtControllerReady.Set(0)
}
