package tpm

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util"
)

func HasDevice(vmiSpec *v1.VirtualMachineInstanceSpec) bool {
	return vmiSpec.Domain.Devices.TPM != nil &&
		(vmiSpec.Domain.Devices.TPM.Enabled == nil || *vmiSpec.Domain.Devices.TPM.Enabled)
}

func HasPersistentDevice(vmiSpec *v1.VirtualMachineInstanceSpec) bool {
	if !HasDevice(vmiSpec) {
		return false
	}
	persistent := vmiSpec.Domain.Devices.TPM.Persistent
	// A declarative state PVC implies TPM state is kept unless explicitly opted out.
	if util.HasDeclarativeVMStateSpec(vmiSpec) {
		return persistent == nil || *persistent
	}
	return persistent != nil && *persistent
}
