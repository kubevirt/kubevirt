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

package v1

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/log"
)

var _ = Describe("PodSelectors", func() {

	It("should detect if VMIs want guaranteed QOS", func() {
		vmi := NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Resources = ResourceRequirements{
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
		vmi.Spec.Domain.Resources = ResourceRequirements{
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
			affinity := UpdateAntiAffinityFromVMINode(pod, vmi)
			newSelector := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0]
			Expect(newSelector).ToNot(BeNil())
			Expect(len(newSelector.MatchExpressions)).To(Equal(1))
			Expect(len(newSelector.MatchExpressions[0].Values)).To(Equal(1))
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

			affinity := UpdateAntiAffinityFromVMINode(pod, vmi)

			Expect(affinity.NodeAffinity).ToNot(BeNil())
			Expect(affinity.PodAffinity).ToNot(BeNil())
			Expect(affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution).ToNot(BeNil())
			Expect(affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).ToNot(BeNil())
			Expect(affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(HaveLen(2))

			selector := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0]
			Expect(selector).ToNot(BeNil())
			Expect(len(selector.MatchExpressions)).To(Equal(1))
			Expect(len(selector.MatchExpressions[0].Values)).To(Equal(1))
			Expect(selector.MatchExpressions[0].Values[0]).To(Equal("test-node"))

			selector = affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[1]
			Expect(selector).ToNot(BeNil())
			Expect(len(selector.MatchExpressions)).To(Equal(2))
			Expect(len(selector.MatchExpressions[1].Values)).To(Equal(1))
			Expect(selector.MatchExpressions[1].Values[0]).To(Equal("test-node"))
		})
	})

	Context("RunStrategy Rules", func() {
		var vm = VirtualMachine{}
		BeforeEach(func() {
			vm = VirtualMachine{
				TypeMeta:   k8smetav1.TypeMeta{APIVersion: GroupVersion.String(), Kind: "VirtualMachine"},
				ObjectMeta: k8smetav1.ObjectMeta{Name: "testvm"}}
		})

		It("should fail if both Running and RunStrategy are defined", func() {
			notRunning := false
			runStrategy := RunStrategyAlways
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
			Expect(runStrategy).To(Equal(RunStrategyHalted))
		})

		It("should convert running=true to RunStrategyAlways", func() {
			running := true
			vm.Spec.Running = &running
			vm.Spec.RunStrategy = nil

			runStrategy, err := vm.RunStrategy()
			Expect(err).ToNot(HaveOccurred())
			Expect(runStrategy).To(Equal(RunStrategyAlways))
		})

		table.DescribeTable("should return RunStrategy", func(runStrategy VirtualMachineRunStrategy) {
			vm.Spec.Running = nil
			vm.Spec.RunStrategy = &runStrategy

			newRunStrategy, err := vm.RunStrategy()
			Expect(err).ToNot(HaveOccurred())
			Expect(newRunStrategy).To(Equal(runStrategy))
		},
			table.Entry(string(RunStrategyAlways), RunStrategyAlways),
			table.Entry(string(RunStrategyHalted), RunStrategyHalted),
			table.Entry(string(RunStrategyManual), RunStrategyManual),
			table.Entry(string(RunStrategyRerunOnFailure), RunStrategyRerunOnFailure),
		)

		It("should default to RunStrategyHalted", func() {
			vm.Spec.Running = nil
			vm.Spec.RunStrategy = nil

			runStrategy, err := vm.RunStrategy()
			Expect(err).ToNot(HaveOccurred())
			Expect(runStrategy).To(Equal(RunStrategyHalted))
		})
	})
})

func TestSelectors(t *testing.T) {
	log.Log.SetIOWriter(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "PodSelectors")
}
