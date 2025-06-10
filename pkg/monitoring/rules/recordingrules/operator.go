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
)

var operatorRecordingRules = []operatorrules.RecordingRule{
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_memory_delta_from_requested_bytes",
			Help: "The delta between the pod with highest memory working set or rss and its requested memory for each container, virt-controller, virt-handler, virt-api and virt-operator.",
			ConstLabels: map[string]string{
				"reason": "memory_working_set_delta_from_request",
			},
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("topk by(container)(1,max by(container, namespace, node)(container_memory_working_set_bytes{container=~\"virt-controller|virt-api|virt-handler|virt-operator\"}  - on(pod) group_left(node) (kube_pod_container_resource_requests{ container=~\"virt-controller|virt-api|virt-handler|virt-operator\",resource=\"memory\"})))"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_memory_delta_from_requested_bytes",
			Help: "The delta between the pod with highest memory working set or rss and its requested memory for each container, virt-controller, virt-handler, virt-api and virt-operator.",
			ConstLabels: map[string]string{
				"reason": "memory_rss_delta_from_request",
			},
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("topk by(container)(1,max by(container, namespace, node)(container_memory_rss{container=~\"virt-controller|virt-api|virt-handler|virt-operator\"}  - on(pod) group_left(node) (kube_pod_container_resource_requests{ container=~\"virt-controller|virt-api|virt-handler|virt-operator\",resource=\"memory\"})))"),
	},
}
