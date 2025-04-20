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
 */

package matcher

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
)

const (
	someMsg         = "Some message"
	someOtherMsg    = "Some other message"
	someReason      = "SomeReason"
	someOtherReason = "SomeOtherReason"
)

var _ = Describe("Condition matcher", func() {
	readyPod := &k8sv1.Pod{
		Status: k8sv1.PodStatus{
			Conditions: []k8sv1.PodCondition{
				{
					Type:    k8sv1.PodReady,
					Status:  k8sv1.ConditionTrue,
					Message: someMsg,
					Reason:  someReason,
				},
			},
		},
	}
	notReadyPod := readyPod.DeepCopy()
	notReadyPod.Status.Conditions[0].Status = k8sv1.ConditionFalse

	missingReadyPod := readyPod.DeepCopy()
	missingReadyPod.Status.Conditions = []k8sv1.PodCondition{}

	pausedVMI := &v1.VirtualMachineInstance{
		Status: v1.VirtualMachineInstanceStatus{
			Conditions: []v1.VirtualMachineInstanceCondition{
				{
					Type:    v1.VirtualMachineInstancePaused,
					Status:  k8sv1.ConditionTrue,
					Message: someMsg,
					Reason:  someReason,
				},
			},
		},
	}

	notPausedVMI := pausedVMI.DeepCopy()
	notPausedVMI.Status.Conditions[0].Status = k8sv1.ConditionFalse

	missingPausedVMI := pausedVMI.DeepCopy()
	missingPausedVMI.Status.Conditions = []v1.VirtualMachineInstanceCondition{
		{
			Type:    v1.VirtualMachineInstanceAgentConnected,
			Status:  k8sv1.ConditionTrue,
			Message: someMsg,
			Reason:  someReason,
		},
	}

	missingConditionsVMI := pausedVMI.DeepCopy()
	missingConditionsVMI.Status.Conditions = []v1.VirtualMachineInstanceCondition{}

	var nilVMI *v1.VirtualMachineInstance = nil

	Context("Missing or false", func() {
		DescribeTable("should work with", func(matcher types.GomegaMatcher, obj interface{}, shouldMatch bool) {
			match, err := matcher.Match(obj)
			Expect(err).ToNot(HaveOccurred())
			Expect(match).To(Equal(shouldMatch))
		},
			Entry("pod that has positive condition", HaveConditionMissingOrFalse(k8sv1.PodReady), readyPod, false),
			Entry("pod that has negative condition", HaveConditionMissingOrFalse(k8sv1.PodReady), notReadyPod, true),
			Entry("pod that is missing condition", HaveConditionMissingOrFalse(k8sv1.PodReady), missingReadyPod, true),

			Entry("vmi that has positive condition", HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused), pausedVMI, false),
			Entry("vmi that has negative condition", HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused), notPausedVMI, true),
			Entry("vmi that is missing condition", HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused), missingPausedVMI, true),
			Entry("vmi that is missing conditions", HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused), missingConditionsVMI, true),

			Entry("condition type as string", HaveConditionMissingOrFalse("Paused"), notPausedVMI, true),
			Entry("with nil object", HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused), nilVMI, false),
			Entry("with nil", HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused), nil, false),
			Entry("with nil as condition type", HaveConditionMissingOrFalse(nil), nil, false),
		)
	})

	Context("True", func() {
		DescribeTable("should work with", func(matcher types.GomegaMatcher, obj interface{}, shouldMatch bool) {
			match, err := matcher.Match(obj)
			Expect(err).ToNot(HaveOccurred())
			Expect(match).To(Equal(shouldMatch))
		},
			Entry("pod that has positive condition", HaveConditionTrue(k8sv1.PodReady), readyPod, true),
			Entry("pod that has negative condition", HaveConditionTrue(k8sv1.PodReady), notReadyPod, false),
			Entry("pod that is missing condition", HaveConditionTrue(k8sv1.PodReady), missingReadyPod, false),

			Entry("vmi that has positive condition", HaveConditionTrue(v1.VirtualMachineInstancePaused), pausedVMI, true),
			Entry("vmi that has negative condition", HaveConditionTrue(v1.VirtualMachineInstancePaused), notPausedVMI, false),
			Entry("vmi that is missing condition", HaveConditionTrue(v1.VirtualMachineInstancePaused), missingPausedVMI, false),
			Entry("vmi that is missing conditions", HaveConditionTrue(v1.VirtualMachineInstancePaused), missingConditionsVMI, false),

			Entry("condition type as string", HaveConditionTrue("Paused"), notPausedVMI, false),
			Entry("with nil object", HaveConditionTrue(v1.VirtualMachineInstancePaused), nilVMI, false),
			Entry("with nil", HaveConditionTrue(v1.VirtualMachineInstancePaused), nil, false),
			Entry("with nil as condition type", HaveConditionTrue(nil), nil, false),
		)
	})

	Context("False", func() {
		DescribeTable("should work with", func(matcher types.GomegaMatcher, obj interface{}, shouldMatch bool) {
			match, err := matcher.Match(obj)
			Expect(err).ToNot(HaveOccurred())
			Expect(match).To(Equal(shouldMatch))
		},
			Entry("pod that has positive condition", HaveConditionFalse(k8sv1.PodReady), readyPod, false),
			Entry("pod that has negative condition", HaveConditionFalse(k8sv1.PodReady), notReadyPod, true),
			Entry("pod that is missing condition", HaveConditionFalse(k8sv1.PodReady), missingReadyPod, false),

			Entry("vmi that has positive condition", HaveConditionFalse(v1.VirtualMachineInstancePaused), pausedVMI, false),
			Entry("vmi that has negative condition", HaveConditionFalse(v1.VirtualMachineInstancePaused), notPausedVMI, true),
			Entry("vmi that is missing condition", HaveConditionFalse(v1.VirtualMachineInstancePaused), missingPausedVMI, false),
			Entry("vmi that is missing conditions", HaveConditionFalse(v1.VirtualMachineInstancePaused), missingConditionsVMI, false),

			Entry("condition type as string", HaveConditionFalse("Paused"), notPausedVMI, true),
			Entry("with nil object", HaveConditionFalse(v1.VirtualMachineInstancePaused), nilVMI, false),
			Entry("with nil", HaveConditionFalse(v1.VirtualMachineInstancePaused), nil, false),
			Entry("with nil as condition type", HaveConditionFalse(nil), nil, false),
		)
	})

	Context("False with message", func() {
		DescribeTable("should work with", func(matcher types.GomegaMatcher, obj interface{}, shouldMatch bool) {
			match, err := matcher.Match(obj)
			Expect(err).ToNot(HaveOccurred())
			Expect(match).To(Equal(shouldMatch))
		},
			Entry("pod that has positive condition and expectedMessage", HaveConditionFalseWithMessage(k8sv1.PodReady, someMsg), readyPod, false),
			Entry("pod that has positive condition and not expectedMessage", HaveConditionFalseWithMessage(k8sv1.PodReady, someOtherMsg), readyPod, false),
			Entry("pod that has negative condition and expectedMessage", HaveConditionFalseWithMessage(k8sv1.PodReady, someMsg), notReadyPod, true),
			Entry("pod that has negative condition and not expectedMessage", HaveConditionFalseWithMessage(k8sv1.PodReady, someOtherMsg), notReadyPod, false),
			Entry("pod that is missing condition", HaveConditionFalseWithMessage(k8sv1.PodReady, someMsg), missingReadyPod, false),

			Entry("vmi that has positive condition and expectedMessage", HaveConditionFalseWithMessage(v1.VirtualMachineInstancePaused, someMsg), pausedVMI, false),
			Entry("vmi that has positive condition and not expectedMessage", HaveConditionFalseWithMessage(v1.VirtualMachineInstancePaused, someOtherMsg), pausedVMI, false),
			Entry("vmi that has negative condition and expectedMessage", HaveConditionFalseWithMessage(v1.VirtualMachineInstancePaused, someMsg), notPausedVMI, true),
			Entry("vmi that has negative condition and not expectedMessage", HaveConditionFalseWithMessage(v1.VirtualMachineInstancePaused, someOtherMsg), notPausedVMI, false),
			Entry("vmi that is missing condition", HaveConditionFalseWithMessage(v1.VirtualMachineInstancePaused, someMsg), missingPausedVMI, false),
			Entry("vmi that is missing conditions", HaveConditionFalseWithMessage(v1.VirtualMachineInstancePaused, someMsg), missingConditionsVMI, false),

			Entry("condition type as string", HaveConditionFalseWithMessage("Paused", someMsg), notPausedVMI, true),
			Entry("with nil object", HaveConditionFalseWithMessage(v1.VirtualMachineInstancePaused, someMsg), nilVMI, false),
			Entry("with nil", HaveConditionFalseWithMessage(v1.VirtualMachineInstancePaused, someMsg), nil, false),
			Entry("with nil as condition type", HaveConditionFalseWithMessage(nil, someMsg), nil, false),
		)
	})

	Context("True with message", func() {
		DescribeTable("should work with", func(matcher types.GomegaMatcher, obj interface{}, shouldMatch bool) {
			match, err := matcher.Match(obj)
			Expect(err).ToNot(HaveOccurred())
			Expect(match).To(Equal(shouldMatch))
		},
			Entry("pod that has positive condition and expectedMessage", HaveConditionTrueWithMessage(k8sv1.PodReady, someMsg), readyPod, true),
			Entry("pod that has positive condition and not expectedMessage", HaveConditionTrueWithMessage(k8sv1.PodReady, someOtherMsg), readyPod, false),
			Entry("pod that has negative condition and expectedMessage", HaveConditionTrueWithMessage(k8sv1.PodReady, someMsg), notReadyPod, false),
			Entry("pod that has negative condition and not expectedMessage", HaveConditionTrueWithMessage(k8sv1.PodReady, someOtherMsg), notReadyPod, false),
			Entry("pod that is missing condition", HaveConditionTrueWithMessage(k8sv1.PodReady, someMsg), missingReadyPod, false),

			Entry("vmi that has positive condition and expectedMessage", HaveConditionTrueWithMessage(v1.VirtualMachineInstancePaused, someMsg), pausedVMI, true),
			Entry("vmi that has positive condition and not expectedMessage", HaveConditionTrueWithMessage(v1.VirtualMachineInstancePaused, someOtherMsg), pausedVMI, false),
			Entry("vmi that has negative condition and expectedMessage", HaveConditionTrueWithMessage(v1.VirtualMachineInstancePaused, someMsg), notPausedVMI, false),
			Entry("vmi that has negative condition and not expectedMessage", HaveConditionTrueWithMessage(v1.VirtualMachineInstancePaused, someOtherMsg), notPausedVMI, false),
			Entry("vmi that is missing condition", HaveConditionTrueWithMessage(v1.VirtualMachineInstancePaused, someMsg), missingPausedVMI, false),
			Entry("vmi that is missing conditions", HaveConditionTrueWithMessage(v1.VirtualMachineInstancePaused, someMsg), missingConditionsVMI, false),

			Entry("condition type as string", HaveConditionTrueWithMessage("Paused", someMsg), pausedVMI, true),
			Entry("with nil object", HaveConditionTrueWithMessage(v1.VirtualMachineInstancePaused, someMsg), nilVMI, false),
			Entry("with nil", HaveConditionTrueWithMessage(v1.VirtualMachineInstancePaused, someMsg), nil, false),
			Entry("with nil as condition type", HaveConditionTrueWithMessage(nil, someMsg), nil, false),
		)
	})

	Context("True with reason", func() {
		DescribeTable("should work with", func(matcher types.GomegaMatcher, obj interface{}, shouldMatch bool) {
			match, err := matcher.Match(obj)
			Expect(err).ToNot(HaveOccurred())
			Expect(match).To(Equal(shouldMatch))
		},
			Entry("pod that has positive condition and expectedReason", HaveConditionTrueWithReason(k8sv1.PodReady, someReason), readyPod, true),
			Entry("pod that has positive condition and not expectedReason", HaveConditionTrueWithReason(k8sv1.PodReady, someOtherReason), readyPod, false),
			Entry("pod that has negative condition and expectedReason", HaveConditionTrueWithReason(k8sv1.PodReady, someReason), notReadyPod, false),
			Entry("pod that has negative condition and not expectedReason", HaveConditionTrueWithReason(k8sv1.PodReady, someOtherReason), notReadyPod, false),
			Entry("pod that is missing condition", HaveConditionTrueWithReason(k8sv1.PodReady, someReason), missingReadyPod, false),

			Entry("vmi that has positive condition and expectedReason", HaveConditionTrueWithReason(v1.VirtualMachineInstancePaused, someReason), pausedVMI, true),
			Entry("vmi that has positive condition and not expectedReason", HaveConditionTrueWithReason(v1.VirtualMachineInstancePaused, someOtherReason), pausedVMI, false),
			Entry("vmi that has negative condition and expectedReason", HaveConditionTrueWithReason(v1.VirtualMachineInstancePaused, someReason), notPausedVMI, false),
			Entry("vmi that has negative condition and not expectedReason", HaveConditionTrueWithReason(v1.VirtualMachineInstancePaused, someOtherReason), notPausedVMI, false),
			Entry("vmi that is missing condition", HaveConditionTrueWithReason(v1.VirtualMachineInstancePaused, someReason), missingPausedVMI, false),
			Entry("vmi that is missing conditions", HaveConditionTrueWithReason(v1.VirtualMachineInstancePaused, someReason), missingConditionsVMI, false),

			Entry("condition type as string", HaveConditionTrueWithReason("Paused", someReason), pausedVMI, true),
			Entry("with nil object", HaveConditionTrueWithReason(v1.VirtualMachineInstancePaused, someReason), nilVMI, false),
			Entry("with nil", HaveConditionTrueWithReason(v1.VirtualMachineInstancePaused, someReason), nil, false),
			Entry("with nil as condition type", HaveConditionTrueWithReason(nil, someReason), nil, false),
		)
	})
})
