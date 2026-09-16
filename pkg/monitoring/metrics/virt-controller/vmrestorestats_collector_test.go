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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("VM Restore Stats Collector", func() {
	newVMRestore := func(name, namespace, uid, vm string, complete *bool) *snapshotv1.VirtualMachineRestore {
		restore := &snapshotv1.VirtualMachineRestore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				UID:       types.UID(uid),
			},
			Spec: snapshotv1.VirtualMachineRestoreSpec{
				Target: corev1.TypedLocalObjectReference{
					Kind: "VirtualMachine",
					Name: vm,
				},
				VirtualMachineSnapshotName: "snapshot-" + vm,
			},
		}
		if complete != nil {
			restore.Status = &snapshotv1.VirtualMachineRestoreStatus{
				Complete: complete,
			}
		}
		return restore
	}

	It("should emit no series when there are no restores", func() {
		Expect(reportVMRestoreStats(nil)).To(BeEmpty())
		Expect(reportVMRestoreStats([]*snapshotv1.VirtualMachineRestore{})).To(BeEmpty())
	})

	It("should emit one series per restore with matching uid and target vm", func() {
		restore := newVMRestore("restore-1", "ns-1", "uid-1", "vm-1", nil)

		results := reportVMRestoreStats([]*snapshotv1.VirtualMachineRestore{restore})
		Expect(results).To(HaveLen(1))
		Expect(results[0].Metric.GetOpts().Name).To(Equal("kubevirt_vmrestore_info"))
		Expect(results[0].Value).To(Equal(1.0))
		Expect(results[0].Labels).To(Equal([]string{
			"ns-1", "restore-1", "uid-1", "vm-1", "false",
		}))
	})

	DescribeTable("should set complete from status", func(complete *bool, expected string) {
		restore := newVMRestore("restore-1", "ns-1", "uid-1", "vm-1", complete)

		results := reportVMRestoreStats([]*snapshotv1.VirtualMachineRestore{restore})
		Expect(results).To(HaveLen(1))
		Expect(results[0].Labels[4]).To(Equal(expected))
	},
		Entry("nil status", nil, "false"),
		Entry("complete false", pointer.P(false), "false"),
		Entry("complete true", pointer.P(true), "true"),
	)

	It("should emit one series per restore and drop deleted objects from the list", func() {
		live := newVMRestore("restore-live", "ns-1", "uid-live", "target-vm", pointer.P(false))
		done := newVMRestore("restore-done", "ns-1", "uid-done", "target-vm", pointer.P(true))

		results := reportVMRestoreStats([]*snapshotv1.VirtualMachineRestore{live, done})
		Expect(results).To(HaveLen(2))
		Expect(results[0].Labels).To(Equal([]string{
			"ns-1", "restore-live", "uid-live", "target-vm", "false",
		}))
		Expect(results[1].Labels).To(Equal([]string{
			"ns-1", "restore-done", "uid-done", "target-vm", "true",
		}))

		Expect(reportVMRestoreStats([]*snapshotv1.VirtualMachineRestore{done})).To(HaveLen(1))
		Expect(reportVMRestoreStats([]*snapshotv1.VirtualMachineRestore{done})[0].Labels[2]).To(Equal("uid-done"))
	})

	It("should emit no series when the restore store is unset", func() {
		orig := stores
		defer func() { stores = orig }()

		stores = nil
		Expect(vmRestoreStatsCollectorCallback()).To(BeEmpty())
		stores = &Stores{}
		Expect(vmRestoreStatsCollectorCallback()).To(BeEmpty())
	})

	It("should list restores from the store", func() {
		orig := stores
		defer func() { stores = orig }()

		informer, _ := testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineRestore{})
		Expect(informer.GetStore().Add(newVMRestore("restore-1", "ns-1", "uid-1", "vm-1", nil))).To(Succeed())
		stores = &Stores{VMRestore: informer.GetStore()}

		results := vmRestoreStatsCollectorCallback()
		Expect(results).To(HaveLen(1))
		Expect(results[0].Value).To(Equal(1.0))
		Expect(results[0].Labels).To(Equal([]string{
			"ns-1", "restore-1", "uid-1", "vm-1", "false",
		}))
	})
})
