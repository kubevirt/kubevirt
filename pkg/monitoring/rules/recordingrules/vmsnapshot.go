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

var vmsnapshotRecordingRules = []operatorrules.RecordingRule{
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "pvc:kubevirt_vmsnapshot_labels:info",
			Help: "Returns the labels of the persistent volume claims that are used for restoring virtual machines.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr: intstr.FromString(
			"label_replace(label_replace(" +
				"kube_persistentvolumeclaim_labels" +
				"{label_restore_kubevirt_io_source_vm_name!='', label_restore_kubevirt_io_source_vm_namespace!=''} == 1," +
				"'vm_namespace', '$1', 'label_restore_kubevirt_io_source_vm_namespace', '(.*)'), " +
				"'vm_name', '$1', 'label_restore_kubevirt_io_source_vm_name', '(.*)')",
		),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "vm:kubevirt_vmsnapshot_disks_restored:sum",
			Help: "Returns the total number of virtual machine disks restored from the source virtual machine.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("sum by(vm_name, vm_namespace) (pvc:kubevirt_vmsnapshot_labels:info)"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "vm:kubevirt_vmsnapshot_restored_bytes:sum",
			Help: "Returns the amount of space in bytes restored from the source virtual machine.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr: intstr.FromString(
			"sum by(vm_name, vm_namespace) (kube_persistentvolumeclaim_resource_requests_storage_bytes * " +
				"on(persistentvolumeclaim, namespace) group_left(vm_name, vm_namespace) pvc:kubevirt_vmsnapshot_labels:info)"),
	},
}
