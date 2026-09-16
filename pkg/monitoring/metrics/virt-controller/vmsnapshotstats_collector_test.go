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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("VM Snapshot Stats Collector", func() {
	createdAt := metav1.NewTime(time.Unix(1700000000, 0))

	newVMSnapshot := func(
		name, namespace, uid, vm string,
		phase snapshotv1.VirtualMachineSnapshotPhase,
		created metav1.Time,
		readyToUse *bool,
	) *snapshotv1.VirtualMachineSnapshot {
		return &snapshotv1.VirtualMachineSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:              name,
				Namespace:         namespace,
				UID:               types.UID(uid),
				CreationTimestamp: created,
			},
			Spec: snapshotv1.VirtualMachineSnapshotSpec{
				Source: corev1.TypedLocalObjectReference{
					APIGroup: pointer.P("kubevirt.io"),
					Kind:     "VirtualMachine",
					Name:     vm,
				},
			},
			Status: &snapshotv1.VirtualMachineSnapshotStatus{
				Phase:      phase,
				ReadyToUse: readyToUse,
			},
		}
	}

	It("should emit no series when there are no snapshots", func() {
		Expect(reportVMSnapshotStats(nil)).To(BeEmpty())
		Expect(reportVMSnapshotStats([]*snapshotv1.VirtualMachineSnapshot{})).To(BeEmpty())
	})

	It("should emit info and create-date with matching uid and source vm", func() {
		snapshot := newVMSnapshot("snap-1", "ns-1", "uid-1", "vm-1", snapshotv1.Succeeded, createdAt, pointer.P(true))

		results := reportVMSnapshotStats([]*snapshotv1.VirtualMachineSnapshot{snapshot})
		Expect(results).To(HaveLen(2))

		Expect(results[0].Metric.GetOpts().Name).To(Equal("kubevirt_vmsnapshot_info"))
		Expect(results[0].Value).To(Equal(1.0))
		Expect(results[0].Labels).To(Equal([]string{
			"ns-1", "snap-1", "uid-1", "vm-1", "succeeded", "true",
		}))

		Expect(results[1].Metric.GetOpts().Name).To(Equal("kubevirt_vmsnapshot_create_date_timestamp_seconds"))
		Expect(results[1].Value).To(Equal(float64(createdAt.Unix())))
		Expect(results[1].Labels).To(Equal([]string{"snap-1", "ns-1"}))
	})

	DescribeTable("should set phase from status", func(phase snapshotv1.VirtualMachineSnapshotPhase, expected string) {
		snapshot := newVMSnapshot("snap-1", "ns-1", "uid-1", "vm-1", phase, createdAt, nil)

		results := reportVMSnapshotStats([]*snapshotv1.VirtualMachineSnapshot{snapshot})
		Expect(results[0].Labels[4]).To(Equal(expected))
	},
		Entry("InProgress", snapshotv1.InProgress, "inprogress"),
		Entry("Succeeded", snapshotv1.Succeeded, "succeeded"),
		Entry("Failed", snapshotv1.Failed, "failed"),
		Entry("Deleting", snapshotv1.Deleting, "deleting"),
		Entry("Unknown", snapshotv1.Unknown, "unknown"),
		Entry("unset", snapshotv1.PhaseUnset, ""),
	)

	It("should emit info with empty phase when status is nil", func() {
		snapshot := newVMSnapshot("snap-1", "ns-1", "uid-1", "vm-1", snapshotv1.InProgress, createdAt, pointer.P(true))
		snapshot.Status = nil

		results := reportVMSnapshotStats([]*snapshotv1.VirtualMachineSnapshot{snapshot})
		Expect(results[0].Labels[4]).To(Equal(""))
		Expect(results[0].Labels[5]).To(Equal("false"))
	})

	DescribeTable("should set ready_to_use from status", func(readyToUse *bool, expected string) {
		snapshot := newVMSnapshot("snap-1", "ns-1", "uid-1", "vm-1", snapshotv1.Succeeded, createdAt, readyToUse)

		results := reportVMSnapshotStats([]*snapshotv1.VirtualMachineSnapshot{snapshot})
		Expect(results[0].Labels[5]).To(Equal(expected))
	},
		Entry("nil readyToUse", nil, "false"),
		Entry("readyToUse false", pointer.P(false), "false"),
		Entry("readyToUse true", pointer.P(true), "true"),
	)

	It("should omit create-date when creationTimestamp is zero", func() {
		snapshot := newVMSnapshot("snap-1", "ns-1", "uid-1", "vm-1", snapshotv1.InProgress, metav1.Time{}, nil)

		results := reportVMSnapshotStats([]*snapshotv1.VirtualMachineSnapshot{snapshot})
		Expect(results).To(HaveLen(1))
		Expect(results[0].Metric.GetOpts().Name).To(Equal("kubevirt_vmsnapshot_info"))
	})

	It("should emit one series set per snapshot and drop deleted objects from the list", func() {
		inProgress := newVMSnapshot("snap-live", "ns-1", "uid-live", "source-vm", snapshotv1.InProgress, createdAt, nil)
		failed := newVMSnapshot("snap-failed", "ns-1", "uid-failed", "source-vm", snapshotv1.Failed, createdAt, pointer.P(false))

		results := reportVMSnapshotStats([]*snapshotv1.VirtualMachineSnapshot{inProgress, failed})
		Expect(results).To(HaveLen(4))
		Expect(results[0].Labels).To(Equal([]string{
			"ns-1", "snap-live", "uid-live", "source-vm", "inprogress", "false",
		}))
		Expect(results[2].Labels).To(Equal([]string{
			"ns-1", "snap-failed", "uid-failed", "source-vm", "failed", "false",
		}))

		remaining := reportVMSnapshotStats([]*snapshotv1.VirtualMachineSnapshot{failed})
		Expect(remaining).To(HaveLen(2))
		Expect(remaining[0].Labels[2]).To(Equal("uid-failed"))
	})
})
