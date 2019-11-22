package util

import (
	v1 "kubevirt.io/client-go/api/v1"
)

const ExtensionAPIServerAuthenticationConfigMap = "extension-apiserver-authentication"
const RequestHeaderClientCAFileKey = "requestheader-client-ca-file"
const VirtShareDir = "/var/run/kubevirt"
const VirtNetworkDir = "/var/run/kubevirt-network"
const VirtLibDir = "/var/lib/kubevirt"
const HostRootMount = "/proc/1/root/"
const CPUManagerOS3Path = HostRootMount + "var/lib/origin/openshift.local.volumes/cpu_manager_state"
const CPUManagerPath = HostRootMount + "var/lib/kubelet/cpu_manager_state"

func IsSRIOVVmi(vmi *v1.VirtualMachineInstance) bool {
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.SRIOV != nil {
			return true
		}
	}
	return false
}

// Check if a VMI spec requests GPU
func IsGPUVMI(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.Devices.GPUs != nil && len(vmi.Spec.Domain.Devices.GPUs) != 0 {
		return true
	}
	return false
}
