package matcher

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Condition matcher", func() {
	readyPod := &k8sv1.Pod{
		Status: k8sv1.PodStatus{
			Conditions: []k8sv1.PodCondition{
				{
					Type:   k8sv1.PodReady,
					Status: k8sv1.ConditionTrue,
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
					Type:   v1.VirtualMachineInstancePaused,
					Status: k8sv1.ConditionTrue,
				},
			},
		},
	}

	notPausedVMI := pausedVMI.DeepCopy()
	notPausedVMI.Status.Conditions[0].Status = k8sv1.ConditionFalse

	missingPausedVMI := pausedVMI.DeepCopy()
	missingPausedVMI.Status.Conditions = []v1.VirtualMachineInstanceCondition{
		{
			Type:   v1.VirtualMachineInstanceAgentConnected,
			Status: k8sv1.ConditionTrue,
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

})
