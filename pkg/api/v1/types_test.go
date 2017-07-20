/*
 * This file is part of the kubevirt project
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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package v1

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
)

var _ = Describe("PodSelectors", func() {
	Context("Pod affinity rules", func() {
		var (
			pod      = &v1.Pod{}
			affinity = &v1.Affinity{}
		)

		BeforeEach(func() {
			pod = &v1.Pod{
				Spec: v1.PodSpec{
					NodeName: "testnode",
				},
			}
			affinity = &v1.Affinity{
				NodeAffinity: &v1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
						NodeSelectorTerms: []v1.NodeSelectorTerm{
							{
								MatchExpressions: []v1.NodeSelectorRequirement{
									{
										Key:      "kubernetes.io/hostname",
										Operator: v1.NodeSelectorOpNotIn,
										Values:   []string{pod.Spec.NodeName},
									},
								},
							},
						},
					},
				},
			}
		})

		AfterEach(func() {
		})

		It("should work", func() {
			vm := NewMinimalVM("testvm")
			vm.Status.NodeName = "test-node"
			affinity := AntiAffinityFromVMNode(vm)
			newSelector := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0]
			Expect(newSelector).ToNot(BeNil())
			Expect(len(newSelector.MatchExpressions)).To(Equal(1))
			Expect(len(newSelector.MatchExpressions[0].Values)).To(Equal(1))
			Expect(newSelector.MatchExpressions[0].Values[0]).To(Equal("test-node"))

		})
	})
})

func TestSelectors(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PodSelectors")
}
