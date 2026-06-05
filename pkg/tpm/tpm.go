package tpm

import v1 "kubevirt.io/api/core/v1"

func HasDevice(vmiSpec *v1.VirtualMachineInstanceSpec) bool {
	return vmiSpec.Domain.Devices.TPM != nil &&
		(vmiSpec.Domain.Devices.TPM.Enabled == nil || *vmiSpec.Domain.Devices.TPM.Enabled)
}

func HasPersistentDevice(vmiSpec *v1.VirtualMachineInstanceSpec) bool {
	return HasDevice(vmiSpec) &&
		vmiSpec.Domain.Devices.TPM.Persistent != nil &&
		*vmiSpec.Domain.Devices.TPM.Persistent
}
