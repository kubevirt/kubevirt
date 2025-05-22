/*
Copyright 2023 The KubeVirt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package alerts

import (
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func virtApiAlerts(namespace string) []promv1.Rule {
	return []promv1.Rule{
		{
			Alert: "VirtAPIDown",
			Expr:  intstr.FromString("kubevirt_virt_api_up == 0"),
			For:   ptr.To(promv1.Duration("10m")),
			Annotations: map[string]string{
				"summary": "All virt-api servers are down.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "LowVirtAPICount",
			Expr:  intstr.FromString("(kubevirt_allocatable_nodes > 1) and (kubevirt_virt_api_up < 2)"),
			For:   ptr.To(promv1.Duration("60m")),
			Annotations: map[string]string{
				"summary": "More than one virt-api should be running if more than one worker nodes exist.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "VirtApiRESTErrorsBurst",
			Expr:  intstr.FromString(getErrorRatio(namespace, "virt-api", "(4|5)[0-9][0-9]", 5) + " >= 0.8"),
			For:   ptr.To(promv1.Duration("5m")),
			Annotations: map[string]string{
				"summary": getRestCallsFailedWarning(80, "virt-api", durationFiveMinutes),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "KubeVirtDeprecatedAPIRequested",
			Expr:  intstr.FromString("sum by (resource,group,version) ((round(increase(kubevirt_api_request_deprecated_total{verb!~\"LIST|WATCH\"}[10m])) > 0 and kubevirt_api_request_deprecated_total{verb!~\"LIST|WATCH\"} offset 10m) or (kubevirt_api_request_deprecated_total{verb!~\"LIST|WATCH\"} != 0 unless kubevirt_api_request_deprecated_total{verb!~\"LIST|WATCH\"} offset 10m))"),
			Annotations: map[string]string{
				"description": "Detected requests to the deprecated {{ $labels.resource }}.{{ $labels.group }}/{{ $labels.version }} API.",
				"summary":     "Detected {{ $value }} requests in the last 10 minutes.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "info",
				operatorHealthImpactLabelKey: "none",
			},
		},
	}
}
