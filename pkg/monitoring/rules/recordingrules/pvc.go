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

var pvcRecordingRules = []operatorrules.RecordingRule{
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "pvc:kubevirt_suspected_orphaned_storage_bytes:info",
			Help: "Suspected orphaned persistent volume claims with their requested storage size in bytes. " +
				"A PVC is considered suspected orphaned when it is not mounted by any pod and not allocated " +
				"to any KubeVirt virtual machine disk.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr: intstr.FromString(
			"((kube_persistentvolumeclaim_info unless on(namespace,persistentvolumeclaim) " +
				"(kube_pod_spec_volumes_persistentvolumeclaims or kubevirt_vm_disk_allocated_size_bytes)) * " +
				"on(namespace,persistentvolumeclaim) group_left() " +
				"kube_persistentvolumeclaim_resource_requests_storage_bytes)",
		),
	},
}
