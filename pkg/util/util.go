package util

import (
	"fmt"
	"os"
	"strings"

	v1 "kubevirt.io/client-go/apis/core/v1"
)

const (
	ExtensionAPIServerAuthenticationConfigMap = "extension-apiserver-authentication"
	RequestHeaderClientCAFileKey              = "requestheader-client-ca-file"
	VirtShareDir                              = "/var/run/kubevirt"
	VirtPrivateDir                            = "/var/run/kubevirt-private"
	VirtLibDir                                = "/var/lib/kubevirt"
	KubeletPodsDir                            = "/var/lib/kubelet/pods"
	HostRootMount                             = "/proc/1/root/"
	CPUManagerOS3Path                         = HostRootMount + "var/lib/origin/openshift.local.volumes/cpu_manager_state"
	CPUManagerPath                            = HostRootMount + "var/lib/kubelet/cpu_manager_state"
)

const NonRootUID = 107
const NonRootUserString = "qemu"
const RootUser = 0

func IsNonRootVMI(vmi *v1.VirtualMachineInstance) bool {
	_, ok := vmi.Annotations[v1.NonRootVMIAnnotation]
	return ok
}

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

// Check if a VMI spec requests VirtIO-FS
func IsVMIVirtiofsEnabled(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.Devices.Filesystems != nil {
		for _, fs := range vmi.Spec.Domain.Devices.Filesystems {
			if fs.Virtiofs != nil {
				return true
			}
		}
	}
	return false
}

// Check if a VMI spec requests a HostDevice
func IsHostDevVMI(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.Devices.HostDevices != nil && len(vmi.Spec.Domain.Devices.HostDevices) != 0 {
		return true
	}
	return false
}

// Check if a VMI spec requests a VFIO device
func IsVFIOVMI(vmi *v1.VirtualMachineInstance) bool {

	if IsHostDevVMI(vmi) || IsGPUVMI(vmi) || IsSRIOVVmi(vmi) {
		return true
	}
	return false
}

// WantVirtioNetDevice checks whether a VMI references at least one "virtio" network interface.
// Note that the reference can be explicit or implicit (unspecified nic models defaults to "virtio").
func WantVirtioNetDevice(vmi *v1.VirtualMachineInstance) bool {
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.Model == "" || iface.Model == "virtio" {
			return true
		}
	}
	return false
}

// NeedVirtioNetDevice checks whether a VMI requires the presence of the "virtio" net device.
// This happens when the VMI wants to use a "virtio" network interface, and software emulation is disallowed.
func NeedVirtioNetDevice(vmi *v1.VirtualMachineInstance, allowEmulation bool) bool {
	return WantVirtioNetDevice(vmi) && !allowEmulation
}

// UseSoftwareEmulationForDevice determines whether to fallback to software emulation for the given device.
// This happens when the given device doesn't exist, and software emulation is enabled.
func UseSoftwareEmulationForDevice(devicePath string, allowEmulation bool) (bool, error) {
	if !allowEmulation {
		return false, nil
	}

	_, err := os.Stat(devicePath)
	if err == nil {
		return false, nil
	}
	if os.IsNotExist(err) {
		return true, nil
	}
	return false, err
}

func ResourceNameToEnvVar(prefix string, resourceName string) string {
	varName := strings.ToUpper(resourceName)
	varName = strings.Replace(varName, "/", "_", -1)
	varName = strings.Replace(varName, ".", "_", -1)
	return fmt.Sprintf("%s_%s", prefix, varName)
}

// Checks if kernel boot is defined in a valid way
func HasKernelBootContainerImage(vmi *v1.VirtualMachineInstance) bool {
	if vmi == nil {
		return false
	}

	vmiFirmware := vmi.Spec.Domain.Firmware
	if (vmiFirmware == nil) || (vmiFirmware.KernelBoot == nil) || (vmiFirmware.KernelBoot.Container == nil) {
		return false
	}

	return true
}

func HasHugePages(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Hugepages != nil
}
