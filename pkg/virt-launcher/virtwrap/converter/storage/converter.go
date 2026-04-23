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

package storage

import (
	"slices"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

func SetDiskDriver(disk *api.Disk, driverType string, discard bool) {
	disk.Driver.Type = driverType
	disk.Driver.ErrorPolicy = v1.DiskErrorPolicyStop
	if discard {
		disk.Driver.Discard = "unmap"
	}
}

func convertVolumeWithCBT(volumeName, cbtPath string, isBlock bool, disk *api.Disk, volumesDiscardIgnore []string) error {
	SetDiskDriver(disk, "qcow2", !slices.Contains(volumesDiscardIgnore, volumeName))

	disk.Type = "file"
	disk.Source.File = cbtPath
	disk.Source.DataStore = &api.DataStore{
		Format: &api.DataStoreFormat{
			Type: "raw",
		},
	}

	if isBlock {
		disk.Source.Name = volumeName
		disk.Source.DataStore.Type = "block"
		disk.Source.DataStore.Source = &api.DiskSource{
			Dev: GetBlockDeviceVolumePath(volumeName),
		}
	} else {
		disk.Source.DataStore.Type = "file"
		disk.Source.DataStore.Source = &api.DiskSource{
			File: GetFilesystemVolumePath(volumeName),
		}
	}

	return nil
}

func convertVolumeWithoutCBT(volumeName string, isBlock bool, disk *api.Disk, volumesDiscardIgnore []string) error {
	SetDiskDriver(disk, "raw", !slices.Contains(volumesDiscardIgnore, volumeName))

	if isBlock {
		disk.Type = "block"
		disk.Source.Name = volumeName
		disk.Source.Dev = GetBlockDeviceVolumePath(volumeName)
	} else {
		disk.Type = "file"
		disk.Source.File = GetFilesystemVolumePath(volumeName)
	}
	return nil
}

func convertHotplugVolumeWithCBT(volumeName, cbtPath string, isBlock bool, disk *api.Disk, volumesDiscardIgnore []string) error {
	SetDiskDriver(disk, "qcow2", !slices.Contains(volumesDiscardIgnore, volumeName))

	disk.Type = "file"
	disk.Source.File = cbtPath
	disk.Source.DataStore = &api.DataStore{
		Format: &api.DataStoreFormat{
			Type: "raw",
		},
	}

	if isBlock {
		disk.Source.DataStore.Type = "block"
		disk.Source.DataStore.Source = &api.DiskSource{
			Dev: GetHotplugBlockDeviceVolumePath(volumeName),
		}
	} else {
		disk.Source.DataStore.Type = "file"
		disk.Source.DataStore.Source = &api.DiskSource{
			File: GetHotplugFilesystemVolumePath(volumeName),
		}
	}

	return nil
}

func convertHotplugVolumeWithoutCBT(volumeName string, isBlock bool, disk *api.Disk, volumesDiscardIgnore []string) error {
	SetDiskDriver(disk, "raw", !slices.Contains(volumesDiscardIgnore, volumeName))

	if isBlock {
		disk.Type = "block"
		disk.Source.Dev = GetHotplugBlockDeviceVolumePath(volumeName)
	} else {
		disk.Type = "file"
		disk.Source.File = GetHotplugFilesystemVolumePath(volumeName)
	}
	return nil
}

func ConvertHotplugVolumeSourceToDisk(volumeName, cbtPath string, isBlock bool, disk *api.Disk, volumesDiscardIgnore []string) error {
	if cbtPath != "" {
		return convertHotplugVolumeWithCBT(volumeName, cbtPath, isBlock, disk, volumesDiscardIgnore)
	}
	return convertHotplugVolumeWithoutCBT(volumeName, isBlock, disk, volumesDiscardIgnore)
}

func ConvertVolumeSourceToDisk(volumeName, cbtPath string, isBlock bool, disk *api.Disk, volumesDiscardIgnore []string) error {
	if cbtPath != "" {
		return convertVolumeWithCBT(volumeName, cbtPath, isBlock, disk, volumesDiscardIgnore)
	}
	return convertVolumeWithoutCBT(volumeName, isBlock, disk, volumesDiscardIgnore)
}

func Convert_v1_PersistentVolumeClaim_To_api_Disk(name string, disk *api.Disk, c *convertertypes.ConverterContext) error {
	return ConvertVolumeSourceToDisk(name, c.ApplyCBT[name], c.IsBlockPVC[name], disk, c.VolumesDiscardIgnore)
}

// Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk converts a Hotplugged PVC to an api disk
func Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk(name string, disk *api.Disk, c *convertertypes.ConverterContext) error {
	return ConvertHotplugVolumeSourceToDisk(name, c.ApplyCBT[name], c.IsBlockPVC[name], disk, c.VolumesDiscardIgnore)
}

func Convert_v1_DataVolume_To_api_Disk(name string, disk *api.Disk, c *convertertypes.ConverterContext) error {
	return ConvertVolumeSourceToDisk(name, c.ApplyCBT[name], c.IsBlockDV[name], disk, c.VolumesDiscardIgnore)
}

// Convert_v1_Hotplug_DataVolume_To_api_Disk converts a Hotplugged DataVolume to an api disk
func Convert_v1_Hotplug_DataVolume_To_api_Disk(name string, disk *api.Disk, c *convertertypes.ConverterContext) error {
	return ConvertHotplugVolumeSourceToDisk(name, c.ApplyCBT[name], c.IsBlockDV[name], disk, c.VolumesDiscardIgnore)
}
