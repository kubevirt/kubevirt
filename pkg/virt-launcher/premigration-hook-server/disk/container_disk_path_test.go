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
	"kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server/disk"
)

var _ = Describe("ContainerDiskPathHook", func() {
	const (
		legacyContainerDiskPath = "/var/run/kubevirt/container-disks/disk_2.img"
		namedContainerDiskPath  = "/var/run/kubevirt/container-disks/disk_r0.img"
		overlayPath             = "/var/run/kubevirt/ephemeral-disks/r0/disk.qcow2"
	)

	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		vmi = libvmi.New(libvmi.WithContainerDisk("r0", "someimage"))
	})

	It("should repoint a legacy backing store at the volume name based path", func() {
		domain := domainWithContainerDisk("ua-r0", legacyContainerDiskPath)

		Expect(disk.ContainerDiskPathHook(nil, vmi, domain)).To(Succeed())
		Expect(domain.Devices.Disks[0].BackingStore.Source.File.File).To(Equal(namedContainerDiskPath))
		Expect(domain.Devices.Disks[0].Source.File.File).To(Equal(overlayPath), "the overlay must not be rewritten")
	})

	It("should be a no-op for a domain already using volume name based paths", func() {
		domain := domainWithContainerDisk("ua-r0", namedContainerDiskPath)

		Expect(disk.ContainerDiskPathHook(nil, vmi, domain)).To(Succeed())
		Expect(domain.Devices.Disks[0].BackingStore.Source.File.File).To(Equal(namedContainerDiskPath))
	})

	It("should not touch a disk that is not a container disk", func() {
		domain := domainWithContainerDisk("ua-pvc", legacyContainerDiskPath)

		Expect(disk.ContainerDiskPathHook(nil, vmi, domain)).To(Succeed())
		Expect(domain.Devices.Disks[0].BackingStore.Source.File.File).To(Equal(legacyContainerDiskPath))
	})

	It("should not touch a container disk without a backing store", func() {
		domain := domainWithContainerDisk("ua-r0", legacyContainerDiskPath)
		domain.Devices.Disks[0].BackingStore = nil

		Expect(disk.ContainerDiskPathHook(nil, vmi, domain)).To(Succeed())
		Expect(domain.Devices.Disks[0].Source.File.File).To(Equal(overlayPath))
	})

	It("should do nothing when the VMI has no container disks", func() {
		vmi = libvmi.New(libvmi.WithPersistentVolumeClaim("pvc", "pvc"))
		domain := domainWithContainerDisk("ua-r0", legacyContainerDiskPath)

		Expect(disk.ContainerDiskPathHook(nil, vmi, domain)).To(Succeed())
		Expect(domain.Devices.Disks[0].BackingStore.Source.File.File).To(Equal(legacyContainerDiskPath))
	})
})

func domainWithContainerDisk(alias, backingStorePath string) *libvirtxml.Domain {
	return &libvirtxml.Domain{
		Devices: &libvirtxml.DomainDeviceList{
			Disks: []libvirtxml.DomainDisk{
				{
					Alias: &libvirtxml.DomainAlias{Name: alias},
					Source: &libvirtxml.DomainDiskSource{
						File: &libvirtxml.DomainDiskSourceFile{
							File: "/var/run/kubevirt/ephemeral-disks/r0/disk.qcow2",
						},
					},
					BackingStore: &libvirtxml.DomainDiskBackingStore{
						Source: &libvirtxml.DomainDiskSource{
							File: &libvirtxml.DomainDiskSourceFile{
								File: backingStorePath,
							},
						},
					},
				},
			},
		},
	}
}
