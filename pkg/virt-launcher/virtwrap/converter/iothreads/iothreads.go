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
	for _, diskDevice := range vmi.Spec.Domain.Devices.Disks {
		if diskDevice.DedicatedIOThread != nil && *diskDevice.DedicatedIOThread {
			return true
		}
	}
	return false
}

func getIOThreadsCountType(vmi *v1.VirtualMachineInstance) (ioThreadCount, autoThreads int) {
	dedicatedThreads := 0

	var threadPoolLimit int
	policy := vmi.Spec.Domain.IOThreadsPolicy
	switch {
	case policy == nil:
		threadPoolLimit = 1
	case *policy == v1.IOThreadsPolicyShared:
		threadPoolLimit = 1
	case *policy == v1.IOThreadsPolicyAuto:
		// When IOThreads policy is set to auto and we've allocated a dedicated
		// pCPU for the emulator thread, we can place IOThread and Emulator thread in the same pCPU
		if vmi.IsCPUDedicated() && vmi.Spec.Domain.CPU.IsolateEmulatorThread {
			threadPoolLimit = 1
		} else {
			numCPUs := 1
			// Requested CPU's is guaranteed to be no greater than the limit
			if cpuRequests, ok := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU]; ok {
				numCPUs = int(cpuRequests.Value())
			} else if cpuLimit, ok := vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU]; ok {
				numCPUs = int(cpuLimit.Value())
			}

			threadPoolLimit = numCPUs * 2
		}
	case *policy == v1.IOThreadsPolicySupplementalPool:
		if vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount != nil {
			ioThreadCount = int(*vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount)
		}
		return
	}

	for _, diskDevice := range vmi.Spec.Domain.Devices.Disks {
		if diskDevice.DedicatedIOThread != nil && *diskDevice.DedicatedIOThread {
			dedicatedThreads += 1
		} else {
			autoThreads += 1
		}
	}

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
			if vmi.Spec.Domain.Devices.Disks[i].DedicatedIOThread != nil && *vmi.Spec.Domain.Devices.Disks[i].DedicatedIOThread {
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
