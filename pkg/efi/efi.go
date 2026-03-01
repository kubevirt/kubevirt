package efi

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
)

func HasDevice(vmiSpec *v1.VirtualMachineInstanceSpec) bool {
	return vmiSpec.Domain.Firmware != nil && vmiSpec.Domain.Firmware.Bootloader != nil &&
		vmiSpec.Domain.Firmware.Bootloader.EFI != nil
}

func HasPersistentDevice(vmiSpec *v1.VirtualMachineInstanceSpec) bool {
	return HasDevice(vmiSpec) &&
		vmiSpec.Domain.Firmware.Bootloader.EFI.Persistent != nil &&
		*vmiSpec.Domain.Firmware.Bootloader.EFI.Persistent
}

func GetEFIVarsFileName(vmi *v1.VirtualMachineInstance) string {
	return fmt.Sprintf("%s_VARS.fd", vmi.Name)
}
