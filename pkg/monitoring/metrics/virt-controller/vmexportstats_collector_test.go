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

	exportv1 "kubevirt.io/api/export/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("VM Export Stats Collector", func() {
	newVMExport := func(
		name, namespace, uid, source, sourceKind string,
		phase exportv1.VirtualMachineExportPhase,
		withStatus bool,
	) *exportv1.VirtualMachineExport {
		vmExport := &exportv1.VirtualMachineExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				UID:       types.UID(uid),
			},
			Spec: exportv1.VirtualMachineExportSpec{
				Source: k8sv1.TypedLocalObjectReference{
					Kind: sourceKind,
					Name: source,
				},
			},
		}
		if withStatus {
			vmExport.Status = &exportv1.VirtualMachineExportStatus{
				Phase: phase,
			}
		}
		return vmExport
	}

	It("should emit no series when there are no exports", func() {
		Expect(reportVMExportStats(nil)).To(BeEmpty())
		Expect(reportVMExportStats([]*exportv1.VirtualMachineExport{})).To(BeEmpty())
	})

	It("should emit one series per export with matching uid and source", func() {
		vmExport := newVMExport("export-1", "ns-1", "uid-1", "source-vm", "VirtualMachine", exportv1.Pending, true)

		results := reportVMExportStats([]*exportv1.VirtualMachineExport{vmExport})
		Expect(results).To(HaveLen(1))
		Expect(results[0].Metric.GetOpts().Name).To(Equal("kubevirt_vmexport_info"))
		Expect(results[0].Value).To(Equal(1.0))
		Expect(results[0].Labels).To(Equal([]string{
			"ns-1", "export-1", "uid-1", "source-vm", "source-vm", "VirtualMachine", "pending",
		}))
	})

	DescribeTable("should set phase from status", func(withStatus bool, phase exportv1.VirtualMachineExportPhase, expected string) {
		vmExport := newVMExport("export-1", "ns-1", "uid-1", "source-pvc", "PersistentVolumeClaim", phase, withStatus)

		results := reportVMExportStats([]*exportv1.VirtualMachineExport{vmExport})
		Expect(results).To(HaveLen(1))
		Expect(results[0].Labels[3]).To(Equal(""))
		Expect(results[0].Labels[6]).To(Equal(expected))
	},
		Entry("nil status", false, exportv1.VirtualMachineExportPhase(""), "unset"),
		Entry("empty phase", true, exportv1.VirtualMachineExportPhase(""), "unset"),
		Entry("pending", true, exportv1.Pending, "pending"),
		Entry("ready", true, exportv1.Ready, "ready"),
		Entry("terminated", true, exportv1.Terminated, "terminated"),
		Entry("skipped", true, exportv1.Skipped, "skipped"),
	)

	It("should emit one series per export and drop deleted objects from the list", func() {
		live := newVMExport("export-live", "ns-1", "uid-live", "source-vm", "VirtualMachine", exportv1.Ready, true)
		done := newVMExport("export-done", "ns-1", "uid-done", "source-snap", "VirtualMachineSnapshot", exportv1.Terminated, true)

		results := reportVMExportStats([]*exportv1.VirtualMachineExport{live, done})
		Expect(results).To(HaveLen(2))
		Expect(results[0].Labels).To(Equal([]string{
			"ns-1", "export-live", "uid-live", "source-vm", "source-vm", "VirtualMachine", "ready",
		}))
		Expect(results[1].Labels).To(Equal([]string{
			"ns-1", "export-done", "uid-done", "", "source-snap", "VirtualMachineSnapshot", "terminated",
		}))

		remaining := reportVMExportStats([]*exportv1.VirtualMachineExport{done})
		Expect(remaining).To(HaveLen(1))
		Expect(remaining[0].Labels[2]).To(Equal("uid-done"))
	})

	It("should set vm from status.virtualMachineName for snapshot sources", func() {
		vmExport := newVMExport(
			"export-1", "ns-1", "uid-1", "source-snap", "VirtualMachineSnapshot",
			exportv1.Ready, true,
		)
		vmExport.Status.VirtualMachineName = pointer.P("source-vm")

		results := reportVMExportStats([]*exportv1.VirtualMachineExport{vmExport})
		Expect(results).To(HaveLen(1))
		Expect(results[0].Labels).To(Equal([]string{
			"ns-1", "export-1", "uid-1", "source-vm", "source-snap", "VirtualMachineSnapshot", "ready",
		}))
	})

	It("should emit ttl expiration timestamp when status.ttlExpirationTime is set", func() {
		expiresAt := metav1.NewTime(time.Unix(1700003600, 0))
		vmExport := newVMExport("export-1", "ns-1", "uid-1", "source-vm", "VirtualMachine", exportv1.Ready, true)
		vmExport.Status.TTLExpirationTime = &expiresAt

		results := reportVMExportStats([]*exportv1.VirtualMachineExport{vmExport})
		Expect(results).To(HaveLen(2))
		Expect(results[1].Metric.GetOpts().Name).To(Equal("kubevirt_vmexport_ttl_expiration_timestamp_seconds"))
		Expect(results[1].Value).To(Equal(float64(expiresAt.Unix())))
		Expect(results[1].Labels).To(Equal([]string{"export-1", "ns-1"}))
	})

	It("should omit ttl expiration timestamp when status is nil", func() {
		vmExport := newVMExport("export-1", "ns-1", "uid-1", "source-vm", "VirtualMachine", exportv1.Pending, false)

		results := reportVMExportStats([]*exportv1.VirtualMachineExport{vmExport})
		Expect(results).To(HaveLen(1))
		Expect(results[0].Metric.GetOpts().Name).To(Equal("kubevirt_vmexport_info"))
	})
})
