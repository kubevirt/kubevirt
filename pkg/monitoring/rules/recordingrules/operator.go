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

var operatorRecordingRules = []operatorrules.RecordingRule{
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "container:kubevirt_memory_delta_from_requested_bytes:max",
			Help: "The delta between the pod with highest memory working set or rss and its requested memory for each container, " +
				"virt-controller, virt-handler, virt-api, virt-operator and compute(virt-launcher).",
			ConstLabels: map[string]string{
				"reason": "memory_working_set_delta_from_request",
			},
		},
		MetricType: operatormetrics.GaugeType,
		Expr: intstr.FromString(
			"topk by(container)(1,max by(container, namespace, node, pod)" +
				"(container_memory_working_set_bytes{container=~\"virt-controller|virt-api|virt-handler|virt-operator|compute\", pod=~\"virt-.*\"} - " +
				"on(pod) group_left(node) " +
				"(kube_pod_container_resource_requests{" +
				"container=~\"virt-controller|virt-api|virt-handler|virt-operator|compute\",resource=\"memory\"" +
				"})))",
		),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "container:kubevirt_memory_delta_from_requested_bytes:max",
			Help: "The delta between the pod with highest memory working set or rss and its requested memory for each container, " +
				"virt-controller, virt-handler, virt-api, virt-operator and compute(virt-launcher).",
			ConstLabels: map[string]string{
				"reason": "memory_rss_delta_from_request",
			},
		},
		MetricType: operatormetrics.GaugeType,
		Expr: intstr.FromString(
			"topk by(container)(1,max by(container, namespace, node, pod)" +
				"(container_memory_rss{container=~\"virt-controller|virt-api|virt-handler|virt-operator|compute\", pod=~\"virt-.*\"} - " +
				"on(pod) group_left(node) " +
				"(kube_pod_container_resource_requests{" +
				"container=~\"virt-controller|virt-api|virt-handler|virt-operator|compute\",resource=\"memory\"" +
				"})))",
		),
	},
}
