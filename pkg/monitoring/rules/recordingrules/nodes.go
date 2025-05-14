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
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"github.com/machadovilaca/operator-observability/pkg/operatorrules"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var nodesRecordingRules = []operatorrules.RecordingRule{
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_allocatable_nodes",
			Help: "The number of allocatable nodes in the cluster.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("count(count (kube_node_status_allocatable) by (node))"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_nodes_with_kvm",
			Help: "The number of nodes in the cluster that have the devices.kubevirt.io/kvm resource available.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("count(kube_node_status_allocatable{resource=\"devices_kubevirt_io_kvm\"} != 0) or vector(0)"),
	},
}
