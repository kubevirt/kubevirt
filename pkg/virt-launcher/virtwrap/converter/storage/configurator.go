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

	"kubevirt.io/kubevirt/pkg/config"
	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"
	"kubevirt.io/kubevirt/pkg/os/disk"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/iothreads"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/virtio"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

type OptimalBlockIODetectFunc func(disk *api.Disk) (*api.BlockIO, error)

type diskOption func(*DiskConfigurator)

type DiskConfigurator struct {
	architecture         string
	virtioModel          string
	useLaunchSecuritySEV bool
	useLaunchSecurityPV  bool
	volumesDiscardIgnore []string
	hotplugVolumes       map[string]v1.VolumeStatus
	permanentVolumes     map[string]v1.VolumeStatus
	isBlockPVC           map[string]bool
	isBlockDV            map[string]bool
	applyCBT             map[string]string
	disksInfo            map[string]*disk.DiskInfo
	ephemeralDiskCreator ephemeraldisk.EphemeralDiskCreatorInterface
	detectOptimalBlockIO OptimalBlockIODetectFunc
}

func NewDiskConfigurator(c *convertertypes.ConverterContext, options ...diskOption) DiskConfigurator {
	d := DiskConfigurator{
		architecture:         c.Architecture.GetArchitecture(),
		virtioModel:          virtio.InterpretTransitionalModelType(&c.UseVirtioTransitional, c.Architecture.GetArchitecture()),
		useLaunchSecuritySEV: c.UseLaunchSecuritySEV,
		useLaunchSecurityPV:  c.UseLaunchSecurityPV,
		volumesDiscardIgnore: c.VolumesDiscardIgnore,
		hotplugVolumes:       c.HotplugVolumes,
		permanentVolumes:     c.PermanentVolumes,
		isBlockPVC:           c.IsBlockPVC,
		isBlockDV:            c.IsBlockDV,
		applyCBT:             c.ApplyCBT,
		disksInfo:            c.DisksInfo,
		ephemeralDiskCreator: c.EphemeraldiskCreator,
		detectOptimalBlockIO: getOptimalBlockIO,
	}
	for _, f := range options {
		f(&d)
	}
	return d
}

func (d DiskConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	hasIOThreads := iothreads.HasIOThreads(vmi)
	var autoThreads int
	if hasIOThreads {
		_, autoThreads = iothreads.GetIOThreadsCountType(vmi)
	}

	volumeIndices := map[string]int{}
	volumes := map[string]*v1.Volume{}
	for i, volume := range vmi.Spec.Volumes {
		volumes[volume.Name] = volume.DeepCopy()
		volumeIndices[volume.Name] = i
	}

	var numBlkQueues *uint
	virtioBlkMQRequested := (vmi.Spec.Domain.Devices.BlockMultiQueue != nil) && (*vmi.Spec.Domain.Devices.BlockMultiQueue)
	cpuTopology := vcpu.GetCPUTopology(vmi)
	cpuCount := vcpu.CalculateRequestedVCPUs(cpuTopology)
	vcpus := uint(cpuCount)
	if vcpus == 0 {
		vcpus = uint(1)
	}

	if virtioBlkMQRequested {
		numBlkQueues = &vcpus
	}

	volumeStatusMap := make(map[string]v1.VolumeStatus)
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		volumeStatusMap[volumeStatus.Name] = volumeStatus
	}

	prefixMap := newDeviceNamer(vmi.Status.VolumeStatus, vmi.Spec.Domain.Devices.Disks)
	currentAutoThread := uint(1)
	currentDedicatedThread := uint(autoThreads + 1)
	supplementalIOThreads := iothreads.BuildSupplementalPoolIOThreads(vmi)
	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		newDisk := api.Disk{}
		emptyCDRom := false

		err := d.convert_v1_Disk_To_api_Disk(&disk, &newDisk, prefixMap, numBlkQueues, volumeStatusMap)
		if err != nil {
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
		switch {
		case emptyCDRom:
			err = convert_v1_Missing_Volume_To_api_Disk(&newDisk)
		case hpOk:
			err = d.convert_v1_Hotplug_Volume_To_api_Disk(volume, &newDisk)
		default:
			err = d.convert_v1_Volume_To_api_Disk(volume, &newDisk, volumeIndices[disk.Name], vmi.Namespace, vmi.Name)
		}

		if err != nil {
			return err
		}

		if err := convert_v1_BlockSize_To_api_BlockIO(&disk, &newDisk, d.architecture, d.detectOptimalBlockIO); err != nil {
			return err
		}

		_, isPermVolume := d.permanentVolumes[disk.Name]
		// if len(d.permanentVolumes) == 0, it means the vmi is not ready yet, add all disks
		permReady := isPermVolume || len(d.permanentVolumes) == 0
		hotplugReady := hpOk && (hpStatus.Phase == v1.HotplugVolumeMounted || hpStatus.Phase == v1.VolumeReady)

		if permReady || hotplugReady || emptyCDRom {
			domain.Spec.Devices.Disks = append(domain.Spec.Devices.Disks, newDisk)
		}
		if err := setErrorPolicy(&disk, &newDisk); err != nil {
			return err
		}
		if hasIOThreads {
			currentDedicatedThread, currentAutoThread = assignDiskIOThread(&disk, &newDisk, supplementalIOThreads, autoThreads, currentDedicatedThread, currentAutoThread)
		}
	}

	return nil
}

func (d DiskConfigurator) convert_v1_Disk_To_api_Disk(diskDevice *v1.Disk, disk *api.Disk, prefixMap map[string]deviceNamer, numQueues *uint, volumeStatusMap map[string]v1.VolumeStatus) error {
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
			disk.Model = d.virtioModel
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
		}
	}
	if numQueues != nil && disk.Target.Bus == v1.DiskBusVirtio {
		disk.Driver.Queues = numQueues
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

func (d DiskConfigurator) convert_v1_Volume_To_api_Disk(source *v1.Volume, disk *api.Disk, diskIndex int, vmiNamespace, vmiName string) error {
	if source.ContainerDisk != nil {
		info := d.disksInfo[source.Name]
		if info == nil {
			return fmt.Errorf("no disk info provided for volume %s", source.Name)
		}
		return convert_v1_ContainerDiskSource_To_api_Disk(source.Name, source.ContainerDisk, disk, d.ephemeralDiskCreator.GetFilePath(source.Name), diskIndex, info.Format)
	}

	if source.CloudInitNoCloud != nil || source.CloudInitConfigDrive != nil {
		return convert_v1_CloudInitSource_To_api_Disk(source.VolumeSource, disk, vmiNamespace, vmiName)
	}

	if source.Sysprep != nil {
		return convert_v1_SysprepSource_To_api_Disk(source.Name, disk)
	}

	if source.HostDisk != nil {
		return convert_v1_HostDisk_To_api_Disk(source.Name, source.HostDisk.Path, d.applyCBT[source.Name], disk)
	}

	if source.PersistentVolumeClaim != nil {
		return convertVolumeSourceToDisk(source.Name, d.applyCBT[source.Name], d.isBlockPVC[source.Name], disk, d.volumesDiscardIgnore)
	}

	if source.DataVolume != nil {
		return convertVolumeSourceToDisk(source.Name, d.applyCBT[source.Name], d.isBlockDV[source.Name], disk, d.volumesDiscardIgnore)
	}

	if source.Ephemeral != nil {
		return convert_v1_EphemeralVolumeSource_To_api_Disk(source.Name, d.ephemeralDiskCreator.GetFilePath(source.Name), d.isBlockPVC[source.Name], disk, d.volumesDiscardIgnore)
	}
	if source.EmptyDisk != nil {
		return convert_v1_EmptyDiskSource_To_api_Disk(source.Name, source.EmptyDisk, disk)
	}
	if source.ConfigMap != nil {
		return convert_v1_Config_To_api_Disk(source.Name, disk, config.ConfigMap)
	}
	if source.Secret != nil {
		return convert_v1_Config_To_api_Disk(source.Name, disk, config.Secret)
	}
	if source.DownwardAPI != nil {
		return convert_v1_Config_To_api_Disk(source.Name, disk, config.DownwardAPI)
	}
	if source.ServiceAccount != nil {
		return convert_v1_Config_To_api_Disk(source.Name, disk, config.ServiceAccount)
	}
	if source.DownwardMetrics != nil {
		return convert_v1_DownwardMetricSource_To_api_Disk(disk, d.virtioModel)
	}

	return fmt.Errorf("disk %s references an unsupported source", disk.Alias.GetName())
}

func (d DiskConfigurator) convert_v1_Hotplug_Volume_To_api_Disk(source *v1.Volume, disk *api.Disk) error {
	// This is here because virt-handler before passing the VMI here replaces all PVCs with host disks in
	// hostdisk.ReplacePVCByHostDisk not quite sure why, but it broken hot plugging PVCs
	if source.HostDisk != nil {
		return convertHotplugVolumeSourceToDisk(source.Name, d.applyCBT[source.Name], d.isBlockPVC[source.Name], disk, d.volumesDiscardIgnore)
	}

	if source.PersistentVolumeClaim != nil {
		return convertHotplugVolumeSourceToDisk(source.Name, d.applyCBT[source.Name], d.isBlockPVC[source.Name], disk, d.volumesDiscardIgnore)
	}

	if source.DataVolume != nil {
		return convertHotplugVolumeSourceToDisk(source.Name, d.applyCBT[source.Name], d.isBlockDV[source.Name], disk, d.volumesDiscardIgnore)
	}
	return fmt.Errorf("hotplug disk %s references an unsupported source", disk.Alias.GetName())
}

func DiskWithOptimalBlockIODetector(f OptimalBlockIODetectFunc) diskOption {
	return func(d *DiskConfigurator) {
		d.detectOptimalBlockIO = f
	}
}
