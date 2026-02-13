package util

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
)

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

var miscCapacityPath = filepath.Join(HostRootMount, "/sys/fs/cgroup/misc.capacity")

// GetMiscCapacity reads /sys/fs/cgroup/misc.capacity to return a map where keys
// are the resource type names and values are their respective capacity limits.
// Note SEV-SNP and SEV-ES share the same capacity pool, e.g. "sev_es 99"
func GetMiscCapacity() (map[string]int, error) {
	caps := make(map[string]int)

	content, err := os.ReadFile(miscCapacityPath)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}

		capacityKey := fields[0]
		capacity, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, err
		}
		caps[capacityKey] = capacity
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return caps, nil
}
