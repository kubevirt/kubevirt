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
			Entry("[test_cid:31544]pod that has positive condition", HaveConditionMissingOrFalse(k8sv1.PodReady), readyPod, false),
			Entry("[test_cid:26868]pod that has negative condition", HaveConditionMissingOrFalse(k8sv1.PodReady), notReadyPod, true),
			Entry("[test_cid:12569]pod that is missing condition", HaveConditionMissingOrFalse(k8sv1.PodReady), missingReadyPod, true),

			Entry("[test_cid:18618]vmi that has positive condition", HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused), pausedVMI, false),
			Entry("[test_cid:36106]vmi that has negative condition", HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused), notPausedVMI, true),
			Entry("[test_cid:27909]vmi that is missing condition", HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused), missingPausedVMI, true),
			Entry("[test_cid:33201]vmi that is missing conditions", HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused), missingConditionsVMI, true),

			Entry("[test_cid:39899]condition type as string", HaveConditionMissingOrFalse("Paused"), notPausedVMI, true),
			Entry("[test_cid:30654]with nil object", HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused), nilVMI, false),
			Entry("[test_cid:33261]with nil", HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused), nil, false),
			Entry("[test_cid:18229]with nil as condition type", HaveConditionMissingOrFalse(nil), nil, false),
		)
	})

	Context("True", func() {
		DescribeTable("should work with", func(matcher types.GomegaMatcher, obj interface{}, shouldMatch bool) {
			match, err := matcher.Match(obj)
			Expect(err).ToNot(HaveOccurred())
			Expect(match).To(Equal(shouldMatch))
		},
			Entry("[test_cid:24831]pod that has positive condition", HaveConditionTrue(k8sv1.PodReady), readyPod, true),
			Entry("[test_cid:33636]pod that has negative condition", HaveConditionTrue(k8sv1.PodReady), notReadyPod, false),
			Entry("[test_cid:37723]pod that is missing condition", HaveConditionTrue(k8sv1.PodReady), missingReadyPod, false),

			Entry("[test_cid:38525]vmi that has positive condition", HaveConditionTrue(v1.VirtualMachineInstancePaused), pausedVMI, true),
			Entry("[test_cid:29790]vmi that has negative condition", HaveConditionTrue(v1.VirtualMachineInstancePaused), notPausedVMI, false),
			Entry("[test_cid:40828]vmi that is missing condition", HaveConditionTrue(v1.VirtualMachineInstancePaused), missingPausedVMI, false),
			Entry("[test_cid:29820]vmi that is missing conditions", HaveConditionTrue(v1.VirtualMachineInstancePaused), missingConditionsVMI, false),

			Entry("[test_cid:37984]condition type as string", HaveConditionTrue("Paused"), notPausedVMI, false),
			Entry("[test_cid:11312]with nil object", HaveConditionTrue(v1.VirtualMachineInstancePaused), nilVMI, false),
			Entry("[test_cid:35674]with nil", HaveConditionTrue(v1.VirtualMachineInstancePaused), nil, false),
			Entry("[test_cid:23308]with nil as condition type", HaveConditionTrue(nil), nil, false),
		)
	})

	Context("False", func() {
		DescribeTable("should work with", func(matcher types.GomegaMatcher, obj interface{}, shouldMatch bool) {
			match, err := matcher.Match(obj)
			Expect(err).ToNot(HaveOccurred())
			Expect(match).To(Equal(shouldMatch))
		},
			Entry("[test_cid:17560]pod that has positive condition", HaveConditionFalse(k8sv1.PodReady), readyPod, false),
			Entry("[test_cid:20561]pod that has negative condition", HaveConditionFalse(k8sv1.PodReady), notReadyPod, true),
			Entry("[test_cid:26728]pod that is missing condition", HaveConditionFalse(k8sv1.PodReady), missingReadyPod, false),

			Entry("[test_cid:27588]vmi that has positive condition", HaveConditionFalse(v1.VirtualMachineInstancePaused), pausedVMI, false),
			Entry("[test_cid:10561]vmi that has negative condition", HaveConditionFalse(v1.VirtualMachineInstancePaused), notPausedVMI, true),
			Entry("[test_cid:19061]vmi that is missing condition", HaveConditionFalse(v1.VirtualMachineInstancePaused), missingPausedVMI, false),
			Entry("[test_cid:28466]vmi that is missing conditions", HaveConditionFalse(v1.VirtualMachineInstancePaused), missingConditionsVMI, false),

			Entry("[test_cid:12884]condition type as string", HaveConditionFalse("Paused"), notPausedVMI, true),
			Entry("[test_cid:37366]with nil object", HaveConditionFalse(v1.VirtualMachineInstancePaused), nilVMI, false),
			Entry("[test_cid:21081]with nil", HaveConditionFalse(v1.VirtualMachineInstancePaused), nil, false),
			Entry("[test_cid:39824]with nil as condition type", HaveConditionFalse(nil), nil, false),
		)
	})

})
