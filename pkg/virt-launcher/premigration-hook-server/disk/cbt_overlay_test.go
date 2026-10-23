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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server/disk"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/storage"
)

var _ = Describe("CBTOverlayHook", func() {
	const (
		volumeName   = "datavolume"
		sourceSize   = int64(2028994560)
		fsImagePath  = "/var/run/kubevirt-private/vmi-disks/datavolume/disk.img"
		blockDevPath = "/dev/datavolume"
	)

	var (
		vmi             *v1.VirtualMachineInstance
		createCalled    int
		capturedOverlay string
		capturedImage   string
		capturedBlock   bool
		capturedSize    int64
		originalCreate  func(string, string, int64) error
	)

	BeforeEach(func() {
		vmi = libvmi.New(
			libvmi.WithNamespace("default"),
			libvmi.WithPersistentVolumeClaim(volumeName, "test-pvc"),
			libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithChangedBlockTracking(&v1.ChangedBlockTrackingStatus{
					State: v1.ChangedBlockTrackingEnabled,
				}),
			)),
		)

		createCalled = 0
		capturedOverlay = ""
		capturedImage = ""
		capturedBlock = false
		capturedSize = 0

		originalCreate = storage.CreateQCOW2Overlay
		storage.CreateQCOW2Overlay = func(overlayPath, imagePath string, size int64) error {
			createCalled++
			capturedOverlay = overlayPath
			capturedImage = imagePath
			capturedBlock = len(imagePath) >= 5 && imagePath[:5] == "/dev/"
			capturedSize = size
			return nil
		}
	})

	AfterEach(func() {
		storage.CreateQCOW2Overlay = originalCreate
	})

	domainWithDataStore := func(backend *libvirtxml.DomainDiskSource) *libvirtxml.Domain {
		dom := &libvirtxml.Domain{
			Devices: &libvirtxml.DomainDeviceList{
				Disks: []libvirtxml.DomainDisk{
					{
						Alias: &libvirtxml.DomainAlias{Name: "ua-" + volumeName},
						Source: &libvirtxml.DomainDiskSource{
							File: &libvirtxml.DomainDiskSourceFile{File: "/var/lib/libvirt/qemu/cbt/" + volumeName + ".qcow2"},
							DataStore: &libvirtxml.DomainDiskDataStore{
								Source: backend,
							},
						},
					},
				},
			},
		}
		Expect(storage.SetMigrationDiskSizes(dom, map[string]int64{volumeName: sourceSize})).To(Succeed())
		return dom
	}

	It("should skip when CBT is not enabled", func() {
		vmi.Status.ChangedBlockTracking = nil
		domain := domainWithDataStore(&libvirtxml.DomainDiskSource{
			File: &libvirtxml.DomainDiskSourceFile{File: fsImagePath},
		})

		Expect(disk.CBTOverlayHook(nil, vmi, domain)).To(Succeed())
		Expect(createCalled).To(Equal(0))
	})

	It("should skip disks without dataStore", func() {
		domain := &libvirtxml.Domain{
			Devices: &libvirtxml.DomainDeviceList{
				Disks: []libvirtxml.DomainDisk{
					{
						Alias: &libvirtxml.DomainAlias{Name: "ua-" + volumeName},
						Source: &libvirtxml.DomainDiskSource{
							File: &libvirtxml.DomainDiskSourceFile{File: fsImagePath},
						},
					},
				},
			},
		}

		Expect(disk.CBTOverlayHook(nil, vmi, domain)).To(Succeed())
		Expect(createCalled).To(Equal(0))
	})

	DescribeTable("should size overlay from DestXML metadata",
		func(backend *libvirtxml.DomainDiskSource, expectBlock bool, expectPath string) {
			domain := domainWithDataStore(backend)

			Expect(disk.CBTOverlayHook(nil, vmi, domain)).To(Succeed())
			Expect(createCalled).To(Equal(1))
			Expect(capturedImage).To(Equal(expectPath))
			Expect(capturedBlock).To(Equal(expectBlock))
			Expect(capturedSize).To(Equal(sourceSize))
			Expect(capturedOverlay).To(Equal(cbt.GetQCOW2OverlayPath(vmi, volumeName)))

			diskSrc := domain.Devices.Disks[0].Source
			Expect(diskSrc.File).NotTo(BeNil())
			Expect(diskSrc.File.File).To(Equal(capturedOverlay))
			Expect(diskSrc.Block).To(BeNil())
			Expect(diskSrc.Slices).To(BeNil())
			Expect(diskSrc.DataStore.Source.Slices).To(BeNil())
			Expect(domain.Metadata).To(BeNil())
		},
		Entry("filesystem dataStore", &libvirtxml.DomainDiskSource{
			File: &libvirtxml.DomainDiskSourceFile{File: fsImagePath},
		}, false, fsImagePath),
		Entry("block dataStore", &libvirtxml.DomainDiskSource{
			Block: &libvirtxml.DomainDiskSourceBlock{Dev: blockDevPath},
		}, true, blockDevPath),
	)

	It("should return error when DestXML metadata size is missing", func() {
		domain := &libvirtxml.Domain{
			Devices: &libvirtxml.DomainDeviceList{
				Disks: []libvirtxml.DomainDisk{
					{
						Alias: &libvirtxml.DomainAlias{Name: "ua-" + volumeName},
						Source: &libvirtxml.DomainDiskSource{
							File: &libvirtxml.DomainDiskSourceFile{File: "/var/lib/libvirt/qemu/cbt/" + volumeName + ".qcow2"},
							DataStore: &libvirtxml.DomainDiskDataStore{
								Source: &libvirtxml.DomainDiskSource{
									File: &libvirtxml.DomainDiskSourceFile{File: fsImagePath},
								},
							},
						},
					},
				},
			},
		}

		err := disk.CBTOverlayHook(nil, vmi, domain)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing source virtual size in DestXML metadata"))
		Expect(createCalled).To(Equal(0))
	})
})
