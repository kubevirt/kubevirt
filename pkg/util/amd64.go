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

package util

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
