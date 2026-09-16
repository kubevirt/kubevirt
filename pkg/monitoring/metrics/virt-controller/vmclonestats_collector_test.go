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

package virtcontroller

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	clonev1 "kubevirt.io/api/clone/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("VM Clone Stats Collector", func() {
	newVMClone := func(
		name, namespace, uid, source, sourceKind, target string,
		phase clonev1.VirtualMachineClonePhase,
		statusTarget *string,
	) *clonev1.VirtualMachineClone {
		vmClone := &clonev1.VirtualMachineClone{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				UID:       types.UID(uid),
			},
			Spec: clonev1.VirtualMachineCloneSpec{
				Source: &k8sv1.TypedLocalObjectReference{
					Kind: sourceKind,
					Name: source,
				},
			},
			Status: clonev1.VirtualMachineCloneStatus{
				Phase:      phase,
				TargetName: statusTarget,
			},
		}
		if target != "" {
			vmClone.Spec.Target = &k8sv1.TypedLocalObjectReference{
				Kind: "VirtualMachine",
				Name: target,
			}
		}
		return vmClone
	}

	It("should emit no series when there are no clones", func() {
		Expect(reportVMCloneStats(nil)).To(BeEmpty())
		Expect(reportVMCloneStats([]*clonev1.VirtualMachineClone{})).To(BeEmpty())
	})

	It("should emit one series per clone with matching uid, source, and target", func() {
		vmClone := newVMClone(
			"clone-1", "ns-1", "uid-1", "source-vm", "VirtualMachine", "target-vm",
			clonev1.Succeeded, nil,
		)

		results := reportVMCloneStats([]*clonev1.VirtualMachineClone{vmClone})
		Expect(results).To(HaveLen(1))
		Expect(results[0].Metric.GetOpts().Name).To(Equal("kubevirt_vmclone_info"))
		Expect(results[0].Value).To(Equal(1.0))
		Expect(results[0].Labels).To(Equal([]string{
			"ns-1", "clone-1", "uid-1", "source-vm", "VirtualMachine", "target-vm", "", "", "succeeded",
		}))
	})

	It("should use status targetName when spec target is unset", func() {
		vmClone := newVMClone(
			"clone-1", "ns-1", "uid-1", "source-snap", "VirtualMachineSnapshot", "",
			clonev1.CreatingTargetVM, pointer.P("generated-target"),
		)

		results := reportVMCloneStats([]*clonev1.VirtualMachineClone{vmClone})
		Expect(results).To(HaveLen(1))
		Expect(results[0].Labels).To(Equal([]string{
			"ns-1", "clone-1", "uid-1", "source-snap", "VirtualMachineSnapshot",
			"generated-target", "", "", "creatingtargetvm",
		}))
	})

	DescribeTable("should set phase from status", func(phase clonev1.VirtualMachineClonePhase, expected string) {
		vmClone := newVMClone(
			"clone-1", "ns-1", "uid-1", "source-vm", "VirtualMachine", "target-vm",
			phase, nil,
		)

		results := reportVMCloneStats([]*clonev1.VirtualMachineClone{vmClone})
		Expect(results).To(HaveLen(1))
		Expect(results[0].Labels[8]).To(Equal(expected))
	},
		Entry("unset", clonev1.PhaseUnset, "unset"),
		Entry("snapshot in progress", clonev1.SnapshotInProgress, "snapshotinprogress"),
		Entry("creating target vm", clonev1.CreatingTargetVM, "creatingtargetvm"),
		Entry("restore in progress", clonev1.RestoreInProgress, "restoreinprogress"),
		Entry("succeeded", clonev1.Succeeded, "succeeded"),
		Entry("failed", clonev1.Failed, "failed"),
		Entry("unknown", clonev1.Unknown, "unknown"),
	)

	It("should emit one series per clone and drop deleted objects from the list", func() {
		live := newVMClone(
			"clone-live", "ns-1", "uid-live", "source-vm", "VirtualMachine", "target-a",
			clonev1.RestoreInProgress, nil,
		)
		done := newVMClone(
			"clone-done", "ns-1", "uid-done", "source-vm", "VirtualMachine", "target-b",
			clonev1.Succeeded, nil,
		)

		results := reportVMCloneStats([]*clonev1.VirtualMachineClone{live, done})
		Expect(results).To(HaveLen(2))
		Expect(results[0].Labels[2]).To(Equal("uid-live"))
		Expect(results[1].Labels[2]).To(Equal("uid-done"))

		remaining := reportVMCloneStats([]*clonev1.VirtualMachineClone{done})
		Expect(remaining).To(HaveLen(1))
		Expect(remaining[0].Labels[2]).To(Equal("uid-done"))
	})

	It("should set snapshot_name and restore_name from status", func() {
		vmClone := newVMClone(
			"clone-1", "ns-1", "uid-1", "source-vm", "VirtualMachine", "target-vm",
			clonev1.RestoreInProgress, nil,
		)
		vmClone.Status.SnapshotName = pointer.P("snap-1")
		vmClone.Status.RestoreName = pointer.P("restore-1")

		results := reportVMCloneStats([]*clonev1.VirtualMachineClone{vmClone})
		Expect(results).To(HaveLen(1))
		Expect(results[0].Labels).To(Equal([]string{
			"ns-1", "clone-1", "uid-1", "source-vm", "VirtualMachine", "target-vm",
			"snap-1", "restore-1", "restoreinprogress",
		}))
	})

	It("should emit create-date when creationTimestamp is set", func() {
		createdAt := metav1.NewTime(time.Unix(1700000000, 0))
		vmClone := newVMClone(
			"clone-1", "ns-1", "uid-1", "source-vm", "VirtualMachine", "target-vm",
			clonev1.Succeeded, nil,
		)
		vmClone.CreationTimestamp = createdAt

		results := reportVMCloneStats([]*clonev1.VirtualMachineClone{vmClone})
		Expect(results).To(HaveLen(2))
		Expect(results[1].Metric.GetOpts().Name).To(Equal("kubevirt_vmclone_create_date_timestamp_seconds"))
		Expect(results[1].Value).To(Equal(float64(createdAt.Unix())))
		Expect(results[1].Labels).To(Equal([]string{"clone-1", "ns-1"}))
	})

	It("should omit create-date when creationTimestamp is zero", func() {
		vmClone := newVMClone(
			"clone-1", "ns-1", "uid-1", "source-vm", "VirtualMachine", "target-vm",
			clonev1.Succeeded, nil,
		)

		results := reportVMCloneStats([]*clonev1.VirtualMachineClone{vmClone})
		Expect(results).To(HaveLen(1))
		Expect(results[0].Metric.GetOpts().Name).To(Equal("kubevirt_vmclone_info"))
	})
})
