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

var vmiRecordingRules = []operatorrules.RecordingRule{
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_phase_count",
			Help: "Sum of VMIs per phase and node. `phase` can be one of the following: [`Pending`, `Scheduling`, `Scheduled`, `Running`, `Succeeded`, `Failed`, `Unknown`].",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("sum by (node, phase, os, workload, flavor, instance_type, preference, guest_os_kernel_release, guest_os_machine, guest_os_arch, guest_os_name, guest_os_version_id) (kubevirt_vmi_info)"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_memory_used_bytes",
			Help: "Amount of `used` memory as seen by the domain.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("kubevirt_vmi_memory_available_bytes-kubevirt_vmi_memory_usable_bytes"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "vmi:kubevirt_vmi_vcpu:count",
			Help: "The number of the VMI vCPUs.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("count by (namespace, name, node) (kubevirt_vmi_vcpu_seconds_total)"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_guest_vcpu_queue",
			Help: "Guest queue length.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("clamp_min(kubevirt_vmi_guest_load_1m - vmi:kubevirt_vmi_vcpu:count, 0)"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "vmi:kubevirt_vmi_memory_headroom_ratio:sum",
			Help: "Usable memory to available memory ratio per VMI (aggregated by name, namespace).",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("sum by (name, namespace) (kubevirt_vmi_memory_usable_bytes) / sum by (name, namespace) (kubevirt_vmi_memory_available_bytes)"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "vmi:kubevirt_vmi_pgmajfaults:rate5m",
			Help: "Rate of major page faults over 5 minutes per VMI (aggregated by name, namespace).",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("sum by (name, namespace) (rate(kubevirt_vmi_memory_pgmajfault_total[5m]))"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "vmi:kubevirt_vmi_swap_traffic_bytes:rate5m",
			Help: "Total swap I/O traffic rate over 5 minutes per VMI (swap in + swap out, aggregated by name, namespace).",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("sum by (name, namespace) (rate(kubevirt_vmi_memory_swap_in_traffic_bytes[5m])) + sum by (name, namespace) (rate(kubevirt_vmi_memory_swap_out_traffic_bytes[5m]))"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "vmi:kubevirt_vmi_memory_available_bytes:sum",
			Help: "Sum of available memory bytes per VMI (aggregated by name, namespace).",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("sum by (name, namespace) (kubevirt_vmi_memory_available_bytes)"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "vmi:kubevirt_vmi_pgmajfaults:rate30m",
			Help: "Rate of major page faults over 30 minutes per VMI (aggregated by name, namespace).",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("sum by (name, namespace) (rate(kubevirt_vmi_memory_pgmajfault_total[30m]))"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "vmi:kubevirt_vmi_swap_traffic_bytes:rate30m",
			Help: "Total swap I/O traffic rate over 30 minutes per VMI (swap in + swap out, aggregated by name, namespace).",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("sum by (name, namespace) (rate(kubevirt_vmi_memory_swap_in_traffic_bytes[30m])) + sum by (name, namespace) (rate(kubevirt_vmi_memory_swap_out_traffic_bytes[30m]))"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_data_total_bytes",
			Help: "[Deprecated] Replaced by kubevirt_vmi_migration_data_bytes_total.",
		},
		MetricType: operatormetrics.CounterType,
		Expr:       intstr.FromString("kubevirt_vmi_migration_data_bytes_total"),
	},
}
