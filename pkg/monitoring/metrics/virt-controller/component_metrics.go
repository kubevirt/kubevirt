/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package virtcontroller

import (
	ioprometheusclient "github.com/prometheus/client_model/go"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

var (
	componentMetrics = []operatormetrics.Metric{
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
