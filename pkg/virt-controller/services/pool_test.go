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

package services

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Pool-aware launcher image selection", func() {

	Context("overrideLauncherImage", func() {
		It("should replace default launcher image in containers", func() {
			pod := &k8sv1.Pod{
				Spec: k8sv1.PodSpec{
					Containers: []k8sv1.Container{
						{Name: "compute", Image: "default-launcher:v1"},
						{Name: "sidecar", Image: "other-image:v1"},
					},
					InitContainers: []k8sv1.Container{
						{Name: "init", Image: "default-launcher:v1"},
					},
				},
			}
			overrideLauncherImage(pod, "default-launcher:v1", "custom-launcher:v2")
			Expect(pod.Spec.Containers[0].Image).To(Equal("custom-launcher:v2"))
			Expect(pod.Spec.Containers[1].Image).To(Equal("other-image:v1"))
			Expect(pod.Spec.InitContainers[0].Image).To(Equal("custom-launcher:v2"))
		})

		It("should not modify anything when default equals override", func() {
			pod := &k8sv1.Pod{
				Spec: k8sv1.PodSpec{
					Containers: []k8sv1.Container{
						{Name: "compute", Image: "launcher:v1"},
					},
				},
			}
			overrideLauncherImage(pod, "launcher:v1", "launcher:v1")
			Expect(pod.Spec.Containers[0].Image).To(Equal("launcher:v1"))
		})
	})

	Context("mergePoolNodeSelector", func() {
		It("should add node affinity when none exists", func() {
			pod := &k8sv1.Pod{
				Spec: k8sv1.PodSpec{},
			}
			mergePoolNodeSelector(pod, map[string]string{"gpu": "true"})
			Expect(pod.Spec.Affinity).NotTo(BeNil())
			Expect(pod.Spec.Affinity.NodeAffinity).NotTo(BeNil())
			ns := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
			Expect(ns).NotTo(BeNil())
			Expect(ns.NodeSelectorTerms).To(HaveLen(1))
			Expect(ns.NodeSelectorTerms[0].MatchExpressions).To(ContainElement(k8sv1.NodeSelectorRequirement{
				Key:      "gpu",
				Operator: k8sv1.NodeSelectorOpIn,
				Values:   []string{"true"},
			}))
		})

		It("should merge into existing node affinity", func() {
			pod := &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: k8sv1.PodSpec{
					Affinity: &k8sv1.Affinity{
						NodeAffinity: &k8sv1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
								NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
									{
										MatchExpressions: []k8sv1.NodeSelectorRequirement{
											{Key: "existing", Operator: k8sv1.NodeSelectorOpIn, Values: []string{"val"}},
										},
									},
								},
							},
						},
					},
				},
			}
			mergePoolNodeSelector(pod, map[string]string{"gpu": "true"})
			ns := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
			Expect(ns.NodeSelectorTerms).To(HaveLen(1))
			Expect(ns.NodeSelectorTerms[0].MatchExpressions).To(HaveLen(2))
		})

		It("should do nothing when nodeSelector is empty", func() {
			pod := &k8sv1.Pod{}
			mergePoolNodeSelector(pod, nil)
			Expect(pod.Spec.Affinity).To(BeNil())
		})
	})

	Context("pool annotation", func() {
		It("should annotate pod with pool name when pool matches", func() {
			pod := &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			}
			pod.Annotations[v1.WorkerPoolLabel] = "gpu-pool"
			Expect(pod.Annotations[v1.WorkerPoolLabel]).To(Equal("gpu-pool"))
		})
	})
})
