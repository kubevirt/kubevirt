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

/*
 * arm64 utilities are in the webhooks package because they are used both
 * by validation and mutation webhooks.
 */
package webhooks

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
)

var _false bool = false

const (
	defaultCPUModel = v1.CPUModeHostPassthrough
)

// verifyInvalidSetting verify if VMI spec contain unavailable setting for arm64, check following items:
// 1. if setting bios boot
// 2. if use uefi secure boot
// 3. if use host-model for cpu model
func verifyInvalidSetting(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) (metav1.StatusCause, bool) {
	if spec.Domain.Firmware != nil && spec.Domain.Firmware.Bootloader != nil {
		if spec.Domain.Firmware.Bootloader.BIOS != nil {
			return metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: "Arm64 does not support bios boot, please change to uefi boot",
				Field:   field.Child("domain", "firmware", "bootloader", "bios").String(),
			}, false
		}
		if spec.Domain.Firmware.Bootloader.EFI != nil {
			// When EFI is enable, secureboot is enabled by default, so here check two condition
			// 1 is EFI is enabled without Secureboot setting
			// 2 is both EFI and Secureboot enabled
			if spec.Domain.Firmware.Bootloader.EFI.SecureBoot == nil || (spec.Domain.Firmware.Bootloader.EFI.SecureBoot != nil && *spec.Domain.Firmware.Bootloader.EFI.SecureBoot) {
				return metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueNotSupported,
					Message: "UEFI secure boot is currently not supported on aarch64 Arch",
					Field:   field.Child("domain", "firmware", "bootloader", "efi", "secureboot").String(),
				}, false
			}
		}
	}
	if spec.Domain.CPU != nil && (&spec.Domain.CPU.Model != nil) && spec.Domain.CPU.Model == "host-model" {
		return metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: "Arm64 not support host model well",
			Field:   field.Child("domain", "cpu", "model").String(),
		}, false
	}
	return metav1.StatusCause{}, true
}

// setDefaultCPUModel set default cpu model to host-passthrough
func setDefaultCPUModel(vmi *v1.VirtualMachineInstance) {
	if vmi.Spec.Domain.CPU == nil {
		vmi.Spec.Domain.CPU = &v1.CPU{}
	}

	vmi.Spec.Domain.CPU.Model = defaultCPUModel
}

// setDefaultBootloader set default bootloader to uefi boot
func setDefaultBootloader(vmi *v1.VirtualMachineInstance) {
	if vmi.Spec.Domain.Firmware == nil || vmi.Spec.Domain.Firmware.Bootloader == nil {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}
		if vmi.Spec.Domain.Firmware.Bootloader == nil {
			vmi.Spec.Domain.Firmware.Bootloader = &v1.Bootloader{}
		}
		vmi.Spec.Domain.Firmware.Bootloader.EFI = &v1.EFI{}
		vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot = &_false
	}
}

// ValidateVirtualMachineInstanceArm64Setting is validation function for validating-webhook
func ValidateVirtualMachineInstanceArm64Setting(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if cause, ok := verifyInvalidSetting(field, spec); !ok {
		causes = append(causes, cause)
	}
	return causes
}

// SetVirtualMachineInstanceArm64Defaults is mutating function for mutating-webhook
func SetVirtualMachineInstanceArm64Defaults(vmi *v1.VirtualMachineInstance) error {
	path := k8sfield.NewPath("spec")
	if cause, ok := verifyInvalidSetting(path, &vmi.Spec); ok {
		setDefaultCPUModel(vmi)
		setDefaultBootloader(vmi)
	} else {
		return fmt.Errorf("%s", cause.Message)
	}
	return nil
}
