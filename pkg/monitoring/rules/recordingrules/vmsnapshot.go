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

var vmsnapshotRecordingRules = []operatorrules.RecordingRule{
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vmsnapshot_persistentvolumeclaim_labels",
			Help: "Returns the labels of the persistent volume claims that are used for restoring virtual machines.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("label_replace(label_replace(kube_persistentvolumeclaim_labels{label_restore_kubevirt_io_source_vm_name!='', label_restore_kubevirt_io_source_vm_namespace!=''} == 1, 'vm_namespace', '$1', 'label_restore_kubevirt_io_source_vm_namespace', '(.*)'), 'vm_name', '$1', 'label_restore_kubevirt_io_source_vm_name', '(.*)')"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vmsnapshot_disks_restored_from_source",
			Help: "Returns the total number of virtual machine disks restored from the source virtual machine.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("sum by(vm_name, vm_namespace) (kubevirt_vmsnapshot_persistentvolumeclaim_labels)"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vmsnapshot_disks_restored_from_source_bytes",
			Help: "Returns the amount of space in bytes restored from the source virtual machine.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("sum by(vm_name, vm_namespace) (kube_persistentvolumeclaim_resource_requests_storage_bytes * on(persistentvolumeclaim, namespace) group_left(vm_name, vm_namespace) kubevirt_vmsnapshot_persistentvolumeclaim_labels)"),
	},
}
