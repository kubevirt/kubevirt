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
		{
			MetricsOpts: operatormetrics.MetricOpts{
				Name: "kubevirt_virt_api_up",
				Help: "The number of virt-api pods that are up.",
			},
			MetricType: operatormetrics.GaugeType,
			Expr: intstr.FromString(
				fmt.Sprintf("sum(up{namespace='%s', pod=~'virt-api-.*'}) or vector(0)", namespace),
			),
		},
		{
			MetricsOpts: operatormetrics.MetricOpts{
				Name: "kubevirt_virt_controller_up",
				Help: "The number of virt-controller pods that are up.",
			},
			MetricType: operatormetrics.GaugeType,
			Expr: intstr.FromString(
				fmt.Sprintf("sum(kube_pod_status_phase{pod=~'virt-controller-.*', namespace='%s', phase='Running'}) or vector(0)", namespace),
			),
		},
		{
			MetricsOpts: operatormetrics.MetricOpts{
				Name: "kubevirt_virt_controller_ready",
				Help: "The number of virt-controller pods that are ready.",
			},
			MetricType: operatormetrics.GaugeType,
			Expr: intstr.FromString(
				fmt.Sprintf("count(kube_pod_status_ready{pod=~'virt-controller-.*', namespace='%s', condition='true'} + on(pod, namespace) kubevirt_virt_controller_ready_status{namespace='%s'}) or vector(0)", namespace, namespace),
			),
		},
		{
			MetricsOpts: operatormetrics.MetricOpts{
				Name: "kubevirt_virt_operator_up",
				Help: "The number of virt-operator pods that are up.",
			},
			MetricType: operatormetrics.GaugeType,
			Expr: intstr.FromString(
				fmt.Sprintf("sum(up{namespace='%s', pod=~'virt-operator-.*'}) or vector(0)", namespace),
			),
		},
		{
			MetricsOpts: operatormetrics.MetricOpts{
				Name: "kubevirt_virt_operator_ready",
				Help: "The number of virt-operator pods that are ready.",
			},
			MetricType: operatormetrics.GaugeType,
			Expr: intstr.FromString(
				fmt.Sprintf("sum(kube_pod_status_ready{pod=~'virt-operator-.*', condition='true', namespace='%s'} * on (pod) kubevirt_virt_operator_ready_status{namespace='%s'}) or vector(0)", namespace, namespace),
			),
		},
		{
			MetricsOpts: operatormetrics.MetricOpts{
				Name: "kubevirt_virt_operator_leading",
				Help: "The number of virt-operator pods that are leading.",
			},
			MetricType: operatormetrics.GaugeType,
			Expr: intstr.FromString(
				fmt.Sprintf("sum(kubevirt_virt_operator_leading_status{namespace='%s'})", namespace),
			),
		},
		{
			MetricsOpts: operatormetrics.MetricOpts{
				Name: "kubevirt_virt_handler_up",
				Help: "The number of virt-handler pods that are up.",
			},
			MetricType: operatormetrics.GaugeType,
			Expr:       intstr.FromString(fmt.Sprintf("sum(up{pod=~'virt-handler-.*', namespace='%s'}) or vector(0)", namespace)),
		},
	}
}
