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

package alerts

import (
	"fmt"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

// excludedFilesystemTypesRegex contains filesystem types that should be ignored by space-usage alerts.
// Rationale:
// - Read-only image filesystems often appear 100% used and are not actionable for capacity (iso9660/CDFS, udf, squashfs, cramfs).
// - Pseudo/kernel/ephemeral filesystems are not meaningful indicators of guest disk pressure (e.g., tmpfs, proc, sysfs, cgroup*, overlay, fuse.*).
// Keep this list aligned with common node_exporter exclusions to minimize noise in alerts.
const excludedFilesystemTypesRegex = "CDFS|iso9660|udf|squashfs|cramfs|tmpfs|devtmpfs|proc|sysfs|selinuxfs|securityfs|pstore|debugfs|tracefs|configfs|binfmt_misc|bpf|devpts|mqueue|nsfs|rpc_pipefs|ramfs|rootfs|overlay|cgroup.*|fuse\\\\..*|fusectl"

var (
	fiftyMB = resource.MustParse("50Mi")

	vmsAlerts = []promv1.Rule{
		{
			Alert: "OrphanedVirtualMachineInstances",
			Expr:  intstr.FromString("(((max by (node) (kube_pod_status_ready{condition='true',pod=~'virt-handler.*'} * on(pod) group_left(node) max by(pod,node)(kube_pod_info{pod=~'virt-handler.*',node!=''})) ) == 1) or (count by (node)( kube_pod_info{pod=~'virt-launcher.*',node!=''})*0)) == 0"),
			For:   ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				"summary": "No ready virt-handler pod detected on node {{ $labels.node }} with running vmis for more than 10 minutes",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "VMCannotBeEvicted",
			Expr:  intstr.FromString("kubevirt_vmi_non_evictable * on(name, namespace) group_left() kubevirt_vmi_info{phase='running'} == 1"),
			For:   ptr.To(promv1.Duration("1m")),
			Annotations: map[string]string{
				"description": "Eviction policy for VirtualMachine {{ $labels.name }} in namespace {{ $labels.namespace }} (on node {{ $labels.node }}) is set to Live Migration but the VM is not migratable",
				"summary":     "The VM's eviction strategy is set to Live Migration but the VM is not migratable",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "KubeVirtVMIExcessiveMigrations",
			Expr:  intstr.FromString("sum by (vmi, namespace) (topk by (vmi, namespace, vmim) (1, max_over_time(kubevirt_vmi_migration_succeeded[1d]))) >= 12"),
			Annotations: map[string]string{
				"description": "VirtualMachineInstance {{ $labels.vmi }} in namespace {{ $labels.namespace }} has been migrated more than 12 times during the last 24 hours",
				"summary":     "An excessive amount of migrations have been detected on a VirtualMachineInstance in the last 24 hours.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "OutdatedVirtualMachineInstanceWorkloads",
			Expr:  intstr.FromString("kubevirt_vmi_number_of_outdated != 0"),
			For:   ptr.To(promv1.Duration("24h")),
			Annotations: map[string]string{
				"summary": "Some running VMIs are still active in outdated pods after KubeVirt control plane update has completed.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "GuestVCPUQueueHighWarning",
			Expr:  intstr.FromString("kubevirt_vmi_guest_vcpu_queue > 10"),
			Annotations: map[string]string{
				"description": "VirtualMachineInstance {{ $labels.name }} CPU queue length > 10",
				"summary":     "Guest vCPU Queue within collection cycle > 10",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "GuestVCPUQueueHighCritical",
			Expr:  intstr.FromString("kubevirt_vmi_guest_vcpu_queue > 20"),
			Annotations: map[string]string{
				"description": "VirtualMachineInstance {{ $labels.name }} CPU queue length > 20",
				"summary":     "Guest vCPU Queue within collection cycle > 20",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "VirtualMachineStuckInUnhealthyState",
			Expr:  intstr.FromString("sum by (name, namespace, status)(kubevirt_vm_info{status='provisioning'}==1 or kubevirt_vm_info{status='starting'} == 1 or kubevirt_vm_info{status='terminating'} == 1 or kubevirt_vm_info{status_group='error'} == 1) unless on(name, namespace) kubevirt_vmi_info"),
			For:   ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				"summary":     "Virtual machine in {{ $labels.status }} state for more than 10 minutes",
				"description": "Virtual machine {{ $labels.name }} in namespace {{ $labels.namespace }} has been in {{ $labels.status }} state for more than 10 minutes.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "VirtualMachineStuckOnNode",
			Expr:  intstr.FromString("sum by (name, namespace, status, node)((kubevirt_vm_info{status='starting'} == 1 or kubevirt_vm_info{status='stopping'} == 1 or kubevirt_vm_info{status='terminating'} == 1 or (kubevirt_vm_info{status_group='error'} == 1 and on(name, namespace) kubevirt_vmi_info) ) * on(name, namespace) group_left(node) kubevirt_vmi_info)"),
			For:   ptr.To(promv1.Duration("5m")),
			Annotations: map[string]string{
				"summary":     "Virtual machine stuck in unhealthy state for more than 5 minutes",
				"description": "Virtual machine {{ $labels.name }} in namespace {{ $labels.namespace }} on node {{ $labels.node }} has been in {{ $labels.status }} state for more than 5 minutes. This may indicate issues with the VM lifecycle on the target node.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "GuestFilesystemAlmostOutOfSpace",
			Expr:  intstr.FromString(fmt.Sprintf("(kubevirt_vmi_filesystem_used_bytes{file_system_type!~'%s',mount_point!='System Reserved'} / kubevirt_vmi_filesystem_capacity_bytes{file_system_type!~'%s',mount_point!='System Reserved'})*100 >= 85 < 95", excludedFilesystemTypesRegex, excludedFilesystemTypesRegex)),
			For:   ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				"summary":     "Guest filesystem is running out of space",
				"description": "VirtualMachineInstance {{ $labels.name }} in namespace {{ $labels.namespace }} has filesystem {{ $labels.disk_name }} ({{ $labels.mount_point }}) usage above 85% (current: {{ $value }}%).",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "KubeVirtVMGuestMemoryPressure",
			Expr:  intstr.FromString("(((vmi:kubevirt_vmi_memory_headroom_ratio:sum < 0.05) and (vmi:kubevirt_vmi_pgmajfaults:rate5m > 5 or (vmi:kubevirt_vmi_swap_traffic_bytes:rate5m > 1048576)) and (vmi:kubevirt_vmi_memory_available_bytes:sum > 0)) * on(name, namespace) group_left(vm) label_replace(label_replace(kubevirt_vmi_info{phase='running'}, 'vm', '$1', 'name', '(.+)'), 'name', '$1', 'vmi_pod', '(.+)'))"),
			For:   ptr.To(promv1.Duration("5m")),
			Annotations: map[string]string{
				"description": "VirtualMachine {{ $labels.vm }} in namespace {{ $labels.namespace }} is under memory pressure: low usable memory with elevated major faults and/or swap IO.",
				"summary":     "The VirtualMachine is under memory pressure (possible thrashing)",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "GuestFilesystemAlmostOutOfSpace",
			Expr:  intstr.FromString(fmt.Sprintf("(kubevirt_vmi_filesystem_used_bytes{file_system_type!~'%s',mount_point!='System Reserved'} / kubevirt_vmi_filesystem_capacity_bytes{file_system_type!~'%s',mount_point!='System Reserved'})*100 >= 95", excludedFilesystemTypesRegex, excludedFilesystemTypesRegex)),
			Annotations: map[string]string{
				"summary":     "Guest filesystem is critically low on space",
				"description": "VirtualMachineInstance {{ $labels.name }} in namespace {{ $labels.namespace }} has filesystem {{ $labels.disk_name }} ({{ $labels.mount_point }}) usage above 95% (current: {{ $value }}%).",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "VirtualMachineInstanceHasEphemeralHotplugVolume",
			Expr:  intstr.FromString("kubevirt_vmi_contains_ephemeral_hotplug_volume == 1"),
			Annotations: map[string]string{
				"summary":     "Virtual Machine Instance has Ephemeral Hotplug Volume(s). Ephemeral Hotplugs are deprecated and must be converted to persistent volumes! In a future release, feature gate `DeclarativeHotplugVolumes` will replace `HotplugVolumes` and as a result, any remaining ephemeral hotplug volumes will be automatically unplugged",
				"description": "Virtual Machine Instance {{ $labels.name }} in namespace {{ $labels.namespace }} has ephemeral hotplug volume(s) {{ $labels.volume_name }}.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "KubeVirtVMGuestMemoryAvailableLow",
			Expr:  intstr.FromString("(((vmi:kubevirt_vmi_memory_headroom_ratio:sum < 0.03) and (vmi:kubevirt_vmi_swap_traffic_bytes:rate30m < 2048) and (vmi:kubevirt_vmi_pgmajfaults:rate30m < 1) and (vmi:kubevirt_vmi_memory_available_bytes:sum > 0)) * on(name, namespace) group_left(vm) label_replace(label_replace(kubevirt_vmi_info{phase='running'}, 'vm', '$1', 'name', '(.+)'), 'name', '$1', 'vmi_pod', '(.+)'))"),
			For:   ptr.To(promv1.Duration("30m")),
			Annotations: map[string]string{
				"description": "VirtualMachine {{ $labels.vm }} in namespace {{ $labels.namespace }} has very low available memory (<3% headroom) for 30 minutes with no meaningful swap IO (likely no swap).",
				"summary":     "The VirtualMachine has low available memory without swap",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "info",
				operatorHealthImpactLabelKey: "none",
			},
		},
	}
)
