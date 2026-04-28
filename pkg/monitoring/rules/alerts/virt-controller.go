/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
			Expr:  intstr.FromString("cluster:kubevirt_virt_controller_ready:sum < cluster:kubevirt_virt_controller_pods_running:count"),
			For:   ptr.To(promv1.Duration("10m")),
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
			Expr:  intstr.FromString("cluster:kubevirt_virt_controller_ready:sum == 0"),
			For:   ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				summaryAnnotationKey: "No ready virt-controller was detected for the last 10 min.",
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
				"cluster:kubevirt_virt_controller_up:sum / on() kube_deployment_spec_replicas{deployment='virt-controller', namespace='%s'} < 0.75",
				namespace,
			)),
			For: ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				summaryAnnotationKey: "Less than 75% of desired virt-controller pods are ready.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
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
