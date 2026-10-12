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

package disk_test

import (
	"libvirt.org/go/libvirtxml"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server/disk"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/storage"
)

var _ = Describe("Hook", func() {
	It("should create CBT overlays against the rewritten disk paths", func() {
		var dataPaths []string
		originalCreate := storage.CreateQCOW2Overlay
		storage.CreateQCOW2Overlay = func(_, data string) error {
			dataPaths = append(dataPaths, data)
			return nil
		}
		DeferCleanup(func() { storage.CreateQCOW2Overlay = originalCreate })

		sourceNs, targetNs := sourceNamespace, targetNamespace
		vmi := libvmi.New(
			libvmi.WithNamespace(sourceNamespace),
			libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithMigrationState(v1.VirtualMachineInstanceMigrationState{
					SourceState: &v1.VirtualMachineInstanceMigrationSourceState{
						VirtualMachineInstanceCommonMigrationState: v1.VirtualMachineInstanceCommonMigrationState{DomainNamespace: &sourceNs},
					},
					TargetState: &v1.VirtualMachineInstanceMigrationTargetState{
						VirtualMachineInstanceCommonMigrationState: v1.VirtualMachineInstanceCommonMigrationState{DomainNamespace: &targetNs},
					},
				}),
			)),
		)
		domain := &libvirtxml.Domain{Devices: &libvirtxml.DomainDeviceList{Disks: []libvirtxml.DomainDisk{{
			Alias: &libvirtxml.DomainAlias{Name: "ua-disk0"},
			Source: &libvirtxml.DomainDiskSource{
				File: &libvirtxml.DomainDiskSourceFile{File: "/var/lib/libvirt/qemu/cbt/disk0.qcow2"},
				DataStore: &libvirtxml.DomainDiskDataStore{Source: &libvirtxml.DomainDiskSource{
					File: &libvirtxml.DomainDiskSourceFile{File: "/data/" + sourceNamespace + "/disk.img"},
				}},
			},
		}}}}

		Expect(disk.Hook(nil, vmi, domain)).To(Succeed())
		Expect(dataPaths).To(ConsistOf("/data/" + targetNamespace + "/disk.img"))
	})
})
