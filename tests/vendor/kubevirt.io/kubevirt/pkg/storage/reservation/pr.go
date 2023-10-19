package reservation

import (
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"
)

const (
	sourceDaemonsPath     = "/var/run/kubevirt/daemons"
	hostSourceDaemonsPath = "/proc/1/root" + sourceDaemonsPath
	prHelperDir           = "pr"
	prHelperSocket        = "pr-helper.sock"
	prResourceName        = "pr-helper"
)

func GetPrResourceName() string {
	return prResourceName
}

func GetPrHelperSocketDir() string {
	return filepath.Join(sourceDaemonsPath, prHelperDir)
}

func GetPrHelperHostSocketDir() string {
	return filepath.Join(hostSourceDaemonsPath, prHelperDir)
}

func GetPrHelperSocketPath() string {
	return filepath.Join(GetPrHelperSocketDir(), prHelperSocket)
}

func GetPrHelperSocket() string {
	return prHelperSocket
}

func HasVMIPersistentReservation(vmi *v1.VirtualMachineInstance) bool {
	return HasVMISpecPersistentReservation(&vmi.Spec)
}

func HasVMISpecPersistentReservation(vmiSpec *v1.VirtualMachineInstanceSpec) bool {
	for _, disk := range vmiSpec.Domain.Devices.Disks {
		if disk.DiskDevice.LUN != nil && disk.DiskDevice.LUN.Reservation {
			return true
		}
	}
	return false
}
