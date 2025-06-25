/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package virt_handler

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
