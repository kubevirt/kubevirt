/*
Copyright 2023 The KubeVirt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package alerts

import (
	"fmt"

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
			Alert: "KubeVirtComponentExceedsRequestedMemory",
			Expr: intstr.FromString(
				// In 'container_memory_working_set_bytes', 'container=""' filters the accumulated metric for the pod slice to measure total Memory usage for all containers within the pod
				fmt.Sprintf(`((kube_pod_container_resource_requests{namespace="%s",container=~"virt-controller|virt-api|virt-handler|virt-operator",resource="memory"}) - on(pod) group_left(node) container_memory_working_set_bytes{container="",namespace="%s"}) < 0`, namespace, namespace)),
			For: ptr.To(promv1.Duration("5m")),
			Annotations: map[string]string{
				"description": "Container {{ $labels.container }} in pod {{ $labels.pod }} memory usage exceeds the memory requested",
				"summary":     "The container is using more memory than what is defined in the containers resource requests",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "KubeVirtComponentExceedsRequestedCPU",
			Expr: intstr.FromString(
				// In 'container_cpu_usage_seconds_total', 'container=""' filters the accumulated metric for the pod slice to measure total CPU usage for all containers within the pod
				fmt.Sprintf(`((kube_pod_container_resource_requests{namespace="%s",container=~"virt-controller|virt-api|virt-handler|virt-operator",resource="cpu"}) - on(pod) sum(rate(container_cpu_usage_seconds_total{container="",namespace="%s"}[5m])) by (pod)) < 0`, namespace, namespace),
			),
			For: ptr.To(promv1.Duration("5m")),
			Annotations: map[string]string{
				"description": "Pod {{ $labels.pod }} cpu usage exceeds the CPU requested",
				"summary":     "The containers in the pod are using more CPU than what is defined in the containers resource requests",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
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
