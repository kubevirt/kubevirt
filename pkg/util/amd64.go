package util

import v1 "kubevirt.io/api/core/v1"

// Check if a VMI spec requests AMD SEV
func IsSEVVMI(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.LaunchSecurity != nil && (vmi.Spec.Domain.LaunchSecurity.SEV != nil || vmi.Spec.Domain.LaunchSecurity.SNP != nil)
}

// Check if VMI spec requests AMD SEV-ES
func IsSEVESVMI(vmi *v1.VirtualMachineInstance) bool {
	if !IsSEVVMI(vmi) {
		return false
	}
	if vmi.Spec.Domain.LaunchSecurity.SEV == nil {
		return false
	}
	if vmi.Spec.Domain.LaunchSecurity.SEV.Policy == nil {
		return false
	}
	if vmi.Spec.Domain.LaunchSecurity.SEV.Policy.EncryptedState == nil {
		return false
	}
	return *vmi.Spec.Domain.LaunchSecurity.SEV.Policy.EncryptedState
}

// Check if a VMI spec requests AMD SEV-SNP
func IsSEVSNPVMI(vmi *v1.VirtualMachineInstance) bool {
	if !IsSEVVMI(vmi) {
		return false
	}
	if vmi.Spec.Domain.LaunchSecurity.SNP == nil {
		return false
	}

	return true
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
