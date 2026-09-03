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
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func virtHandlerAlerts(namespace string) []promv1.Rule {
	return []promv1.Rule{
		{
			// Node-level virt-handler gap, not a per-VM alert.
			// Aggregate by node only so virt-launcher namespaces
			// cannot split the series and produce false positives.
			Alert: "OrphanedVirtualMachineInstances",
			Expr: intstr.FromString(
				"(((max by (node) (kube_pod_status_ready{condition='true',pod=~'virt-handler.*'} " +
					"* on(pod) group_left(node) max by(pod,node)(kube_pod_info{pod=~'virt-handler.*',node!=''})) ) == 1) " +
					"or (count by (node)( kube_pod_info{pod=~'virt-launcher.*',node!=''})*0)) == 0",
			),
			For: ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				"summary": "No ready virt-handler pod detected on node {{ $labels.node }} with running vmis for more than 10 minutes",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
				namespaceAlertLabelKey:       namespace,
			},
		},
		{
			Alert: "VirtHandlerDaemonSetRolloutFailing",
			Expr: intstr.FromString(
				fmt.Sprintf("(%s - %s) != 0",
					fmt.Sprintf("kube_daemonset_status_number_ready{namespace='%s', daemonset='virt-handler'}", namespace),
					fmt.Sprintf("kube_daemonset_status_desired_number_scheduled{namespace='%s', daemonset='virt-handler'}", namespace))),
			For: ptr.To(promv1.Duration("15m")),
			Annotations: map[string]string{
				"summary": "Some virt-handlers failed to roll out",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "VirtHandlerRESTErrorsBurst",
			Expr:  intstr.FromString(getErrorRatio(namespace, "virt-handler", "(4|5)[0-9][0-9]", 5) + " >= 0.8"),
			For:   ptr.To(promv1.Duration("5m")),
			Annotations: map[string]string{
				"summary": getRestCallsFailedWarning(80, "virt-handler", durationFiveMinutes),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
	}
}
