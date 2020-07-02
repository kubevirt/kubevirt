package util

import (
	"fmt"
	"strings"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	clientutil "kubevirt.io/client-go/util"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/rbac"
)

const ExtensionAPIServerAuthenticationConfigMap = "extension-apiserver-authentication"
const RequestHeaderClientCAFileKey = "requestheader-client-ca-file"
const VirtShareDir = "/var/run/kubevirt"
const VirtPrivateDir = "/var/run/kubevirt-private"
const VirtLibDir = "/var/lib/kubevirt"
const KubeletPodsDir = "/var/lib/kubelet/pods"
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
func FilterKubevirtLabels(labels map[string]string) map[string]string {
	m := make(map[string]string)
	if len(labels) == 0 {
		// Return the empty map to avoid edge cases
		return m
	}
	for label, value := range labels {
		if strings.HasPrefix(label, "kubevirt.io") {
			m[label] = value
		}
	}
	return m
}

func GetAllowedServiceAccounts() map[string]struct{} {
	ns, err := clientutil.GetNamespace()
	logger := log.DefaultLogger()

	if err != nil {
		logger.Info("Failed to get namespace. Fallback to default: 'kubevirt'")
		ns = "kubevirt"
	}

	// system:serviceaccount:{namespace}:{kubevirt-component}
	prefix := fmt.Sprintf("%s:%s:%s", "system", "serviceaccount", ns)
	return map[string]struct{}{
		fmt.Sprintf("%s:%s", prefix, rbac.ApiServiceAccountName):        {},
		fmt.Sprintf("%s:%s", prefix, rbac.HandlerServiceAccountName):    {},
		fmt.Sprintf("%s:%s", prefix, rbac.ControllerServiceAccountName): {},
	}
}
