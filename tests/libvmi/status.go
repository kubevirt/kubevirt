package libvmi

import (
	v1 "kubevirt.io/api/core/v1"
)

func IndexInterfaceStatusByName(vmi *v1.VirtualMachineInstance) map[string]v1.VirtualMachineInstanceNetworkInterface {
	interfaceStatusByName := map[string]v1.VirtualMachineInstanceNetworkInterface{}
	for _, interfaceStatus := range vmi.Status.Interfaces {
		interfaceStatusByName[interfaceStatus.Name] = interfaceStatus
	}
	return interfaceStatusByName
}
