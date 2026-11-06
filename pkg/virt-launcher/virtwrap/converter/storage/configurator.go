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
	osdisk "kubevirt.io/kubevirt/pkg/os/disk"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/iothreads"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

type OptimalBlockIODetectFunc func(disk *api.Disk) (*api.BlockIO, error)

type diskOption func(*DiskConfigurator)

type DiskConfigurator struct {
	architecture                   string
	virtioModel                    string
	useLaunchSecuritySEV           bool
	useLaunchSecurityPV            bool
	volumesDiscardIgnore           []string
	hotplugVolumes                 map[string]v1.VolumeStatus
	permanentVolumes               map[string]v1.VolumeStatus
	isBlockPVC                     map[string]bool
	isBlockDV                      map[string]bool
	applyCBT                       map[string]string
	disksInfo                      map[string]*osdisk.DiskInfo
	ephemeralDiskCreator           ephemeraldisk.EphemeralDiskCreatorInterface
	detectOptimalBlockIO           OptimalBlockIODetectFunc
	scsiMultiIOThreadEnabled       bool
	multiIOThreadAutoPolicyEnabled bool
}

func NewDiskConfigurator(options ...diskOption) DiskConfigurator {
	d := DiskConfigurator{
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
	ioThreadPool, currentAutoThread, currentDedicatedThread, useSupplementalPool := d.buildIOThreadPool(vmi, autoThreads, vcpus)

	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		newDisk, shouldAppend, err := d.buildDisk(&disk, volumes, volumeIndices, prefixMap, numBlkQueues, volumeStatusMap, vmi)
		if err != nil {
			return err
		}

		if shouldAppend {
			domain.Spec.Devices.Disks = append(domain.Spec.Devices.Disks, newDisk)
		}

		if err := setErrorPolicy(&disk, &newDisk); err != nil {
			return err
		}

		if hasIOThreads {
			currentDedicatedThread, currentAutoThread = assignDiskIOThread(
				&disk, &newDisk, ioThreadPool, autoThreads,
				currentDedicatedThread, currentAutoThread, useSupplementalPool,
			)
		}
	}

	return nil
}

func (d DiskConfigurator) buildIOThreadPool(
	vmi *v1.VirtualMachineInstance, autoThreads int, vcpus uint,
) (pool *api.DiskIOThreads, currentAutoThread, currentDedicatedThread uint, useSupplementalPool bool) {
	currentAutoThread = uint(1)
	currentDedicatedThread = uint(autoThreads + 1)
	// SCSIMultiIOThread feature gate and MultiIOThreadAutoPolicy kubevirt config
	// must both be enabled to allocate the entire auto thread pool to each disk
	useMultiIOAuto := d.scsiMultiIOThreadEnabled && d.multiIOThreadAutoPolicyEnabled
	if vmi.Spec.Domain.IOThreadsPolicy == nil {
		return pool, currentAutoThread, currentDedicatedThread, false
	}
	switch {
	case *vmi.Spec.Domain.IOThreadsPolicy == v1.IOThreadsPolicySupplementalPool:
		pool = iothreads.BuildIOThreadPool(int(*vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount))
		useSupplementalPool = true
	case *vmi.Spec.Domain.IOThreadsPolicy == v1.IOThreadsPolicyAuto && useMultiIOAuto:
		autoPoolSize := iothreads.AutoPoolSize(autoThreads, vcpus)
		pool = iothreads.BuildIOThreadPool(autoPoolSize)
		currentDedicatedThread = uint(autoPoolSize + 1)
	}
	return pool, currentAutoThread, currentDedicatedThread, useSupplementalPool
}

func (d DiskConfigurator) resolveVolume(
	diskDevice *v1.Disk, volumes map[string]*v1.Volume,
) (volume *v1.Volume, emptyCDRom bool, err error) {
	volume = volumes[diskDevice.Name]
	if volume == nil {
		if diskDevice.CDRom == nil {
			return nil, false, fmt.Errorf("no matching volume with name %s found", diskDevice.Name)
		}
		return nil, true, nil
	}
	return volume, false, nil
}

func (d DiskConfigurator) convertVolumeToDisk(
	volume *v1.Volume, newDisk *api.Disk, emptyCDRom, hpOk bool, diskIndex int, vmi *v1.VirtualMachineInstance,
) error {
	switch {
	case emptyCDRom:
		return convertV1MissingVolumeToAPIDisk(newDisk)
	case hpOk:
		return d.convertV1HotplugVolumeToAPIDisk(volume, newDisk)
	default:
		return d.convertV1VolumeToAPIDisk(volume, newDisk, diskIndex, vmi.Namespace, vmi.Name)
	}
}

func (d DiskConfigurator) shouldAppendDisk(diskName string, hpOk, emptyCDRom bool, hpStatus v1.VolumeStatus) bool {
	_, isPermVolume := d.permanentVolumes[diskName]
	permReady := isPermVolume || len(d.permanentVolumes) == 0
	hotplugReady := hpOk && (hpStatus.Phase == v1.HotplugVolumeMounted || hpStatus.Phase == v1.VolumeReady)
	return permReady || hotplugReady || emptyCDRom
}

func (d DiskConfigurator) buildDisk(
	diskDevice *v1.Disk,
	volumes map[string]*v1.Volume,
	volumeIndices map[string]int,
	prefixMap map[string]deviceNamer,
	numBlkQueues *uint,
	volumeStatusMap map[string]v1.VolumeStatus,
	vmi *v1.VirtualMachineInstance,
) (api.Disk, bool, error) {
	var newDisk api.Disk
	if err := d.convertV1DiskToAPIDisk(diskDevice, &newDisk, prefixMap, numBlkQueues, volumeStatusMap); err != nil {
		return newDisk, false, err
	}

	volume, emptyCDRom, resolveErr := d.resolveVolume(diskDevice, volumes)
	if resolveErr != nil {
		return newDisk, false, resolveErr
	}

	hpStatus, hpOk := d.hotplugVolumes[diskDevice.Name]
	if err := d.convertVolumeToDisk(volume, &newDisk, emptyCDRom, hpOk, volumeIndices[diskDevice.Name], vmi); err != nil {
		return newDisk, false, err
	}

	if err := convertV1BlockSizeToAPIBlockIO(diskDevice, &newDisk, d.architecture, d.detectOptimalBlockIO); err != nil {
		return newDisk, false, err
	}

	return newDisk, d.shouldAppendDisk(diskDevice.Name, hpOk, emptyCDRom, hpStatus), nil
}

func (d DiskConfigurator) convertV1DiskToAPIDisk(
	diskDevice *v1.Disk,
	disk *api.Disk,
	prefixMap map[string]deviceNamer,
	numQueues *uint,
	volumeStatusMap map[string]v1.VolumeStatus,
) error {
	switch {
	case diskDevice.Disk != nil:
		if err := d.configureDiskDevice(diskDevice, disk, prefixMap); err != nil {
			return err
		}
	case diskDevice.LUN != nil:
		configureLUNDevice(diskDevice, disk, prefixMap)
	case diskDevice.CDRom != nil:
		configureCDRomDevice(diskDevice, disk, prefixMap)
	}
	disk.Driver = &api.DiskDriver{
		Name:  "qemu",
		Cache: string(diskDevice.Cache),
		IO:    diskDevice.IO,
	}
	d.applyPVCDiskInfo(diskDevice, disk, volumeStatusMap)
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

func (d DiskConfigurator) applyPVCDiskInfo(diskDevice *v1.Disk, disk *api.Disk, volumeStatusMap map[string]v1.VolumeStatus) {
	if diskDevice.Disk == nil && diskDevice.LUN == nil {
		return
	}
	if !slices.Contains(d.volumesDiscardIgnore, diskDevice.Name) {
		disk.Driver.Discard = "unmap"
	}
	volumeStatus, ok := volumeStatusMap[diskDevice.Name]
	if ok && volumeStatus.PersistentVolumeClaimInfo != nil {
		disk.FilesystemOverhead = volumeStatus.PersistentVolumeClaimInfo.FilesystemOverhead
		disk.Capacity = storagetypes.GetDiskCapacity(volumeStatus.PersistentVolumeClaimInfo)
	}
}

func (d DiskConfigurator) configureDiskDevice(diskDevice *v1.Disk, disk *api.Disk, prefixMap map[string]deviceNamer) error {
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
	disk.ReadOnly = toAPIReadOnly(diskDevice.Disk.ReadOnly)
	disk.Serial = diskDevice.Serial
	if diskDevice.Shareable != nil && *diskDevice.Shareable {
		if diskDevice.Cache == "" {
			diskDevice.Cache = v1.CacheNone
		}
		if diskDevice.Cache != v1.CacheNone && diskDevice.Cache != v1.CacheDirectSync {
			return fmt.Errorf("a sharable disk requires cache mode none or directsync, got: %v", diskDevice.Cache)
		}
		disk.Shareable = &api.Shareable{}
	}
	return nil
}

func configureLUNDevice(diskDevice *v1.Disk, disk *api.Disk, prefixMap map[string]deviceNamer) {
	var unit int
	disk.Device = "lun"
	disk.Target.Bus = diskDevice.LUN.Bus
	disk.Target.Device, unit = makeDeviceName(diskDevice.Name, diskDevice.LUN.Bus, prefixMap)
	if diskDevice.LUN.Bus == "scsi" {
		assignDiskToSCSIController(disk, unit)
	}
	disk.ReadOnly = toAPIReadOnly(diskDevice.LUN.ReadOnly)
	if diskDevice.LUN.Reservation {
		setReservation(disk)
	}
}

func configureCDRomDevice(diskDevice *v1.Disk, disk *api.Disk, prefixMap map[string]deviceNamer) {
	disk.Device = "cdrom"
	disk.Target.Tray = string(diskDevice.CDRom.Tray)
	disk.Target.Bus = diskDevice.CDRom.Bus
	disk.Target.Device, _ = makeDeviceName(diskDevice.Name, diskDevice.CDRom.Bus, prefixMap)
	if diskDevice.CDRom.ReadOnly != nil {
		disk.ReadOnly = toAPIReadOnly(*diskDevice.CDRom.ReadOnly)
	} else {
		disk.ReadOnly = toAPIReadOnly(true)
	}
}

func (d DiskConfigurator) convertV1VolumeToAPIDisk(source *v1.Volume, disk *api.Disk, diskIndex int, vmiNamespace, vmiName string) error {
	if source.ContainerDisk != nil {
		info := d.disksInfo[source.Name]
		if info == nil {
			return fmt.Errorf("no disk info provided for volume %s", source.Name)
		}
		return convertV1ContainerDiskSourceToAPIDisk(disk, d.ephemeralDiskCreator.GetFilePath(source.Name), diskIndex, info.Format)
	}

	if source.CloudInitNoCloud != nil || source.CloudInitConfigDrive != nil {
		return convertV1CloudInitSourceToAPIDisk(source.VolumeSource, disk, vmiNamespace, vmiName)
	}

	if source.Sysprep != nil {
		return convertV1SysprepSourceToAPIDisk(source.Name, disk)
	}

	if source.HostDisk != nil {
		return convertV1HostDiskToAPIDisk(source.Name, source.HostDisk.Path, disk)
	}

	if source.PersistentVolumeClaim != nil {
		return convertVolumeSourceToDisk(source.Name, d.applyCBT[source.Name], d.isBlockPVC[source.Name], disk, d.volumesDiscardIgnore)
	}

	if source.DataVolume != nil {
		return convertVolumeSourceToDisk(source.Name, d.applyCBT[source.Name], d.isBlockDV[source.Name], disk, d.volumesDiscardIgnore)
	}

	if source.Ephemeral != nil {
		return convertV1EphemeralVolumeSourceToAPIDisk(
			source.Name,
			d.ephemeralDiskCreator.GetFilePath(source.Name),
			d.isBlockPVC[source.Name],
			disk,
			d.volumesDiscardIgnore,
		)
	}
	if source.EmptyDisk != nil {
		return convertV1EmptyDiskSourceToAPIDisk(source.Name, source.EmptyDisk, disk)
	}
	if configType, ok := configVolumeType(source); ok {
		return convertV1ConfigToAPIDisk(source.Name, disk, configType)
	}
	if source.DownwardMetrics != nil {
		return convertV1DownwardMetricSourceToAPIDisk(disk, d.virtioModel)
	}

	return fmt.Errorf("disk %s references an unsupported source", disk.Alias.GetName())
}

func (d DiskConfigurator) convertV1HotplugVolumeToAPIDisk(source *v1.Volume, disk *api.Disk) error {
	if source.PersistentVolumeClaim != nil {
		return convertHotplugVolumeSourceToDisk(source.Name, d.applyCBT[source.Name], d.isBlockPVC[source.Name], disk, d.volumesDiscardIgnore)
	}

	if source.DataVolume != nil {
		return convertHotplugVolumeSourceToDisk(source.Name, d.applyCBT[source.Name], d.isBlockDV[source.Name], disk, d.volumesDiscardIgnore)
	}
	return fmt.Errorf("hotplug disk %s references an unsupported source", disk.Alias.GetName())
}

func DiskWithArchitecture(architecture string) diskOption {
	return func(d *DiskConfigurator) {
		d.architecture = architecture
	}
}

func DiskWithVirtioModel(virtioModel string) diskOption {
	return func(d *DiskConfigurator) {
		d.virtioModel = virtioModel
	}
}

func DiskWithUseLaunchSecuritySEV(useLaunchSecuritySEV bool) diskOption {
	return func(d *DiskConfigurator) {
		d.useLaunchSecuritySEV = useLaunchSecuritySEV
	}
}

func DiskWithUseLaunchSecurityPV(useLaunchSecurityPV bool) diskOption {
	return func(d *DiskConfigurator) {
		d.useLaunchSecurityPV = useLaunchSecurityPV
	}
}

func DiskWithVolumesDiscardIgnore(volumesDiscardIgnore []string) diskOption {
	return func(d *DiskConfigurator) {
		d.volumesDiscardIgnore = volumesDiscardIgnore
	}
}

func DiskWithHotplugVolumes(hotplugVolumes map[string]v1.VolumeStatus) diskOption {
	return func(d *DiskConfigurator) {
		d.hotplugVolumes = hotplugVolumes
	}
}

func DiskWithPermanentVolumes(permanentVolumes map[string]v1.VolumeStatus) diskOption {
	return func(d *DiskConfigurator) {
		d.permanentVolumes = permanentVolumes
	}
}

func DiskWithIsBlockPVC(isBlockPVC map[string]bool) diskOption {
	return func(d *DiskConfigurator) {
		d.isBlockPVC = isBlockPVC
	}
}

func DiskWithIsBlockDV(isBlockDV map[string]bool) diskOption {
	return func(d *DiskConfigurator) {
		d.isBlockDV = isBlockDV
	}
}

func DiskWithApplyCBT(applyCBT map[string]string) diskOption {
	return func(d *DiskConfigurator) {
		d.applyCBT = applyCBT
	}
}

func DiskWithDisksInfo(disksInfo map[string]*osdisk.DiskInfo) diskOption {
	return func(d *DiskConfigurator) {
		d.disksInfo = disksInfo
	}
}

func DiskWithEphemeralDiskCreator(ephemeralDiskCreator ephemeraldisk.EphemeralDiskCreatorInterface) diskOption {
	return func(d *DiskConfigurator) {
		d.ephemeralDiskCreator = ephemeralDiskCreator
	}
}

func DiskWithOptimalBlockIODetector(f OptimalBlockIODetectFunc) diskOption {
	return func(d *DiskConfigurator) {
		d.detectOptimalBlockIO = f
	}
}

func DiskWithScsiMultiIOThreadEnabled(enabled bool) diskOption {
	return func(d *DiskConfigurator) {
		d.scsiMultiIOThreadEnabled = enabled
	}
}

func DiskWithMultiIOThreadAutoPolicyEnabled(enabled bool) diskOption {
	return func(d *DiskConfigurator) {
		d.multiIOThreadAutoPolicyEnabled = enabled
	}
}

func configVolumeType(source *v1.Volume) (config.Type, bool) {
	switch {
	case source.ConfigMap != nil:
		return config.ConfigMap, true
	case source.Secret != nil:
		return config.Secret, true
	case source.DownwardAPI != nil:
		return config.DownwardAPI, true
	case source.ServiceAccount != nil:
		return config.ServiceAccount, true
	}
	return "", false
}
