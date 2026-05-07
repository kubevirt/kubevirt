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
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/virtio"
)

func Convert_v1_Volume_To_api_Disk(source *v1.Volume, disk *api.Disk, c *convertertypes.ConverterContext, diskIndex int) error {

	if source.ContainerDisk != nil {
		return Convert_v1_ContainerDiskSource_To_api_Disk(source.Name, source.ContainerDisk, disk, c, diskIndex)
	}

	if source.CloudInitNoCloud != nil || source.CloudInitConfigDrive != nil {
		return Convert_v1_CloudInitSource_To_api_Disk(source.VolumeSource, disk, c)
	}

	if source.Sysprep != nil {
		return Convert_v1_SysprepSource_To_api_Disk(source.Name, disk)
	}

	if source.HostDisk != nil {
		return Convert_v1_HostDisk_To_api_Disk(source.Name, source.HostDisk.Path, disk, c)
	}

	if source.PersistentVolumeClaim != nil {
		return Convert_v1_PersistentVolumeClaim_To_api_Disk(source.Name, disk, c)
	}

	if source.DataVolume != nil {
		return Convert_v1_DataVolume_To_api_Disk(source.Name, disk, c)
	}

	if source.Ephemeral != nil {
		return Convert_v1_EphemeralVolumeSource_To_api_Disk(source.Name, disk, c)
	}
	if source.EmptyDisk != nil {
		return Convert_v1_EmptyDiskSource_To_api_Disk(source.Name, source.EmptyDisk, disk)
	}
	if source.ConfigMap != nil {
		return Convert_v1_Config_To_api_Disk(source.Name, disk, config.ConfigMap)
	}
	if source.Secret != nil {
		return Convert_v1_Config_To_api_Disk(source.Name, disk, config.Secret)
	}
	if source.DownwardAPI != nil {
		return Convert_v1_Config_To_api_Disk(source.Name, disk, config.DownwardAPI)
	}
	if source.ServiceAccount != nil {
		return Convert_v1_Config_To_api_Disk(source.Name, disk, config.ServiceAccount)
	}
	if source.DownwardMetrics != nil {
		return Convert_v1_DownwardMetricSource_To_api_Disk(disk, c)
	}

	return fmt.Errorf("disk %s references an unsupported source", disk.Alias.GetName())
}

// Convert_v1_Hotplug_Volume_To_api_Disk convers a hotplug volume to an api disk
func Convert_v1_Hotplug_Volume_To_api_Disk(source *v1.Volume, disk *api.Disk, c *convertertypes.ConverterContext) error {
	// This is here because virt-handler before passing the VMI here replaces all PVCs with host disks in
	// hostdisk.ReplacePVCByHostDisk not quite sure why, but it broken hot plugging PVCs
	if source.HostDisk != nil {
		return Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk(source.Name, disk, c)
	}

	if source.PersistentVolumeClaim != nil {
		return Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk(source.Name, disk, c)
	}

	if source.DataVolume != nil {
		return Convert_v1_Hotplug_DataVolume_To_api_Disk(source.Name, disk, c)
	}
	return fmt.Errorf("hotplug disk %s references an unsupported source", disk.Alias.GetName())
}

// Convert_v1_Missing_Volume_To_api_Disk sets defaults when no volume for disk (cdrom, floppy, etc) is provided
func Convert_v1_Missing_Volume_To_api_Disk(disk *api.Disk) error {
	disk.Type = "block"
	disk.Driver.Type = "raw"
	disk.Driver.Discard = "unmap"
	return nil
}

func Convert_v1_Config_To_api_Disk(volumeName string, disk *api.Disk, configType config.Type) error {
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
	setDiskDriver(disk, "raw", !slices.Contains(volumesDiscardIgnore, volumeName))

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
	setDiskDriver(disk, "raw", !slices.Contains(volumesDiscardIgnore, volumeName))

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

// Convert_v1_FilesystemVolumeSource_To_api_Disk takes a FS source and builds the domain Disk representation
func Convert_v1_FilesystemVolumeSource_To_api_Disk(volumeName string, disk *api.Disk, volumesDiscardIgnore []string) error {
	disk.Type = "file"
	setDiskDriver(disk, "raw", false)
	disk.Source.File = GetFilesystemVolumePath(volumeName)
	if !slices.Contains(volumesDiscardIgnore, volumeName) {
		disk.Driver.Discard = "unmap"
	}
	return nil
}

func Convert_v1_BlockVolumeSource_To_api_Disk(volumeName string, disk *api.Disk, volumesDiscardIgnore []string) error {
	disk.Type = "block"
	setDiskDriver(disk, "raw", !slices.Contains(volumesDiscardIgnore, volumeName))
	disk.Source.Name = volumeName
	disk.Source.Dev = GetBlockDeviceVolumePath(volumeName)
	return nil
}

func Convert_v1_HostDisk_To_api_Disk(volumeName string, path string, disk *api.Disk, c *convertertypes.ConverterContext) error {
	disk.Type = "file"
	if cbtPath, ok := c.ApplyCBT[volumeName]; ok {
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

func Convert_v1_SysprepSource_To_api_Disk(volumeName string, disk *api.Disk) error {
	if disk.Type == "lun" {
		return fmt.Errorf(DeviceTypeNotCompatibleFmt, disk.Alias.GetName())
	}

	disk.Source.File = config.GetSysprepDiskPath(volumeName)
	disk.Type = "file"
	disk.Driver.Type = "raw"
	return nil
}

func Convert_v1_CloudInitSource_To_api_Disk(source v1.VolumeSource, disk *api.Disk, c *convertertypes.ConverterContext) error {
	if disk.Type == "lun" {
		return fmt.Errorf(DeviceTypeNotCompatibleFmt, disk.Alias.GetName())
	}

	var dataSource cloudinit.DataSourceType
	if source.CloudInitNoCloud != nil {
		dataSource = cloudinit.DataSourceNoCloud
	} else if source.CloudInitConfigDrive != nil {
		dataSource = cloudinit.DataSourceConfigDrive
	} else {
		return fmt.Errorf("Only nocloud and configdrive are valid cloud-init volumes")
	}

	disk.Source.File = cloudinit.GetIsoFilePath(dataSource, c.VirtualMachine.Name, c.VirtualMachine.Namespace)
	disk.Type = "file"
	setDiskDriver(disk, "raw", false)
	return nil
}

func Convert_v1_DownwardMetricSource_To_api_Disk(disk *api.Disk, c *convertertypes.ConverterContext) error {
	disk.Type = "file"
	disk.ReadOnly = ToAPIReadOnly(true)
	disk.Driver = &api.DiskDriver{
		Type: "raw",
		Name: "qemu",
	}
	// This disk always needs `virtio`. Validation ensures that bus is unset or is already virtio
	disk.Model = virtio.InterpretTransitionalModelType(&c.UseVirtioTransitional, c.Architecture.GetArchitecture())
	disk.Source = api.DiskSource{
		File: config.DownwardMetricDisk,
	}
	return nil
}

func Convert_v1_EmptyDiskSource_To_api_Disk(volumeName string, _ *v1.EmptyDiskSource, disk *api.Disk) error {
	if disk.Type == "lun" {
		return fmt.Errorf(DeviceTypeNotCompatibleFmt, disk.Alias.GetName())
	}

	disk.Type = "file"
	disk.Source.File = emptydisk.NewEmptyDiskCreator().FilePathForVolumeName(volumeName)
	setDiskDriver(disk, "qcow2", true)

	return nil
}

func Convert_v1_ContainerDiskSource_To_api_Disk(volumeName string, _ *v1.ContainerDiskSource, disk *api.Disk, c *convertertypes.ConverterContext, diskIndex int) error {
	if disk.Type == "lun" {
		return fmt.Errorf(DeviceTypeNotCompatibleFmt, disk.Alias.GetName())
	}
	disk.Type = "file"
	setDiskDriver(disk, "qcow2", true)
	disk.Source.File = c.EphemeraldiskCreator.GetFilePath(volumeName)
	disk.BackingStore = &api.BackingStore{
		Format: &api.BackingStoreFormat{},
		Source: &api.DiskSource{},
	}

	source := containerdisk.GetDiskTargetPathFromLauncherView(diskIndex)
	if info := c.DisksInfo[volumeName]; info != nil {
		disk.BackingStore.Format.Type = info.Format
	} else {
		return fmt.Errorf("no disk info provided for volume %s", volumeName)
	}
	disk.BackingStore.Source.File = source
	disk.BackingStore.Type = "file"

	return nil
}

func Convert_v1_EphemeralVolumeSource_To_api_Disk(volumeName string, disk *api.Disk, c *convertertypes.ConverterContext) error {
	disk.Type = "file"
	setDiskDriver(disk, "qcow2", true)
	disk.Source.File = c.EphemeraldiskCreator.GetFilePath(volumeName)
	disk.BackingStore = &api.BackingStore{
		Format: &api.BackingStoreFormat{},
		Source: &api.DiskSource{},
	}

	backingDisk := &api.Disk{Driver: &api.DiskDriver{}}
	if c.IsBlockPVC[volumeName] {
		if err := Convert_v1_BlockVolumeSource_To_api_Disk(volumeName, backingDisk, c.VolumesDiscardIgnore); err != nil {
			return err
		}
	} else {
		if err := Convert_v1_FilesystemVolumeSource_To_api_Disk(volumeName, backingDisk, c.VolumesDiscardIgnore); err != nil {
			return err
		}
	}
	disk.BackingStore.Format.Type = backingDisk.Driver.Type
	disk.BackingStore.Source = &backingDisk.Source
	disk.BackingStore.Type = backingDisk.Type

	return nil
}
