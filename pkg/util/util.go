package util

import (
	"fmt"
	"path/filepath"
	"strings"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/vmitrait"
)

const (
	VirtShareDir            = "/var/run/kubevirt"
	VirtImageVolumeDir      = "/var/run/kubevirt-image-volume"
	VirtKernelBootVolumeDir = "/var/run/kubevirt-kernel-boot"
	VirtPrivateDir          = "/var/run/kubevirt-private"
	KubeletRoot             = "/var/lib/kubelet"
	KubeletPodsDir          = KubeletRoot + "/pods"
	HostRootMount           = "/proc/1/root/"

	NonRootUID = 107
	RootUser   = 0

	// extensive log verbosity threshold after which libvirt debug logs will be enabled
	EXT_LOG_VERBOSITY_THRESHOLD         = 5
	ENV_VAR_SHARED_FILESYSTEM_PATHS     = "SHARED_FILESYSTEM_PATHS"
	ENV_VAR_LIBVIRT_DEBUG_LOGS          = "LIBVIRT_DEBUG_LOGS"
	ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY = "VIRT_LAUNCHER_LOG_VERBOSITY"

	// VMStatePVCMountPath is where VMState PVC is mounted
	VMStatePVCMountPath = VirtPrivateDir + "/vm-state"

	// Canonical subdirectories/files inside the VirtualMachineState PVC.
	VMStateDirTPM          = "tpm"
	VMStateDirEFI          = "efi"
	VMStateDirCBT          = "cbt"
	VMStateDirMeta         = "meta"
	VMStateDirSwtpmLocalca = "swtpm-localca"
	VMStateEFIVarsFile     = "efi_vars.fd"

	// Old layout directory names, migrated to the canonical layout on first boot.
	VMStateDirNVRAMLegacy = "nvram" // holds <vmname>_VARS.fd, moved to efi/efi_vars.fd
	VMStateDirSwtpmLegacy = "swtpm" // holds <uuid>/tpm2, moved to tpm/

	// VMStateFileLayout is the file inside meta/ recording the PVC's canonical layout version.
	VMStateFileLayout = "layout"

	// VMStateLayoutVersion is the current layout version; bump it when the canonical layout changes.
	VMStateLayoutVersion = 1

	// VMStateFileCBTOwner records the UID of cbt/'s last owner, to detect adoption by another VM.
	VMStateFileCBTOwner = "cbt-owner"
)

func HasDeclarativeVMState(vmi *v1.VirtualMachineInstance) bool {
	state := vmi.Spec.VirtualMachineState
	return state != nil && (state.VolumeClaimTemplate != nil || state.Source != nil)
}

func VMStateCanonicalEFIVarsPath() string {
	return filepath.Join(VMStatePVCMountPath, VMStateDirEFI, VMStateEFIVarsFile)
}

func UseLaunchSecurity(vmi *v1.VirtualMachineInstance) bool {
	return IsSEVVMI(vmi) || IsSecureExecutionVMI(vmi) || IsTDXVMI(vmi)
}

func ResourceNameToEnvVar(prefix string, resourceName string) string {
	varName := strings.ToUpper(resourceName)
	varName = strings.ReplaceAll(varName, "/", "_")
	varName = strings.ReplaceAll(varName, ".", "_")
	return fmt.Sprintf("%s_%s", prefix, varName)
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

func PathForSwtpmLocalca(vmi *v1.VirtualMachineInstance) string {
	localCaPath := "/var/lib/swtpm-localca"
	if vmitrait.IsNonRoot(vmi) {
		localCaPath = filepath.Join(VirtPrivateDir, "var", "lib", "swtpm-localca")
	}
	return localCaPath
}
