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
 * Copyright the KubeVirt Authors.
 *
 */

package webhooks

import (
	"fmt"
	"slices"

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
	secType := ""
	secTypeCount := 0
	if launchSecurity.SEV != nil {
		secType = "SEV"
		secTypeCount++
		fg = featuregate.WorkloadEncryptionSEV
	}
	if launchSecurity.TDX != nil {
		secType = "TDX"
		secTypeCount++
		fg = featuregate.WorkloadEncryptionTDX
	}
	if secTypeCount != 1 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "One and only one launchSecurity type should be set",
			Field:   field.Child("launchSecurity").String(),
		})
	}

	if (launchSecurity.SEV != nil && !config.WorkloadEncryptionSEVEnabled()) || (launchSecurity.TDX != nil && !config.WorkloadEncryptionTDXEnabled()) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s feature gate is not enabled in kubevirt-config", fg),
			Field:   field.Child("launchSecurity").String(),
		})
	}

	if launchSecurity.SEV != nil || launchSecurity.TDX != nil {
		firmware := spec.Domain.Firmware
		if firmware == nil || firmware.Bootloader == nil || firmware.Bootloader.EFI == nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s requires OVMF (UEFI)", secType),
				Field:   field.Child("launchSecurity").String(),
			})
		}

		if launchSecurity.SEV != nil && (firmware.Bootloader.EFI.SecureBoot == nil || *firmware.Bootloader.EFI.SecureBoot) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s does not work along with SecureBoot", secType),
				Field:   field.Child("launchSecurity").String(),
			})
		}

		startStrategy := spec.StartStrategy
		if launchSecurity.SEV != nil && launchSecurity.SEV.Attestation != nil && (startStrategy == nil || *startStrategy != v1.StartStrategyPaused) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("SEV attestation requires VMI StartStrategy '%s'", v1.StartStrategyPaused),
				Field:   field.Child("launchSecurity").String(),
			})
		}

		for _, iface := range spec.Domain.Devices.Interfaces {
			if iface.BootOrder != nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("%s does not work with bootable NICs: %s", secType, iface.Name),
					Field:   field.Child("launchSecurity").String(),
				})
			}
		}
	}
	return causes
}
