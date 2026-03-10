package util

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/api/core/v1"
	vmipredicates "kubevirt.io/api/core/v1/predicates"
	generatedscheme "kubevirt.io/client-go/kubevirt/scheme"
	"kubevirt.io/client-go/log"
)

const (
	ExtensionAPIServerAuthenticationConfigMap = "extension-apiserver-authentication"
	RequestHeaderClientCAFileKey              = "requestheader-client-ca-file"
	VirtShareDir                              = "/var/run/kubevirt"
	VirtImageVolumeDir                        = "/var/run/kubevirt-image-volume"
	VirtKernelBootVolumeDir                   = "/var/run/kubevirt-kernel-boot"
	VirtPrivateDir                            = "/var/run/kubevirt-private"
	KubeletRoot                               = "/var/lib/kubelet"
	KubeletPodsDir                            = KubeletRoot + "/pods"
	HostRootMount                             = "/proc/1/root/"
	ContainerBinary                           = "/container-disk-binary"

	NonRootUID        = 107
	NonRootUserString = "qemu"
	RootUser          = 0

	// extensive log verbosity threshold after which libvirt debug logs will be enabled
	EXT_LOG_VERBOSITY_THRESHOLD         = 5
	ENV_VAR_SHARED_FILESYSTEM_PATHS     = "SHARED_FILESYSTEM_PATHS"
	ENV_VAR_LIBVIRT_DEBUG_LOGS          = "LIBVIRT_DEBUG_LOGS"
	ENV_VAR_VIRTIOFSD_DEBUG_LOGS        = "VIRTIOFSD_DEBUG_LOGS"
	ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY = "VIRT_LAUNCHER_LOG_VERBOSITY"
)

func ResourceNameToEnvVar(prefix string, resourceName string) string {
	varName := strings.ToUpper(resourceName)
	varName = strings.Replace(varName, "/", "_", -1)
	varName = strings.Replace(varName, ".", "_", -1)
	return fmt.Sprintf("%s_%s", prefix, varName)
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
				logger.V(4).Infof("disk size is not 1MiB-aligned. Adjusting from %d down to %d.", size, newSize)
			}
		}
		return newSize
	}

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
	const memoryDumpOverhead = 100 * 1024 * 1024
	domain := vmi.Spec.Domain
	vmiMemoryReq := domain.Resources.Requests.Memory()
	expectedPvcSize := resource.NewQuantity(int64(memoryDumpOverhead), vmiMemoryReq.Format)
	expectedPvcSize.Add(*vmiMemoryReq)
	return expectedPvcSize
}

// GenerateVMExportToken creates a cryptographically secure token for VM export
func GenerateVMExportToken() (string, error) {
	const alphanums = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	const tokenLen = 20
	ret := make([]byte, tokenLen)
	for i := range ret {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphanums))))
		if err != nil {
			return "", err
		}
		ret[i] = alphanums[num.Int64()]
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

func PathForSwtpm(vmi *v1.VirtualMachineInstance) string {
	swtpmPath := "/var/lib/libvirt/swtpm"
	if vmipredicates.IsNonRootVMI(vmi) {
		swtpmPath = filepath.Join(VirtPrivateDir, "libvirt", "qemu", "swtpm")
	}

	return swtpmPath
}

func PathForSwtpmLocalca(vmi *v1.VirtualMachineInstance) string {
	localCaPath := "/var/lib/swtpm-localca"
	if vmipredicates.IsNonRootVMI(vmi) {
		localCaPath = filepath.Join(VirtPrivateDir, "var", "lib", "swtpm-localca")
	}

	return localCaPath
}

func PathForNVram(vmi *v1.VirtualMachineInstance) string {
	nvramPath := "/var/lib/libvirt/qemu/nvram"
	if vmipredicates.IsNonRootVMI(vmi) {
		nvramPath = filepath.Join(VirtPrivateDir, "libvirt", "qemu", "nvram")
	}

	return nvramPath
}
