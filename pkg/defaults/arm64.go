/* Licensed under the Apache License, Version 2.0 (the "License");
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
 * Copyright 2021
 *
 */
package defaults

import (
	v1 "kubevirt.io/api/core/v1"
)

var _false bool = false

const (
	defaultCPUModelArm64 = v1.CPUModeHostPassthrough
)

// setDefaultArm64CPUModel set default cpu model to host-passthrough
func setDefaultArm64CPUModel(spec *v1.VirtualMachineInstanceSpec) {
	if spec.Domain.CPU == nil {
		spec.Domain.CPU = &v1.CPU{}
	}

	if spec.Domain.CPU.Model == "" {
		spec.Domain.CPU.Model = defaultCPUModelArm64
	}
}

// setDefaultArm64Bootloader set default bootloader to uefi boot
func setDefaultArm64Bootloader(spec *v1.VirtualMachineInstanceSpec) {
	if spec.Domain.Firmware == nil || spec.Domain.Firmware.Bootloader == nil {
		if spec.Domain.Firmware == nil {
			spec.Domain.Firmware = &v1.Firmware{}
		}
		if spec.Domain.Firmware.Bootloader == nil {
			spec.Domain.Firmware.Bootloader = &v1.Bootloader{}
		}
		spec.Domain.Firmware.Bootloader.EFI = &v1.EFI{}
		spec.Domain.Firmware.Bootloader.EFI.SecureBoot = &_false
	}
}

// setDefaultArm64DisksBus set default Disks Bus, because sata is not supported by qemu-kvm of Arm64
func setDefaultArm64DisksBus(spec *v1.VirtualMachineInstanceSpec) {
	bus := v1.DiskBusVirtio

	for i := range spec.Domain.Devices.Disks {
		disk := &spec.Domain.Devices.Disks[i].DiskDevice

		if disk.Disk != nil && disk.Disk.Bus == "" {
			disk.Disk.Bus = bus
		}
		if disk.CDRom != nil && disk.CDRom.Bus == "" {
			disk.CDRom.Bus = bus
		}
		if disk.LUN != nil && disk.LUN.Bus == "" {
			disk.LUN.Bus = bus
		}
	}

}

// SetArm64Defaults is mutating function for mutating-webhook
func SetArm64Defaults(spec *v1.VirtualMachineInstanceSpec) {
	setDefaultArm64CPUModel(spec)
	setDefaultArm64Bootloader(spec)
	setDefaultArm64DisksBus(spec)
}

func IsARM64(vmiSpec *v1.VirtualMachineInstanceSpec) bool {
	return vmiSpec.Architecture == "arm64"
}
