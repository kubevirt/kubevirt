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

package storage_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/storage"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

var _ = Describe("Disk source conversion", func() {
	newDisk := func() *api.Disk {
		return &api.Disk{Driver: &api.DiskDriver{}}
	}

	newContext := func() *convertertypes.ConverterContext {
		return &convertertypes.ConverterContext{
			IsBlockPVC: make(map[string]bool),
			IsBlockDV:  make(map[string]bool),
		}
	}

	type converterFunc func(string, *api.Disk, *convertertypes.ConverterContext) error

	DescribeTable("should convert hotplug PVC/DV to disk",
		func(fn converterFunc, volumeName string, isBlock, discardIgnore bool) {
			c := newContext()
			if isBlock {
				c.IsBlockPVC[volumeName] = true
				c.IsBlockDV[volumeName] = true
			}
			if discardIgnore {
				c.VolumesDiscardIgnore = []string{volumeName}
			}

			disk := newDisk()
			Expect(fn(volumeName, disk, c)).To(Succeed())

			Expect(disk.Driver.Type).To(Equal("raw"))
			Expect(disk.Driver.ErrorPolicy).To(Equal(v1.DiskErrorPolicyStop))

			if isBlock {
				Expect(disk.Type).To(Equal("block"))
				Expect(disk.Source.Dev).To(Equal(storage.GetHotplugBlockDeviceVolumePath(volumeName)))
			} else {
				Expect(disk.Type).To(Equal("file"))
				Expect(disk.Source.File).To(Equal(storage.GetHotplugFilesystemVolumePath(volumeName)))
			}

			if discardIgnore {
				Expect(disk.Driver.Discard).To(BeEmpty())
			} else {
				Expect(disk.Driver.Discard).To(Equal("unmap"))
			}
		},
		Entry("filesystem PVC", storage.Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk, "test-fs-pvc", false, false),
		Entry("block PVC", storage.Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk, "test-block-pvc", true, false),
		Entry("discard-ignore PVC", storage.Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk, "test-discard", false, true),
		Entry("filesystem DV", storage.Convert_v1_Hotplug_DataVolume_To_api_Disk, "test-fs-dv", false, false),
		Entry("block DV", storage.Convert_v1_Hotplug_DataVolume_To_api_Disk, "test-block-dv", true, false),
		Entry("discard-ignore DV", storage.Convert_v1_Hotplug_DataVolume_To_api_Disk, "test-discard", false, true),
	)

	DescribeTable("should convert non-hotplug PVC/DV to disk",
		func(fn converterFunc, volumeName string, isBlock bool) {
			c := newContext()
			if isBlock {
				c.IsBlockPVC[volumeName] = true
				c.IsBlockDV[volumeName] = true
			}

			disk := newDisk()
			Expect(fn(volumeName, disk, c)).To(Succeed())

			Expect(disk.Driver.Type).To(Equal("raw"))
			Expect(disk.Driver.ErrorPolicy).To(Equal(v1.DiskErrorPolicyStop))
			Expect(disk.Driver.Discard).To(Equal("unmap"))

			if isBlock {
				Expect(disk.Type).To(Equal("block"))
				Expect(disk.Source.Dev).To(Equal(storage.GetBlockDeviceVolumePath(volumeName)))
				Expect(disk.Source.Name).To(Equal(volumeName))
			} else {
				Expect(disk.Type).To(Equal("file"))
				Expect(disk.Source.File).To(Equal(storage.GetFilesystemVolumePath(volumeName)))
			}
		},
		Entry("filesystem PVC", storage.Convert_v1_PersistentVolumeClaim_To_api_Disk, "test-fs-pvc", false),
		Entry("block PVC", storage.Convert_v1_PersistentVolumeClaim_To_api_Disk, "test-block-pvc", true),
		Entry("filesystem DV", storage.Convert_v1_DataVolume_To_api_Disk, "test-fs-dv", false),
		Entry("block DV", storage.Convert_v1_DataVolume_To_api_Disk, "test-block-dv", true),
	)

	DescribeTable("should convert filesystem/block volume source to disk",
		func(fn func(string, *api.Disk, []string) error, volumeName string, isBlock, discardIgnore bool) {
			var volumesDiscardIgnore []string
			if discardIgnore {
				volumesDiscardIgnore = []string{volumeName}
			}

			disk := newDisk()
			Expect(fn(volumeName, disk, volumesDiscardIgnore)).To(Succeed())

			Expect(disk.Driver.Type).To(Equal("raw"))
			Expect(disk.Driver.ErrorPolicy).To(Equal(v1.DiskErrorPolicyStop))

			if isBlock {
				Expect(disk.Type).To(Equal("block"))
				Expect(disk.Source.Dev).To(Equal(storage.GetBlockDeviceVolumePath(volumeName)))
				Expect(disk.Source.Name).To(Equal(volumeName))
			} else {
				Expect(disk.Type).To(Equal("file"))
				Expect(disk.Source.File).To(Equal(storage.GetFilesystemVolumePath(volumeName)))
			}

			if discardIgnore {
				Expect(disk.Driver.Discard).To(BeEmpty())
			} else {
				Expect(disk.Driver.Discard).To(Equal("unmap"))
			}
		},
		Entry("filesystem", storage.Convert_v1_FilesystemVolumeSource_To_api_Disk, "test-fs", false, false),
		Entry("filesystem discard-ignore", storage.Convert_v1_FilesystemVolumeSource_To_api_Disk, "test-fs", false, true),
		Entry("block", storage.Convert_v1_BlockVolumeSource_To_api_Disk, "test-block", true, false),
		Entry("block discard-ignore", storage.Convert_v1_BlockVolumeSource_To_api_Disk, "test-block", true, true),
	)

	DescribeTable("should convert volume with CBT to disk with datastore",
		func(fn converterFunc, volumeName string, isBlock, isHotplug bool) {
			cbtPath := "/var/lib/libvirt/qemu/cbt/" + volumeName + ".qcow2"
			c := newContext()
			c.ApplyCBT = map[string]string{volumeName: cbtPath}
			if isBlock {
				c.IsBlockPVC[volumeName] = true
				c.IsBlockDV[volumeName] = true
			}

			disk := newDisk()
			Expect(fn(volumeName, disk, c)).To(Succeed())

			Expect(disk.Driver.Type).To(Equal("qcow2"))
			Expect(disk.Driver.ErrorPolicy).To(Equal(v1.DiskErrorPolicyStop))
			Expect(disk.Driver.Discard).To(Equal("unmap"))
			Expect(disk.Type).To(Equal("file"))
			Expect(disk.Source.File).To(Equal(cbtPath))

			Expect(disk.Source.DataStore).ToNot(BeNil())
			Expect(disk.Source.DataStore.Format).ToNot(BeNil())
			Expect(disk.Source.DataStore.Format.Type).To(Equal("raw"))
			Expect(disk.Source.DataStore.Source).ToNot(BeNil())

			if isBlock {
				Expect(disk.Source.DataStore.Type).To(Equal("block"))
				if isHotplug {
					Expect(disk.Source.DataStore.Source.Dev).To(Equal(storage.GetHotplugBlockDeviceVolumePath(volumeName)))
				} else {
					Expect(disk.Source.DataStore.Source.Dev).To(Equal(storage.GetBlockDeviceVolumePath(volumeName)))
					Expect(disk.Source.Name).To(Equal(volumeName))
				}
			} else {
				Expect(disk.Source.DataStore.Type).To(Equal("file"))
				if isHotplug {
					Expect(disk.Source.DataStore.Source.File).To(Equal(storage.GetHotplugFilesystemVolumePath(volumeName)))
				} else {
					Expect(disk.Source.DataStore.Source.File).To(Equal(storage.GetFilesystemVolumePath(volumeName)))
				}
			}
		},
		Entry("filesystem PVC", storage.Convert_v1_PersistentVolumeClaim_To_api_Disk, "test-pvc", false, false),
		Entry("block PVC", storage.Convert_v1_PersistentVolumeClaim_To_api_Disk, "test-pvc", true, false),
		Entry("filesystem DV", storage.Convert_v1_DataVolume_To_api_Disk, "test-dv", false, false),
		Entry("block DV", storage.Convert_v1_DataVolume_To_api_Disk, "test-dv", true, false),
		Entry("hotplug filesystem PVC", storage.Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk, "test-pvc", false, true),
		Entry("hotplug block PVC", storage.Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk, "test-pvc", true, true),
		Entry("hotplug filesystem DV", storage.Convert_v1_Hotplug_DataVolume_To_api_Disk, "test-dv", false, true),
		Entry("hotplug block DV", storage.Convert_v1_Hotplug_DataVolume_To_api_Disk, "test-dv", true, true),
	)
})
