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

package virthandler

import (
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GuestTerminated condition", func() {
	It("should return nil without a domain", func() {
		Expect(guestTerminatedCondition(nil)).To(BeNil())
	})

	It("should report false for an active domain without a termination event", func() {
		domain := &api.Domain{Status: api.DomainStatus{Status: api.Running}}

		condition := guestTerminatedCondition(domain)

		Expect(condition).ToNot(BeNil())
		Expect(condition.Type).To(Equal(v1.VirtualMachineInstanceGuestTerminated))
		Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
		Expect(condition.Reason).To(Equal(v1.GuestNotTerminatedReason))
		Expect(condition.Message).To(Equal("Guest has not terminated"))
	})

	It("should report true for a normalized termination event", func() {
		timestamp := metav1.Now()
		domain := &api.Domain{
			Status: api.DomainStatus{
				Status: api.Shutoff,
				TerminationEvent: &api.TerminationEvent{
					Reason:    api.TerminationReasonGuestShutdown,
					Timestamp: timestamp,
				},
			},
		}

		condition := guestTerminatedCondition(domain)

		Expect(condition).ToNot(BeNil())
		Expect(condition.Type).To(Equal(v1.VirtualMachineInstanceGuestTerminated))
		Expect(condition.Status).To(Equal(k8sv1.ConditionTrue))
		Expect(condition.Reason).To(Equal(string(api.TerminationReasonGuestShutdown)))
		Expect(condition.Message).To(Equal(api.TerminationReasonGuestShutdown.Message()))
		Expect(condition.LastProbeTime).To(Equal(timestamp))
		Expect(condition.LastTransitionTime).To(Equal(timestamp))
	})

	It("should return nil for a terminal domain without a normalized termination event", func() {
		domain := &api.Domain{Status: api.DomainStatus{Status: api.Shutoff}}

		Expect(guestTerminatedCondition(domain)).To(BeNil())
	})
})

var _ = Describe("Guest OS termination metrics transition", func() {
	It("should increment for the first true condition", func() {
		condition := &v1.VirtualMachineInstanceCondition{
			Status: k8sv1.ConditionTrue,
			Reason: string(api.TerminationReasonGuestShutdown),
		}

		Expect(shouldIncGuestOSTermination(nil, condition)).To(BeTrue())
	})

	It("should not increment for false conditions", func() {
		condition := &v1.VirtualMachineInstanceCondition{
			Status: k8sv1.ConditionFalse,
			Reason: v1.GuestNotTerminatedReason,
		}

		Expect(shouldIncGuestOSTermination(nil, condition)).To(BeFalse())
	})

	It("should increment when condition changes from false to true", func() {
		oldCondition := &v1.VirtualMachineInstanceCondition{
			Status: k8sv1.ConditionFalse,
			Reason: v1.GuestNotTerminatedReason,
		}
		condition := &v1.VirtualMachineInstanceCondition{
			Status: k8sv1.ConditionTrue,
			Reason: string(api.TerminationReasonPlatformRequestedShutdown),
		}

		Expect(shouldIncGuestOSTermination(oldCondition, condition)).To(BeTrue())
	})

	It("should increment when the true reason changes", func() {
		oldCondition := &v1.VirtualMachineInstanceCondition{
			Status: k8sv1.ConditionTrue,
			Reason: string(api.TerminationReasonGuestShutdown),
		}
		condition := &v1.VirtualMachineInstanceCondition{
			Status: k8sv1.ConditionTrue,
			Reason: string(api.TerminationReasonHostShutdown),
		}

		Expect(shouldIncGuestOSTermination(oldCondition, condition)).To(BeTrue())
	})

	It("should not increment for the same true reason", func() {
		oldCondition := &v1.VirtualMachineInstanceCondition{
			Status: k8sv1.ConditionTrue,
			Reason: string(api.TerminationReasonGuestShutdown),
		}
		condition := &v1.VirtualMachineInstanceCondition{
			Status: k8sv1.ConditionTrue,
			Reason: string(api.TerminationReasonGuestShutdown),
		}

		Expect(shouldIncGuestOSTermination(oldCondition, condition)).To(BeFalse())
	})
})
