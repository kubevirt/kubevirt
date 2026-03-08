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

// All deprecated recording rules centralized here, aliasing the new names
var deprecatedRecordingRules = []operatorrules.RecordingRule{
	// Nodes
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_allocatable_nodes",
			Help: "[Deprecated] Replaced by cluster:kubevirt_nodes_allocatable:count.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("cluster:kubevirt_nodes_allocatable:count"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_nodes_with_kvm",
			Help: "[Deprecated] Replaced by cluster:kubevirt_nodes_with_kvm:count.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("cluster:kubevirt_nodes_with_kvm:count"),
	},
	// API
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_api_request_deprecated_total",
			Help: "[Deprecated] Replaced by cluster:kubevirt_api_request_deprecated_total:sum.",
		},
		MetricType: operatormetrics.CounterType,
		Expr:       intstr.FromString("cluster:kubevirt_api_request_deprecated_total:sum"),
	},
	// VM
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_number_of_vms",
			Help: "[Deprecated] Replaced by namespace:kubevirt_vm:sum.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("namespace:kubevirt_vm:sum"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vm_container_memory_request_margin_based_on_rss_bytes",
			Help: "[Deprecated] Replaced by pod_container:kubevirt_vm_memory_request_margin_based_on_rss_bytes:sum.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("pod_container:kubevirt_vm_memory_request_margin_based_on_rss_bytes:sum"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vm_container_memory_request_margin_based_on_working_set_bytes",
			Help: "[Deprecated] Replaced by pod_container:kubevirt_vm_memory_request_margin_based_on_working_set_bytes:sum.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("pod_container:kubevirt_vm_memory_request_margin_based_on_working_set_bytes:sum"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vm_created_total",
			Help: "[Deprecated] The total number of VMs created by namespace, since install.",
		},
		MetricType: operatormetrics.CounterType,
		Expr:       intstr.FromString("sum by (namespace) (kubevirt_vm_created_by_pod_total)"),
	},
	// VMI
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_guest_vcpu_queue",
			Help: "[Deprecated] Replaced by vmi:kubevirt_vmi_guest_queue_length:sum.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("vmi:kubevirt_vmi_guest_queue_length:sum"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_memory_used_bytes",
			Help: "[Deprecated] Replaced by vmi:kubevirt_vmi_memory_used_bytes:sum.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("vmi:kubevirt_vmi_memory_used_bytes:sum"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_phase_count",
			Help: "[Deprecated] Replaced by node:kubevirt_vmi_phase:sum.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("node:kubevirt_vmi_phase:sum"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_data_total_bytes",
			Help: "[Deprecated] Replaced by kubevirt_vmi_migration_data_bytes_total.",
		},
		MetricType: operatormetrics.CounterType,
		Expr:       intstr.FromString("kubevirt_vmi_migration_data_bytes_total"),
	},
	// Virt components
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_virt_api_up",
			Help: "[Deprecated] Replaced by cluster:kubevirt_virt_api_up:sum.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("cluster:kubevirt_virt_api_up:sum"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_virt_controller_up",
			Help: "[Deprecated] Replaced by cluster:kubevirt_virt_controller_up:sum.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("cluster:kubevirt_virt_controller_up:sum"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_virt_controller_ready",
			Help: "[Deprecated] Replaced by cluster:kubevirt_virt_controller_ready:sum.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("cluster:kubevirt_virt_controller_ready:sum"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_virt_operator_up",
			Help: "[Deprecated] Replaced by cluster:kubevirt_virt_operator_up:sum.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("cluster:kubevirt_virt_operator_up:sum"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_virt_operator_ready",
			Help: "[Deprecated] Replaced by cluster:kubevirt_virt_operator_ready:sum.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("cluster:kubevirt_virt_operator_ready:sum"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_virt_operator_leading",
			Help: "[Deprecated] Replaced by cluster:kubevirt_virt_operator_leading:sum.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("cluster:kubevirt_virt_operator_leading:sum"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_virt_handler_up",
			Help: "[Deprecated] Replaced by cluster:kubevirt_virt_handler_up:sum.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("cluster:kubevirt_virt_handler_up:sum"),
	},
	// VM Snapshot
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vmsnapshot_persistentvolumeclaim_labels",
			Help: "[Deprecated] Replaced by pvc:kubevirt_vmsnapshot_labels:info.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("pvc:kubevirt_vmsnapshot_labels:info"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vmsnapshot_disks_restored_from_source",
			Help: "[Deprecated] Replaced by vm:kubevirt_vmsnapshot_disks_restored:sum.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("vm:kubevirt_vmsnapshot_disks_restored:sum"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vmsnapshot_disks_restored_from_source_bytes",
			Help: "[Deprecated] Replaced by vm:kubevirt_vmsnapshot_restored_bytes:sum.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("vm:kubevirt_vmsnapshot_restored_bytes:sum"),
	},
	// Operator
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_memory_delta_from_requested_bytes",
			Help: "[Deprecated] Replaced by container:kubevirt_memory_delta_from_requested_bytes:max.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("container:kubevirt_memory_delta_from_requested_bytes:max"),
	},
}
