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

var vmRecordingRules = []operatorrules.RecordingRule{
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vm_container_memory_request_margin_based_on_working_set_bytes",
			Help: "Difference between requested memory and working set for VM containers (request margin). Can be negative when usage exceeds request.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("sum by(pod, container, namespace) (kube_pod_container_resource_requests{pod=~'virt-launcher-.*', container='compute', resource='memory'}- on(pod,container, namespace) max by(pod, container, namespace) (container_memory_working_set_bytes{pod=~'virt-launcher-.*', container='compute'}))"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vm_container_memory_request_margin_based_on_rss_bytes",
			Help: "Difference between requested memory and rss for VM containers (request margin). Can be negative when usage exceeds request.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("sum by(pod, container, namespace) (kube_pod_container_resource_requests{pod=~'virt-launcher-.*', container='compute', resource='memory'}- on(pod,container, namespace) container_memory_rss{pod=~'virt-launcher-.*', container='compute'})"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_number_of_vms",
			Help: "The number of VMs in the cluster by namespace.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("sum by (namespace) (count by (name,namespace) (kubevirt_vm_error_status_last_transition_timestamp_seconds + kubevirt_vm_migrating_status_last_transition_timestamp_seconds + kubevirt_vm_non_running_status_last_transition_timestamp_seconds + kubevirt_vm_running_status_last_transition_timestamp_seconds + kubevirt_vm_starting_status_last_transition_timestamp_seconds))"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vm_created_total",
			Help: "The total number of VMs created by namespace, since install.",
		},
		MetricType: operatormetrics.CounterType,
		Expr:       intstr.FromString("sum by (namespace) (kubevirt_vm_created_by_pod_total)"),
	},
}
