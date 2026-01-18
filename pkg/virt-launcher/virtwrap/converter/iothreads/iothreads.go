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

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	k8sv1 "k8s.io/api/core/v1"
)

func HasIOThreads(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.IOThreadsPolicy != nil {
		return true
	}
	return slices.ContainsFunc(vmi.Spec.Domain.Devices.Disks, HasDedicatedIOThread)
}

func HasDedicatedIOThread(disk v1.Disk) bool {
	return disk.DedicatedIOThread != nil && *disk.DedicatedIOThread
}

func CalculateThreadAllocation(vmi *v1.VirtualMachineInstance) (uint, uint) {
	if isSupplementalPolicy(vmi) {
		if vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount != nil {
			ioThreadCount := *vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount
			return uint(ioThreadCount), uint(ioThreadCount)
		}
		return 0, 0
	}

	var sharedCount, dedicatedCount int
	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		if HasDedicatedIOThread(disk) {
			dedicatedCount++
		} else {
			sharedCount++
		}
	}

	poolLimit := getThreadPoolLimit(vmi)

	poolSize := sharedCount
	if (poolSize + dedicatedCount) > poolLimit {
		poolSize = poolLimit - dedicatedCount
	}
	if poolSize < 1 {
		poolSize = 1
	}

	return uint(poolSize), uint(poolSize + dedicatedCount)
}

func getThreadPoolLimit(vmi *v1.VirtualMachineInstance) int {
	policy := vmi.Spec.Domain.IOThreadsPolicy

	if policy == nil || *policy == v1.IOThreadsPolicyShared {
		return 1
	}

	if *policy == v1.IOThreadsPolicyAuto {
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
	}

	return 1
}

func SupplementalIOThreads(vmi *v1.VirtualMachineInstance, poolSize uint) *api.DiskIOThreads {
	if !isSupplementalPolicy(vmi) {
		return nil
	}

	supplementalIOThreads := &api.DiskIOThreads{}
	for id := uint(1); id <= poolSize; id++ {
		supplementalIOThreads.IOThread = append(supplementalIOThreads.IOThread, api.DiskIOThread{Id: uint32(id)})
	}

	return supplementalIOThreads
}

func isSupplementalPolicy(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.IOThreadsPolicy != nil && *vmi.Spec.Domain.IOThreadsPolicy == v1.IOThreadsPolicySupplementalPool
}
