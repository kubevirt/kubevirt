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

package vfio

import (
	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
)

const (
	// MMIOOverheadBytes is the MMIO overhead added to the memlock limit
	// for VFIO devices. It originates from x86 systems where it represents
	// reserved MMIO space and matches libvirt's qemuDomainGetMemLockLimitBytes.
	MMIOOverheadBytes = 1024 * 1024 * 1024
)

// CalculateMemlockLimit computes the memlock limit needed for VFIO device
// passthrough, matching libvirt's qemuDomainGetMemLockLimitBytes formula:
// numDevices * guest_memory + 1GiB. Returns 0 if no VFIO devices are
// present. The value is in bytes.
func CalculateMemlockLimit(vmi *v1.VirtualMachineInstance) int64 {
	numDevices := CountDevices(vmi)
	if numDevices == 0 {
		return 0
	}
	return int64(numDevices)*getVirtualMemoryBytes(vmi) + MMIOOverheadBytes
}

// CalculateMemlockExtraBytes returns the additional memlock bytes needed
// beyond what GetMemoryOverhead already provides (which includes
// 1 * guestMemory + 1GiB MMIO). Returns (N-1) * guestMemory for N > 1
// devices, or 0 otherwise.
func CalculateMemlockExtraBytes(vmi *v1.VirtualMachineInstance) int64 {
	numDevices := CountDevices(vmi)
	if numDevices <= 1 {
		return 0
	}
	return int64(numDevices-1) * getVirtualMemoryBytes(vmi)
}

// CountDevices returns the total number of VFIO devices (GPUs,
// HostDevices, and SRIOV interfaces) in the VMI spec.
func CountDevices(vmi *v1.VirtualMachineInstance) int {
	count := len(vmi.Spec.Domain.Devices.GPUs) + len(vmi.Spec.Domain.Devices.HostDevices)
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.SRIOV != nil {
			count++
		}
	}
	return count
}

func getVirtualMemoryBytes(vmi *v1.VirtualMachineInstance) int64 {
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Guest != nil {
		return vmi.Spec.Domain.Memory.Guest.Value()
	}
	if req, ok := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]; ok {
		return req.Value()
	}
	if lim, ok := vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceMemory]; ok {
		return lim.Value()
	}
	return 0
}
