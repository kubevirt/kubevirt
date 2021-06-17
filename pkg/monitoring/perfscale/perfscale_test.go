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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package perfscale

import (
	"time"

	"github.com/onsi/ginkgo/extensions/table"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/client-go/api/v1"
)

var _ = BeforeSuite(func() {
})

var _ = Describe("VMI phase transition time histogram", func() {
	Context("Transition Time calculations", func() {
		table.DescribeTable("time diff calculations", func(expectedVal float64, curPhase v1.VirtualMachineInstancePhase, oldPhase v1.VirtualMachineInstancePhase) {
			var oldVMI *v1.VirtualMachineInstance

			vmi := createVMISForPhaseTransitionTime(curPhase, oldPhase, expectedVal*1000, true)

			if oldPhase != "" {
				oldVMI = vmi.DeepCopy()
				oldVMI.Status.Phase = oldPhase
			}

			diffSeconds, err := getTransitionTimeSeconds(false, oldVMI, vmi)
			Expect(err).To(BeNil())

			Expect(diffSeconds).To(Equal(expectedVal))

		},
			table.Entry("Time between running and scheduled", 5.0, v1.Running, v1.Scheduled),
			table.Entry("Time between running and scheduled using fraction of a second", .5, v1.Running, v1.Scheduled),
			table.Entry("Time between scheduled and scheduling", 2.0, v1.Scheduled, v1.Scheduling),
			table.Entry("Time between scheduling and pending", 1.0, v1.Scheduling, v1.Pending),
			table.Entry("Time between scheduling and creation", 3.0, v1.Scheduling, v1.VmPhaseUnset),
			table.Entry("Time between scheduling and creation when timestamps are within one second of another", 0.0, v1.Scheduling, v1.VmPhaseUnset),
		)
	})

})

func createVMISForPhaseTransitionTime(phase v1.VirtualMachineInstancePhase, oldPhase v1.VirtualMachineInstancePhase, offset float64, hasTransitionTime bool) *v1.VirtualMachineInstance {

	now := metav1.NewTime(time.Now())

	old := metav1.NewTime(now.Time.Add(-time.Duration(int64(offset)) * time.Millisecond))

	creation := old
	if oldPhase != "" {
		creation = metav1.NewTime(old.Time.Add(-time.Duration(int64(offset)) * time.Millisecond))
	}

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
