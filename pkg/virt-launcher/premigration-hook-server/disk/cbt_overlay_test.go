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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"libvirt.org/go/libvirtxml"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	osdisk "kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server/disk"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/storage"
)

var _ = Describe("CBTOverlayHook", func() {
	const (
		volumeName     = "datavolume"
		sourceSize     = int64(2028994560)
		targetDataSize = int64(3221225472)
		fsImagePath    = "/var/run/kubevirt-private/vmi-disks/datavolume/disk.img"
		blockDevPath   = "/dev/datavolume"
	)

	var (
		vmi             *v1.VirtualMachineInstance
		createCalled    int
		capturedOverlay string
		capturedImage   string
		capturedBlock   bool
		capturedSize    int64
		originalCreate  func(string, string, bool, int64) error
		originalInfo    func(string) (*osdisk.DiskInfo, error)
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
		storage.CreateQCOW2Overlay = func(overlayPath, imagePath string, blockDev bool, size int64) error {
			createCalled++
			capturedOverlay = overlayPath
			capturedImage = imagePath
			capturedBlock = blockDev
			capturedSize = size
			return nil
		}

		originalInfo = disk.GetDiskInfo
		disk.GetDiskInfo = func(path string) (*osdisk.DiskInfo, error) {
			return &osdisk.DiskInfo{VirtualSize: targetDataSize}, nil
		}
	})

	AfterEach(func() {
		storage.CreateQCOW2Overlay = originalCreate
		disk.GetDiskInfo = originalInfo
	})

	domainWithDataStore := func(backend *libvirtxml.DomainDiskSource) *libvirtxml.Domain {
		return &libvirtxml.Domain{
			Devices: &libvirtxml.DomainDeviceList{
				Disks: []libvirtxml.DomainDisk{
					{
						Alias: &libvirtxml.DomainAlias{Name: "ua-" + volumeName},
						Source: &libvirtxml.DomainDiskSource{
							File: &libvirtxml.DomainDiskSourceFile{File: "/var/lib/libvirt/qemu/cbt/" + volumeName + ".qcow2"},
							Slices: &libvirtxml.DomainDiskSlices{
								Slices: []libvirtxml.DomainDiskSlice{
									{Type: "storage", Offset: 0, Size: uint(sourceSize)},
								},
							},
							DataStore: &libvirtxml.DomainDiskDataStore{
								Source: backend,
							},
						},
					},
				},
			},
		}
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

	DescribeTable("should size overlay from source slices not target dataStore",
		func(backend *libvirtxml.DomainDiskSource, expectBlock bool, expectPath string) {
			domain := domainWithDataStore(backend)

			Expect(disk.CBTOverlayHook(nil, vmi, domain)).To(Succeed())
			Expect(createCalled).To(Equal(1))
			Expect(capturedImage).To(Equal(expectPath))
			Expect(capturedBlock).To(Equal(expectBlock))
			Expect(capturedSize).To(Equal(sourceSize))
			Expect(capturedSize).NotTo(Equal(targetDataSize))
			Expect(capturedOverlay).To(Equal(cbt.GetQCOW2OverlayPath(vmi, volumeName)))

			diskSrc := domain.Devices.Disks[0].Source
			Expect(diskSrc.File).NotTo(BeNil())
			Expect(diskSrc.File.File).To(Equal(capturedOverlay))
			Expect(diskSrc.Block).To(BeNil())
			Expect(diskSrc.Slices).To(BeNil())
			Expect(diskSrc.DataStore.Source.Slices).To(BeNil())
		},
		Entry("filesystem dataStore", &libvirtxml.DomainDiskSource{
			File: &libvirtxml.DomainDiskSourceFile{File: fsImagePath},
		}, false, fsImagePath),
		Entry("block dataStore", &libvirtxml.DomainDiskSource{
			Block: &libvirtxml.DomainDiskSourceBlock{Dev: blockDevPath},
		}, true, blockDevPath),
	)

	It("should fall back to MigratedVolumes source capacity when slices are absent", func() {
		vmi.Status.MigratedVolumes = []v1.StorageMigratedVolumeInfo{{
			VolumeName: volumeName,
			SourcePVCInfo: &v1.PersistentVolumeClaimInfo{
				ClaimName:  "src",
				Capacity:   k8sv1.ResourceList{k8sv1.ResourceStorage: *resource.NewQuantity(sourceSize, resource.BinarySI)},
				Requests:   k8sv1.ResourceList{k8sv1.ResourceStorage: *resource.NewQuantity(sourceSize, resource.BinarySI)},
				VolumeMode: pointer.P(k8sv1.PersistentVolumeFilesystem),
			},
			DestinationPVCInfo: &v1.PersistentVolumeClaimInfo{
				ClaimName:  "dst",
				Capacity:   k8sv1.ResourceList{k8sv1.ResourceStorage: *resource.NewQuantity(targetDataSize, resource.BinarySI)},
				Requests:   k8sv1.ResourceList{k8sv1.ResourceStorage: *resource.NewQuantity(targetDataSize, resource.BinarySI)},
				VolumeMode: pointer.P(k8sv1.PersistentVolumeBlock),
			},
		}}
		domain := domainWithDataStore(&libvirtxml.DomainDiskSource{
			Block: &libvirtxml.DomainDiskSourceBlock{Dev: blockDevPath},
		})
		domain.Devices.Disks[0].Source.Slices = nil

		Expect(disk.CBTOverlayHook(nil, vmi, domain)).To(Succeed())
		Expect(capturedSize).To(Equal(sourceSize))
	})

	It("should return error when disk info lookup fails and no source size is available", func() {
		disk.GetDiskInfo = func(path string) (*osdisk.DiskInfo, error) {
			return nil, fmt.Errorf("qemu-img failed")
		}
		domain := domainWithDataStore(&libvirtxml.DomainDiskSource{
			File: &libvirtxml.DomainDiskSourceFile{File: fsImagePath},
		})
		domain.Devices.Disks[0].Source.Slices = nil

		err := disk.CBTOverlayHook(nil, vmi, domain)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to get disk info"))
		Expect(createCalled).To(Equal(0))
	})
})
