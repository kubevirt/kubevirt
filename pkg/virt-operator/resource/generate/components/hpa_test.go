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
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("export-proxy HPA", func() {
	exportProxyDeployment := func() *appsv1.Deployment {
		config := &util.KubeVirtDeploymentConfig{
			Namespace: "kubevirt",
		}
		return NewExportProxyDeployment(config, "", "", "")
	}

	It("uses resource CPU metrics by default", func() {
		hpa := NewExportProxyHorizontalPodAutoscaler(exportProxyDeployment(), ExportProxyHPAMetricsProfileResource)
		Expect(hpa.Annotations[ExportProxyHPAMetricsProfileAnnotation]).To(Equal(string(ExportProxyHPAMetricsProfileResource)))
		Expect(hpa.Spec.Metrics).To(HaveLen(1))
		Expect(hpa.Spec.Metrics[0].Type).To(Equal(autoscalingv2.ResourceMetricSourceType))
		Expect(hpa.Spec.Metrics[0].Resource.Name).To(Equal(corev1.ResourceCPU))
		Expect(*hpa.Spec.Metrics[0].Resource.Target.AverageUtilization).To(Equal(exportProxyHPATargetCPUUtilization))
	})

	It("uses custom transfer metrics when requested", func() {
		hpa := NewExportProxyHorizontalPodAutoscaler(exportProxyDeployment(), ExportProxyHPAMetricsProfileCustomMetrics)
		Expect(hpa.Annotations[ExportProxyHPAMetricsProfileAnnotation]).To(Equal(string(ExportProxyHPAMetricsProfileCustomMetrics)))
		Expect(hpa.Spec.Metrics).To(HaveLen(2))
		Expect(hpa.Spec.Metrics[0].Pods.Metric.Name).To(Equal(ExportProxyActiveTransfersMetricName))
		Expect(hpa.Spec.Metrics[1].Object.Metric.Name).To(Equal(ExportProxyActiveTransfersPodMaxMetricName))
	})
})
