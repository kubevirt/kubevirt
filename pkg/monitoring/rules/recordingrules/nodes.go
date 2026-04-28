/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package recordingrules

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var nodesRecordingRules = []operatorrules.RecordingRule{
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "cluster:kubevirt_non_schedulable_nodes:sum",
			Help: "The number of non-schedulable nodes in the cluster.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("sum (count by (node) (kube_node_role{role=~'arbiter'})) or vector(0)"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "cluster:kubevirt_nodes_allocatable:count",
			Help: "The number of allocatable nodes in the cluster.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("count(count (kube_node_status_allocatable) by (node)) - cluster:kubevirt_non_schedulable_nodes:sum"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "cluster:kubevirt_nodes_with_kvm:count",
			Help: "The number of nodes in the cluster that have the devices.kubevirt.io/kvm resource available.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("count(kube_node_status_allocatable{resource=\"devices_kubevirt_io_kvm\"} != 0) or vector(0)"),
	},
}
