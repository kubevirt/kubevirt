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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package api

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("PodSelectors", func() {
	It("should detect if VMIs want guaranteed QOS", func() {
		vmi := NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Resources = v1.ResourceRequirements{
			Requests: k8sv1.ResourceList{
				k8sv1.ResourceCPU:    resource.MustParse("1"),
				k8sv1.ResourceMemory: resource.MustParse("64M"),
			},
			Limits: k8sv1.ResourceList{
				k8sv1.ResourceCPU:    resource.MustParse("1"),
				k8sv1.ResourceMemory: resource.MustParse("64M"),
			},
		}
		Expect(vmi.WantsToHaveQOSGuaranteed()).To(BeTrue())
	})

	It("should detect if VMIs don't want guaranteed QOS", func() {
		vmi := NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Resources = v1.ResourceRequirements{
			Requests: k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64M"),
			},
			Limits: k8sv1.ResourceList{
				k8sv1.ResourceCPU:    resource.MustParse("1"),
				k8sv1.ResourceMemory: resource.MustParse("64M"),
			},
		}
		Expect(vmi.WantsToHaveQOSGuaranteed()).To(BeFalse())
	})

	Context("Pod affinity rules", func() {
		It("should work", func() {
			vmi := NewMinimalVMI("testvmi")
			vmi.Status.NodeName = "test-node"
			pod := &k8sv1.Pod{}
			affinity := v1.UpdateAntiAffinityFromVMINode(pod, vmi)
			newSelector := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0]
			Expect(newSelector).ToNot(BeNil())
			Expect(newSelector.MatchExpressions).To(HaveLen(1))
			Expect(newSelector.MatchExpressions[0].Values).To(HaveLen(1))
			Expect(newSelector.MatchExpressions[0].Values[0]).To(Equal("test-node"))
		})

		It("should merge", func() {
			vmi := NewMinimalVMI("testvmi")
			vmi.Status.NodeName = "test-node"

			existingTerm := k8sv1.NodeSelectorTerm{}
			secondExistingTerm := k8sv1.NodeSelectorTerm{
				MatchExpressions: []k8sv1.NodeSelectorRequirement{
					{},
				},
			}

			pod := &k8sv1.Pod{
				Spec: k8sv1.PodSpec{
					Affinity: &k8sv1.Affinity{
						PodAffinity: &k8sv1.PodAffinity{},
						NodeAffinity: &k8sv1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
								NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
									existingTerm,
									secondExistingTerm,
								},
							},
						},
					},
				},
			}

			affinity := v1.UpdateAntiAffinityFromVMINode(pod, vmi)

			Expect(affinity.NodeAffinity).ToNot(BeNil())
			Expect(affinity.PodAffinity).ToNot(BeNil())
			Expect(affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution).ToNot(BeNil())
			Expect(affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).ToNot(BeNil())
			Expect(affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(HaveLen(2))

			selector := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0]
			Expect(selector).ToNot(BeNil())
			Expect(selector.MatchExpressions).To(HaveLen(1))
			Expect(selector.MatchExpressions[0].Values).To(HaveLen(1))
			Expect(selector.MatchExpressions[0].Values[0]).To(Equal("test-node"))

			selector = affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[1]
			Expect(selector).ToNot(BeNil())
			Expect(selector.MatchExpressions).To(HaveLen(2))
			Expect(selector.MatchExpressions[1].Values).To(HaveLen(1))
			Expect(selector.MatchExpressions[1].Values[0]).To(Equal("test-node"))
		})
	})

	Context("RunStrategy Rules", func() {
		vm := v1.VirtualMachine{}
		BeforeEach(func() {
			vm = v1.VirtualMachine{
				TypeMeta:   k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachine"},
				ObjectMeta: k8smetav1.ObjectMeta{Name: "testvm"},
			}
		})

		It("should fail if both Running and RunStrategy are defined", func() {
			notRunning := false
			runStrategy := v1.RunStrategyAlways
			vm.Spec.Running = &notRunning
			vm.Spec.RunStrategy = &runStrategy

			_, err := vm.RunStrategy()
			Expect(err).To(HaveOccurred())
		})

		It("should convert running=false to Halted", func() {
			notRunning := false
			vm.Spec.Running = &notRunning
			vm.Spec.RunStrategy = nil

			runStrategy, err := vm.RunStrategy()
			Expect(err).ToNot(HaveOccurred())
			Expect(runStrategy).To(Equal(v1.RunStrategyHalted))
		})

		It("should convert running=true to RunStrategyAlways", func() {
			running := true
			vm.Spec.Running = &running
			vm.Spec.RunStrategy = nil

			runStrategy, err := vm.RunStrategy()
			Expect(err).ToNot(HaveOccurred())
			Expect(runStrategy).To(Equal(v1.RunStrategyAlways))
		})

		DescribeTable("should return RunStrategy", func(runStrategy v1.VirtualMachineRunStrategy) {
			vm.Spec.Running = nil
			vm.Spec.RunStrategy = &runStrategy

			newRunStrategy, err := vm.RunStrategy()
			Expect(err).ToNot(HaveOccurred())
			Expect(newRunStrategy).To(Equal(runStrategy))
		},
			Entry(string(v1.RunStrategyAlways), v1.RunStrategyAlways),
			Entry(string(v1.RunStrategyHalted), v1.RunStrategyHalted),
			Entry(string(v1.RunStrategyManual), v1.RunStrategyManual),
			Entry(string(v1.RunStrategyRerunOnFailure), v1.RunStrategyRerunOnFailure),
		)

		It("should default to RunStrategyHalted", func() {
			vm.Spec.Running = nil
			vm.Spec.RunStrategy = nil

			runStrategy, err := vm.RunStrategy()
			Expect(err).ToNot(HaveOccurred())
			Expect(runStrategy).To(Equal(v1.RunStrategyHalted))
		})
	})
})
