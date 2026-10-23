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

package virtwrap

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"
	api2 "kubevirt.io/client-go/api"
	"kubevirt.io/client-go/log"

	osdisk "kubevirt.io/kubevirt/pkg/os/disk"
	virtpointer "kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/disksource"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/testing"
)

const (
	existingFilePath    = "/dev/null"
	nonExistingFilePath = "/disks/disk.img"
)

var _ = Describe("getBlockResizeArgs", func() {
	const (
		gb          = 1024 * 1024 * 1024
		fakePercent = v1.Percent("0.05")
	)

	It("should let libvirt infer size for direct block device (LUKS-safe)", func() {
		disk := api.Disk{
			Source: api.DiskSource{Dev: "/dev/vda"},
			Alias:  api.NewUserDefinedAlias("test-disk"),
		}
		ds := disksource.Resolve(disk)
		rszArgs, ok := getBlockResizeArgs(disk, ds)
		Expect(ok).To(BeTrue())
		Expect(rszArgs.size).To(Equal(uint64(0)))
		Expect(rszArgs.flags).To(Equal(libvirt.DOMAIN_BLOCK_RESIZE_BYTES | libvirt.DOMAIN_BLOCK_RESIZE_CAPACITY))
	})

	It("should let libvirt infer size for overlay on block (CAPACITY)", func() {
		disk := api.Disk{
			Source: api.DiskSource{
				File:      "/test/overlay.qcow2",
				DataStore: &api.DataStore{Type: "block", Source: &api.DiskSource{Dev: "/dev/vda"}},
			},
			Alias: api.NewUserDefinedAlias("test-disk"),
		}
		ds := disksource.Resolve(disk)
		rszArgs, ok := getBlockResizeArgs(disk, ds)
		Expect(ok).To(BeTrue())
		Expect(rszArgs.size).To(Equal(uint64(0)))
		Expect(rszArgs.flags).To(Equal(libvirt.DOMAIN_BLOCK_RESIZE_BYTES | libvirt.DOMAIN_BLOCK_RESIZE_CAPACITY))
	})

	It("should return possibleGuestSize for file backed disk", func() {
		disk := api.Disk{
			FilesystemOverhead: virtpointer.P(fakePercent),
			Capacity:           virtpointer.P(int64(2 * gb)),
			Source:             api.DiskSource{File: existingFilePath},
			Alias:              api.NewUserDefinedAlias("test-disk"),
		}
		ds := disksource.Resolve(disk)
		rszArgs, ok := getBlockResizeArgs(disk, ds)
		Expect(ok).To(BeTrue())
		Expect(rszArgs.size).To(BeNumerically(">", 0))
		Expect(rszArgs.flags).To(Equal(libvirt.DOMAIN_BLOCK_RESIZE_BYTES))
	})

	It("should return false for file backed disk when source does not exist", func() {
		disk := api.Disk{
			FilesystemOverhead: virtpointer.P(fakePercent),
			Capacity:           virtpointer.P(int64(2 * gb)),
			Source:             api.DiskSource{File: nonExistingFilePath},
			Alias:              api.NewUserDefinedAlias("test-disk"),
		}
		ds := disksource.Resolve(disk)
		_, ok := getBlockResizeArgs(disk, ds)
		Expect(ok).To(BeFalse())
	})

	It("should return false for file backed disk without capacity", func() {
		disk := api.Disk{
			Source: api.DiskSource{File: existingFilePath},
			Alias:  api.NewUserDefinedAlias("test-disk"),
		}
		ds := disksource.Resolve(disk)
		_, ok := getBlockResizeArgs(disk, ds)
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("expandDisksOnline", func() {
	var (
		ctrl                     *gomock.Controller
		mockLibvirt              *testing.Libvirt
		manager                  *LibvirtDomainManager
		origGetDiskInfoForResize func(string) (*osdisk.DiskInfo, error)
	)

	const gb = 1024 * 1024 * 1024

	pvcBackedVMI := func(diskName string) *v1.VirtualMachineInstance {
		vmi := api2.NewMinimalVMI("test-vmi")
		vmi.Status.VolumeStatus = []v1.VolumeStatus{
			{
				Name:                      diskName,
				PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{},
			},
		}
		return vmi
	}

	overlayBlockDomain := func(capacity int64) *api.Domain {
		return &api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					Disks: []api.Disk{{
						Source: api.DiskSource{
							File:      "/test/overlay.qcow2",
							DataStore: &api.DataStore{Type: "block", Source: &api.DiskSource{Dev: "/dev/vda"}},
						},
						Alias:    api.NewUserDefinedAlias("disk0"),
						Capacity: &capacity,
					}},
				},
			},
		}
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockLibvirt = testing.NewLibvirt(ctrl)
		manager = &LibvirtDomainManager{
			guestDiskSizes: map[string]int64{},
		}
		origGetDiskInfoForResize = getDiskInfoForResize
		getDiskInfoForResize = func(path string) (*osdisk.DiskInfo, error) {
			Fail("getDiskInfoForResize must not be called")
			return nil, nil
		}
	})

	AfterEach(func() {
		getDiskInfoForResize = origGetDiskInfoForResize
	})

	It("should seed map with PVC capacity on first sync without calling BlockResize", func() {
		capacity := int64(2 * gb)
		domain := &api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					Disks: []api.Disk{{
						Source:   api.DiskSource{Dev: "/dev/vda"},
						Alias:    api.NewUserDefinedAlias("disk0"),
						Capacity: &capacity,
					}},
				},
			},
		}
		vmi := pvcBackedVMI("disk0")

		manager.expandDisksOnline(mockLibvirt.VirtDomain, domain, vmi)

		Expect(manager.guestDiskSizes["disk0"]).To(Equal(capacity))
	})

	It("should not call BlockResize when capacity unchanged", func() {
		capacity := int64(2 * gb)
		manager.guestDiskSizes["disk0"] = capacity

		domain := &api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					Disks: []api.Disk{{
						Source:   api.DiskSource{Dev: "/dev/vda"},
						Alias:    api.NewUserDefinedAlias("disk0"),
						Capacity: &capacity,
					}},
				},
			},
		}
		vmi := pvcBackedVMI("disk0")

		manager.expandDisksOnline(mockLibvirt.VirtDomain, domain, vmi)
	})

	// Verifies that for block devices we pass size=0 with DOMAIN_BLOCK_RESIZE_CAPACITY
	// so libvirt infers the real device size. This is required for LUKS-encrypted volumes
	// where PVC capacity includes the 16MiB LUKS header.
	// See https://github.com/kubevirt/kubevirt/pull/16366
	It("should call BlockResize with size 0 when block device capacity changes (LUKS-safe)", func() {
		oldCapacity := int64(1 * gb)
		newCapacity := int64(2 * gb)
		manager.guestDiskSizes["disk0"] = oldCapacity

		domain := &api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					Disks: []api.Disk{{
						Source:   api.DiskSource{Dev: "/dev/vda"},
						Alias:    api.NewUserDefinedAlias("disk0"),
						Capacity: &newCapacity,
					}},
				},
			},
		}
		vmi := pvcBackedVMI("disk0")

		mockLibvirt.DomainEXPECT().BlockResize("/dev/vda", uint64(0),
			libvirt.DOMAIN_BLOCK_RESIZE_BYTES|libvirt.DOMAIN_BLOCK_RESIZE_CAPACITY).Times(1).Return(nil)

		manager.expandDisksOnline(mockLibvirt.VirtDomain, domain, vmi)

		Expect(manager.guestDiskSizes["disk0"]).To(Equal(newCapacity))
	})

	It("should call BlockResize with the overlay path and CAPACITY for overlay-on-block when capacity changes", func() {
		oldCapacity := int64(1 * gb)
		newCapacity := int64(2 * gb)
		manager.guestDiskSizes["disk0"] = oldCapacity

		domain := overlayBlockDomain(newCapacity)
		vmi := pvcBackedVMI("disk0")

		mockLibvirt.DomainEXPECT().BlockResize("/test/overlay.qcow2", uint64(0),
			libvirt.DOMAIN_BLOCK_RESIZE_BYTES|libvirt.DOMAIN_BLOCK_RESIZE_CAPACITY).Times(1).Return(nil)

		manager.expandDisksOnline(mockLibvirt.VirtDomain, domain, vmi)

		Expect(manager.guestDiskSizes["disk0"]).To(Equal(newCapacity))
	})

	unsupportedErr := libvirt.Error{Code: libvirt.ERR_OPERATION_UNSUPPORTED}

	It("should fall back to explicit size (BYTES) when CAPACITY is unsupported for overlay-on-block", func() {
		manager.guestDiskSizes["disk0"] = int64(1 * gb)
		domain := overlayBlockDomain(int64(2 * gb))
		vmi := pvcBackedVMI("disk0")

		const stubbedVirtualSize = int64(5 * gb)
		getDiskInfoForResize = func(path string) (*osdisk.DiskInfo, error) {
			Expect(path).To(Equal("/dev/vda"))
			return &osdisk.DiskInfo{VirtualSize: stubbedVirtualSize}, nil
		}

		gomock.InOrder(
			mockLibvirt.DomainEXPECT().BlockResize("/test/overlay.qcow2", uint64(0),
				libvirt.DOMAIN_BLOCK_RESIZE_BYTES|libvirt.DOMAIN_BLOCK_RESIZE_CAPACITY).Times(1).Return(unsupportedErr),
			mockLibvirt.DomainEXPECT().BlockResize("/test/overlay.qcow2", uint64(stubbedVirtualSize),
				libvirt.DOMAIN_BLOCK_RESIZE_BYTES).Times(1).Return(nil),
		)

		manager.expandDisksOnline(mockLibvirt.VirtDomain, domain, vmi)

		Expect(manager.guestDiskSizes["disk0"]).To(Equal(int64(2 * gb)))
	})

	It("should not update the map when the fallback BlockResize also fails", func() {
		manager.guestDiskSizes["disk0"] = int64(1 * gb)
		domain := overlayBlockDomain(int64(2 * gb))
		vmi := pvcBackedVMI("disk0")

		getDiskInfoForResize = func(path string) (*osdisk.DiskInfo, error) {
			return &osdisk.DiskInfo{VirtualSize: int64(5 * gb)}, nil
		}

		gomock.InOrder(
			mockLibvirt.DomainEXPECT().BlockResize("/test/overlay.qcow2", uint64(0),
				libvirt.DOMAIN_BLOCK_RESIZE_BYTES|libvirt.DOMAIN_BLOCK_RESIZE_CAPACITY).Times(1).Return(unsupportedErr),
			mockLibvirt.DomainEXPECT().BlockResize("/test/overlay.qcow2", uint64(5*gb),
				libvirt.DOMAIN_BLOCK_RESIZE_BYTES).Times(1).Return(fmt.Errorf("fallback resize failed")),
		)

		manager.expandDisksOnline(mockLibvirt.VirtDomain, domain, vmi)

		Expect(manager.guestDiskSizes["disk0"]).To(Equal(int64(1 * gb)))
	})

	It("should not call the fallback BlockResize when getDiskInfoForResize fails", func() {
		manager.guestDiskSizes["disk0"] = int64(1 * gb)
		domain := overlayBlockDomain(int64(2 * gb))
		vmi := pvcBackedVMI("disk0")

		getDiskInfoForResize = func(path string) (*osdisk.DiskInfo, error) {
			return nil, fmt.Errorf("qemu-img info failed")
		}

		mockLibvirt.DomainEXPECT().BlockResize("/test/overlay.qcow2", uint64(0),
			libvirt.DOMAIN_BLOCK_RESIZE_BYTES|libvirt.DOMAIN_BLOCK_RESIZE_CAPACITY).Times(1).Return(unsupportedErr)

		manager.expandDisksOnline(mockLibvirt.VirtDomain, domain, vmi)

		Expect(manager.guestDiskSizes["disk0"]).To(Equal(int64(1 * gb)))
	})

	DescribeTable("should not call the fallback BlockResize when the data-file virtual size is not positive",
		func(virtualSize int64) {
			manager.guestDiskSizes["disk0"] = int64(1 * gb)
			domain := overlayBlockDomain(int64(2 * gb))
			vmi := pvcBackedVMI("disk0")

			getDiskInfoForResize = func(path string) (*osdisk.DiskInfo, error) {
				return &osdisk.DiskInfo{VirtualSize: virtualSize}, nil
			}

			mockLibvirt.DomainEXPECT().BlockResize("/test/overlay.qcow2", uint64(0),
				libvirt.DOMAIN_BLOCK_RESIZE_BYTES|libvirt.DOMAIN_BLOCK_RESIZE_CAPACITY).Times(1).Return(unsupportedErr)

			manager.expandDisksOnline(mockLibvirt.VirtDomain, domain, vmi)

			Expect(manager.guestDiskSizes["disk0"]).To(Equal(int64(1 * gb)))
		},
		Entry("zero virtual size", int64(0)),
		Entry("negative virtual size", int64(-1)),
	)

	It("should not fall back on a non-unsupported error for overlay-on-block", func() {
		manager.guestDiskSizes["disk0"] = int64(1 * gb)
		domain := overlayBlockDomain(int64(2 * gb))
		vmi := pvcBackedVMI("disk0")

		internalErr := libvirt.Error{Code: libvirt.ERR_INTERNAL_ERROR}
		mockLibvirt.DomainEXPECT().BlockResize("/test/overlay.qcow2", uint64(0),
			libvirt.DOMAIN_BLOCK_RESIZE_BYTES|libvirt.DOMAIN_BLOCK_RESIZE_CAPACITY).Times(1).Return(internalErr)

		manager.expandDisksOnline(mockLibvirt.VirtDomain, domain, vmi)

		Expect(manager.guestDiskSizes["disk0"]).To(Equal(int64(1 * gb)))
	})

	It("should not fall back for a direct block device even on an unsupported error", func() {
		manager.guestDiskSizes["disk0"] = int64(1 * gb)
		newCapacity := int64(2 * gb)
		domain := &api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					Disks: []api.Disk{{
						Source:   api.DiskSource{Dev: "/dev/vda"},
						Alias:    api.NewUserDefinedAlias("disk0"),
						Capacity: &newCapacity,
					}},
				},
			},
		}
		vmi := pvcBackedVMI("disk0")

		mockLibvirt.DomainEXPECT().BlockResize("/dev/vda", uint64(0),
			libvirt.DOMAIN_BLOCK_RESIZE_BYTES|libvirt.DOMAIN_BLOCK_RESIZE_CAPACITY).Times(1).Return(unsupportedErr)

		manager.expandDisksOnline(mockLibvirt.VirtDomain, domain, vmi)

		Expect(manager.guestDiskSizes["disk0"]).To(Equal(int64(1 * gb)))
	})

	// resizeWithFallback is called directly: the guard on the data-file being a block device
	// is exercised without involving stat/statfs on a real file.
	It("should not fall back for overlay-on-file on an unsupported error", func() {
		ds := disksource.Resolve(api.Disk{
			Source: api.DiskSource{
				File:      "/test/overlay.qcow2",
				DataStore: &api.DataStore{Type: "file", Source: &api.DiskSource{File: "/test/disk.img"}},
			},
		})
		Expect(ds.HasOverlay()).To(BeTrue())
		Expect(ds.BackendIsBlock()).To(BeFalse())
		rszArgs := blockResizeArgs{size: uint64(2 * gb), flags: libvirt.DOMAIN_BLOCK_RESIZE_BYTES}

		mockLibvirt.DomainEXPECT().BlockResize("/test/overlay.qcow2", rszArgs.size, rszArgs.flags).
			Times(1).Return(unsupportedErr)

		err := resizeWithFallback(mockLibvirt.VirtDomain, ds, rszArgs, log.DefaultLogger())

		Expect(err).To(Equal(unsupportedErr))
	})

	It("should only seed guestDiskSizes for overlay-on-block on first sync without calling BlockResize", func() {
		capacity := int64(2 * gb)
		domain := overlayBlockDomain(capacity)
		vmi := pvcBackedVMI("disk0")

		manager.expandDisksOnline(mockLibvirt.VirtDomain, domain, vmi)

		Expect(manager.guestDiskSizes["disk0"]).To(Equal(capacity))
	})

	It("should not call BlockResize when capacity decreases", func() {
		oldCapacity := int64(2 * gb)
		newCapacity := int64(1 * gb)
		manager.guestDiskSizes["disk0"] = oldCapacity

		domain := &api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					Disks: []api.Disk{{
						Source:   api.DiskSource{Dev: "/dev/vda"},
						Alias:    api.NewUserDefinedAlias("disk0"),
						Capacity: &newCapacity,
					}},
				},
			},
		}
		vmi := pvcBackedVMI("disk0")

		manager.expandDisksOnline(mockLibvirt.VirtDomain, domain, vmi)

		Expect(manager.guestDiskSizes["disk0"]).To(Equal(oldCapacity))
	})

	It("should not update map when BlockResize fails", func() {
		oldCapacity := int64(1 * gb)
		newCapacity := int64(2 * gb)
		manager.guestDiskSizes["disk0"] = oldCapacity

		domain := &api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					Disks: []api.Disk{{
						Source:   api.DiskSource{Dev: "/dev/vda"},
						Alias:    api.NewUserDefinedAlias("disk0"),
						Capacity: &newCapacity,
					}},
				},
			},
		}
		vmi := pvcBackedVMI("disk0")

		mockLibvirt.DomainEXPECT().BlockResize(gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1).Return(fmt.Errorf("resize failed"))

		manager.expandDisksOnline(mockLibvirt.VirtDomain, domain, vmi)

		Expect(manager.guestDiskSizes["disk0"]).To(Equal(oldCapacity))
	})

	It("should track PVC capacity for file-backed disks", func() {
		capacity := int64(42 * gb)
		domain := &api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					Disks: []api.Disk{{
						FilesystemOverhead: virtpointer.P(v1.Percent("0.05")),
						Capacity:           &capacity,
						Source:             api.DiskSource{File: existingFilePath},
						Alias:              api.NewUserDefinedAlias("disk0"),
					}},
				},
			},
		}
		vmi := pvcBackedVMI("disk0")

		manager.expandDisksOnline(mockLibvirt.VirtDomain, domain, vmi)

		Expect(manager.guestDiskSizes["disk0"]).To(Equal(capacity))
	})

	It("should skip disks without PVC backing", func() {
		capacity := int64(2 * gb)
		domain := &api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					Disks: []api.Disk{{
						Source:   api.DiskSource{Dev: "/dev/vda"},
						Alias:    api.NewUserDefinedAlias("disk0"),
						Capacity: &capacity,
					}},
				},
			},
		}
		vmi := api2.NewMinimalVMI("test-vmi")

		manager.expandDisksOnline(mockLibvirt.VirtDomain, domain, vmi)

		Expect(manager.guestDiskSizes).To(BeEmpty())
	})

	It("should skip disks without capacity", func() {
		domain := &api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					Disks: []api.Disk{{
						Source: api.DiskSource{Dev: "/dev/vda"},
						Alias:  api.NewUserDefinedAlias("disk0"),
					}},
				},
			},
		}
		vmi := pvcBackedVMI("disk0")

		manager.expandDisksOnline(mockLibvirt.VirtDomain, domain, vmi)

		Expect(manager.guestDiskSizes).To(BeEmpty())
	})
})
