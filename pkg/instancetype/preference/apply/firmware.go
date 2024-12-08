//nolint:gocyclo
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
package apply

import (
	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

func applyFirmwarePreferences(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	if preferenceSpec.Firmware == nil {
		return
	}

	firmware := preferenceSpec.Firmware
	if vmiSpec.Domain.Firmware == nil {
		vmiSpec.Domain.Firmware = &virtv1.Firmware{}
	}

	vmiFirmware := vmiSpec.Domain.Firmware

	if vmiFirmware.Bootloader == nil {
		vmiFirmware.Bootloader = &virtv1.Bootloader{}
	}

	if firmware.PreferredUseBios != nil &&
		*firmware.PreferredUseBios &&
		vmiFirmware.Bootloader.BIOS == nil &&
		vmiFirmware.Bootloader.EFI == nil {
		vmiFirmware.Bootloader.BIOS = &virtv1.BIOS{}
	}

	if firmware.PreferredUseBiosSerial != nil && vmiFirmware.Bootloader.BIOS != nil && vmiFirmware.Bootloader.BIOS.UseSerial == nil {
		vmiFirmware.Bootloader.BIOS.UseSerial = pointer.P(*firmware.PreferredUseBiosSerial)
	}

	if vmiFirmware.Bootloader.EFI == nil && vmiFirmware.Bootloader.BIOS == nil && firmware.PreferredEfi != nil {
		vmiFirmware.Bootloader.EFI = firmware.PreferredEfi.DeepCopy()
		// When using PreferredEfi return early to avoid applying DeprecatedPreferredUseEfi or DeprecatedPreferredUseSecureBoot below
		return
	}

	if firmware.DeprecatedPreferredUseEfi != nil &&
		*firmware.DeprecatedPreferredUseEfi &&
		vmiFirmware.Bootloader.EFI == nil &&
		vmiFirmware.Bootloader.BIOS == nil {
		vmiFirmware.Bootloader.EFI = &virtv1.EFI{}
	}

	if firmware.DeprecatedPreferredUseSecureBoot != nil && vmiFirmware.Bootloader.EFI != nil && vmiFirmware.Bootloader.EFI.SecureBoot == nil {
		vmiFirmware.Bootloader.EFI.SecureBoot = pointer.P(*firmware.DeprecatedPreferredUseSecureBoot)
	}
}
