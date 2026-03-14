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

// Checks if CPU pinning has been requested
func IsCPUDedicated(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.CPU != nil && vmi.Spec.Domain.CPU.DedicatedCPUPlacement
}

func IsBootloaderEFI(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.Firmware != nil && vmi.Spec.Domain.Firmware.Bootloader != nil &&
		vmi.Spec.Domain.Firmware.Bootloader.EFI != nil
}

// WantsToHaveQOSGuaranteed checks if cpu and memory limits and requests are identical on the VMI.
// This is the indicator that people want a VMI with QOS of guaranteed
// If memory limit is set but not its corresponding request, we will eventually set request=limit
func WantsToHaveQOSGuaranteed(vmi *v1.VirtualMachineInstance) bool {
	resources := vmi.Spec.Domain.Resources
	memoryWantsIt := (resources.Requests.Memory().IsZero() && !resources.Limits.Memory().IsZero()) ||
		(!resources.Requests.Memory().IsZero() && resources.Requests.Memory().Cmp(*resources.Limits.Memory()) == 0)
	cpuWantsIt := !resources.Requests.Cpu().IsZero() && resources.Requests.Cpu().Cmp(*resources.Limits.Cpu()) == 0
	return memoryWantsIt && cpuWantsIt
}

// ShouldStartPaused returns true if VMI should be started in paused state
func ShouldStartPaused(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.StartStrategy != nil && *vmi.Spec.StartStrategy == v1.StartStrategyPaused
}

func IsRealtimeEnabled(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.CPU != nil && vmi.Spec.Domain.CPU.Realtime != nil
}

// IsHighPerformanceVMI returns true if the VMI is considered as high performance.
// A VMI is considered as high performance if one of the following is true:
// - the vmi requests a dedicated cpu
// - the realtime flag is enabled
// - the vmi requests hugepages
func IsHighPerformanceVMI(vmi *v1.VirtualMachineInstance) bool {
	if IsCPUDedicated(vmi) || IsRealtimeEnabled(vmi) {
		return true
	}

	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Hugepages != nil {
		return true
	}

	return false
}

func IsDecentralizedMigration(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.TargetState != nil &&
		vmi.Status.MigrationState.SourceState != nil &&
		((vmi.Status.MigrationState.SourceState.SyncAddress == nil && vmi.Status.MigrationState.TargetState.SyncAddress != nil) ||
			(vmi.Status.MigrationState.SourceState.SyncAddress != nil && vmi.Status.MigrationState.TargetState.SyncAddress == nil))
}

func IsNonRootVMI(vmi *v1.VirtualMachineInstance) bool {
	_, ok := vmi.Annotations[v1.DeprecatedNonRootVMIAnnotation]

	nonRoot := vmi.Status.RuntimeUser != 0
	return ok || nonRoot
}

// Check if a VMI spec requests GPU
func IsGPUVMI(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.Devices.GPUs != nil && len(vmi.Spec.Domain.Devices.GPUs) != 0
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
	return vmi.Spec.Domain.Devices.HostDevices != nil && len(vmi.Spec.Domain.Devices.HostDevices) != 0
}

func isSRIOVVmi(vmi *v1.VirtualMachineInstance) bool {
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.SRIOV != nil {
			return true
		}
	}
	return false
}

// Check if a VMI spec requests a VFIO device
func IsVFIOVMI(vmi *v1.VirtualMachineInstance) bool {
	return IsHostDevVMI(vmi) || IsGPUVMI(vmi) || isSRIOVVmi(vmi)
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

// Check if a VMI spec requests Intel TDX
func IsTDXVMI(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.LaunchSecurity != nil && vmi.Spec.Domain.LaunchSecurity.TDX != nil
}

// Check if a VMI spec requests Secure Execution
func IsSecureExecutionVMI(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.LaunchSecurity != nil && vmi.Spec.Architecture == "s390x"
}
