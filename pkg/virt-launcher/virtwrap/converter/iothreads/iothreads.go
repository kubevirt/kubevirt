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

package iothreads

import (
	"slices"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	k8sv1 "k8s.io/api/core/v1"
)

const (
	defaultIOThread = uint(1)
)

func HasIOThreads(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.IOThreadsPolicy != nil {
		return true
	}
	return slices.ContainsFunc(vmi.Spec.Domain.Devices.Disks, hasDedicatedIOThread)
}

func hasDedicatedIOThread(disk v1.Disk) bool {
	return disk.DedicatedIOThread != nil && *disk.DedicatedIOThread
}

func getIOThreadsCountType(vmi *v1.VirtualMachineInstance) (ioThreadCount, autoThreads int) {
	dedicatedThreads := 0

	if vmi.Spec.Domain.IOThreadsPolicy != nil &&
		*vmi.Spec.Domain.IOThreadsPolicy == v1.IOThreadsPolicySupplementalPool &&
		vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount != nil {
		return int(*vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount), 0
	}

	for _, diskDevice := range vmi.Spec.Domain.Devices.Disks {
		if diskDevice.DedicatedIOThread != nil && *diskDevice.DedicatedIOThread {
			dedicatedThreads += 1
		} else {
			autoThreads += 1
		}
	}

	threadPoolLimit := getThreadPoolLimit(vmi)

	if (autoThreads + dedicatedThreads) > threadPoolLimit {
		autoThreads = threadPoolLimit - dedicatedThreads
		// We need at least one shared thread
		if autoThreads < 1 {
			autoThreads = 1
		}
	}

	ioThreadCount = autoThreads + dedicatedThreads
	return
}

func getThreadPoolLimit(vmi *v1.VirtualMachineInstance) int {
	policy := vmi.Spec.Domain.IOThreadsPolicy

	switch {
	case policy == nil, *policy == v1.IOThreadsPolicyShared:
		return 1
	case *policy == v1.IOThreadsPolicyAuto:
		// When IOThreads policy is set to auto and we've allocated a dedicated
		// pCPU for the emulator thread, we can place IOThread and Emulator thread in the same pCPU
		if vmi.IsCPUDedicated() && vmi.Spec.Domain.CPU.IsolateEmulatorThread {
			return 1
		}
		numCPUs := 1
		// Requested CPU's is guaranteed to be no greater than the limit
		if req, ok := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU]; ok {
			numCPUs = int(req.Value())
		} else if lim, ok := vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU]; ok {
			numCPUs = int(lim.Value())
		}
		return numCPUs * 2
	default:
		return 0
	}
}

func SetIOThreads(vmi *v1.VirtualMachineInstance, domain *api.Domain, vcpus uint) {
	if !HasIOThreads(vmi) {
		return
	}
	currentAutoThread := defaultIOThread
	ioThreadCount, autoThreads := getIOThreadsCountType(vmi)
	if ioThreadCount != 0 {
		if domain.Spec.IOThreads == nil {
			domain.Spec.IOThreads = &api.IOThreads{}
		}
		domain.Spec.IOThreads.IOThreads = uint(ioThreadCount)
	}
	if vmi.Spec.Domain.IOThreadsPolicy != nil &&
		*vmi.Spec.Domain.IOThreadsPolicy == v1.IOThreadsPolicySupplementalPool {
		iothreads := &api.DiskIOThreads{}
		for id := 1; id <= int(*vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount); id++ {
			iothreads.IOThread = append(iothreads.IOThread, api.DiskIOThread{Id: uint32(id)})
		}
		for i, disk := range domain.Spec.Devices.Disks {
			// Only disks with virtio bus support IOThreads
			if disk.Target.Bus == v1.DiskBusVirtio {
				domain.Spec.Devices.Disks[i].Driver.IOThreads = iothreads
			}
		}
	} else {
		currentDedicatedThread := uint(autoThreads + 1)
		for i, disk := range domain.Spec.Devices.Disks {
			// Only disks with virtio bus support IOThreads
			if disk.Target.Bus == v1.DiskBusVirtio {
				if vmi.Spec.Domain.Devices.Disks[i].DedicatedIOThread != nil && *vmi.Spec.Domain.Devices.Disks[i].DedicatedIOThread {
					domain.Spec.Devices.Disks[i].Driver.IOThread = pointer.P(currentDedicatedThread)
					currentDedicatedThread += 1
				} else {
					domain.Spec.Devices.Disks[i].Driver.IOThread = pointer.P(currentAutoThread)
					// increment the threadId to be used next but wrap around at the thread limit
					// the odd math here is because thread ID's start at 1, not 0
					currentAutoThread = (currentAutoThread % uint(autoThreads)) + 1
				}
			}
		}
	}

	// Virtio-scsi doesn't support IO threads yet, only the SCSI controller supports.
	setIOThreadSCSIController := false
	for i, disk := range domain.Spec.Devices.Disks {
		// Only disks with virtio bus support IOThreads
		if disk.Target.Bus == v1.DiskBusSCSI {
			if hasDedicatedIOThread(vmi.Spec.Domain.Devices.Disks[i]) {
				setIOThreadSCSIController = true
				break
			}
		}
	}
	if !setIOThreadSCSIController {
		return
	}
	for i, controller := range domain.Spec.Devices.Controllers {
		if controller.Type == "scsi" {
			if controller.Driver == nil {
				domain.Spec.Devices.Controllers[i].Driver = &api.ControllerDriver{}
			}
			domain.Spec.Devices.Controllers[i].Driver.IOThread = pointer.P(currentAutoThread)
			domain.Spec.Devices.Controllers[i].Driver.Queues = pointer.P(vcpus)
		}
	}
}
