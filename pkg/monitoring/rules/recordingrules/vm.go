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
			Name: "kubevirt_vm_container_free_memory_bytes_based_on_working_set_bytes",
			Help: "The current available memory of the VM containers based on the working set.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("sum by(pod, container, namespace) (kube_pod_container_resource_requests{pod=~'virt-launcher-.*', container='compute', resource='memory'}- on(pod,container, namespace) max by(pod, container, namespace) (container_memory_working_set_bytes{pod=~'virt-launcher-.*', container='compute'}))"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vm_container_free_memory_bytes_based_on_rss",
			Help: "The current available memory of the VM containers based on the rss.",
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
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vm_allocated_cpu_cores",
			Help: "The number of CPU cores allocated to each VM.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr: intstr.FromString(`
			kubevirt_vm_resource_requests{resource="cpu", unit="cores"}
		`),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vm_allocated_cpu_sockets",
			Help: "The number of CPU sockets allocated to each VM, with default value of 1.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr: intstr.FromString(`
			kubevirt_vm_resource_requests{resource="cpu", unit="sockets"}
			or
			(kubevirt_vm_allocated_cpu_cores * 0 + 1)
		`),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vm_allocated_cpu_threads",
			Help: "The number of CPU threads per core allocated to each VM, with default value of 1.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr: intstr.FromString(`
			kubevirt_vm_resource_requests{resource="cpu", unit="threads"}
			or
			(kubevirt_vm_allocated_cpu_cores * 0 + 1)
		`),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vm_vcpu_count",
			Help: "The total number of vCPUs (virtual CPUs) for each VM, calculated as cores × sockets × threads.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr: intstr.FromString(`
			kubevirt_vm_allocated_cpu_cores * on(cluster, namespace, name) kubevirt_vm_allocated_cpu_sockets * on(cluster, namespace, name) kubevirt_vm_allocated_cpu_threads
		`),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vm_requested_memory_bytes",
			Help: "The requested memory in bytes for each VM, grouped by name and namespace.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr: intstr.FromString(`
			max by (name, namespace) (
				kubevirt_vm_resource_requests{resource="memory"}
			)
		`),
	},
}
