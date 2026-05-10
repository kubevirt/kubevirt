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

//nolint:gocyclo
func ConvertDisks(
	vmi *v1.VirtualMachineInstance, domain *api.Domain, c *convertertypes.ConverterContext,
) error {
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

	numBlkQueues := calculateBlkQueues(vmi)

	volumeStatusMap := make(map[string]v1.VolumeStatus)
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		volumeStatusMap[volumeStatus.Name] = volumeStatus
	}

	prefixMap := NewDeviceNamer(vmi.Status.VolumeStatus, vmi.Spec.Domain.Devices.Disks)
	currentAutoThread := uint(1)
	currentDedicatedThread := uint(autoThreads) + 1 //nolint:gosec // autoThreads is always non-negative
	supplementalIOThreads := iothreads.SupplementalPoolThreadCount(vmi)
	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		newDisk := api.Disk{}
		emptyCDRom := false

		err := ConvertV1DiskToAPIDisk(c, &disk, &newDisk, prefixMap, numBlkQueues, volumeStatusMap)
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

		hpStatus, hpOk := c.HotplugVolumes[disk.Name]
		switch {
		case emptyCDRom:
			err = ConvertV1MissingVolumeToAPIDisk(&newDisk)
		case hpOk:
			err = ConvertV1HotplugVolumeToAPIDisk(volume, &newDisk, c)
		default:
			err = ConvertV1VolumeToAPIDisk(volume, &newDisk, c, volumeIndices[disk.Name])
		}

		if err != nil {
			return err
		}

		if err := ConvertV1BlockSizeToAPIBlockIO(&disk, &newDisk, c.Architecture.GetArchitecture()); err != nil {
			return err
		}

		_, isPermVolume := c.PermanentVolumes[disk.Name]
		// if len(c.PermanentVolumes) == 0, it means the vmi is not ready yet, add all disks
		permReady := isPermVolume || len(c.PermanentVolumes) == 0
		hotplugReady := hpOk && (hpStatus.Phase == v1.HotplugVolumeMounted || hpStatus.Phase == v1.VolumeReady)

		if permReady || hotplugReady || emptyCDRom {
			domain.Spec.Devices.Disks = append(domain.Spec.Devices.Disks, newDisk)
		}
		if err := SetErrorPolicy(&disk, &newDisk); err != nil {
			return err
		}
		if hasIOThreads {
			currentDedicatedThread, currentAutoThread = AssignDiskIOThread(
				&disk, &newDisk, supplementalIOThreads, autoThreads,
				currentDedicatedThread, currentAutoThread,
			)
		}
	}

	return nil
}

func calculateBlkQueues(vmi *v1.VirtualMachineInstance) *uint {
	virtioBlkMQRequested := vmi.Spec.Domain.Devices.BlockMultiQueue != nil &&
		*vmi.Spec.Domain.Devices.BlockMultiQueue
	if !virtioBlkMQRequested {
		return nil
	}

	cpuTopology := vcpu.GetCPUTopology(vmi)
	cpuCount := vcpu.CalculateRequestedVCPUs(cpuTopology)
	vcpus := uint(cpuCount)
	if vcpus == 0 {
		vcpus = uint(1)
	}
	return &vcpus
}
