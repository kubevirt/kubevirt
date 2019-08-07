package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	v1 "kubevirt.io/client-go/api/v1"
)

const ExtensionAPIServerAuthenticationConfigMap = "extension-apiserver-authentication"
const RequestHeaderClientCAFileKey = "requestheader-client-ca-file"
const VirtShareDir = "/var/run/kubevirt"
const VirtLibDir = "/var/lib/kubevirt"
const GpuDevice = "nvidia.com/"

func IsSRIOVVmi(vmi *v1.VirtualMachineInstance) bool {
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.SRIOV != nil {
			return true
		}
	}
	return false
}

func IsNvidiaGpuVmi(vmi *v1.VirtualMachineInstance) bool {
	for key := range vmi.Spec.Domain.Resources.Requests {
		if strings.HasPrefix(string(key), GpuDevice) {
			return true
		}
	}

	for key := range vmi.Spec.Domain.Resources.Limits {
		if strings.HasPrefix(string(key), GpuDevice) {
			return true
		}
	}
	return false
}
