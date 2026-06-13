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

package webhooks

import (
	"encoding/base64"
	"fmt"
	"slices"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

// ValidateVirtualMachineInstanceAmd64Setting is a validation function for validating-webhook on Amd64
func ValidateVirtualMachineInstanceAmd64Setting(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var statusCauses []metav1.StatusCause
	validateWatchdogAmd64(field, spec, &statusCauses)
	validateVideoTypeAmd64(field, spec, &statusCauses)
	return statusCauses
}

func validateVideoTypeAmd64(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, statusCauses *[]metav1.StatusCause) {
	if spec.Domain.Devices.Video == nil {
		return
	}

	videoType := spec.Domain.Devices.Video.Type

	validTypes := []string{"vga", "cirrus", "virtio", "ramfb", "bochs"}
	if !slices.Contains(validTypes, videoType) {
		*statusCauses = append(*statusCauses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("video model '%s' is not supported on amd64 architecture", videoType),
			Field:   field.Child("domain", "devices", "video").Child("type").String(),
		})
	}
}

func validateWatchdogAmd64(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, statusCauses *[]metav1.StatusCause) {
	watchdog := spec.Domain.Devices.Watchdog
	if watchdog == nil {
		return
	}

	if !isOnlyI6300ESBWatchdog(watchdog) {
		*statusCauses = append(*statusCauses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: "amd64 only supports I6300ESB watchdog device",
			Field:   field.Child("domain", "devices", "watchdog").String(),
		})
	}
}

func isOnlyI6300ESBWatchdog(watchdog *v1.Watchdog) bool {
	return watchdog.WatchdogDevice.I6300ESB != nil && watchdog.WatchdogDevice.Diag288 == nil
}

func ValidateLaunchSecurityAmd64(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause
	launchSecurity := spec.Domain.LaunchSecurity

	fg := ""
	var selectedTypes []string
	if launchSecurity.SEV != nil {
		selectedTypes = append(selectedTypes, "SEV")
		fg = featuregate.WorkloadEncryptionSEV
	}
	if launchSecurity.SNP != nil {
		selectedTypes = append(selectedTypes, "SNP")
		fg = featuregate.WorkloadEncryptionSEV
	}
	if launchSecurity.TDX != nil {
		selectedTypes = append(selectedTypes, "TDX")
		fg = featuregate.WorkloadEncryptionTDX
	}

	// We always get a valid launchSecurity type after this check
	if len(selectedTypes) != 1 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeForbidden,
			Message: "One and only one launchSecurity type can be set",
			Field:   field.Child("launchSecurity").String(),
		})
	} else if ((launchSecurity.SEV != nil || launchSecurity.SNP != nil) && !config.WorkloadEncryptionSEVEnabled()) ||
		(launchSecurity.TDX != nil && !config.WorkloadEncryptionTDXEnabled()) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s feature gate is not enabled in kubevirt-config", fg),
			Field:   field.Child("launchSecurity").String(),
		})
	} else {
		features := spec.Domain.Features
		if launchSecurity.TDX != nil &&
			(features != nil && features.SMM != nil && (features.SMM.Enabled == nil || *features.SMM.Enabled)) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "TDX does not work along with SMM",
				Field:   field.Child("launchSecurity").String(),
			})
		}

		firmware := spec.Domain.Firmware
		if firmware == nil || firmware.Bootloader == nil || firmware.Bootloader.EFI == nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s requires OVMF (UEFI)", selectedTypes[0]),
				Field:   field.Child("launchSecurity").String(),
			})
		} else {
			efi := firmware.Bootloader.EFI
			if (launchSecurity.SEV != nil || launchSecurity.SNP != nil) &&
				(efi.SecureBoot == nil || *efi.SecureBoot) {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s does not work along with SecureBoot", selectedTypes[0]),
					Field:   field.Child("launchSecurity").String(),
				})
			}

			if (launchSecurity.SNP != nil || launchSecurity.TDX != nil) &&
				(efi.Persistent != nil && *efi.Persistent) {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s does not work along with Persistent EFI variables", selectedTypes[0]),
					Field:   field.Child("launchSecurity").String(),
				})
			}
		}

		startStrategy := spec.StartStrategy
		if launchSecurity.SEV != nil &&
			(startStrategy == nil || *startStrategy != v1.StartStrategyPaused) {
			if launchSecurity.SEV.Attestation != nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("SEV attestation requires VMI StartStrategy '%s'", v1.StartStrategyPaused),
					Field:   field.Child("launchSecurity").String(),
				})
			}
		}
		if launchSecurity.SNP != nil && launchSecurity.SNP.Policy != "" {
			// Check if policy is a valid decimal or hex value
			dec, err := strconv.ParseUint(launchSecurity.SNP.Policy, 0, 64)
			if err != nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s is not a valid SEV-SNP Policy Config", launchSecurity.SNP.Policy),
					Field:   field.Child("launchSecurity", "snp").String(),
				})
			}
			// Ensure bit 17 (0x20000) is set to 1, which is required by AMD SEV-SNP specification
			policyBitReserved := uint64(1 << 17)
			if dec&policyBitReserved != policyBitReserved && err == nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s SEV-SNP Policy Config must have bit 17 (0x%X) set to 1", launchSecurity.SNP.Policy, policyBitReserved),
					Field:   field.Child("launchSecurity", "snp").String(),
				})
			}
		}

		if launchSecurity.SNP != nil && launchSecurity.SNP.HostData != "" {
			// libvirt expects hostData as base64-encoded data that decodes to
			// exactly 32 bytes.
			decoded, err := base64.StdEncoding.DecodeString(launchSecurity.SNP.HostData)
			if err != nil || len(decoded) != 32 {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s is not a valid SEV-SNP HostData value, must be base64-encoded data that decodes to exactly 32 bytes", launchSecurity.SNP.HostData),
					Field:   field.Child("launchSecurity", "snp").Child("hostData").String(),
				})
			}
		}

		if launchSecurity.SNP != nil {
			// IdBlock and IdAuth must be set together; AuthorKey is optional but
			// requires both IdBlock and IdAuth when enabled.
			hasAuthorKey := launchSecurity.SNP.AuthorKey != nil && *launchSecurity.SNP.AuthorKey
			hasIdBlock := launchSecurity.SNP.IdBlock != ""
			hasIdAuth := launchSecurity.SNP.IdAuth != ""

			if hasIdBlock != hasIdAuth {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "IdBlock and IdAuth must be set together for guest identity attestation",
					Field:   field.Child("launchSecurity", "snp").String(),
				})
			}
			if hasAuthorKey && !(hasIdBlock && hasIdAuth) {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "AuthorKey requires both IdBlock and IdAuth to be set",
					Field:   field.Child("launchSecurity", "snp").String(),
				})
			}
			// Validate base64 encoding and length, IdBlock, and IdAuth
			if hasIdBlock {
				decodedIdBlock, err := base64.StdEncoding.DecodeString(launchSecurity.SNP.IdBlock)
				if err != nil || len(decodedIdBlock) != 96 {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("%s is not a valid SEV-SNP IdBlock value, must be base64-encoded data that decodes to exactly 96 bytes", launchSecurity.SNP.IdBlock),
						Field:   field.Child("launchSecurity", "snp").Child("idBlock").String(),
					})
				}
			}
			if hasIdAuth {
				decodedIdAuth, err := base64.StdEncoding.DecodeString(launchSecurity.SNP.IdAuth)
				if err != nil || len(decodedIdAuth) != 4096 {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("%s is not a valid SEV-SNP IdAuth value, must be base64-encoded data that decodes to exactly 4096 bytes", launchSecurity.SNP.IdAuth),
						Field:   field.Child("launchSecurity", "snp").Child("idAuth").String(),
					})
				}
			}
		}

		if launchSecurity.SNP != nil && launchSecurity.SNP.KernelHashes != nil {
			// Measured direct boot requires the kernel/initrd to be provided directly
			if spec.Domain.Firmware == nil ||
				spec.Domain.Firmware.KernelBoot == nil ||
				spec.Domain.Firmware.KernelBoot.Container == nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "KernelHashes requires direct kernel boot configuration (spec.domain.firmware.kernelBoot)",
					Field:   field.Child("launchSecurity", "snp", "kernelHashes").String(),
				})
			}
		}

		for _, iface := range spec.Domain.Devices.Interfaces {
			if iface.BootOrder != nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s does not work with bootable NICs: %s", selectedTypes[0], iface.Name),
					Field:   field.Child("launchSecurity").String(),
				})
			}
		}
	}

	return causes
}
