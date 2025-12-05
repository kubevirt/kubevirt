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
 */

package recordingrules

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hypervisor"
)

func nodesRecordingRules(hypervisorName string) []operatorrules.RecordingRule {
	rules := []operatorrules.RecordingRule{
		{
			MetricsOpts: operatormetrics.MetricOpts{
				Name: "kubevirt_allocatable_nodes",
				Help: "The number of allocatable nodes in the cluster.",
			},
			MetricType: operatormetrics.GaugeType,
			Expr:       intstr.FromString("count(count (kube_node_status_allocatable) by (node))"),
		},
	}

	// Generate generic hypervisor metric based on the configured hypervisor
	hypervisorDevice := hypervisor.NewHypervisor(hypervisorName).GetDevice()
	resourceName := "devices_kubevirt_io_" + hypervisorDevice

	rules = append(rules, operatorrules.RecordingRule{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_nodes_with_hypervisor",
			Help: "The number of nodes in the cluster that have the configured hypervisor resource available.",
			ConstLabels: map[string]string{
				"hypervisor": hypervisorName,
			},
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("count(kube_node_status_allocatable{resource=\"" + resourceName + "\"} != 0) or vector(0)"),
	})

	// Keep the older kubevirt_nodes_with_kvm metric for backward compatibility
	// However mark it as deprecated
	if hypervisorDevice == v1.KvmHypervisorName {
		rules = append(rules, operatorrules.RecordingRule{
			MetricsOpts: operatormetrics.MetricOpts{
				Name: "kubevirt_nodes_with_kvm",
				Help: "DEPRECATED: The number of nodes in the cluster that have the KVM hypervisor resource available. Use kubevirt_nodes_with_hypervisor instead.",
				ConstLabels: map[string]string{
					"deprecated": "true",
				},
			},
			MetricType: operatormetrics.GaugeType,
			Expr:       intstr.FromString("count(kube_node_status_allocatable{resource=\"devices_kubevirt_io_kvm\"} != 0) or vector(0)"),
		})
	}

	return rules
}
