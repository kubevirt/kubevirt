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

const (
	diskTypeBlock = "block"
	diskTypeFile  = "file"
)

func SetDiskDriver(disk *api.Disk, driverType string, discard bool) {
	disk.Driver.Type = driverType
	disk.Driver.ErrorPolicy = v1.DiskErrorPolicyStop
	if discard {
		disk.Driver.Discard = "unmap"
	}
}

func setDiskSource(disk *api.Disk, volumeName string, isBlock, isHotplug bool, devPath string) {
	if isBlock {
		disk.Type = diskTypeBlock
		disk.Source.Dev = devPath
		if !isHotplug {
			disk.Source.Name = volumeName
		}
	} else {
		disk.Type = diskTypeFile
		disk.Source.File = devPath
	}
}

func newDataStore(isBlock bool, devPath string) *api.DataStore {
	ds := &api.DataStore{
		Format: &api.DataStoreFormat{Type: "raw"},
	}
	if isBlock {
		ds.Type = diskTypeBlock
		ds.Source = &api.DiskSource{Dev: devPath}
	} else {
		ds.Type = diskTypeFile
		ds.Source = &api.DiskSource{File: devPath}
	}
	return ds
}

func resolveStorageBackend(volumeName, cbtPath string, isBlock, isHotplug bool, disk *api.Disk, volumesDiscardIgnore []string) {
	discard := !slices.Contains(volumesDiscardIgnore, volumeName)
	devPath := GetVolumeImagePath(volumeName, isBlock, isHotplug)
	if cbtPath != "" {
		SetDiskDriver(disk, "qcow2", discard)
		disk.Type = diskTypeFile
		disk.Source.File = cbtPath
		disk.Source.DataStore = newDataStore(isBlock, devPath)
		if isBlock && !isHotplug {
			disk.Source.Name = volumeName
		}
	} else {
		SetDiskDriver(disk, "raw", discard)
		setDiskSource(disk, volumeName, isBlock, isHotplug, devPath)
	}
}

func Convert_v1_PersistentVolumeClaim_To_api_Disk(name string, disk *api.Disk, c *convertertypes.ConverterContext) error { //nolint:staticcheck,lll
	resolveStorageBackend(name, c.ApplyCBT[name], c.IsBlockPVC[name], false, disk, c.VolumesDiscardIgnore)
	return nil
}

func Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk(name string, disk *api.Disk, c *convertertypes.ConverterContext) error { //nolint:staticcheck,lll
	resolveStorageBackend(name, c.ApplyCBT[name], c.IsBlockPVC[name], true, disk, c.VolumesDiscardIgnore)
	return nil
}

func Convert_v1_DataVolume_To_api_Disk(name string, disk *api.Disk, c *convertertypes.ConverterContext) error { //nolint:staticcheck
	resolveStorageBackend(name, c.ApplyCBT[name], c.IsBlockDV[name], false, disk, c.VolumesDiscardIgnore)
	return nil
}

func Convert_v1_Hotplug_DataVolume_To_api_Disk(name string, disk *api.Disk, c *convertertypes.ConverterContext) error { //nolint:staticcheck
	resolveStorageBackend(name, c.ApplyCBT[name], c.IsBlockDV[name], true, disk, c.VolumesDiscardIgnore)
	return nil
}

func Convert_v1_FilesystemVolumeSource_To_api_Disk(volumeName string, disk *api.Disk, volumesDiscardIgnore []string) error { //nolint:staticcheck,lll
	resolveStorageBackend(volumeName, "", false, false, disk, volumesDiscardIgnore)
	return nil
}

func Convert_v1_BlockVolumeSource_To_api_Disk(volumeName string, disk *api.Disk, volumesDiscardIgnore []string) error { //nolint:staticcheck,lll
	resolveStorageBackend(volumeName, "", true, false, disk, volumesDiscardIgnore)
	return nil
}
