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
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

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
	}
)
