/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
				Name: "cluster:kubevirt_virt_api_up:sum",
				Help: "The number of virt-api pods that are up.",
			},
			MetricType: operatormetrics.GaugeType,
			Expr: intstr.FromString(
				fmt.Sprintf("sum(up{namespace='%s', pod=~'virt-api-.*'}) or vector(0)", namespace),
			),
		},
		{
			MetricsOpts: operatormetrics.MetricOpts{
				Name: "cluster:kubevirt_virt_controller_pods_running:count",
				Help: "The number of virt-controller pods that are running.",
			},
			MetricType: operatormetrics.GaugeType,
			Expr: intstr.FromString(
				fmt.Sprintf("count(kube_pod_status_phase{pod=~'virt-controller-.*', namespace='%s', phase='Running'} == 1) or vector(0)", namespace),
			),
		},
		{
			MetricsOpts: operatormetrics.MetricOpts{
				Name: "cluster:kubevirt_virt_controller_up:sum",
				Help: "The number of virt-controller pods that are up.",
			},
			MetricType: operatormetrics.GaugeType,
			Expr: intstr.FromString(
				fmt.Sprintf("sum(up{pod=~'virt-controller-.*', namespace='%s'}) or vector(0)", namespace),
			),
		},
		{
			MetricsOpts: operatormetrics.MetricOpts{
				Name: "cluster:kubevirt_virt_controller_ready:sum",
				Help: "The number of virt-controller pods that are ready.",
			},
			MetricType: operatormetrics.GaugeType,
			Expr: intstr.FromString(
				fmt.Sprintf("sum(kube_pod_status_ready{pod=~'virt-controller-.*', namespace='%s', condition='true'} * "+
					" on(pod, namespace) kubevirt_virt_controller_ready_status{namespace='%s'}) or vector(0)", namespace, namespace),
			),
		},
		{
			MetricsOpts: operatormetrics.MetricOpts{
				Name: "cluster:kubevirt_virt_operator_up:sum",
				Help: "The number of virt-operator pods that are up.",
			},
			MetricType: operatormetrics.GaugeType,
			Expr: intstr.FromString(
				fmt.Sprintf("sum(up{namespace='%s', pod=~'virt-operator-.*'}) or vector(0)", namespace),
			),
		},
		{
			MetricsOpts: operatormetrics.MetricOpts{
				Name: "cluster:kubevirt_virt_operator_pods_running:count",
				Help: "The number of virt-operator pods that are running.",
			},
			MetricType: operatormetrics.GaugeType,
			Expr: intstr.FromString(
				fmt.Sprintf("count(kube_pod_status_phase{pod=~'virt-operator-.*', namespace='%s', phase='Running'} == 1) or vector(0)", namespace),
			),
		},
		{
			MetricsOpts: operatormetrics.MetricOpts{
				Name: "cluster:kubevirt_virt_operator_ready:sum",
				Help: "The number of virt-operator pods that are ready.",
			},
			MetricType: operatormetrics.GaugeType,
			Expr: intstr.FromString(
				fmt.Sprintf("sum(kube_pod_status_ready{pod=~'virt-operator-.*', condition='true', namespace='%s'} * "+
					"on (pod) kubevirt_virt_operator_ready_status{namespace='%s'}) or vector(0)", namespace, namespace),
			),
		},
		{
			MetricsOpts: operatormetrics.MetricOpts{
				Name: "cluster:kubevirt_virt_operator_leading:sum",
				Help: "The number of virt-operator pods that are leading.",
			},
			MetricType: operatormetrics.GaugeType,
			Expr: intstr.FromString(
				fmt.Sprintf("sum(kubevirt_virt_operator_leading_status{namespace='%s'})", namespace),
			),
		},
		{
			MetricsOpts: operatormetrics.MetricOpts{
				Name: "cluster:kubevirt_virt_handler_up:sum",
				Help: "The number of virt-handler pods that are up.",
			},
			MetricType: operatormetrics.GaugeType,
			Expr:       intstr.FromString(fmt.Sprintf("sum(up{pod=~'virt-handler-.*', namespace='%s'}) or vector(0)", namespace)),
		},
	}
}
