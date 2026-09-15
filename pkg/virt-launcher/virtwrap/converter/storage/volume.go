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
	"fmt"
	"slices"

	v1 "kubevirt.io/api/core/v1"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/config"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/emptydisk"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/storage/volumepath"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const deviceTypeNotCompatibleFmt = "device %s is of type lun. Not compatible with a file based disk"

func toApiReadOnly(src bool) *api.ReadOnly {
	if src {
		return &api.ReadOnly{}
	}
	return nil
}

func setDiskDriver(disk *api.Disk, driverType string, discard bool) {
	disk.Driver.Type = driverType
	disk.Driver.ErrorPolicy = v1.DiskErrorPolicyStop
	if discard {
		disk.Driver.Discard = "unmap"
	}
}

func convertVolumeWithCBT(volumeName, cbtPath string, isBlock bool, disk *api.Disk, volumesDiscardIgnore []string) error {
	setDiskDriver(disk, "qcow2", !slices.Contains(volumesDiscardIgnore, volumeName))

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
			Dev: volumepath.BlockDevice(volumeName),
		}
	} else {
		disk.Source.DataStore.Type = "file"
		disk.Source.DataStore.Source = &api.DiskSource{
			File: volumepath.Filesystem(volumeName),
		}
	}

	return nil
}

func convertVolumeWithoutCBT(volumeName string, isBlock bool, disk *api.Disk, volumesDiscardIgnore []string) error {
	setDiskDriver(disk, "raw", !slices.Contains(volumesDiscardIgnore, volumeName))

	if isBlock {
		disk.Type = "block"
		disk.Source.Name = volumeName
		disk.Source.Dev = volumepath.BlockDevice(volumeName)
	} else {
		disk.Type = "file"
		disk.Source.File = volumepath.Filesystem(volumeName)
	}
	return nil
}

func convertHotplugVolumeWithCBT(volumeName, cbtPath string, isBlock bool, disk *api.Disk, volumesDiscardIgnore []string) error {
	setDiskDriver(disk, "qcow2", !slices.Contains(volumesDiscardIgnore, volumeName))

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
			Dev: volumepath.HotplugBlockDevice(volumeName),
		}
	} else {
		disk.Source.DataStore.Type = "file"
		disk.Source.DataStore.Source = &api.DiskSource{
			File: volumepath.HotplugFilesystem(volumeName),
		}
	}

	return nil
}

func convertHotplugVolumeWithoutCBT(volumeName string, isBlock bool, disk *api.Disk, volumesDiscardIgnore []string) error {
	setDiskDriver(disk, "raw", !slices.Contains(volumesDiscardIgnore, volumeName))

	if isBlock {
		disk.Type = "block"
		disk.Source.Dev = volumepath.HotplugBlockDevice(volumeName)
	} else {
		disk.Type = "file"
		disk.Source.File = volumepath.HotplugFilesystem(volumeName)
	}
	return nil
}

func convertHotplugVolumeSourceToDisk(volumeName, cbtPath string, isBlock bool, disk *api.Disk, volumesDiscardIgnore []string) error {
	if cbtPath != "" {
		return convertHotplugVolumeWithCBT(volumeName, cbtPath, isBlock, disk, volumesDiscardIgnore)
	}
	return convertHotplugVolumeWithoutCBT(volumeName, isBlock, disk, volumesDiscardIgnore)
}

func convertVolumeSourceToDisk(volumeName, cbtPath string, isBlock bool, disk *api.Disk, volumesDiscardIgnore []string) error {
	if cbtPath != "" {
		return convertVolumeWithCBT(volumeName, cbtPath, isBlock, disk, volumesDiscardIgnore)
	}
	return convertVolumeWithoutCBT(volumeName, isBlock, disk, volumesDiscardIgnore)
}

// convert_v1_FilesystemVolumeSource_To_api_Disk takes a FS source and builds the domain Disk representation
func convert_v1_FilesystemVolumeSource_To_api_Disk(volumeName string, disk *api.Disk, volumesDiscardIgnore []string) error {
	disk.Type = "file"
	setDiskDriver(disk, "raw", false)
	disk.Source.File = volumepath.Filesystem(volumeName)
	if !slices.Contains(volumesDiscardIgnore, volumeName) {
		disk.Driver.Discard = "unmap"
	}
	return nil
}

func convert_v1_BlockVolumeSource_To_api_Disk(volumeName string, disk *api.Disk, volumesDiscardIgnore []string) error {
	disk.Type = "block"
	setDiskDriver(disk, "raw", !slices.Contains(volumesDiscardIgnore, volumeName))
	disk.Source.Name = volumeName
	disk.Source.Dev = volumepath.BlockDevice(volumeName)
	return nil
}

func convert_v1_HostDisk_To_api_Disk(volumeName, path, cbtPath string, disk *api.Disk) error {
	disk.Type = "file"
	if cbtPath != "" {
		disk.Driver.Type = "qcow2"
		disk.Source.File = cbtPath
		disk.Source.DataStore = &api.DataStore{
			Type: "file",
			Format: &api.DataStoreFormat{
				Type: "raw",
			},
			Source: &api.DiskSource{
				File: hostdisk.GetMountedHostDiskPath(volumeName, path),
			},
		}
	} else {
		disk.Driver.Type = "raw"
		disk.Source.File = hostdisk.GetMountedHostDiskPath(volumeName, path)
	}
	disk.Driver.ErrorPolicy = v1.DiskErrorPolicyStop
	return nil
}

func convert_v1_SysprepSource_To_api_Disk(volumeName string, disk *api.Disk) error {
	if disk.Type == "lun" {
		return fmt.Errorf(deviceTypeNotCompatibleFmt, disk.Alias.GetName())
	}

	disk.Source.File = config.GetSysprepDiskPath(volumeName)
	disk.Type = "file"
	disk.Driver.Type = "raw"
	return nil
}

func convert_v1_CloudInitSource_To_api_Disk(source v1.VolumeSource, disk *api.Disk, vmiNamespace, vmiName string) error {
	if disk.Type == "lun" {
		return fmt.Errorf(deviceTypeNotCompatibleFmt, disk.Alias.GetName())
	}

	var dataSource cloudinit.DataSourceType
	if source.CloudInitNoCloud != nil {
		dataSource = cloudinit.DataSourceNoCloud
	} else if source.CloudInitConfigDrive != nil {
		dataSource = cloudinit.DataSourceConfigDrive
	} else {
		return fmt.Errorf("Only nocloud and configdrive are valid cloud-init volumes")
	}

	disk.Source.File = cloudinit.GetIsoFilePath(dataSource, vmiName, vmiNamespace)
	disk.Type = "file"
	setDiskDriver(disk, "raw", false)
	return nil
}

func convert_v1_DownwardMetricSource_To_api_Disk(disk *api.Disk, virtioModel string) error {
	disk.Type = "file"
	disk.ReadOnly = toApiReadOnly(true)
	disk.Driver = &api.DiskDriver{
		Type: "raw",
		Name: "qemu",
	}
	// This disk always needs `virtio`. Validation ensures that bus is unset or is already virtio
	disk.Model = virtioModel
	disk.Source = api.DiskSource{
		File: config.DownwardMetricDisk,
	}
	return nil
}

func convert_v1_EmptyDiskSource_To_api_Disk(volumeName string, _ *v1.EmptyDiskSource, disk *api.Disk) error {
	if disk.Type == "lun" {
		return fmt.Errorf(deviceTypeNotCompatibleFmt, disk.Alias.GetName())
	}

	disk.Type = "file"
	disk.Source.File = emptydisk.NewEmptyDiskCreator().FilePathForVolumeName(volumeName)
	setDiskDriver(disk, "qcow2", true)

	return nil
}

func convert_v1_ContainerDiskSource_To_api_Disk(volumeName string, _ *v1.ContainerDiskSource, disk *api.Disk, ephemeralDiskPath string, diskFormat string) error {
	if disk.Type == "lun" {
		return fmt.Errorf(deviceTypeNotCompatibleFmt, disk.Alias.GetName())
	}
	disk.Type = "file"
	setDiskDriver(disk, "qcow2", true)
	disk.Source.File = ephemeralDiskPath
	disk.BackingStore = &api.BackingStore{
		Format: &api.BackingStoreFormat{},
		Source: &api.DiskSource{},
	}

	source := containerdisk.GetDiskTargetPathFromLauncherView(volumeName)
	disk.BackingStore.Format.Type = diskFormat
	disk.BackingStore.Source.File = source
	disk.BackingStore.Type = "file"

	return nil
}

func convert_v1_EphemeralVolumeSource_To_api_Disk(volumeName, ephemeralDiskPath string, isBlock bool, disk *api.Disk, volumesDiscardIgnore []string) error {
	disk.Type = "file"
	setDiskDriver(disk, "qcow2", true)
	disk.Source.File = ephemeralDiskPath
	disk.BackingStore = &api.BackingStore{
		Format: &api.BackingStoreFormat{},
		Source: &api.DiskSource{},
	}

	backingDisk := &api.Disk{Driver: &api.DiskDriver{}}
	if isBlock {
		if err := convert_v1_BlockVolumeSource_To_api_Disk(volumeName, backingDisk, volumesDiscardIgnore); err != nil {
			return err
		}
	} else {
		if err := convert_v1_FilesystemVolumeSource_To_api_Disk(volumeName, backingDisk, volumesDiscardIgnore); err != nil {
			return err
		}
	}
	disk.BackingStore.Format.Type = backingDisk.Driver.Type
	disk.BackingStore.Source = &backingDisk.Source
	disk.BackingStore.Type = backingDisk.Type

	return nil
}

// convert_v1_Missing_Volume_To_api_Disk sets defaults when no volume for disk (cdrom, floppy, etc) is provided
func convert_v1_Missing_Volume_To_api_Disk(disk *api.Disk) error {
	disk.Type = "block"
	disk.Driver.Type = "raw"
	disk.Driver.Discard = "unmap"
	return nil
}

func convert_v1_Config_To_api_Disk(volumeName string, disk *api.Disk, configType config.Type) error {
	disk.Type = "file"
	setDiskDriver(disk, "raw", false)
	switch configType {
	case config.ConfigMap:
		disk.Source.File = config.GetConfigMapDiskPath(volumeName)
	case config.Secret:
		disk.Source.File = config.GetSecretDiskPath(volumeName)
	case config.DownwardAPI:
		disk.Source.File = config.GetDownwardAPIDiskPath(volumeName)
	case config.ServiceAccount:
		disk.Source.File = config.GetServiceAccountDiskPath()
	default:
		return fmt.Errorf("Cannot convert config '%s' to disk, unrecognized type", configType)
	}

	return nil
}
