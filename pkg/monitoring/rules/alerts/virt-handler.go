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
	"fmt"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func virtHandlerAlerts(namespace string) []promv1.Rule {
	return []promv1.Rule{
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
