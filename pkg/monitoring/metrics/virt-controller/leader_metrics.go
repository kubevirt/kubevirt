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
	leaderMetrics = []operatormetrics.Metric{
		outdatedVirtualMachineInstanceWorkloads,
	}

	outdatedVirtualMachineInstanceWorkloads = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_number_of_outdated",
			Help: "Indication for the total number of VirtualMachineInstance workloads " +
				"that are not running within the most up-to-date version of the virt-launcher environment.",
		},
	)
)

func SetOutdatedVirtualMachineInstanceWorkloads(value int) {
	outdatedVirtualMachineInstanceWorkloads.Set(float64(value))
}

func GetOutdatedVirtualMachineInstanceWorkloads() (int, error) {
	dto := &ioprometheusclient.Metric{}
	if err := outdatedVirtualMachineInstanceWorkloads.Write(dto); err != nil {
		return 0, err
	}

	return int(dto.GetGauge().GetValue()), nil
}
