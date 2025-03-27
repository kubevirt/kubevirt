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
 * Copyright 2019 Red Hat, Inc.
 *
 */
package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/api"

	v12 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("VirtualMachineInstance ConditionManager", func() {
	var vmi *v1.VirtualMachineInstance
	var cm *VirtualMachineInstanceConditionManager
	var pc1 *v12.PodCondition
	var pc2 *v12.PodCondition

	BeforeEach(func() {
		vmi = api.NewMinimalVMI("test")

		pc1 = &v12.PodCondition{
			Type:   v12.PodScheduled,
			Status: v12.ConditionFalse,
		}
		pc2 = &v12.PodCondition{
			Type:   v12.PodScheduled,
			Status: v12.ConditionTrue,
		}

		cm = NewVirtualMachineInstanceConditionManager()
	})

	When("Adding a condition", func() {
		It("should report condition available", func() {
			cm.AddPodCondition(vmi, pc1)
			Expect(cm.HasCondition(vmi, v1.VirtualMachineInstanceConditionType(pc1.Type))).To(BeTrue())
		})

		It("should report different condition not available", func() {
			cm.AddPodCondition(vmi, pc1)
			Expect(cm.HasCondition(vmi, v1.VirtualMachineInstanceConditionType(v12.PodInitialized))).To(BeFalse())
		})

		When("adding a 2nd condition of same type", func() {
			It("should only have 1 condition", func() {
				cm.AddPodCondition(vmi, pc1)
				cm.AddPodCondition(vmi, pc2)
				Expect(vmi.Status.Conditions).To(HaveLen(1))
			})
		})
	})

	When("VMI is nil", func() {
		It("should gracefully report condition not available", func() {
			var vmi2 *v1.VirtualMachineInstance
			Expect(cm.HasCondition(vmi2, v1.VirtualMachineInstanceConditionType(pc1.Type))).To(BeFalse())
		})
	})

	When("Updating a condition", func() {
		var vc1 *v1.VirtualMachineInstanceCondition
		BeforeEach(func() {
			vc1 = &v1.VirtualMachineInstanceCondition{
				Type:    v1.VirtualMachineInstanceReady,
				Status:  v12.ConditionFalse,
				Reason:  "A reason",
				Message: "A message",
			}

			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{*vc1}
		})

		It("should update the condition if status has changed", func() {
			vc2 := &v1.VirtualMachineInstanceCondition{
				Type:   v1.VirtualMachineInstanceReady,
				Status: v12.ConditionTrue,
			}

			cm.UpdateCondition(vmi, vc2)
			Expect(vmi.Status.Conditions).To(HaveLen(1))
			Expect(cm.GetCondition(vmi, vc1.Type)).To(Equal(vc2))
		})

		It("should update the condition if the reason has changed", func() {
			vc2 := &v1.VirtualMachineInstanceCondition{
				Type:    v1.VirtualMachineInstanceReady,
				Status:  v12.ConditionFalse,
				Reason:  "A different reason",
				Message: "A different message",
			}

			cm.UpdateCondition(vmi, vc2)
			Expect(vmi.Status.Conditions).To(HaveLen(1))
			Expect(cm.GetCondition(vmi, vc1.Type)).To(Equal(vc2))
		})

		It("shouldn't update the condition if both status and reason hasn't changed", func() {
			vc2 := &v1.VirtualMachineInstanceCondition{
				Type:    v1.VirtualMachineInstanceReady,
				Status:  v12.ConditionFalse,
				Reason:  "A reason",
				Message: "A different message",
			}

			cm.UpdateCondition(vmi, vc2)
			Expect(vmi.Status.Conditions).To(HaveLen(1))
			Expect(cm.GetCondition(vmi, vc1.Type)).To(Equal(vc1))
		})
	})
})
