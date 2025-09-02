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

package alerts

import (
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func systemAlerts(namespace string) []promv1.Rule {
	return []promv1.Rule{
		{
			Alert: "LowKVMNodesCount",
			Expr:  intstr.FromString("(kubevirt_allocatable_nodes > 1) and (kubevirt_nodes_with_kvm < 2)"),
			For:   ptr.To(promv1.Duration("5m")),
			Annotations: map[string]string{
				"description": "Low number of nodes with KVM resource available.",
				"summary":     "At least two nodes with kvm resource required for VM live migration.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "KubeVirtNoAvailableNodesToRunVMs",
			Expr:  intstr.FromString("((sum(kube_node_status_allocatable{resource='devices_kubevirt_io_kvm'}) or on() vector(0)) == 0 and (sum(kubevirt_configuration_emulation_enabled) or on() vector(0)) == 0) or (sum(kube_node_labels{label_kubevirt_io_schedulable='true'}) or on() vector(0)) == 0"),
			For:   ptr.To(promv1.Duration("5m")),
			Annotations: map[string]string{
				"summary": "There are no available nodes in the cluster to run VMs.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "critical",
			},
		},
	}
}
