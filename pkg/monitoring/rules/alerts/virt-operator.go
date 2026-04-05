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

func virtOperatorAlerts(namespace string) []promv1.Rule {
	return []promv1.Rule{
		{
			Alert: "VirtOperatorDown",
			Expr:  intstr.FromString("cluster:kubevirt_virt_operator_up:sum == 0"),
			For:   ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				summaryAnnotationKey: "All virt-operator servers are down.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "LowVirtOperatorCount",
			Expr: intstr.FromString(
				fmt.Sprintf(
					"cluster:kubevirt_virt_operator_up:sum / on() "+
						"kube_deployment_spec_replicas{deployment='virt-operator', namespace='%s'} < 0.75",
					namespace,
				),
			),
			For: ptr.To(promv1.Duration("60m")),
			Annotations: map[string]string{
				summaryAnnotationKey: "Less than 75% of desired virt-operator pods are running.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "VirtOperatorRESTErrorsBurst",
			Expr:  intstr.FromString(getErrorRatio(namespace, "virt-operator", "(4|5)[0-9][0-9]", fiveMinutes) + " >= 0.8"),
			For:   ptr.To(promv1.Duration("5m")),
			Annotations: map[string]string{
				summaryAnnotationKey: getRestCallsFailedWarning(eightyPercent, "virt-operator", fiveMinutes),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "LowReadyVirtOperatorsCount",
			Expr:  intstr.FromString("cluster:kubevirt_virt_operator_ready:sum < cluster:kubevirt_virt_operator_pods_running:count"),
			For:   ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				summaryAnnotationKey: "Some virt-operators are running but not ready.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "NoReadyVirtOperator",
			Expr:  intstr.FromString("cluster:kubevirt_virt_operator_ready:sum == 0"),
			For:   ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				summaryAnnotationKey: "No ready virt-operator was detected for the last 10 min.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "NoLeadingVirtOperator",
			Expr:  intstr.FromString("cluster:kubevirt_virt_operator_leading:sum == 0"),
			For:   ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				summaryAnnotationKey: "No leading virt-operator was detected for the last 10 min.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
	}
}
