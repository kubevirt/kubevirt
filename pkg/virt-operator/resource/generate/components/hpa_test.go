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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("export-proxy HPA", func() {
	const namespace = "kubevirt"

	It("uses resource CPU and memory metrics by default", func() {
		hpa := NewExportProxyHorizontalPodAutoscaler(namespace, ExportProxyHPAMetricsProfileResource)
		Expect(hpa.Namespace).To(Equal(namespace))
		Expect(hpa.Annotations[ExportProxyHPAMetricsProfileAnnotation]).To(Equal(string(ExportProxyHPAMetricsProfileResource)))
		Expect(hpa.Spec.ScaleTargetRef.Name).To(Equal(VirtExportProxyName))
		Expect(hpa.Spec.Metrics).To(HaveLen(2))
		Expect(hpa.Spec.Metrics[0].Type).To(Equal(autoscalingv2.ResourceMetricSourceType))
		Expect(hpa.Spec.Metrics[0].Resource.Name).To(Equal(corev1.ResourceCPU))
		Expect(*hpa.Spec.Metrics[0].Resource.Target.AverageUtilization).To(Equal(exportProxyHPATargetCPUUtilization))
		Expect(hpa.Spec.Metrics[1].Type).To(Equal(autoscalingv2.ResourceMetricSourceType))
		Expect(hpa.Spec.Metrics[1].Resource.Name).To(Equal(corev1.ResourceMemory))
		Expect(*hpa.Spec.Metrics[1].Resource.Target.AverageUtilization).To(Equal(exportProxyHPATargetMemoryUtilization))
	})

	It("uses custom transfer metrics when requested", func() {
		hpa := NewExportProxyHorizontalPodAutoscaler(namespace, ExportProxyHPAMetricsProfileCustomMetrics)
		Expect(hpa.Annotations[ExportProxyHPAMetricsProfileAnnotation]).To(Equal(string(ExportProxyHPAMetricsProfileCustomMetrics)))
		Expect(hpa.Spec.Metrics).To(HaveLen(4))
		Expect(hpa.Spec.Metrics[0].Pods.Metric.Name).To(Equal(ExportProxyActiveTransfersMetricName))
		Expect(hpa.Spec.Metrics[1].Object.Metric.Name).To(Equal(ExportProxyActiveTransfersPodMaxMetricName))
		Expect(hpa.Spec.Metrics[1].Object.DescribedObject.Name).To(Equal(namespace))
		Expect(hpa.Spec.Metrics[2].Type).To(Equal(autoscalingv2.ResourceMetricSourceType))
		Expect(hpa.Spec.Metrics[2].Resource.Name).To(Equal(corev1.ResourceCPU))
		Expect(hpa.Spec.Metrics[3].Type).To(Equal(autoscalingv2.ResourceMetricSourceType))
		Expect(hpa.Spec.Metrics[3].Resource.Name).To(Equal(corev1.ResourceMemory))
	})
})
