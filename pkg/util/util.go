package util

import (
	"os/exec"

	v1 "kubevirt.io/client-go/api/v1"
)

const ExtensionAPIServerAuthenticationConfigMap = "extension-apiserver-authentication"
const RequestHeaderClientCAFileKey = "requestheader-client-ca-file"
const VirtShareDir = "/var/run/kubevirt"
const VirtPrivateDir = "/var/run/kubevirt-private"
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

func isIpv6Disabled() bool {
	res, err := exec.Command("cat", "/proc/sys/net/ipv6/conf/default/disable_ipv6").Output()
	return err != nil || string(res) == "1"
}

// GetIPBindAddress returns IP bind address (either 0.0.0.0 or [::] according sysctl disable_ipv6)
func GetIPBindAddress() string {
	if isIpv6Disabled() {
		return "0.0.0.0"
	}

	return "[::]"
}

// GetLoopbackAddress returns the loopback IP address (either 127.0.0.1 or [::1] according sysctl disable_ipv6)
func GetLoopbackAddress() string {
	if isIpv6Disabled() {
		return "127.0.0.1"
	}

	return "[::1]"
}
