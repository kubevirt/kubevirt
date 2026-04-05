/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package virtcontroller

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("VMI phase transition time histogram", func() {
	Context("Transition Time calculations", func() {
		DescribeTable("time diff calculations", func(expectedVal float64, curPhase, oldPhase v1.VirtualMachineInstancePhase) {
			var oldVMI *v1.VirtualMachineInstance

			vmi := createVMISForPhaseTransitionTime(curPhase, oldPhase, expectedVal*1000, true)

			oldVMI = vmi.DeepCopy()
			oldVMI.Status.Phase = oldPhase

			diffSeconds, err := getVMITransitionTimeSeconds(false, false, oldVMI, vmi)
			Expect(err).ToNot(HaveOccurred())

			Expect(diffSeconds).To(Equal(expectedVal))
		},
			Entry("Time between running and scheduled", 5.0, v1.Running, v1.Scheduled),
			Entry("Time between running and scheduled using fraction of a second", .5, v1.Running, v1.Scheduled),
			Entry("Time between scheduled and scheduling", 2.0, v1.Scheduled, v1.Scheduling),
			Entry("Time between scheduling and pending", 1.0, v1.Scheduling, v1.Pending),
			Entry("Time between running and failed", 1.0, v1.Running, v1.Failed),
		)
	})

	Context("Time since Create/Delete calculations", func() {
		DescribeTable("time diff calculations", func(expectedVal float64, curPhase, oldPhase v1.VirtualMachineInstancePhase, creation bool) {
			var oldVMI *v1.VirtualMachineInstance

			vmi := createVMISForPhaseTransitionTime(curPhase, oldPhase, expectedVal*1000, true)
			if !creation {
				vmi.DeletionTimestamp = &vmi.CreationTimestamp
			}

			if oldPhase != "" {
				oldVMI = vmi.DeepCopy()
				oldVMI.Status.Phase = oldPhase
			}

			diffSeconds, err := getVMITransitionTimeSeconds(creation, !creation, oldVMI, vmi)
			Expect(err).ToNot(HaveOccurred())

			// Time since created or deleted timestamp
			// Value should be 2x expectedVal while time between Phases should be
			// 1x expectedVal because the measurement is creationtime -> oldphase -> newphase
			Expect(diffSeconds).To(Equal(2 * expectedVal))
		},
			Entry("Time between creation and pending", 3.0, v1.Pending, v1.VmPhaseUnset, true),
			Entry("Time between creation and running", 5.0, v1.Running, v1.Scheduled, true),
			Entry("Time between creation and scheduling using fraction of a second", .5, v1.Scheduling, v1.Scheduled, true),
			Entry("Time spent deleting a VMI that exited in a failed state", 5.0, v1.Failed, v1.Running, false),
			Entry("Time spent deleting a VMI that exited in a succeeded state", 5.0, v1.Succeeded, v1.Running, false),
		)
	})
})

func createVMISForPhaseTransitionTime(
	phase, oldPhase v1.VirtualMachineInstancePhase,
	offset float64,
	hasTransitionTime bool,
) *v1.VirtualMachineInstance {
	now := metav1.NewTime(time.Now())
	old := metav1.NewTime(now.Time.Add(-time.Duration(int64(offset)) * time.Millisecond))
	creation := metav1.NewTime(old.Time.Add(-time.Duration(int64(offset)) * time.Millisecond))

	vmi := &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "test-ns",
			Name:              "testvmi",
			CreationTimestamp: creation,
		},
		Status: v1.VirtualMachineInstanceStatus{
			NodeName: "testNode",
			Phase:    phase,
		},
	}

	if hasTransitionTime {
		vmi.Status.PhaseTransitionTimestamps = append(vmi.Status.PhaseTransitionTimestamps, v1.VirtualMachineInstancePhaseTransitionTimestamp{
			Phase:                    phase,
			PhaseTransitionTimestamp: now,
		})
	}

	if oldPhase != "" {
		vmi.Status.PhaseTransitionTimestamps = append(vmi.Status.PhaseTransitionTimestamps, v1.VirtualMachineInstancePhaseTransitionTimestamp{
			Phase:                    oldPhase,
			PhaseTransitionTimestamp: old,
		})
	}

	return vmi
}
