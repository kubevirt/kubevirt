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

// virt-exportproxy HPA uses auto-detected metrics: resource CPU/memory by default
// (metrics-server only), or custom transfer metrics plus resource CPU/memory when
// prometheus-adapter exposes the transfer metrics. Resource metrics stay in the
// custom profile so soft CPU/memory admission pressure still drives scale-out when
// active transfers are below the transfer targets. Detection is cached so brief
// adapter outages do not flip the HPA every reconcile.
//
// Working prometheus-adapter rules and operator notes:
// docs/observability/export-proxy-hpa.md
// docs/observability/examples/export-proxy-prometheus-adapter-rules.yaml

package components

import (
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/exportproxy/admission"
	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	VirtExportProxyHPAName = "virt-exportproxy-hpa"

	// ExportProxyHPAMetricsProfileAnnotation records the metric strategy selected for
	// virt-exportproxy-hpa (resource or custom-metrics).
	ExportProxyHPAMetricsProfileAnnotation = "kubevirt.io/export-proxy-hpa-metrics-profile"

	// ExportProxyActiveTransfersMetricName is the Prometheus/custom-metrics name used by
	// the export-proxy HPA average metric. Must match the metric name in
	// pkg/monitoring/metrics/virt-exportproxy/transfer_metrics.go and the adapter
	// rules in docs/observability/examples/export-proxy-prometheus-adapter-rules.yaml.
	ExportProxyActiveTransfersMetricName = "kubevirt_exportproxy_active_transfers"

	// ExportProxyActiveTransfersPodMaxMetricName is a namespace-level custom metric
	// exposing the hottest virt-exportproxy pod, gated so it reports zero when fleet
	// average active transfers is below admission.HPAMaxMetricAverageFloor.
	// Adapter rule: docs/observability/examples/export-proxy-prometheus-adapter-rules.yaml.
	ExportProxyActiveTransfersPodMaxMetricName = "kubevirt_exportproxy_active_transfers_pod_max"

	exportProxyHPAMinReplicas int32 = 2
	exportProxyHPAMaxReplicas int32 = 20

	// Resource HPA AverageUtilization is vs requests (not limits). Export-proxy
	// is Burstable with requests below limits, so targets are >100% of request
	// to scale near soft admission (~70% of limit):
	// CPU 200% of 350m ~= 700m ~= 70% of 1 limit; memory 140% of 256Mi ~= 70% of 512Mi.
	exportProxyHPATargetCPUUtilization    int32 = 200
	exportProxyHPATargetMemoryUtilization int32 = 140
)

// ExportProxyHPAMetricsProfile selects which metric sources virt-exportproxy-hpa uses.
type ExportProxyHPAMetricsProfile string

const (
	ExportProxyHPAMetricsProfileResource      ExportProxyHPAMetricsProfile = "resource"
	ExportProxyHPAMetricsProfileCustomMetrics ExportProxyHPAMetricsProfile = "custom-metrics"
)

// Target average concurrent export transfers per pod before the HPA scales out.
var exportProxyHPATargetActiveTransfersPerPod = resource.NewQuantity(int64(admission.HPATargetAverageTransfers), resource.DecimalExponent)

// Target per-pod maximum active transfers for the gated max metric. Intended as
// average target plus headroom (admission.HPATargetAverageTransfers + 20).
var exportProxyHPATargetMaxActiveTransfersPerPod = resource.NewQuantity(int64(admission.HPATargetMaxTransfers), resource.DecimalExponent)

// NewExportProxyHorizontalPodAutoscaler returns an HPA for virt-exportproxy that scales
// between 2 and 20 replicas using the requested metrics profile.
func NewExportProxyHorizontalPodAutoscaler(namespace string, profile ExportProxyHPAMetricsProfile) *autoscalingv2.HorizontalPodAutoscaler {
	if profile == "" {
		profile = ExportProxyHPAMetricsProfileResource
	}

	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      VirtExportProxyHPAName,
			Labels: map[string]string{
				virtv1.AppLabel: VirtExportProxyHPAName,
			},
			Annotations: map[string]string{
				ExportProxyHPAMetricsProfileAnnotation: string(profile),
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
			Metrics:     exportProxyHPAMetrics(profile, namespace),
		},
	}

	return hpa
}

func exportProxyHPAMetrics(profile ExportProxyHPAMetricsProfile, namespace string) []autoscalingv2.MetricSpec {
	resourceMetrics := exportProxyResourceMetrics()
	if profile == ExportProxyHPAMetricsProfileCustomMetrics {
		// Keep resource metrics so soft CPU/memory admission (which does not
		// raise active_transfers) can still drive scale-out via HPA.
		return append(exportProxyCustomMetrics(namespace), resourceMetrics...)
	}
	return resourceMetrics
}

func exportProxyResourceMetrics() []autoscalingv2.MetricSpec {
	return []autoscalingv2.MetricSpec{
		{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: pointer.P(exportProxyHPATargetCPUUtilization),
				},
			},
		},
		{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceMemory,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: pointer.P(exportProxyHPATargetMemoryUtilization),
				},
			},
		},
	}
}

func exportProxyCustomMetrics(namespace string) []autoscalingv2.MetricSpec {
	targetAverageTransfers := exportProxyHPATargetActiveTransfersPerPod.DeepCopy()
	targetMaxTransfers := exportProxyHPATargetMaxActiveTransfersPerPod.DeepCopy()

	return []autoscalingv2.MetricSpec{
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
					Name:       namespace,
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
	}
}
