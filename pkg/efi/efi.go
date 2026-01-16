package efi

import v1 "kubevirt.io/api/core/v1"

func HasDevice(vmiSpec *v1.VirtualMachineInstanceSpec) bool {
	return vmiSpec.Domain.Firmware != nil && vmiSpec.Domain.Firmware.Bootloader != nil &&
		vmiSpec.Domain.Firmware.Bootloader.EFI != nil
}

func HasPersistentDevice(vmiSpec *v1.VirtualMachineInstanceSpec) bool {
	return HasDevice(vmiSpec) &&
		vmiSpec.Domain.Firmware.Bootloader.EFI.Persistent != nil &&
		*vmiSpec.Domain.Firmware.Bootloader.EFI.Persistent
}
