/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package virthandler

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"libvirt.org/go/libvirtxml"
)

var (
	machineTypeMetrics = []operatormetrics.Metric{
		deprecatedMachineTypeMetric,
	}

	deprecatedMachineTypeMetric = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_node_deprecated_machine_types",
			Help: "List of deprecated machine types based on the capabilities of individual nodes, as detected by virt-handler.",
		},
		[]string{"machine_type", "node"},
	)
)

func ReportDeprecatedMachineTypes(machines []libvirtxml.CapsGuestMachine, nodeName string) {
	for _, machine := range machines {
		if machine.Deprecated == "yes" {
			deprecatedMachineTypeMetric.WithLabelValues(machine.Name, nodeName).Set(1)
		}
	}
}
