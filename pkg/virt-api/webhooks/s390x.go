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

// ValidateVirtualMachineInstanceS390xSetting is a validation function for validating-webhook on s390x
func ValidateVirtualMachineInstanceS390XSetting(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var statusCauses []metav1.StatusCause
	validateWatchdogS390x(field, spec, &statusCauses)
	validateVideoTypeS390x(field, spec, &statusCauses)
	return statusCauses
}

func validateVideoTypeS390x(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, statusCauses *[]metav1.StatusCause) {
	if spec.Domain.Devices.Video == nil {
		return
	}

	videoType := spec.Domain.Devices.Video.Type

	validTypes := []string{"virtio"}
	if !slices.Contains(validTypes, videoType) {
		*statusCauses = append(*statusCauses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("video model '%s' is not supported on s390x architecture", videoType),
			Field:   field.Child("domain", "devices", "video").Child("type").String(),
		})
	}
}

func validateWatchdogS390x(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, statusCauses *[]metav1.StatusCause) {
	watchdog := spec.Domain.Devices.Watchdog
	if watchdog == nil {
		return
	}

	if !isOnlyDiag288Watchdog(watchdog) {
		*statusCauses = append(*statusCauses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: "s390x only supports Diag288 watchdog device",
			Field:   field.Child("domain", "devices", "watchdog").String(),
		})
	}
}

func isOnlyDiag288Watchdog(watchdog *v1.Watchdog) bool {
	return watchdog.WatchdogDevice.Diag288 != nil && watchdog.WatchdogDevice.I6300ESB == nil
}

func IsS390X(spec *v1.VirtualMachineInstanceSpec) bool {
	return spec.Architecture == "s390x"
}

func ValidateLaunchSecurityS390x(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if !config.SecureExecutionEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s feature gate is not enabled in kubevirt-config", featuregate.SecureExecution),
			Field:   field.Child("launchSecurity").String(),
		})
	}

	return causes
}
