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

package components_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

var _ = Describe("Handler Pool DaemonSets", func() {
	newPrimaryHandler := func() *appsv1.DaemonSet {
		return &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      components.VirtHandlerName,
				Namespace: "kubevirt",
				Labels: map[string]string{
					virtv1.AppLabel: components.VirtHandlerName,
				},
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"kubevirt.io": components.VirtHandlerName,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"kubevirt.io": components.VirtHandlerName,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "virt-handler", Image: "registry.example.com/virt-handler:v1.0.0"},
						},
						InitContainers: []corev1.Container{
							{Name: "virt-launcher", Image: "registry.example.com/virt-launcher:v1.0.0"},
						},
					},
				},
			},
		}
	}

	Describe("NewHandlerPoolDaemonSet", func() {
		It("should set pool name and labels", func() {
			pool := virtv1.VirtHandlerPoolConfig{
				Name:              "gpu-pool",
				VirtLauncherImage: "registry.example.com/virt-launcher:gpu",
				NodeSelector:      map[string]string{"nvidia.com/gpu.product": "Tesla-T4"},
				Selector: virtv1.VirtHandlerPoolSelector{
					DeviceNames: []string{"nvidia.com/TU104GL_Tesla_T4"},
				},
			}

			ds := components.NewHandlerPoolDaemonSet(newPrimaryHandler(), pool)

			Expect(ds.Name).To(Equal("virt-handler-gpu-pool"))
			Expect(ds.Labels[virtv1.HandlerPoolLabel]).To(Equal("gpu-pool"))
			Expect(ds.Spec.Template.Labels[virtv1.HandlerPoolLabel]).To(Equal("gpu-pool"))
			Expect(ds.Spec.Selector.MatchLabels[virtv1.HandlerPoolLabel]).To(Equal("gpu-pool"))
		})

		It("should override handler image when specified", func() {
			pool := virtv1.VirtHandlerPoolConfig{
				Name:             "gpu-pool",
				VirtHandlerImage: "registry.example.com/virt-handler:gpu",
				NodeSelector:     map[string]string{"k": "v"},
				Selector:         virtv1.VirtHandlerPoolSelector{DeviceNames: []string{"dev"}},
			}

			ds := components.NewHandlerPoolDaemonSet(newPrimaryHandler(), pool)
			Expect(ds.Spec.Template.Spec.Containers[0].Image).To(Equal("registry.example.com/virt-handler:gpu"))
		})

		It("should keep default handler image when not specified", func() {
			pool := virtv1.VirtHandlerPoolConfig{
				Name:              "gpu-pool",
				VirtLauncherImage: "registry.example.com/virt-launcher:gpu",
				NodeSelector:      map[string]string{"k": "v"},
				Selector:          virtv1.VirtHandlerPoolSelector{DeviceNames: []string{"dev"}},
			}

			ds := components.NewHandlerPoolDaemonSet(newPrimaryHandler(), pool)
			Expect(ds.Spec.Template.Spec.Containers[0].Image).To(Equal("registry.example.com/virt-handler:v1.0.0"))
		})

		It("should override launcher image in init containers", func() {
			pool := virtv1.VirtHandlerPoolConfig{
				Name:              "gpu-pool",
				VirtLauncherImage: "registry.example.com/virt-launcher:gpu",
				NodeSelector:      map[string]string{"k": "v"},
				Selector:          virtv1.VirtHandlerPoolSelector{DeviceNames: []string{"dev"}},
			}

			ds := components.NewHandlerPoolDaemonSet(newPrimaryHandler(), pool)
			Expect(ds.Spec.Template.Spec.InitContainers[0].Image).To(Equal("registry.example.com/virt-launcher:gpu"))
		})

		It("should merge pool nodeSelector with primary handler's nodeSelector", func() {
			primary := newPrimaryHandler()
			primary.Spec.Template.Spec.NodeSelector = map[string]string{"kubernetes.io/os": "linux"}

			pool := virtv1.VirtHandlerPoolConfig{
				Name:              "gpu-pool",
				VirtLauncherImage: "img",
				NodeSelector:      map[string]string{"nvidia.com/gpu.product": "Tesla-T4"},
				Selector:          virtv1.VirtHandlerPoolSelector{DeviceNames: []string{"dev"}},
			}

			ds := components.NewHandlerPoolDaemonSet(primary, pool)
			Expect(ds.Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{
				"kubernetes.io/os":      "linux",
				"nvidia.com/gpu.product": "Tesla-T4",
			}))
		})

		It("should apply nodeSelector when primary has none", func() {
			pool := virtv1.VirtHandlerPoolConfig{
				Name:              "gpu-pool",
				VirtLauncherImage: "img",
				NodeSelector:      map[string]string{"nvidia.com/gpu.product": "Tesla-T4"},
				Selector:          virtv1.VirtHandlerPoolSelector{DeviceNames: []string{"dev"}},
			}

			ds := components.NewHandlerPoolDaemonSet(newPrimaryHandler(), pool)
			Expect(ds.Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"nvidia.com/gpu.product": "Tesla-T4"}))
		})

		It("should not mutate the primary handler", func() {
			primary := newPrimaryHandler()
			pool := virtv1.VirtHandlerPoolConfig{
				Name:              "gpu-pool",
				VirtHandlerImage:  "custom-handler",
				VirtLauncherImage: "custom-launcher",
				NodeSelector:      map[string]string{"k": "v"},
				Selector:          virtv1.VirtHandlerPoolSelector{DeviceNames: []string{"dev"}},
			}

			components.NewHandlerPoolDaemonSet(primary, pool)

			Expect(primary.Name).To(Equal(components.VirtHandlerName))
			Expect(primary.Spec.Template.Spec.Containers[0].Image).To(Equal("registry.example.com/virt-handler:v1.0.0"))
			Expect(primary.Labels).NotTo(HaveKey(virtv1.HandlerPoolLabel))
		})
	})

	Describe("ApplyPoolAntiAffinityToPrimaryHandler", func() {
		It("should add NotIn expressions for pool nodeSelectors", func() {
			handler := newPrimaryHandler()
			pools := []virtv1.VirtHandlerPoolConfig{
				{
					Name:         "gpu-pool",
					NodeSelector: map[string]string{"nvidia.com/gpu.product": "Tesla-T4"},
				},
			}

			components.ApplyPoolAntiAffinityToPrimaryHandler(handler, pools)

			affinity := handler.Spec.Template.Spec.Affinity
			Expect(affinity).NotTo(BeNil())
			Expect(affinity.NodeAffinity).NotTo(BeNil())
			required := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
			Expect(required).NotTo(BeNil())
			Expect(required.NodeSelectorTerms).To(HaveLen(1))
			Expect(required.NodeSelectorTerms[0].MatchExpressions).To(ContainElement(
				corev1.NodeSelectorRequirement{
					Key:      "nvidia.com/gpu.product",
					Operator: corev1.NodeSelectorOpNotIn,
					Values:   []string{"Tesla-T4"},
				},
			))
		})

		It("should do nothing for empty pools", func() {
			handler := newPrimaryHandler()
			components.ApplyPoolAntiAffinityToPrimaryHandler(handler, nil)
			Expect(handler.Spec.Template.Spec.Affinity).To(BeNil())
		})

		It("should handle multiple pools", func() {
			handler := newPrimaryHandler()
			pools := []virtv1.VirtHandlerPoolConfig{
				{Name: "gpu", NodeSelector: map[string]string{"nvidia.com/gpu.product": "Tesla-T4"}},
				{Name: "fpga", NodeSelector: map[string]string{"fpga.intel.com/present": "true"}},
			}

			components.ApplyPoolAntiAffinityToPrimaryHandler(handler, pools)

			required := handler.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
			Expect(required.NodeSelectorTerms).To(HaveLen(1))
			Expect(required.NodeSelectorTerms[0].MatchExpressions).To(HaveLen(2))
		})
	})
})
