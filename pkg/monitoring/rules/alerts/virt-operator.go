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

func virtOperatorAlerts(namespace string) []promv1.Rule {
	return []promv1.Rule{
		{
			Alert: "VirtOperatorDown",
			Expr:  intstr.FromString("kubevirt_virt_operator_up == 0"),
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
			Expr: intstr.FromString(fmt.Sprintf(
				"kubevirt_virt_operator_up / on() kube_deployment_spec_replicas{deployment='virt-operator', namespace='%s'} < 0.75", namespace,
			)),
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
			Expr:  intstr.FromString("kubevirt_virt_operator_ready < cluster:kubevirt_virt_operator_pods_running:count"),
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
			Expr:  intstr.FromString("kubevirt_virt_operator_ready == 0"),
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
			Expr:  intstr.FromString("kubevirt_virt_operator_leading == 0"),
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
