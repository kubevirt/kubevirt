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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package converter

import (
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
)

type IOThreadsPlacer struct {
	dedicatedThreads       int
	autoThreads            int
	threadPoolLimit        int
	currentDedicatedThread uint
	currentAutoThread      uint
}

func NewIOThreadsPlacer(vmi *v1.VirtualMachineInstance) *IOThreadsPlacer {
	var iotp IOThreadsPlacer

	if vmi.Spec.Domain.IOThreadsPolicy != nil {
		if (*vmi.Spec.Domain.IOThreadsPolicy) == v1.IOThreadsPolicyAuto {
			// When IOThreads policy is set to auto and we've allocated a dedicated
			// pCPU for the emulator thread, we can place IOThread and Emulator thread in the same pCPU
			if vmi.IsCPUDedicated() && vmi.Spec.Domain.CPU.IsolateEmulatorThread {
				iotp.threadPoolLimit = 1
			} else {
				numCPUs := 1
				// Requested CPU's is guaranteed to be no greater than the limit
				if cpuRequests, ok := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU]; ok {
					numCPUs = int(cpuRequests.Value())
				} else if cpuLimit, ok := vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU]; ok {
					numCPUs = int(cpuLimit.Value())
				}

				iotp.threadPoolLimit = numCPUs * 2
			}
		}
	}

	for _, diskDevice := range vmi.Spec.Domain.Devices.Disks {
		if diskDevice.DedicatedIOThread != nil &&
			*diskDevice.DedicatedIOThread {
			iotp.dedicatedThreads += 1
		} else {
			iotp.autoThreads += 1
		}
	}

	if (iotp.autoThreads + iotp.dedicatedThreads) > iotp.threadPoolLimit {
		iotp.autoThreads = iotp.threadPoolLimit - iotp.dedicatedThreads
		// We need at least one shared thread
		if iotp.autoThreads < 1 {
			iotp.autoThreads = 1
		}
	}

	iotp.currentDedicatedThread = uint(iotp.autoThreads + 1)
	iotp.currentAutoThread = defaultIOThread

	return &iotp
}

func (iotp *IOThreadsPlacer) IOThreadCount() uint {
	return uint(iotp.autoThreads + iotp.dedicatedThreads)
}

func (iotp *IOThreadsPlacer) CurrentAutoThread() uint {
	return iotp.currentAutoThread
}

func (iotp *IOThreadsPlacer) SetIOThreadToDisk(vmiDisk *v1.Disk, newDisk *api.Disk, c *ConverterContext) {
	if _, ok := c.HotplugVolumes[vmiDisk.Name]; !ok {
		if vmiDisk.DedicatedIOThread != nil && *vmiDisk.DedicatedIOThread {
			newDisk.Driver.IOThread = pointer.P(iotp.currentDedicatedThread)
			iotp.currentDedicatedThread += 1
		} else {
			newDisk.Driver.IOThread = pointer.P(iotp.currentAutoThread)
			// increment the threadId to be used next but wrap around at the thread limit
			// the odd math here is because thread ID's start at 1, not 0
			iotp.currentAutoThread = (iotp.currentAutoThread % uint(iotp.autoThreads)) + 1
		}
	} else {
		newDisk.Driver.IO = v1.IOThreads
	}
}

func CerateDomainDisks(vmi *v1.VirtualMachineInstance, ioThreadPlacer *IOThreadsPlacer, c *ConverterContext) ([]api.Disk, error) {
	var domainDisks []api.Disk

	cpuTopology := vcpu.GetCPUTopology(vmi)
	cpuCount := vcpu.CalculateRequestedVCPUs(cpuTopology)

	var numBlkQueues *uint
	virtioBlkMQRequested := (vmi.Spec.Domain.Devices.BlockMultiQueue != nil) && (*vmi.Spec.Domain.Devices.BlockMultiQueue)
	vcpus := uint(cpuCount)
	if vcpus == 0 {
		vcpus = uint(1)
	}

	if virtioBlkMQRequested {
		numBlkQueues = &vcpus
	}

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

	useIOThreads := UseIOThreads(vmi)
	prefixMap := newDeviceNamer(vmi.Status.VolumeStatus, vmi.Spec.Domain.Devices.Disks)
	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		newDisk := api.Disk{}

		err := Convert_v1_Disk_To_api_Disk(c, &disk, &newDisk, prefixMap, numBlkQueues, volumeStatusMap)
		if err != nil {
			return nil, err
		}
		volume := volumes[disk.Name]
		if volume == nil {
			return nil, fmt.Errorf("no matching volume with name %s found", disk.Name)
		}

		if _, ok := c.HotplugVolumes[disk.Name]; !ok {
			err = Convert_v1_Volume_To_api_Disk(volume, &newDisk, c, volumeIndices[disk.Name])
		} else {
			err = Convert_v1_Hotplug_Volume_To_api_Disk(volume, &newDisk, c)
		}
		if err != nil {
			return nil, err
		}

		if err := Convert_v1_BlockSize_To_api_BlockIO(&disk, &newDisk); err != nil {
			return nil, err
		}

		if useIOThreads {
			ioThreadPlacer.SetIOThreadToDisk(&disk, &newDisk, c)
		}

		if err := setErrorPolicy(&disk, &newDisk); err != nil {
			return nil, err
		}

		hpStatus, hpOk := c.HotplugVolumes[disk.Name]
		// if len(c.PermanentVolumes) == 0, it means the vmi is not ready yet, add all disks
		if _, ok := c.PermanentVolumes[disk.Name]; ok ||
			len(c.PermanentVolumes) == 0 ||
			(hpOk && (hpStatus.Phase == v1.HotplugVolumeMounted || hpStatus.Phase == v1.VolumeReady)) {
			domainDisks = append(domainDisks, newDisk)
		}
	}

	return domainDisks, nil
}

func UseIOThreads(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.IOThreadsPolicy != nil {
		return true
	}

	for _, diskDevice := range vmi.Spec.Domain.Devices.Disks {
		if diskDevice.DedicatedIOThread != nil &&
			*diskDevice.DedicatedIOThread {
			return true
		}
	}

	return false
}

func setErrorPolicy(diskDevice *v1.Disk, disk *api.Disk) error {
	if diskDevice.ErrorPolicy == nil {
		disk.Driver.ErrorPolicy = v1.DiskErrorPolicyStop
		return nil
	}
	switch *diskDevice.ErrorPolicy {
	case v1.DiskErrorPolicyStop, v1.DiskErrorPolicyIgnore, v1.DiskErrorPolicyReport, v1.DiskErrorPolicyEnospace:
		disk.Driver.ErrorPolicy = *diskDevice.ErrorPolicy
	default:
		return fmt.Errorf("error policy %s not recognized", *diskDevice.ErrorPolicy)
	}
	return nil
}
