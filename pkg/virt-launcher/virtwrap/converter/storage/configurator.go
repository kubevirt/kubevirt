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

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/iothreads"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
)

type OptimalBlockIODetectFunc func(disk *api.Disk) (*api.BlockIO, error)

type diskOption func(*DiskConfigurator)

type DiskConfigurator struct {
	c                    *convertertypes.ConverterContext
	detectOptimalBlockIO OptimalBlockIODetectFunc
}

func NewDiskConfigurator(c *convertertypes.ConverterContext, options ...diskOption) DiskConfigurator {
	d := DiskConfigurator{c: c, detectOptimalBlockIO: getOptimalBlockIO}
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

	var ioThreadPool *api.DiskIOThreads
	useMultiIOAuto := d.c.SCSIMultiIOThreadEnabled && d.c.MultiIOThreadAutoPolicyEnabled
	if vmi.Spec.Domain.IOThreadsPolicy != nil {
		if *vmi.Spec.Domain.IOThreadsPolicy == v1.IOThreadsPolicySupplementalPool {
			// vmi admitter requires SupplementalPoolThreadCount to be set when using this thread policy
			ioThreadPool = iothreads.BuildIOThreadPool(int(*vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount))
		} else if *vmi.Spec.Domain.IOThreadsPolicy == v1.IOThreadsPolicyAuto && useMultiIOAuto {
			// when feature gate and config set,
			// auto policy matches virtio-scsi behavior in assigning each disk the entire auto thread pool
			autoPoolSize := min(autoThreads, int(vcpus))
			ioThreadPool = iothreads.BuildIOThreadPool(autoPoolSize)
		}
	}
	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		newDisk := api.Disk{}
		emptyCDRom := false

		err := convert_v1_Disk_To_api_Disk(d.c, &disk, &newDisk, prefixMap, numBlkQueues, volumeStatusMap)
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

		hpStatus, hpOk := d.c.HotplugVolumes[disk.Name]
		switch {
		case emptyCDRom:
			err = convert_v1_Missing_Volume_To_api_Disk(&newDisk)
		case hpOk:
			err = convert_v1_Hotplug_Volume_To_api_Disk(volume, &newDisk, d.c)
		default:
			err = convert_v1_Volume_To_api_Disk(volume, &newDisk, d.c, volumeIndices[disk.Name])
		}

		if err != nil {
			return err
		}

		if err := convert_v1_BlockSize_To_api_BlockIO(&disk, &newDisk, d.c.Architecture.GetArchitecture(), d.detectOptimalBlockIO); err != nil {
			return err
		}

		_, isPermVolume := d.c.PermanentVolumes[disk.Name]
		// if len(d.c.PermanentVolumes) == 0, it means the vmi is not ready yet, add all disks
		permReady := isPermVolume || len(d.c.PermanentVolumes) == 0
		hotplugReady := hpOk && (hpStatus.Phase == v1.HotplugVolumeMounted || hpStatus.Phase == v1.VolumeReady)

		if permReady || hotplugReady || emptyCDRom {
			domain.Spec.Devices.Disks = append(domain.Spec.Devices.Disks, newDisk)
		}
		if err := setErrorPolicy(&disk, &newDisk); err != nil {
			return err
		}
		if hasIOThreads {
			currentDedicatedThread, currentAutoThread = assignDiskIOThread(&disk, &newDisk, ioThreadPool, autoThreads, currentDedicatedThread, currentAutoThread)
		}
	}

	return nil
}

func DiskWithOptimalBlockIODetector(f OptimalBlockIODetectFunc) diskOption {
	return func(d *DiskConfigurator) {
		d.detectOptimalBlockIO = f
	}
}
