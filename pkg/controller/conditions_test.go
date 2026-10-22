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
package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("VirtualMachineInstance ConditionManager", func() {

	var vmi *v1.VirtualMachineInstance
	var cm *VirtualMachineInstanceConditionManager
	var pc1 *v12.PodCondition
	var pc2 *v12.PodCondition

	BeforeEach(func() {
		vmi = libvmi.New()

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

	DescribeTable("should clear stale readiness for a final VMI", func(phase v1.VirtualMachineInstancePhase, status v12.ConditionStatus) {
		vmi.Status.Phase = phase
		if status != "" {
			cm.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{Type: v1.VirtualMachineInstanceReady, Status: status})
		}
		cm.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{Type: v1.VirtualMachineInstanceSynchronized, Status: v12.ConditionTrue})

		cm.SyncReadyConditionForFinalVMI(vmi)

		condition := cm.GetCondition(vmi, v1.VirtualMachineInstanceReady)
		Expect(condition).ToNot(BeNil())
		Expect(condition.Status).To(Equal(v12.ConditionFalse))
		Expect(condition.Reason).To(Equal(v1.GuestNotRunningReason))
		Expect(condition.LastTransitionTime.IsZero()).To(BeFalse())
		Expect(cm.HasConditionWithStatus(vmi, v1.VirtualMachineInstanceSynchronized, v12.ConditionTrue)).To(BeTrue())
		conditions := vmi.DeepCopy().Status.Conditions
		cm.SyncReadyConditionForFinalVMI(vmi)
		Expect(vmi.Status.Conditions).To(Equal(conditions))
	},
		Entry("Succeeded with True readiness", v1.Succeeded, v12.ConditionTrue),
		Entry("Succeeded with Unknown readiness", v1.Succeeded, v12.ConditionUnknown),
		Entry("Succeeded with missing readiness", v1.Succeeded, v12.ConditionStatus("")),
		Entry("Failed with True readiness", v1.Failed, v12.ConditionTrue),
		Entry("Failed with Unknown readiness", v1.Failed, v12.ConditionUnknown),
		Entry("Failed with missing readiness", v1.Failed, v12.ConditionStatus("")),
	)

	DescribeTable("should preserve an existing False Ready condition for a final VMI", func(phase v1.VirtualMachineInstancePhase) {
		vmi.Status.Phase = phase
		condition := &v1.VirtualMachineInstanceCondition{
			Type:               v1.VirtualMachineInstanceReady,
			Status:             v12.ConditionFalse,
			Reason:             v1.PodTerminatingReason,
			Message:            "virt-launcher pod is terminating",
			LastProbeTime:      metav1.Now(),
			LastTransitionTime: metav1.Now(),
		}
		cm.UpdateCondition(vmi, condition)

		cm.SyncReadyConditionForFinalVMI(vmi)

		Expect(cm.GetCondition(vmi, v1.VirtualMachineInstanceReady)).To(Equal(condition))
	},
		Entry("Succeeded", v1.Succeeded),
		Entry("Failed", v1.Failed),
	)

	DescribeTable("should leave readiness unchanged for a non-final VMI", func(phase v1.VirtualMachineInstancePhase) {
		vmi.Status.Phase = phase
		cm.SyncReadyConditionForFinalVMI(vmi)
		Expect(vmi.Status.Conditions).To(BeEmpty())
		condition := &v1.VirtualMachineInstanceCondition{Type: v1.VirtualMachineInstanceReady, Status: v12.ConditionTrue}
		cm.UpdateCondition(vmi, condition)
		cm.SyncReadyConditionForFinalVMI(vmi)
		Expect(cm.GetCondition(vmi, v1.VirtualMachineInstanceReady)).To(Equal(condition))
	},
		Entry("Pending", v1.Pending),
		Entry("Scheduled", v1.Scheduled),
		Entry("Running", v1.Running),
	)

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
