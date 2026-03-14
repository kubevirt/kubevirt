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
 * Copyright The KubeVirt Authors
 *
 */

package predicates

import v1 "kubevirt.io/api/core/v1"

// Check if a VMI spec requests AMD SEV
func IsSEVVMI(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.LaunchSecurity == nil {
		return false
	}
	return vmi.Spec.Domain.LaunchSecurity.SEV != nil || vmi.Spec.Domain.LaunchSecurity.SNP != nil
}

// Check if VMI spec requests AMD SEV-ES
func IsSEVESVMI(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.LaunchSecurity == nil ||
		vmi.Spec.Domain.LaunchSecurity.SEV == nil ||
		vmi.Spec.Domain.LaunchSecurity.SEV.Policy == nil ||
		vmi.Spec.Domain.LaunchSecurity.SEV.Policy.EncryptedState == nil {
		return false
	}
	return *vmi.Spec.Domain.LaunchSecurity.SEV.Policy.EncryptedState
}

// Check if a VMI spec requests AMD SEV-SNP
func IsSEVSNPVMI(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.LaunchSecurity != nil && vmi.Spec.Domain.LaunchSecurity.SNP != nil
}

// Check if a VMI spec requests SEV with attestation
func IsSEVAttestationRequested(vmi *v1.VirtualMachineInstance) bool {
	if !IsSEVVMI(vmi) {
		return false
	}
	// If SEV-SNP is requested, attestation is not applicable
	if IsSEVSNPVMI(vmi) {
		return false
	}
	// Check if SEV is configured before accessing Attestation
	if vmi.Spec.Domain.LaunchSecurity.SEV == nil {
		return false
	}
	return vmi.Spec.Domain.LaunchSecurity.SEV.Attestation != nil
}

// Check if a VMI spec requests Intel TDX
func IsTDXVMI(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.LaunchSecurity != nil && vmi.Spec.Domain.LaunchSecurity.TDX != nil
}

// Check if a VMI spec requests Secure Execution
func IsSecureExecutionVMI(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.LaunchSecurity != nil && vmi.Spec.Architecture == "s390x"
}

func IsNonRootVMI(vmi *v1.VirtualMachineInstance) bool {
	_, ok := vmi.Annotations[v1.DeprecatedNonRootVMIAnnotation]

	nonRoot := vmi.Status.RuntimeUser != 0
	return ok || nonRoot
}

func isSRIOVVmi(vmi *v1.VirtualMachineInstance) bool {
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.SRIOV != nil {
			return true
		}
	}
	return false
}

// Check if a VMI spec requests GPU
func IsGPUVMI(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.Devices.GPUs != nil && len(vmi.Spec.Domain.Devices.GPUs) != 0 {
		return true
	}
	return false
}

// Check if a VMI spec requests VirtIO-FS
func IsVMIVirtiofsEnabled(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.Devices.Filesystems != nil {
		for _, fs := range vmi.Spec.Domain.Devices.Filesystems {
			if fs.Virtiofs != nil {
				return true
			}
		}
	}
	return false
}

// Check if a VMI spec requests a HostDevice
func IsHostDevVMI(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.Devices.HostDevices != nil && len(vmi.Spec.Domain.Devices.HostDevices) != 0 {
		return true
	}
	return false
}

// Check if a VMI spec requests a VFIO device
func IsVFIOVMI(vmi *v1.VirtualMachineInstance) bool {

	if IsHostDevVMI(vmi) || IsGPUVMI(vmi) || isSRIOVVmi(vmi) {
		return true
	}
	return false
}

// Check if a VMI spec requests memory overhead
func RequiresMemoryOverheadReservation(v *v1.VirtualMachineInstance) bool {
	return v.Spec.Domain.Memory != nil &&
		v.Spec.Domain.Memory.ReservedOverhead != nil &&
		v.Spec.Domain.Memory.ReservedOverhead.AddedOverhead != nil
}

// Check if a VMI spec requests locking VM's memory (e.g. for DMA)
func RequiresLockingMemory(v *v1.VirtualMachineInstance) bool {
	return v.Spec.Domain.Memory != nil &&
		v.Spec.Domain.Memory.ReservedOverhead != nil &&
		v.Spec.Domain.Memory.ReservedOverhead.MemLock != nil &&
		*v.Spec.Domain.Memory.ReservedOverhead.MemLock == v1.MemLockRequired
}

func UseLaunchSecurity(vmi *v1.VirtualMachineInstance) bool {
	return IsSEVVMI(vmi) || IsSecureExecutionVMI(vmi) || IsTDXVMI(vmi)
}

func IsAutoAttachVSOCK(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.Devices.AutoattachVSOCK != nil && *vmi.Spec.Domain.Devices.AutoattachVSOCK
}

// Checks if kernel boot is defined in a valid way
func HasKernelBootContainerImage(vmi *v1.VirtualMachineInstance) bool {
	if vmi == nil {
		return false
	}

	vmiFirmware := vmi.Spec.Domain.Firmware
	if (vmiFirmware == nil) || (vmiFirmware.KernelBoot == nil) || (vmiFirmware.KernelBoot.Container == nil) {
		return false
	}

	return true
}
