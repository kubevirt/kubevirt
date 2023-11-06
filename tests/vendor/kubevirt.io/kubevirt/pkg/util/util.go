package util

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/api/core/v1"
	generatedscheme "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/scheme"
	"kubevirt.io/client-go/log"
)

const (
	ExtensionAPIServerAuthenticationConfigMap = "extension-apiserver-authentication"
	RequestHeaderClientCAFileKey              = "requestheader-client-ca-file"
	VirtShareDir                              = "/var/run/kubevirt"
	VirtPrivateDir                            = "/var/run/kubevirt-private"
	VirtLibDir                                = "/var/lib/kubevirt"
	KubeletRoot                               = "/var/lib/kubelet"
	KubeletPodsDir                            = KubeletRoot + "/pods"
	HostRootMount                             = "/proc/1/root/"
	CPUManagerOS3Path                         = HostRootMount + "var/lib/origin/openshift.local.volumes/cpu_manager_state"
	CPUManagerPath                            = HostRootMount + "var/lib/kubelet/cpu_manager_state"

	// Alphanums is the list of alphanumeric characters used to create a securely generated random string
	Alphanums = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	NonRootUID         = 107
	NonRootUserString  = "qemu"
	RootUser           = 0
	memoryDumpOverhead = 100 * 1024 * 1024

	UnprivilegedContainerSELinuxLabel = "system_u:object_r:container_file_t:s0"
)

func IsNonRootVMI(vmi *v1.VirtualMachineInstance) bool {
	_, ok := vmi.Annotations[v1.DeprecatedNonRootVMIAnnotation]

	nonRoot := vmi.Status.RuntimeUser != 0
	return ok || nonRoot
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

// Check if the VMI includes passt network interface(s)
func IsPasstVMI(vmi *v1.VirtualMachineInstance) bool {
	for _, net := range vmi.Spec.Domain.Devices.Interfaces {
		if net.Passt != nil {
			return true
		}
	}
	return false
}

// Check if a VMI spec requests AMD SEV
func IsSEVVMI(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.LaunchSecurity != nil && vmi.Spec.Domain.LaunchSecurity.SEV != nil
}

// Check if a VMI spec requests AMD SEV-ES
func IsSEVESVMI(vmi *v1.VirtualMachineInstance) bool {
	return IsSEVVMI(vmi) &&
		vmi.Spec.Domain.LaunchSecurity.SEV.Policy != nil &&
		vmi.Spec.Domain.LaunchSecurity.SEV.Policy.EncryptedState != nil &&
		*vmi.Spec.Domain.LaunchSecurity.SEV.Policy.EncryptedState == true
}

// Check if a VMI spec requests SEV with attestation
func IsSEVAttestationRequested(vmi *v1.VirtualMachineInstance) bool {
	return IsSEVVMI(vmi) && vmi.Spec.Domain.LaunchSecurity.SEV.Attestation != nil
}

func IsAMD64VMI(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Architecture == "amd64"
}

func IsARM64VMI(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Architecture == "arm64"
}

func IsEFIVMI(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.Firmware != nil &&
		vmi.Spec.Domain.Firmware.Bootloader != nil &&
		vmi.Spec.Domain.Firmware.Bootloader.EFI != nil
}

func IsVmiUsingHyperVReenlightenment(vmi *v1.VirtualMachineInstance) bool {
	if vmi == nil {
		return false
	}

	domainFeatures := vmi.Spec.Domain.Features

	return domainFeatures != nil && domainFeatures.Hyperv != nil && domainFeatures.Hyperv.Reenlightenment != nil &&
		domainFeatures.Hyperv.Reenlightenment.Enabled != nil && *domainFeatures.Hyperv.Reenlightenment.Enabled
}

// WantVirtioNetDevice checks whether a VMI references at least one "virtio" network interface.
// Note that the reference can be explicit or implicit (unspecified nic models defaults to "virtio").
func WantVirtioNetDevice(vmi *v1.VirtualMachineInstance) bool {
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.Model == "" || iface.Model == v1.VirtIO {
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

func NeedTunDevice(vmi *v1.VirtualMachineInstance) bool {
	return (len(vmi.Spec.Domain.Devices.Interfaces) > 0) ||
		(vmi.Spec.Domain.Devices.AutoattachPodInterface == nil) ||
		(*vmi.Spec.Domain.Devices.AutoattachPodInterface == true)
}

func IsAutoAttachVSOCK(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.Devices.AutoattachVSOCK != nil && *vmi.Spec.Domain.Devices.AutoattachVSOCK
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
	if errors.Is(err, os.ErrNotExist) {
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

func IsReadOnlyDisk(disk *v1.Disk) bool {
	isReadOnlyCDRom := disk.CDRom != nil && (disk.CDRom.ReadOnly == nil || *disk.CDRom.ReadOnly == true)

	return isReadOnlyCDRom
}

// AlignImageSizeTo1MiB rounds down the size to the nearest multiple of 1MiB
// A warning or an error may get logged
// The caller is responsible for ensuring the rounded-down size is not 0
func AlignImageSizeTo1MiB(size int64, logger *log.FilteredLogger) int64 {
	remainder := size % (1024 * 1024)
	if remainder == 0 {
		return size
	} else {
		newSize := size - remainder
		if logger != nil {
			if newSize == 0 {
				logger.Errorf("disks must be at least 1MiB, %d bytes is too small", size)
			} else {
				logger.Warningf("disk size is not 1MiB-aligned. Adjusting from %d down to %d.", size, newSize)
			}
		}
		return newSize
	}

}

func MarkAsNonroot(vmi *v1.VirtualMachineInstance) {
	vmi.Status.RuntimeUser = 107
}

func SetDefaultVolumeDisk(spec *v1.VirtualMachineInstanceSpec) {
	diskAndFilesystemNames := make(map[string]struct{})

	for _, disk := range spec.Domain.Devices.Disks {
		diskAndFilesystemNames[disk.Name] = struct{}{}
	}

	for _, fs := range spec.Domain.Devices.Filesystems {
		diskAndFilesystemNames[fs.Name] = struct{}{}
	}

	for _, volume := range spec.Volumes {
		if _, foundDisk := diskAndFilesystemNames[volume.Name]; !foundDisk {
			spec.Domain.Devices.Disks = append(
				spec.Domain.Devices.Disks,
				v1.Disk{
					Name: volume.Name,
				},
			)
		}
	}
}

func CalcExpectedMemoryDumpSize(vmi *v1.VirtualMachineInstance) *resource.Quantity {
	domain := vmi.Spec.Domain
	vmiMemoryReq := domain.Resources.Requests.Memory()
	expectedPvcSize := resource.NewQuantity(int64(memoryDumpOverhead), vmiMemoryReq.Format)
	expectedPvcSize.Add(*vmiMemoryReq)
	return expectedPvcSize
}

// GenerateRandomString creates a securely generated random string using crypto/rand
func GenerateSecureRandomString(n int) (string, error) {
	ret := make([]byte, n)
	for i := range ret {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(Alphanums))))
		if err != nil {
			return "", err
		}
		ret[i] = Alphanums[num.Int64()]
	}

	return string(ret), nil
}

// GenerateKubeVirtGroupVersionKind ensures a provided object registered with KubeVirts generated schema
// has GVK set correctly. This is required as client-go continues to return objects without
// TypeMeta set as set out in the following issue: https://github.com/kubernetes/client-go/issues/413
func GenerateKubeVirtGroupVersionKind(obj runtime.Object) (runtime.Object, error) {
	objCopy := obj.DeepCopyObject()
	gvks, _, err := generatedscheme.Scheme.ObjectKinds(objCopy)
	if err != nil {
		return nil, fmt.Errorf("could not get GroupVersionKind for object: %w", err)
	}
	objCopy.GetObjectKind().SetGroupVersionKind(gvks[0])

	return objCopy, nil
}
