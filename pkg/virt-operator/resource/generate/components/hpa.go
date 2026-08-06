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
 *
 */

// virt-exportproxy HPA requires two custom metrics on custom.metrics.k8s.io.
// Prometheus must scrape kubevirt_exportproxy_active_transfers from virt-exportproxy
// pods; prometheus-adapter (or a platform equivalent) must expose the rules below.
// Merge the rules into the adapter ConfigMap (typically data.config.yaml → rules).
//
// Example prometheus-adapter rules (targets align with pkg/exportproxy/admission):
//
//	rules:
//	- seriesQuery: 'kubevirt_exportproxy_active_transfers{namespace!="",pod!=""}'
//	  resources:
//	    overrides:
//	      namespace:
//	        resource: namespace
//	      pod:
//	        resource: pod
//	  name:
//	    matches: "^kubevirt_exportproxy_active_transfers"
//	    as: "kubevirt_exportproxy_active_transfers"
//	  metricsQuery: 'sum(<<.Series>>{<<.LabelMatchers>>}) by (<<.GroupBy>>)'
//	- seriesQuery: 'kubevirt_exportproxy_active_transfers{namespace!="",pod!=""}'
//	  resources:
//	    overrides:
//	      namespace:
//	        resource: namespace
//	  name:
//	    matches: "^kubevirt_exportproxy_active_transfers"
//	    as: "kubevirt_exportproxy_active_transfers_pod_max"
//	  metricsQuery: |
//	    max(<<.Series>>{<<.LabelMatchers>>}) by (<<.GroupBy>>)
//	    *
//	    (
//	      avg(<<.Series>>{<<.LabelMatchers>>}) by (<<.GroupBy>>) > bool 35
//	    )
//
// The first rule backs the Pods metric (HPA averageValue target 50). The second
// exposes a namespace-level gated max (HPA value target 70); the > bool 35 gate
// matches admission.HPAMaxMetricAverageFloor so hot pods draining after load
// ends do not alone drive scale-out or block scale-down.
//
// Verify before relying on HPA:
//
//	kubectl get apiservice v1beta1.custom.metrics.k8s.io
//	kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1/namespaces/<ns>/pods/*/kubevirt_exportproxy_active_transfers"
//	kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1/namespaces/<ns>/kubevirt_exportproxy_active_transfers_pod_max"

package components

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/exportproxy/admission"
	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	VirtExportProxyHPAName = "virt-exportproxy-hpa"

	// ExportProxyActiveTransfersMetricName is the Prometheus/custom-metrics name used by
	// the export-proxy HPA average metric. Must match the metric name in
	// pkg/monitoring/metrics/virt-exportproxy/transfer_metrics.go.
	ExportProxyActiveTransfersMetricName = "kubevirt_exportproxy_active_transfers"

	// ExportProxyActiveTransfersPodMaxMetricName is a namespace-level custom metric
	// exposing the hottest virt-exportproxy pod, gated so it reports zero when fleet
	// average active transfers is below admission.HPAMaxMetricAverageFloor.
	ExportProxyActiveTransfersPodMaxMetricName = "kubevirt_exportproxy_active_transfers_pod_max"

	exportProxyHPAMinReplicas int32 = 2
	exportProxyHPAMaxReplicas int32 = 20
)

// Target average concurrent export transfers per pod before the HPA scales out.
var exportProxyHPATargetActiveTransfersPerPod = resource.NewQuantity(int64(admission.HPATargetAverageTransfers), resource.DecimalExponent)

// Target per-pod maximum active transfers for the gated max metric. Intended as
// average target plus headroom (admission.HPATargetAverageTransfers + 20).
var exportProxyHPATargetMaxActiveTransfersPerPod = resource.NewQuantity(int64(admission.HPATargetMaxTransfers), resource.DecimalExponent)

// NewExportProxyHorizontalPodAutoscaler returns an HPA for virt-exportproxy that scales
// between 2 and 20 replicas using two custom metrics (HPA uses the highest replica
// recommendation from either metric):
//
//   - Average active transfers per pod (target 50): total fleet load.
//   - Gated per-pod max active transfers (target 70): detects skew while the fleet is
//     loaded; reports zero when average < 35 so hot pods draining after a spike do not
//     block scale-down or force scale-out alone.
//
// Clusters must expose both metrics on custom.metrics.k8s.io via prometheus-adapter
// (or a platform equivalent). The pod-max metric rule should gate on fleet average
// active transfers exceeding admission.HPAMaxMetricAverageFloor.
func NewExportProxyHorizontalPodAutoscaler(deployment *appsv1.Deployment) *autoscalingv2.HorizontalPodAutoscaler {
	targetAverageTransfers := exportProxyHPATargetActiveTransfersPerPod.DeepCopy()
	targetMaxTransfers := exportProxyHPATargetMaxActiveTransfersPerPod.DeepCopy()

	return &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: deployment.Namespace,
			Name:      VirtExportProxyHPAName,
			Labels: map[string]string{
				virtv1.AppLabel: VirtExportProxyHPAName,
			},
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       VirtExportProxyName,
			},
			MinReplicas: pointer.P(exportProxyHPAMinReplicas),
			MaxReplicas: exportProxyHPAMaxReplicas,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.PodsMetricSourceType,
					Pods: &autoscalingv2.PodsMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Name: ExportProxyActiveTransfersMetricName,
						},
						Target: autoscalingv2.MetricTarget{
							Type:         autoscalingv2.AverageValueMetricType,
							AverageValue: &targetAverageTransfers,
						},
					},
				},
				{
					Type: autoscalingv2.ObjectMetricSourceType,
					Object: &autoscalingv2.ObjectMetricSource{
						DescribedObject: autoscalingv2.CrossVersionObjectReference{
							APIVersion: "v1",
							Kind:       "Namespace",
							Name:       deployment.Namespace,
						},
						Metric: autoscalingv2.MetricIdentifier{
							Name: ExportProxyActiveTransfersPodMaxMetricName,
						},
						Target: autoscalingv2.MetricTarget{
							Type:  autoscalingv2.ValueMetricType,
							Value: &targetMaxTransfers,
						},
					},
				},
			},
		},
	}
}
