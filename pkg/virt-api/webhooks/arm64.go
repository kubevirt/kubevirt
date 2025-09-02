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

/*
 * arm64 utilities are in the webhooks package because they are used both
 * by validation and mutation webhooks.
 */
package webhooks

import (
	"fmt"
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
)

// ValidateVirtualMachineInstanceArm64Setting is a validation function for validating-webhook to filter unsupported setting on Arm64
func ValidateVirtualMachineInstanceArm64Setting(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var statusCauses []metav1.StatusCause
	validateBootOptions(field, spec, &statusCauses)
	validateCPUModel(field, spec, &statusCauses)
	validateDiskBus(field, spec, &statusCauses)
	validateWatchdog(field, spec, &statusCauses)
	validateSoundDevice(field, spec, &statusCauses)
	validateVideoTypeArm64(field, spec, &statusCauses)
	return statusCauses
}

func validateVideoTypeArm64(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, statusCauses *[]metav1.StatusCause) {
	if spec.Domain.Devices.Video == nil {
		return
	}

	videoType := spec.Domain.Devices.Video.Type

	validTypes := []string{"virtio", "ramfb"}
	if !slices.Contains(validTypes, videoType) {
		*statusCauses = append(*statusCauses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("video model '%s' is not supported on arm64 architecture", videoType),
			Field:   field.Child("domain", "devices", "video").Child("type").String(),
		})
	}
}

func validateBootOptions(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, statusCauses *[]metav1.StatusCause) {
	if spec.Domain.Firmware != nil && spec.Domain.Firmware.Bootloader != nil {
		if spec.Domain.Firmware.Bootloader.BIOS != nil {
			*statusCauses = append(*statusCauses, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: "Arm64 does not support bios boot, please change to uefi boot",
				Field:   field.Child("domain", "firmware", "bootloader", "bios").String(),
			})
		}
		if spec.Domain.Firmware.Bootloader.EFI != nil {
			// When EFI is enable, secureboot is enabled by default, so here check two condition
			// 1 is EFI is enabled without Secureboot setting
			// 2 is both EFI and Secureboot enabled
			if spec.Domain.Firmware.Bootloader.EFI.SecureBoot == nil || (spec.Domain.Firmware.Bootloader.EFI.SecureBoot != nil && *spec.Domain.Firmware.Bootloader.EFI.SecureBoot) {
				*statusCauses = append(*statusCauses, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueNotSupported,
					Message: "UEFI secure boot is currently not supported on aarch64 Arch",
					Field:   field.Child("domain", "firmware", "bootloader", "efi", "secureboot").String(),
				})
			}
		}
	}
}

func validateCPUModel(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, statusCauses *[]metav1.StatusCause) {
	if spec.Domain.CPU != nil && (&spec.Domain.CPU.Model != nil) && spec.Domain.CPU.Model != "" && spec.Domain.CPU.Model != v1.CPUModeHostPassthrough {
		*statusCauses = append(*statusCauses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("currently, %v is the only model supported on Arm64", v1.CPUModeHostPassthrough),
			Field:   field.Child("domain", "cpu", "model").String(),
		})
	}
}

func validateDiskBus(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, statusCauses *[]metav1.StatusCause) {
	if spec.Domain.Devices.Disks != nil {
		// checkIfBusAvailable: if bus type is nil, virtio, scsi return true, otherwise, return false
		checkIfBusAvailable := func(bus v1.DiskBus) bool {
			if bus == "" || bus == v1.DiskBusVirtio || bus == v1.DiskBusSCSI {
				return true
			}
			return false
		}

		for i, disk := range spec.Domain.Devices.Disks {
			if disk.Disk != nil && !checkIfBusAvailable(disk.Disk.Bus) {
				*statusCauses = append(*statusCauses, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueNotSupported,
					Message: "Arm64 not support this disk bus type, please use virtio or scsi",
					Field:   field.Child("domain", "devices", "disks").Index(i).Child("disk", "bus").String(),
				})
			}
			if disk.CDRom != nil && !checkIfBusAvailable(disk.CDRom.Bus) {
				*statusCauses = append(*statusCauses, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueNotSupported,
					Message: "Arm64 not support this disk bus type, please use virtio or scsi",
					Field:   field.Child("domain", "devices", "disks").Index(i).Child("cdrom", "bus").String(),
				})
			}
			if disk.LUN != nil && !checkIfBusAvailable(disk.LUN.Bus) {
				*statusCauses = append(*statusCauses, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueNotSupported,
					Message: "Arm64 not support this disk bus type, please use virtio or scsi",
					Field:   field.Child("domain", "devices", "disks").Index(i).Child("lun", "bus").String(),
				})
			}
		}
	}
}

func validateWatchdog(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, statusCauses *[]metav1.StatusCause) {
	if spec.Domain.Devices.Watchdog != nil {
		*statusCauses = append(*statusCauses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: "Arm64 not support Watchdog device",
			Field:   field.Child("domain", "devices", "watchdog").String(),
		})
	}
}

func validateSoundDevice(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, statusCauses *[]metav1.StatusCause) {
	if spec.Domain.Devices.Sound != nil {
		*statusCauses = append(*statusCauses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: "Arm64 not support sound device",
			Field:   field.Child("domain", "devices", "sound").String(),
		})
	}
}
