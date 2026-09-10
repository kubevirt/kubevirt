package util

import (
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/vmitrait"
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

	NonRootUID = 107
	RootUser   = 0

	// EXT_LOG_VERBOSITY_THRESHOLD is the log verbosity level above which
	// extended libvirt debug logging is enabled in virt-launcher.
	EXT_LOG_VERBOSITY_THRESHOLD = 5
)

func UseLaunchSecurity(vmi *v1.VirtualMachineInstance) bool {
	return IsSEVVMI(vmi) || IsSecureExecutionVMI(vmi) || IsTDXVMI(vmi)
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


func PathForSwtpm(vmi *v1.VirtualMachineInstance) string {
	swtpmPath := "/var/lib/libvirt/swtpm"
	if vmitrait.IsNonRoot(vmi) {
		swtpmPath = filepath.Join(VirtPrivateDir, "libvirt", "qemu", "swtpm")
	}

	return swtpmPath
}

func PathForNVram(vmi *v1.VirtualMachineInstance) string {
	nvramPath := "/var/lib/libvirt/qemu/nvram"
	if vmitrait.IsNonRoot(vmi) {
		nvramPath = filepath.Join(VirtPrivateDir, "libvirt", "qemu", "nvram")
	}

	return nvramPath
}
