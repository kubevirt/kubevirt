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

var _ = Describe("Worker Pool DaemonSets", func() {

	var primaryDS *appsv1.DaemonSet

	BeforeEach(func() {
		primaryDS = &appsv1.DaemonSet{
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
							virtv1.AppLabel: components.VirtHandlerName,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  components.VirtHandlerName,
								Image: "default-handler:latest",
							},
						},
					},
				},
			},
		}
	})

	Context("NewWorkerPoolDaemonSet", func() {

		It("should create a pool DaemonSet with the correct name", func() {
			pool := &virtv1.WorkerPoolConfig{
				Name:         "gpu",
				NodeSelector: map[string]string{"gpu": "true"},
				Selector:     virtv1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/A100"}},
			}
			ds := components.NewWorkerPoolDaemonSet(primaryDS, pool, "kubevirt")
			Expect(ds.Name).To(Equal("virt-handler-gpu"))
		})

		It("should set the worker pool label on the DaemonSet and template", func() {
			pool := &virtv1.WorkerPoolConfig{
				Name:         "gpu",
				NodeSelector: map[string]string{"gpu": "true"},
				Selector:     virtv1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/A100"}},
			}
			ds := components.NewWorkerPoolDaemonSet(primaryDS, pool, "kubevirt")
			Expect(ds.Labels[virtv1.WorkerPoolLabel]).To(Equal("gpu"))
			Expect(ds.Spec.Template.Labels[virtv1.WorkerPoolLabel]).To(Equal("gpu"))
		})

		It("should override the handler image when specified", func() {
			pool := &virtv1.WorkerPoolConfig{
				Name:             "gpu",
				VirtHandlerImage: "custom-handler:v1",
				NodeSelector:     map[string]string{"gpu": "true"},
				Selector:         virtv1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/A100"}},
			}
			ds := components.NewWorkerPoolDaemonSet(primaryDS, pool, "kubevirt")
			Expect(ds.Spec.Template.Spec.Containers[0].Image).To(Equal("custom-handler:v1"))
		})

		It("should keep the default handler image when no override is specified", func() {
			pool := &virtv1.WorkerPoolConfig{
				Name:         "gpu",
				NodeSelector: map[string]string{"gpu": "true"},
				Selector:     virtv1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/A100"}},
			}
			ds := components.NewWorkerPoolDaemonSet(primaryDS, pool, "kubevirt")
			Expect(ds.Spec.Template.Spec.Containers[0].Image).To(Equal("default-handler:latest"))
		})

		It("should set the pool's nodeSelector", func() {
			pool := &virtv1.WorkerPoolConfig{
				Name:         "gpu",
				NodeSelector: map[string]string{"accelerator": "nvidia", "tier": "gpu"},
				Selector:     virtv1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/A100"}},
			}
			ds := components.NewWorkerPoolDaemonSet(primaryDS, pool, "kubevirt")
			Expect(ds.Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"accelerator": "nvidia", "tier": "gpu"}))
		})

		It("should include the pool label in the selector", func() {
			pool := &virtv1.WorkerPoolConfig{
				Name:         "gpu",
				NodeSelector: map[string]string{"gpu": "true"},
				Selector:     virtv1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/A100"}},
			}
			ds := components.NewWorkerPoolDaemonSet(primaryDS, pool, "kubevirt")
			Expect(ds.Spec.Selector.MatchLabels[virtv1.WorkerPoolLabel]).To(Equal("gpu"))
		})
	})

	Context("ApplyPoolAntiAffinityToPrimaryHandler", func() {

		It("should not modify primary when no pools exist", func() {
			original := primaryDS.DeepCopy()
			components.ApplyPoolAntiAffinityToPrimaryHandler(primaryDS, nil)
			Expect(primaryDS.Spec.Template.Spec.Affinity).To(Equal(original.Spec.Template.Spec.Affinity))
		})

		It("should add anti-affinity for pool node selectors", func() {
			pools := []virtv1.WorkerPoolConfig{
				{
					Name:         "gpu",
					NodeSelector: map[string]string{"gpu": "true"},
					Selector:     virtv1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/A100"}},
				},
			}
			components.ApplyPoolAntiAffinityToPrimaryHandler(primaryDS, pools)

			affinity := primaryDS.Spec.Template.Spec.Affinity
			Expect(affinity).NotTo(BeNil())
			Expect(affinity.NodeAffinity).NotTo(BeNil())
			Expect(affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution).NotTo(BeNil())
			terms := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
			Expect(terms).To(HaveLen(1))
			Expect(terms[0].MatchExpressions).To(ContainElement(corev1.NodeSelectorRequirement{
				Key:      "gpu",
				Operator: corev1.NodeSelectorOpNotIn,
				Values:   []string{"true"},
			}))
		})
	})
})
