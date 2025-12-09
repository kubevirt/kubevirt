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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/config"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/emptydisk"
	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/storage/reservation"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/virtio"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

type DiskConfigurator struct {
	architecture          string
	hotplugVolumes        map[string]v1.VolumeStatus
	permanentVolumes      map[string]v1.VolumeStatus
	disksInfo             map[string]*disk.DiskInfo
	isBlockPVC            map[string]bool
	isBlockDV             map[string]bool
	applyCBT              map[string]string
	useVirtioTransitional bool
	useLaunchSecuritySEV  bool
	useLaunchSecurityPV   bool
	expandDisksEnabled    bool
	useBlkMQ              bool
	vcpus                 uint
	volumesDiscardIgnore  []string
	ephemeralDiskCreator  ephemeraldisk.EphemeralDiskCreatorInterface
}

const (
	deviceTypeNotCompatibleFmt = "device %s is of type lun. Not compatible with a file based disk"
)

type option func(*DiskConfigurator)

func NewDiskConfigurator(options ...option) DiskConfigurator {
	var configurator DiskConfigurator

	for _, f := range options {
		f(&configurator)
	}

	return configurator
}

func (d DiskConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	volumeIndices := map[string]int{}
	volumes := map[string]*v1.Volume{}
	for i, volume := range vmi.Spec.Volumes {
		volumes[volume.Name] = volume.DeepCopy()
		volumeIndices[volume.Name] = i
	}

	volumeStatusMap := make(map[string]v1.VolumeStatus)
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		volumeStatusMap[volumeStatus.Name] = volumeStatus
	}

	prefixMap := newDeviceNamer(vmi.Status.VolumeStatus, vmi.Spec.Domain.Devices.Disks)
	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		newDisk := api.Disk{}
		emptyCDRom := false

		if err := d.Convert_v1_Disk_To_api_Disk(&disk, &newDisk, prefixMap, volumeStatusMap); err != nil {
			return err
		}
		volume := volumes[disk.Name]
		if volume == nil {
			if disk.CDRom == nil {
				return fmt.Errorf("no matching volume with name %s found", disk.Name)
			}
			emptyCDRom = true
		}

		hpStatus, hpOk := d.hotplugVolumes[disk.Name]
		var err error
		switch {
		case emptyCDRom:
			err = d.Convert_v1_Missing_Volume_To_api_Disk(&newDisk)
		case hpOk:
			err = d.Convert_v1_Hotplug_Volume_To_api_Disk(volume, &newDisk)
		default:
			err = d.Convert_v1_Volume_To_api_Disk(vmi, volume, &newDisk, volumeIndices[disk.Name])
		}

		if err != nil {
			return err
		}

		if err := Convert_v1_BlockSize_To_api_BlockIO(&disk, &newDisk); err != nil {
			return err
		}

		_, isPermVolume := d.permanentVolumes[disk.Name]
		// if len(c.PermanentVolumes) == 0, it means the vmi is not ready yet, add all disks
		permReady := isPermVolume || len(d.permanentVolumes) == 0
		hotplugReady := hpOk && (hpStatus.Phase == v1.HotplugVolumeMounted || hpStatus.Phase == v1.VolumeReady)

		if permReady || hotplugReady || emptyCDRom {
			domain.Spec.Devices.Disks = append(domain.Spec.Devices.Disks, newDisk)
		}
		if err := setErrorPolicy(&disk, &newDisk); err != nil {
			return err
		}
	}

	return nil
}

func (d *DiskConfigurator) Convert_v1_Disk_To_api_Disk(diskDevice *v1.Disk, disk *api.Disk, prefixMap map[string]deviceNamer, volumeStatusMap map[string]v1.VolumeStatus) error {
	if diskDevice.Disk != nil {
		var unit int
		disk.Device = "disk"
		disk.Target.Bus = diskDevice.Disk.Bus
		disk.Target.Device, unit = makeDeviceName(diskDevice.Name, diskDevice.Disk.Bus, prefixMap)
		if diskDevice.Disk.Bus == "scsi" {
			assignDiskToSCSIController(disk, unit)
		}
		if diskDevice.Disk.PciAddress != "" {
			if diskDevice.Disk.Bus != v1.DiskBusVirtio {
				return fmt.Errorf("setting a pci address is not allowed for non-virtio bus types, for disk %s", diskDevice.Name)
			}
			addr, err := device.NewPciAddressField(diskDevice.Disk.PciAddress)
			if err != nil {
				return fmt.Errorf("failed to configure disk %s: %v", diskDevice.Name, err)
			}
			disk.Address = addr
		}
		if diskDevice.Disk.Bus == v1.DiskBusVirtio {
			disk.Model = virtio.InterpretTransitionalModelType(&d.useVirtioTransitional, d.architecture)
		}
		disk.ReadOnly = toApiReadOnly(diskDevice.Disk.ReadOnly)
		disk.Serial = diskDevice.Serial
		if diskDevice.Shareable != nil {
			if *diskDevice.Shareable {
				if diskDevice.Cache == "" {
					diskDevice.Cache = v1.CacheNone
				}
				if diskDevice.Cache != v1.CacheNone {
					return fmt.Errorf("a sharable disk requires cache = none got: %v", diskDevice.Cache)
				}
				disk.Shareable = &api.Shareable{}
			}
		}
	} else if diskDevice.LUN != nil {
		var unit int
		disk.Device = "lun"
		disk.Target.Bus = diskDevice.LUN.Bus
		disk.Target.Device, unit = makeDeviceName(diskDevice.Name, diskDevice.LUN.Bus, prefixMap)
		if diskDevice.LUN.Bus == "scsi" {
			assignDiskToSCSIController(disk, unit)
		}
		disk.ReadOnly = toApiReadOnly(diskDevice.LUN.ReadOnly)
		if diskDevice.LUN.Reservation {
			setReservation(disk)
		}
	} else if diskDevice.CDRom != nil {
		disk.Device = "cdrom"
		disk.Target.Tray = string(diskDevice.CDRom.Tray)
		disk.Target.Bus = diskDevice.CDRom.Bus
		disk.Target.Device, _ = makeDeviceName(diskDevice.Name, diskDevice.CDRom.Bus, prefixMap)
		if diskDevice.CDRom.ReadOnly != nil {
			disk.ReadOnly = toApiReadOnly(*diskDevice.CDRom.ReadOnly)
		} else {
			disk.ReadOnly = toApiReadOnly(true)
		}
	}
	disk.Driver = &api.DiskDriver{
		Name:  "qemu",
		Cache: string(diskDevice.Cache),
		IO:    diskDevice.IO,
	}
	if diskDevice.Disk != nil || diskDevice.LUN != nil {
		if !slices.Contains(d.volumesDiscardIgnore, diskDevice.Name) {
			disk.Driver.Discard = "unmap"
		}
		volumeStatus, ok := volumeStatusMap[diskDevice.Name]
		if ok && volumeStatus.PersistentVolumeClaimInfo != nil {
			disk.FilesystemOverhead = volumeStatus.PersistentVolumeClaimInfo.FilesystemOverhead
			disk.Capacity = storagetypes.GetDiskCapacity(volumeStatus.PersistentVolumeClaimInfo)
			disk.ExpandDisksEnabled = d.expandDisksEnabled
		}
	}

	queues := d.numBlkQueues()
	if queues != nil && disk.Target.Bus == v1.DiskBusVirtio {
		disk.Driver.Queues = queues
	}
	disk.Alias = api.NewUserDefinedAlias(diskDevice.Name)
	if diskDevice.BootOrder != nil {
		disk.BootOrder = &api.BootOrder{Order: *diskDevice.BootOrder}
	}
	if (d.useLaunchSecuritySEV || d.useLaunchSecurityPV) && disk.Target.Bus == v1.DiskBusVirtio {
		disk.Driver.IOMMU = "on"
	}

	return nil
}

func (d *DiskConfigurator) Convert_v1_Volume_To_api_Disk(vmi *v1.VirtualMachineInstance, source *v1.Volume, disk *api.Disk, diskIndex int) error {

	if source.ContainerDisk != nil {
		return d.Convert_v1_ContainerDiskSource_To_api_Disk(source.Name, source.ContainerDisk, disk, diskIndex)
	}

	if source.CloudInitNoCloud != nil || source.CloudInitConfigDrive != nil {
		return d.Convert_v1_CloudInitSource_To_api_Disk(vmi, source.VolumeSource, disk)
	}

	if source.Sysprep != nil {
		return d.Convert_v1_SysprepSource_To_api_Disk(source.Name, disk)
	}

	if source.HostDisk != nil {
		return d.Convert_v1_HostDisk_To_api_Disk(source.Name, source.HostDisk.Path, disk)
	}

	if source.PersistentVolumeClaim != nil {
		return d.Convert_v1_PersistentVolumeClaim_To_api_Disk(source.Name, disk)
	}

	if source.DataVolume != nil {
		return d.Convert_v1_DataVolume_To_api_Disk(source.Name, disk)
	}

	if source.Ephemeral != nil {
		return d.Convert_v1_EphemeralVolumeSource_To_api_Disk(source.Name, disk)
	}
	if source.EmptyDisk != nil {
		return d.Convert_v1_EmptyDiskSource_To_api_Disk(source.Name, source.EmptyDisk, disk)
	}
	if source.ConfigMap != nil {
		return d.Convert_v1_Config_To_api_Disk(source.Name, disk, config.ConfigMap)
	}
	if source.Secret != nil {
		return d.Convert_v1_Config_To_api_Disk(source.Name, disk, config.Secret)
	}
	if source.DownwardAPI != nil {
		return d.Convert_v1_Config_To_api_Disk(source.Name, disk, config.DownwardAPI)
	}
	if source.ServiceAccount != nil {
		return d.Convert_v1_Config_To_api_Disk(source.Name, disk, config.ServiceAccount)
	}
	if source.DownwardMetrics != nil {
		return d.Convert_v1_DownwardMetricSource_To_api_Disk(disk)
	}

	return fmt.Errorf("disk %s references an unsupported source", disk.Alias.GetName())
}

func (d *DiskConfigurator) Convert_v1_ContainerDiskSource_To_api_Disk(volumeName string, _ *v1.ContainerDiskSource, disk *api.Disk, diskIndex int) error {
	if disk.Type == "lun" {
		return fmt.Errorf(deviceTypeNotCompatibleFmt, disk.Alias.GetName())
	}
	disk.Type = "file"
	setDiskDriver(disk, "qcow2", true)
	disk.Source.File = d.ephemeralDiskCreator.GetFilePath(volumeName)
	disk.BackingStore = &api.BackingStore{
		Format: &api.BackingStoreFormat{},
		Source: &api.DiskSource{},
	}

	source := containerdisk.GetDiskTargetPathFromLauncherView(diskIndex)
	if info := d.disksInfo[volumeName]; info != nil {
		disk.BackingStore.Format.Type = info.Format
	} else {
		return fmt.Errorf("no disk info provided for volume %s", volumeName)
	}
	disk.BackingStore.Source.File = source
	disk.BackingStore.Type = "file"

	return nil
}

func (d *DiskConfigurator) Convert_v1_CloudInitSource_To_api_Disk(vmi *v1.VirtualMachineInstance, source v1.VolumeSource, disk *api.Disk) error {
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

	disk.Source.File = cloudinit.GetIsoFilePath(dataSource, vmi.Name, vmi.Namespace)
	disk.Type = "file"
	setDiskDriver(disk, "raw", false)
	return nil
}

func (d *DiskConfigurator) Convert_v1_SysprepSource_To_api_Disk(volumeName string, disk *api.Disk) error {
	if disk.Type == "lun" {
		return fmt.Errorf(deviceTypeNotCompatibleFmt, disk.Alias.GetName())
	}

	disk.Source.File = config.GetSysprepDiskPath(volumeName)
	disk.Type = "file"
	disk.Driver.Type = "raw"
	return nil
}

func (d *DiskConfigurator) Convert_v1_HostDisk_To_api_Disk(volumeName string, path string, disk *api.Disk) error {
	disk.Type = "file"
	if cbtPath, ok := d.applyCBT[volumeName]; ok {
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

func (d *DiskConfigurator) Convert_v1_PersistentVolumeClaim_To_api_Disk(name string, disk *api.Disk) error {
	return ConvertVolumeSourceToDisk(name, d.applyCBT[name], d.isBlockPVC[name], disk, d.volumesDiscardIgnore)
}

func (d *DiskConfigurator) Convert_v1_DataVolume_To_api_Disk(name string, disk *api.Disk) error {
	return ConvertVolumeSourceToDisk(name, d.applyCBT[name], d.isBlockDV[name], disk, d.volumesDiscardIgnore)
}

func (d *DiskConfigurator) Convert_v1_EphemeralVolumeSource_To_api_Disk(volumeName string, disk *api.Disk) error {
	disk.Type = "file"
	setDiskDriver(disk, "qcow2", true)
	disk.Source.File = d.ephemeralDiskCreator.GetFilePath(volumeName)
	disk.BackingStore = &api.BackingStore{
		Format: &api.BackingStoreFormat{},
		Source: &api.DiskSource{},
	}

	backingDisk := &api.Disk{Driver: &api.DiskDriver{}}
	if d.isBlockPVC[volumeName] {
		if err := Convert_v1_BlockVolumeSource_To_api_Disk(volumeName, backingDisk, d.volumesDiscardIgnore); err != nil {
			return err
		}
	} else {
		if err := Convert_v1_FilesystemVolumeSource_To_api_Disk(volumeName, backingDisk, d.volumesDiscardIgnore); err != nil {
			return err
		}
	}
	disk.BackingStore.Format.Type = backingDisk.Driver.Type
	disk.BackingStore.Source = &backingDisk.Source
	disk.BackingStore.Type = backingDisk.Type

	return nil
}

func (d *DiskConfigurator) Convert_v1_EmptyDiskSource_To_api_Disk(volumeName string, _ *v1.EmptyDiskSource, disk *api.Disk) error {
	if disk.Type == "lun" {
		return fmt.Errorf(deviceTypeNotCompatibleFmt, disk.Alias.GetName())
	}

	disk.Type = "file"
	disk.Source.File = emptydisk.NewEmptyDiskCreator().FilePathForVolumeName(volumeName)
	setDiskDriver(disk, "qcow2", true)

	return nil
}

func (d *DiskConfigurator) Convert_v1_Config_To_api_Disk(volumeName string, disk *api.Disk, configType config.Type) error {
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

func (d *DiskConfigurator) Convert_v1_DownwardMetricSource_To_api_Disk(disk *api.Disk) error {
	disk.Type = "file"
	disk.ReadOnly = toApiReadOnly(true)
	disk.Driver = &api.DiskDriver{
		Type: "raw",
		Name: "qemu",
	}
	// This disk always needs `virtio`. Validation ensures that bus is unset or is already virtio
	disk.Model = virtio.InterpretTransitionalModelType(&d.useVirtioTransitional, d.architecture)
	disk.Source = api.DiskSource{
		File: config.DownwardMetricDisk,
	}
	return nil
}

// Convert_v1_Missing_Volume_To_api_Disk sets defaults when no volume for disk (cdrom, floppy, etc) is provided
func (d *DiskConfigurator) Convert_v1_Missing_Volume_To_api_Disk(disk *api.Disk) error {
	disk.Type = "block"
	disk.Driver.Type = "raw"
	disk.Driver.Discard = "unmap"
	return nil
}

// Convert_v1_Hotplug_Volume_To_api_Disk convers a hotplug volume to an api disk
func (d *DiskConfigurator) Convert_v1_Hotplug_Volume_To_api_Disk(source *v1.Volume, disk *api.Disk) error {
	// This is here because virt-handler before passing the VMI here replaces all PVCs with host disks in
	// hostdisk.ReplacePVCByHostDisk not quite sure why, but it broken hot plugging PVCs
	if source.HostDisk != nil {
		return d.Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk(source.Name, disk)
	}

	if source.PersistentVolumeClaim != nil {
		return d.Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk(source.Name, disk)
	}

	if source.DataVolume != nil {
		return d.Convert_v1_Hotplug_DataVolume_To_api_Disk(source.Name, disk)
	}
	return fmt.Errorf("hotplug disk %s references an unsupported source", disk.Alias.GetName())
}

// Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk converts a Hotplugged PVC to an api disk
func (d *DiskConfigurator) Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk(name string, disk *api.Disk) error {
	if d.isBlockPVC[name] {
		return d.Convert_v1_Hotplug_BlockVolumeSource_To_api_Disk(name, disk)
	}
	return d.Convert_v1_Hotplug_FilesystemVolumeSource_To_api_Disk(name, disk)
}

// Convert_v1_Hotplug_DataVolume_To_api_Disk converts a Hotplugged DataVolume to an api disk
func (d *DiskConfigurator) Convert_v1_Hotplug_DataVolume_To_api_Disk(name string, disk *api.Disk) error {
	if d.isBlockDV[name] {
		return d.Convert_v1_Hotplug_BlockVolumeSource_To_api_Disk(name, disk)
	}
	return d.Convert_v1_Hotplug_FilesystemVolumeSource_To_api_Disk(name, disk)
}

// Convert_v1_Hotplug_FilesystemVolumeSource_To_api_Disk takes a FS source and builds the KVM Disk representation
func (d *DiskConfigurator) Convert_v1_Hotplug_FilesystemVolumeSource_To_api_Disk(volumeName string, disk *api.Disk) error {
	disk.Type = "file"
	setDiskDriver(disk, "raw", !slices.Contains(d.volumesDiscardIgnore, volumeName))
	disk.Source.File = GetHotplugFilesystemVolumePath(volumeName)
	return nil
}

// Convert_v1_Hotplug_BlockVolumeSource_To_api_Disk takes a block device source and builds the domain Disk representation
func (d *DiskConfigurator) Convert_v1_Hotplug_BlockVolumeSource_To_api_Disk(volumeName string, disk *api.Disk) error {
	disk.Type = "block"
	setDiskDriver(disk, "raw", !slices.Contains(d.volumesDiscardIgnore, volumeName))
	disk.Source.Dev = GetHotplugBlockDeviceVolumePath(volumeName)
	return nil
}

func Convert_v1_BlockVolumeSource_To_api_Disk(volumeName string, disk *api.Disk, volumesDiscardIgnore []string) error {
	disk.Type = "block"
	setDiskDriver(disk, "raw", !slices.Contains(volumesDiscardIgnore, volumeName))
	disk.Source.Name = volumeName
	disk.Source.Dev = GetBlockDeviceVolumePath(volumeName)
	return nil
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

func Convert_v1_BlockSize_To_api_BlockIO(source *v1.Disk, disk *api.Disk) error {
	if source.BlockSize == nil {
		return nil
	}

	if blockSize := source.BlockSize.Custom; blockSize != nil {
		disk.BlockIO = &api.BlockIO{
			LogicalBlockSize:  blockSize.Logical,
			PhysicalBlockSize: blockSize.Physical,
		}
		// TODO: as of the time of writing this, KubeVirt uses libvirt < v11.6.0
		// which means that a discard_granularity value of 0 is omitted.
		// remove this comment once upgraded.
		if blockSize.DiscardGranularity != nil {
			disk.BlockIO.DiscardGranularity = pointer.P(*blockSize.DiscardGranularity)
		}
	} else if matchFeature := source.BlockSize.MatchVolume; matchFeature != nil && (matchFeature.Enabled == nil || *matchFeature.Enabled) {
		blockIO, err := getOptimalBlockIO(disk)
		if err != nil {
			return fmt.Errorf("failed to configure disk with block size detection enabled: %v", err)
		}
		disk.BlockIO = blockIO
	}
	return nil
}

func ConvertVolumeSourceToDisk(volumeName, cbtPath string, isBlock bool, disk *api.Disk, volumesDiscardIgnore []string) error {
	if cbtPath != "" {
		return convertVolumeWithCBT(volumeName, cbtPath, isBlock, disk, volumesDiscardIgnore)
	}
	return convertVolumeWithoutCBT(volumeName, isBlock, disk, volumesDiscardIgnore)
}

func (d *DiskConfigurator) numBlkQueues() *uint {
	if !d.useBlkMQ {
		return nil
	}
	return &d.vcpus
}

func WithArchitecture(architecture string) option {
	return func(d *DiskConfigurator) {
		d.architecture = architecture
	}
}

func WithUseLaunchSecuritySEV(useLaunchSecuritySEV bool) option {
	return func(d *DiskConfigurator) {
		d.useLaunchSecuritySEV = useLaunchSecuritySEV
	}
}

func WithUseLaunchSecurityPV(useLaunchSecurityPV bool) option {
	return func(d *DiskConfigurator) {
		d.useLaunchSecurityPV = useLaunchSecurityPV
	}
}

func WithHotplugVolumes(hotplugVolumes map[string]v1.VolumeStatus) option {
	return func(d *DiskConfigurator) {
		d.hotplugVolumes = hotplugVolumes
	}
}

func WithPermanentVolumes(permanentVolumes map[string]v1.VolumeStatus) option {
	return func(d *DiskConfigurator) {
		d.permanentVolumes = permanentVolumes
	}
}

func WithUseVirtioTransitional(useVirtioTranslation bool) option {
	return func(d *DiskConfigurator) {
		d.useVirtioTransitional = useVirtioTranslation
	}
}

func WithExpandDisksEnabled(expandDisksEnabled bool) option {
	return func(d *DiskConfigurator) {
		d.expandDisksEnabled = expandDisksEnabled
	}
}

func WithVolumesDiscardIgnore(volumesDiscardIgnore []string) option {
	return func(d *DiskConfigurator) {
		d.volumesDiscardIgnore = volumesDiscardIgnore
	}
}

func WithEphemeralDiskCreator(ephemeralDiskCreator ephemeraldisk.EphemeralDiskCreatorInterface) option {
	return func(d *DiskConfigurator) {
		d.ephemeralDiskCreator = ephemeralDiskCreator
	}
}

func WithDisksInfo(disksInfo map[string]*disk.DiskInfo) option {
	return func(d *DiskConfigurator) {
		d.disksInfo = disksInfo
	}
}

func WithApplyCBT(applyCBT map[string]string) option {
	return func(d *DiskConfigurator) {
		d.applyCBT = applyCBT
	}
}

func WithIsBlockPVC(isBlockPVC map[string]bool) option {
	return func(d *DiskConfigurator) {
		d.isBlockPVC = isBlockPVC
	}
}

func WithIsBlockDV(isBlockDV map[string]bool) option {
	return func(d *DiskConfigurator) {
		d.isBlockDV = isBlockDV
	}
}

func WithUseBlkMQ(useBlkMQ bool) option {
	return func(d *DiskConfigurator) {
		d.useBlkMQ = useBlkMQ
	}
}

func WithVcpus(count uint) option {
	return func(d *DiskConfigurator) {
		d.vcpus = count
		if d.vcpus == 0 {
			d.vcpus = 1
		}
	}
}

func GetBlockDeviceVolumePath(volumeName string) string {
	return filepath.Join(string(filepath.Separator), "dev", volumeName)
}

func GetFilesystemVolumePath(volumeName string) string {
	return filepath.Join(string(filepath.Separator), "var", "run", "kubevirt-private", "vmi-disks", volumeName, "disk.img")
}

// GetHotplugFilesystemVolumePath returns the path and file name of a hotplug disk image
func GetHotplugFilesystemVolumePath(volumeName string) string {
	return filepath.Join(string(filepath.Separator), "var", "run", "kubevirt", "hotplug-disks", fmt.Sprintf("%s.img", volumeName))
}

// GetHotplugBlockDeviceVolumePath returns the path and name of a hotplugged block device
func GetHotplugBlockDeviceVolumePath(volumeName string) string {
	return filepath.Join(string(filepath.Separator), "var", "run", "kubevirt", "hotplug-disks", volumeName)
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

func setDiskDriver(disk *api.Disk, driverType string, discard bool) {
	disk.Driver.Type = driverType
	disk.Driver.ErrorPolicy = v1.DiskErrorPolicyStop
	if discard {
		disk.Driver.Discard = "unmap"
	}
}

func toApiReadOnly(src bool) *api.ReadOnly {
	if src {
		return &api.ReadOnly{}
	}
	return nil
}

func makeDeviceName(diskName string, bus v1.DiskBus, prefixMap map[string]deviceNamer) (string, int) {
	prefix := getPrefixFromBus(bus)
	if _, ok := prefixMap[prefix]; !ok {
		prefixMap[prefix] = deviceNamer{
			existingNameMap: make(map[string]string),
			usedDeviceMap:   make(map[string]string),
		}
	}
	deviceNamer := prefixMap[prefix]
	if name, ok := deviceNamer.getExistingVolumeValue(diskName); ok {
		for i := 0; i < 26*26*26; i++ {
			calculatedName := formatDeviceName(prefix, i)
			if calculatedName == name {
				return name, i
			}
		}
		log.Log.Error("Unable to determine index of device")
		return name, 0
	}
	for i := 0; i < 26*26*26; i++ {
		name := formatDeviceName(prefix, i)
		if _, ok := deviceNamer.getExistingTargetValue(name); !ok {
			deviceNamer.existingNameMap[diskName] = name
			deviceNamer.usedDeviceMap[name] = diskName
			return name, i
		}
	}
	return "", 0
}

func assignDiskToSCSIController(disk *api.Disk, unit int) {
	// Ensure we assign this disk to the correct scsi controller
	if disk.Address == nil {
		disk.Address = &api.Address{}
	}
	disk.Address.Type = "drive"
	// This should be the index of the virtio-scsi controller, which is hard coded to 0
	disk.Address.Controller = "0"
	disk.Address.Bus = "0"
	disk.Address.Unit = strconv.Itoa(unit)
}

func formatDeviceName(prefix string, index int) string {
	base := int('z' - 'a' + 1)
	name := ""

	for index >= 0 {
		name = string(rune('a'+(index%base))) + name
		index = (index / base) - 1
	}
	return prefix + name
}

func setReservation(disk *api.Disk) {
	disk.Source.Reservations = &api.Reservations{
		Managed: "no",
		SourceReservations: &api.SourceReservations{
			Type: "unix",
			Path: reservation.GetPrHelperSocketPath(),
			Mode: "client",
		},
	}
}

func setErrorPolicy(diskDevice *v1.Disk, apiDisk *api.Disk) error {
	if diskDevice.ErrorPolicy == nil {
		apiDisk.Driver.ErrorPolicy = v1.DiskErrorPolicyStop
		return nil
	}
	switch *diskDevice.ErrorPolicy {
	case v1.DiskErrorPolicyStop, v1.DiskErrorPolicyIgnore, v1.DiskErrorPolicyReport, v1.DiskErrorPolicyEnospace:
		apiDisk.Driver.ErrorPolicy = *diskDevice.ErrorPolicy
	default:
		return fmt.Errorf("error policy %s not recognized", *diskDevice.ErrorPolicy)
	}
	return nil
}

func getOptimalBlockIO(disk *api.Disk) (*api.BlockIO, error) {
	if disk.Source.Dev != "" {
		return getOptimalBlockIOForDevice(disk.Source.Dev)
	} else if disk.Source.File != "" {
		return getOptimalBlockIOForFile(disk.Source.File)
	}
	return nil, fmt.Errorf("disk is neither a block device nor a file")
}

func getOptimalBlockIOForDevice(path string) (*api.BlockIO, error) {
	safePath, err := safepath.JoinAndResolveWithRelativeRoot("/", path)
	if err != nil {
		return nil, err
	}
	fd, err := safepath.OpenAtNoFollow(safePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s. Reason: %w", safePath, err)
	}
	defer util.CloseIOAndCheckErr(fd, nil)

	f, err := os.OpenFile(fd.SafePath(), os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer util.CloseIOAndCheckErr(f, &err)

	logicalSize, err := unix.IoctlGetUint32(int(f.Fd()), unix.BLKSSZGET)
	if err != nil {
		return nil, fmt.Errorf("unable to get logical block size from device %s: %w", path, err)
	}
	physicalSize, err := unix.IoctlGetUint32(int(f.Fd()), unix.BLKPBSZGET)
	if err != nil {
		return nil, fmt.Errorf("unable to get physical block size from device %s: %w", path, err)
	}

	log.Log.Infof("Detected logical size of %d and physical size of %d for device %s", logicalSize, physicalSize, path)

	if logicalSize == 0 && physicalSize == 0 {
		return nil, fmt.Errorf("block sizes returned by device %v are 0", path)
	}

	discardGranularity, err := getDiscardGranularity(safePath)
	if err != nil {
		return nil, err
	}

	log.Log.Infof("Detected discard granularity of %d for device %v", discardGranularity, path)

	blockIO := &api.BlockIO{
		LogicalBlockSize:   uint(logicalSize),
		PhysicalBlockSize:  uint(physicalSize),
		DiscardGranularity: pointer.P(uint(discardGranularity)),
	}
	if logicalSize == 0 || physicalSize == 0 {
		if logicalSize > physicalSize {
			log.Log.Infof("Invalid physical size %d. Matching it to the logical size %d", physicalSize, logicalSize)
			blockIO.PhysicalBlockSize = uint(logicalSize)
		} else {
			log.Log.Infof("Invalid logical size %d. Matching it to the physical size %d", logicalSize, physicalSize)
			blockIO.LogicalBlockSize = uint(physicalSize)
		}
	}
	if *blockIO.DiscardGranularity%blockIO.LogicalBlockSize != 0 {
		log.Log.Infof("Invalid discard granularity %d. Matching it to physical size %d", *blockIO.DiscardGranularity, blockIO.PhysicalBlockSize)
		blockIO.DiscardGranularity = pointer.P(uint(physicalSize))
	}
	return blockIO, nil
}

func getDiscardGranularity(safePath *safepath.Path) (uint64, error) {
	fileInfo, err := safepath.StatAtNoFollow(safePath)
	if err != nil {
		return 0, fmt.Errorf("could not stat file %s. Reason: %w", safePath.String(), err)
	}
	stat := fileInfo.Sys().(*syscall.Stat_t)
	rdev := uint64(stat.Rdev) //nolint:unconvert // Rdev is uint32 on e.g. MIPS.
	major := unix.Major(rdev)
	minor := unix.Minor(rdev)

	raw, err := os.ReadFile(fmt.Sprintf("/sys/dev/block/%d:%d/queue/discard_granularity", major, minor))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// On the off chance that we can't stat the discard granularity, set it to disabled.
			return 0, nil
		}
		return 0, fmt.Errorf("cannot read discard granularity for device %s: %w", safePath.String(), err)
	}
	discardGranularity, err := strconv.ParseUint(strings.TrimSpace(string(raw)), 10, 0)
	if err != nil {
		return 0, err
	}

	return discardGranularity, err
}

// getOptimalBlockIOForFile determines the optimal sizes based on the filesystem settings
// the VM's disk image is residing on. A filesystem does not differentiate between sizes.
// The physical size will always match the logical size. The rest is up to the filesystem.
func getOptimalBlockIOForFile(path string) (*api.BlockIO, error) {
	var statfs unix.Statfs_t
	if err := unix.Statfs(path, &statfs); err != nil {
		return nil, fmt.Errorf("failed to stat file %v: %v", path, err)
	}
	blockSize := uint(statfs.Bsize)
	return &api.BlockIO{
		LogicalBlockSize:   blockSize,
		PhysicalBlockSize:  blockSize,
		DiscardGranularity: &blockSize,
	}, nil
}
