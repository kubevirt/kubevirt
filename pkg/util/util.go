package util

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

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

var (
	miscCapacityPath = filepath.Join(HostRootMount, "sys/fs/cgroup/misc.capacity")
	miscMaxPath      = filepath.Join(HostRootMount, "sys/fs/cgroup/misc.max")
)

// GetMiscCapacity reads the misc cgroup controller to return a map where keys
// are the resource type names and values are their respective capacity limits.
// Note SEV-SNP and SEV-ES share the same capacity pool, e.g. "sev_es 99"
//
// The kernel never exposes both files at one cgroup: misc.capacity is
// CFTYPE_ONLY_ON_ROOT, misc.max is CFTYPE_NOT_ON_ROOT. A node owning the
// machine reads the real capacity, while a node that is itself a container,
// like a KinD node, has only the limit on its own cgroup to go by. There a key
// being present is the whole signal: the limit is that node's share, not a
// count of how many guests the machine can run, so report one.
func GetMiscCapacity() (map[string]int, error) {
	const defaultLimit = 1

	content, err := os.ReadFile(miscCapacityPath)
	parseValue := strconv.Atoi
	if errors.Is(err, fs.ErrNotExist) {
		content, err = os.ReadFile(miscMaxPath)
		parseValue = func(string) (int, error) { return defaultLimit, nil }
	}
	if err != nil {
		return nil, err
	}

	caps := make(map[string]int)
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}

		capacity, err := parseValue(fields[1])
		if err != nil {
			log.Log.V(4).Infof("Skipping malformed misc.capacity line: %q, err: %v", line, err)
			continue
		}
		caps[fields[0]] = capacity
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return caps, nil
}
