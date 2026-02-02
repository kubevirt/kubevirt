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
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func virtControllerAlerts(namespace string) []promv1.Rule {
	return []promv1.Rule{
		{
			Alert: "LowReadyVirtControllersCount",
			Expr:  intstr.FromString("kubevirt_virt_controller_ready < cluster:kubevirt_virt_controller_pods_running:count"),
			For:   ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				"summary": "Some virt controllers are running but not ready.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "NoReadyVirtController",
			Expr:  intstr.FromString("kubevirt_virt_controller_ready == 0"),
			For:   ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				"summary": "No ready virt-controller was detected for the last 10 min.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "VirtControllerDown",
			Expr:  intstr.FromString("cluster:kubevirt_virt_controller_pods_running:count == 0"),
			For:   ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				"summary": "No running virt-controller was detected for the last 10 min.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "LowVirtControllersCount",
			Expr:  intstr.FromString("(kubevirt_allocatable_nodes > 1) and (kubevirt_virt_controller_ready < 2)"),
			For:   ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				"summary": "More than one virt-controller should be ready if more than one worker node.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "VirtControllerRESTErrorsBurst",
			Expr:  intstr.FromString(getErrorRatio(namespace, "virt-controller", "(4|5)[0-9][0-9]", 5) + " >= 0.8"),
			For:   ptr.To(promv1.Duration("5m")),
			Annotations: map[string]string{
				"summary": getRestCallsFailedWarning(80, "virt-controller", durationFiveMinutes),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
	}
}
