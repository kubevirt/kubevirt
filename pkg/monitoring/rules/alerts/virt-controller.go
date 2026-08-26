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

func virtControllerAlerts(namespace string) []promv1.Rule {
	return []promv1.Rule{
		{
			Alert: "LowReadyVirtControllersCount",
			Expr: intstr.FromString(
				"cluster:kubevirt_virt_controller_ready:sum < cluster:kubevirt_virt_controller_pods_running:count " +
					"and cluster:kubevirt_virt_controller_ready:sum > 0",
			),
			For: ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				summaryAnnotationKey: "Some virt controllers are running but not ready.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "NoReadyVirtController",
			Expr: intstr.FromString(
				"cluster:kubevirt_virt_controller_ready:sum == 0 " +
					"and cluster:kubevirt_virt_controller_pods_running:count > 0",
			),
			For: ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				summaryAnnotationKey: "No ready virt-controller was detected for the last 10 min.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "NoLeadingVirtController",
			Expr:  intstr.FromString("cluster:kubevirt_virt_controller_leading:sum == 0"),
			For:   ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				summaryAnnotationKey: "No leading virt-controller was detected for the last 10 min.",
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
				summaryAnnotationKey: "No running virt-controller was detected for the last 10 min.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "LowVirtControllersCount",
			Expr: intstr.FromString(fmt.Sprintf(
				"cluster:kubevirt_virt_controller_pods_running:count / on() "+
					"kube_deployment_spec_replicas{deployment='virt-controller', namespace='%s'} < 0.75",
				namespace,
			)),
			For: ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				summaryAnnotationKey: "Less than 75% of desired virt-controller pods are running.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			// Cluster-wide count of failed virt-launcher pods, not a
			// per-namespace alert. Do not sum by namespace: that would
			// split the 200-pod threshold across VM namespaces. The
			// install namespace is applied as a static label in Register().
			Alert: "VirtLauncherPodsStuckFailed",
			Expr:  intstr.FromString("sum(kube_pod_status_phase{phase='Failed', pod=~'virt-launcher-.*'}) >= 200"),
			For:   ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				summaryAnnotationKey: "At least 200 virt-launcher pods are stuck in Failed state and not deleted for 10 minutes.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "VirtControllerRESTErrorsBurst",
			Expr:  intstr.FromString(getErrorRatio(namespace, "virt-controller", "(4|5)[0-9][0-9]", fiveMinutes) + " >= 0.8"),
			For:   ptr.To(promv1.Duration("5m")),
			Annotations: map[string]string{
				summaryAnnotationKey: getRestCallsFailedWarning(eightyPercent, "virt-controller", fiveMinutes),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
	}
}
