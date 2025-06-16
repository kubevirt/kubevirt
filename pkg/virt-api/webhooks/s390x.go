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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
)

// ValidateVirtualMachineInstanceS390xSetting is a validation function for validating-webhook on s390x
func ValidateVirtualMachineInstanceS390XSetting(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var statusCauses []metav1.StatusCause
	validateWatchdogS390x(field, spec, &statusCauses)
	return statusCauses
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
