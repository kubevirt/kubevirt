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
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server/disk"
)

const (
	sourceNamespace       = "source-ns"
	targetNamespace       = "target-ns"
	sourceName            = "source-vm"
	targetName            = "target-vm"
	cloudInitPathWithSrc  = "/var/run/kubevirt-ephemeral-disks/cloud-init-data/source-ns/source-vm/noCloud.iso"
	cloudInitPathWithDst  = "/var/run/kubevirt-ephemeral-disks/cloud-init-data/target-ns/target-vm/noCloud.iso"
	cloudInitPathNsOnly   = "/var/run/kubevirt-ephemeral-disks/cloud-init-data/target-ns/source-vm/noCloud.iso"
	cloudInitPathNameOnly = "/var/run/kubevirt-ephemeral-disks/cloud-init-data/source-ns/target-vm/noCloud.iso"
	pathEndingWithSrc     = "/var/run/kubevirt-ephemeral-disks/cloud-init-data/source-ns/source-vm"
	pathEndingWithDst     = "/var/run/kubevirt-ephemeral-disks/cloud-init-data/target-ns/target-vm"
	pvcDiskPath           = "/var/run/kubevirt-private/vmi-disks/pvc/disk.img"
)

var _ = Describe("DiskSourcePathHook", func() {
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		vmi = newMigrationTargetVMI(sourceNamespace, sourceName, targetNamespace, targetName)
	})

	DescribeTable("file path rewriting",
		func(sourceNS, sourceVM, targetNS, targetVM, inputPath, expectedPath string) {
			vmi = newMigrationTargetVMI(sourceNS, sourceVM, targetNS, targetVM)
			domain := domainWithCloudInitDisk(inputPath)
			Expect(disk.DiskSourcePathHook(nil, vmi, domain)).To(Succeed())
			Expect(domain.Devices.Disks[0].Source.File.File).To(Equal(expectedPath))
		},
		Entry("replaces namespace and name", sourceNamespace, sourceName, targetNamespace, targetName, cloudInitPathWithSrc, cloudInitPathWithDst),
		Entry("replaces only namespace when names match", sourceNamespace, sourceName, targetNamespace, sourceName, cloudInitPathWithSrc, cloudInitPathNsOnly),
		Entry("replaces only name when namespaces match", sourceNamespace, sourceName, sourceNamespace, targetName, cloudInitPathWithSrc, cloudInitPathNameOnly),
		Entry("replaces namespace and name when they are the final path segments", sourceNamespace, sourceName, targetNamespace, targetName, pathEndingWithSrc, pathEndingWithDst),
		Entry("does not modify unrelated paths", sourceNamespace, sourceName, targetNamespace, targetName, pvcDiskPath, pvcDiskPath),
		Entry("does not modify when source and target are identical", sourceNamespace, sourceName, sourceNamespace, sourceName, cloudInitPathWithSrc, cloudInitPathWithSrc),
	)

	It("should replace namespace and name in datastore file paths", func() {
		domain := &libvirtxml.Domain{
			Devices: &libvirtxml.DomainDeviceList{
				Disks: []libvirtxml.DomainDisk{
					{
						Alias: &libvirtxml.DomainAlias{Name: "ua-cbt-disk"},
						Source: &libvirtxml.DomainDiskSource{
							DataStore: &libvirtxml.DomainDiskDataStore{
								Source: &libvirtxml.DomainDiskSource{
									File: &libvirtxml.DomainDiskSourceFile{
										File: cloudInitPathWithSrc,
									},
								},
							},
						},
					},
				},
			},
		}

		Expect(disk.DiskSourcePathHook(nil, vmi, domain)).To(Succeed())
		Expect(domain.Devices.Disks[0].Source.DataStore.Source.File.File).To(Equal(cloudInitPathWithDst))
	})

	It("should return nil when domain has no devices", func() {
		Expect(disk.DiskSourcePathHook(nil, vmi, &libvirtxml.Domain{})).To(Succeed())
	})

	It("should not rewrite the name without SourceState.DomainName", func() {
		vmi.Status.MigrationState.SourceState.DomainName = nil
		domain := domainWithCloudInitDisk(cloudInitPathWithSrc)

		Expect(disk.DiskSourcePathHook(nil, vmi, domain)).To(Succeed())
		Expect(domain.Devices.Disks[0].Source.File.File).To(Equal(cloudInitPathNsOnly))
	})

	It("should not treat target namespace as source when SourceState.DomainNamespace is unset", func() {
		// Hook runs on the target VMI; falling back to vmi.Namespace would wrongly
		// skip namespace rewrite. Without SourceState.DomainNamespace, only the name
		// segment (from SourceState.DomainName) may be rewritten.
		vmi.Status.MigrationState.SourceState.DomainNamespace = nil
		domain := domainWithCloudInitDisk(cloudInitPathWithSrc)

		Expect(disk.DiskSourcePathHook(nil, vmi, domain)).To(Succeed())
		Expect(domain.Devices.Disks[0].Source.File.File).To(Equal(cloudInitPathNameOnly))
	})

	Context("early-return when migration state is incomplete", func() {
		It("should return nil when MigrationState is nil", func() {
			vmi = libvmi.New(
				libvmi.WithNamespace(targetNamespace),
				libvmi.WithName(targetName),
				libvmistatus.WithStatus(libvmistatus.New()),
			)
			domain := domainWithCloudInitDisk(cloudInitPathWithSrc)

			Expect(disk.DiskSourcePathHook(nil, vmi, domain)).To(Succeed())
			Expect(domain.Devices.Disks[0].Source.File.File).To(Equal(cloudInitPathWithSrc))
		})

		It("should return nil when TargetState is nil", func() {
			vmi = libvmi.New(
				libvmi.WithNamespace(targetNamespace),
				libvmi.WithName(targetName),
				libvmistatus.WithStatus(libvmistatus.New(
					libvmistatus.WithMigrationState(v1.VirtualMachineInstanceMigrationState{
						TargetState: nil,
					}),
				)),
			)
			domain := domainWithCloudInitDisk(cloudInitPathWithSrc)

			Expect(disk.DiskSourcePathHook(nil, vmi, domain)).To(Succeed())
			Expect(domain.Devices.Disks[0].Source.File.File).To(Equal(cloudInitPathWithSrc))
		})

		It("should return nil when DomainNamespace is nil and names match", func() {
			vmi = libvmi.New(
				libvmi.WithNamespace(sourceNamespace),
				libvmi.WithName(sourceName),
				libvmistatus.WithStatus(libvmistatus.New(
					libvmistatus.WithMigrationState(v1.VirtualMachineInstanceMigrationState{
						SourceState: &v1.VirtualMachineInstanceMigrationSourceState{
							VirtualMachineInstanceCommonMigrationState: v1.VirtualMachineInstanceCommonMigrationState{
								DomainName: pointer.P(sourceName),
							},
						},
						TargetState: &v1.VirtualMachineInstanceMigrationTargetState{
							VirtualMachineInstanceCommonMigrationState: v1.VirtualMachineInstanceCommonMigrationState{
								DomainNamespace: nil,
								DomainName:      pointer.P(sourceName),
							},
						},
					}),
				)),
			)
			domain := domainWithCloudInitDisk(cloudInitPathWithSrc)

			Expect(disk.DiskSourcePathHook(nil, vmi, domain)).To(Succeed())
			Expect(domain.Devices.Disks[0].Source.File.File).To(Equal(cloudInitPathWithSrc))
		})

	})
})

func newMigrationTargetVMI(sourceNS, sourceVM, targetNS, targetVM string) *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithNamespace(targetNS),
		libvmi.WithName(targetVM),
		libvmistatus.WithStatus(libvmistatus.New(
			libvmistatus.WithMigrationState(v1.VirtualMachineInstanceMigrationState{
				SourceState: &v1.VirtualMachineInstanceMigrationSourceState{
					VirtualMachineInstanceCommonMigrationState: v1.VirtualMachineInstanceCommonMigrationState{
						DomainNamespace: pointer.P(sourceNS),
						DomainName:      pointer.P(sourceVM),
					},
				},
				TargetState: &v1.VirtualMachineInstanceMigrationTargetState{
					VirtualMachineInstanceCommonMigrationState: v1.VirtualMachineInstanceCommonMigrationState{
						DomainNamespace: pointer.P(targetNS),
						DomainName:      pointer.P(targetVM),
					},
				},
			}),
		)),
	)
}

func domainWithCloudInitDisk(path string) *libvirtxml.Domain {
	return &libvirtxml.Domain{
		Devices: &libvirtxml.DomainDeviceList{
			Disks: []libvirtxml.DomainDisk{
				{
					Alias: &libvirtxml.DomainAlias{Name: "ua-cloudinit"},
					Source: &libvirtxml.DomainDiskSource{
						File: &libvirtxml.DomainDiskSourceFile{
							File: path,
						},
					},
				},
			},
		},
	}
}
