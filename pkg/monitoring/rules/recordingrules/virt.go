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
	"fmt"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func virtRecordingRules(namespace string) []operatorrules.RecordingRule {
	return []operatorrules.RecordingRule{
		// up
		newRecordingRule(
			"cluster:kubevirt_virt_api_up:sum",
			"The number of virt-api pods that are up.",
			upExpr(namespace, "api"),
		),
		newRecordingRule(
			"cluster:kubevirt_virt_controller_up:sum",
			"The number of virt-controller pods that are up.",
			upExpr(namespace, "controller"),
		),
		newRecordingRule(
			"cluster:kubevirt_virt_operator_up:sum",
			"The number of virt-operator pods that are up.",
			upExpr(namespace, "operator"),
		),
		newRecordingRule(
			"cluster:kubevirt_virt_handler_up:sum",
			"The number of virt-handler pods that are up.",
			upExpr(namespace, "handler"),
		),

		// pods running
		newRecordingRule(
			"cluster:kubevirt_virt_api_pods_running:count",
			"The number of virt-api pods that are running.",
			podsRunningExpr(namespace, "api"),
		),
		newRecordingRule(
			"cluster:kubevirt_virt_controller_pods_running:count",
			"The number of virt-controller pods that are running.",
			podsRunningExpr(namespace, "controller"),
		),
		newRecordingRule(
			"cluster:kubevirt_virt_operator_pods_running:count",
			"The number of virt-operator pods that are running.",
			podsRunningExpr(namespace, "operator"),
		),
		newRecordingRule(
			"cluster:kubevirt_virt_handler_pods_running:count",
			"The number of virt-handler pods that are running.",
			podsRunningExpr(namespace, "handler"),
		),

		// ready
		newRecordingRule(
			"cluster:kubevirt_virt_api_ready:sum",
			"The number of virt-api pods that are ready.",
			kubePodReadyOnlyExpr(namespace, "api"),
		),
		newRecordingRule(
			"cluster:kubevirt_virt_controller_ready:sum",
			"The number of virt-controller pods that are ready.",
			readyExpr(namespace, "controller", "kubevirt_virt_controller_ready_status"),
		),
		newRecordingRule(
			"cluster:kubevirt_virt_operator_ready:sum",
			"The number of virt-operator pods that are ready.",
			readyExpr(namespace, "operator", "kubevirt_virt_operator_ready_status"),
		),
		newRecordingRule(
			"cluster:kubevirt_virt_handler_ready:sum",
			"The number of virt-handler pods that are ready.",
			kubePodReadyOnlyExpr(namespace, "handler"),
		),

		// leading
		newRecordingRule(
			"cluster:kubevirt_virt_operator_leading:sum",
			"The number of virt-operator pods that are leading.",
			fmt.Sprintf("sum(kubevirt_virt_operator_leading_status{namespace='%s'})", namespace),
		),
	}
}

func newRecordingRule(name, help, expr string) operatorrules.RecordingRule {
	return operatorrules.RecordingRule{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: name,
			Help: help,
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString(expr),
	}
}

func upExpr(namespace, component string) string {
	return fmt.Sprintf(
		"sum(up{namespace='%s', pod=~'virt-%s-.*'}) or vector(0)",
		namespace, component,
	)
}

func podsRunningExpr(namespace, component string) string {
	return fmt.Sprintf(
		"count(kube_pod_status_phase{pod=~'virt-%s-.*', namespace='%s', phase='Running'} == 1) or vector(0)",
		component, namespace,
	)
}

func readyExpr(namespace, component, readyStatusMetric string) string {
	return fmt.Sprintf(
		"sum(kube_pod_status_ready{pod=~'virt-%s-.*', namespace='%s', condition='true'} * on(pod, namespace) %s{namespace='%s'}) or vector(0)",
		component, namespace, readyStatusMetric, namespace,
	)
}

func kubePodReadyOnlyExpr(namespace, component string) string {
	return fmt.Sprintf(
		"sum(kube_pod_status_ready{pod=~'virt-%s-.*', namespace='%s', condition='true'}) or vector(0)",
		component, namespace,
	)
}
