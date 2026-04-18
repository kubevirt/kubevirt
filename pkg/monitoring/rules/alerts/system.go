/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
			Expr:  intstr.FromString("(cluster:kubevirt_nodes_allocatable:count > 1) and (cluster:kubevirt_nodes_with_kvm:count < 2)"),
			For:   ptr.To(promv1.Duration("5m")),
			Annotations: map[string]string{
				descriptionAnnotationKey: "Low number of nodes with KVM resource available.",
				summaryAnnotationKey:     "At least two nodes with kvm resource required for VM live migration.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "KubeVirtNoAvailableNodesToRunVMs",
			Expr: intstr.FromString(
				"((cluster:kubevirt_nodes_with_kvm:count or on() vector(0)) == 0 " +
					" and (sum(kubevirt_configuration_emulation_enabled) or on() vector(0)) == 0) " +
					" or (sum(kube_node_labels{label_kubevirt_io_schedulable='true'} * on(node) group_left() (1 - kube_node_spec_unschedulable)) " +
					" or on() vector(0)) == 0"),
			For: ptr.To(promv1.Duration("5m")),
			Annotations: map[string]string{
				summaryAnnotationKey: "There are no available nodes in the cluster to run VMs.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "critical",
			},
		},
	}
}
