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

package components

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	VirtExportProxyHPAName = "virt-exportproxy-hpa"

	// ExportProxyActiveTransfersMetricName is the Prometheus/custom-metrics name used by
	// the export-proxy HPA. Must match the metric name in
	// pkg/monitoring/metrics/virt-exportproxy/transfer_metrics.go.
	ExportProxyActiveTransfersMetricName = "kubevirt_exportproxy_active_transfers"

	exportProxyHPAMinReplicas int32 = 2
	exportProxyHPAMaxReplicas int32 = 20
)

// Target average concurrent export transfers per pod before the HPA scales out.
var exportProxyHPATargetActiveTransfersPerPod = resource.MustParse("50")

// NewExportProxyHorizontalPodAutoscaler returns an HPA for virt-exportproxy that scales
// between 2 and 20 replicas based on average active export transfers per pod.
//
// The HPA uses a Pods custom metric (not CPU/memory from metrics-server). Clusters must
// expose kubevirt_exportproxy_active_transfers on custom.metrics.k8s.io, which requires
// Prometheus to scrape virt-exportproxy metrics and a prometheus-adapter (or platform
// equivalent) configured with a matching rule for this metric name.
func NewExportProxyHorizontalPodAutoscaler(deployment *appsv1.Deployment) *autoscalingv2.HorizontalPodAutoscaler {
	targetAverageTransfers := exportProxyHPATargetActiveTransfersPerPod.DeepCopy()

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
			},
		},
	}
}
