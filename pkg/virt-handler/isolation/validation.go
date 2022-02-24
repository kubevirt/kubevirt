package isolation

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"
	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"

	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
)

const (
	QEMUIMGPath = "/usr/bin/qemu-img"
)

func GetImageInfo(imagePath string, context IsolationResult, config *v1.DiskVerification) (*containerdisk.DiskInfo, error) {
	memoryLimit := fmt.Sprintf("%d", config.MemoryLimit.Value())

	// #nosec g204 no risk to use MountNamespace()  argument as it returns a fixed string of "/proc/<pid>/ns/mnt"
	out, err := virt_chroot.ExecChroot(
		"--user", "qemu", "--memory", memoryLimit, "--cpu", "10", "--mount", context.MountNamespace(), "exec", "--",
		QEMUIMGPath, "info", imagePath, "--output", "json",
	).Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			if len(e.Stderr) > 0 {
				return nil, fmt.Errorf("failed to invoke qemu-img: %v: '%v'", err, string(e.Stderr))
			}
		}
		return nil, fmt.Errorf("failed to invoke qemu-img: %v", err)
	}

	info := &containerdisk.DiskInfo{}
	err = json.Unmarshal(out, info)
	if err != nil {
		return nil, fmt.Errorf("failed to parse disk info: %v", err)
	}
	return info, err
}

func GetFileSize(imagePath string, context IsolationResult) (int64, error) {
	path := filepath.Join(context.MountRoot(), imagePath)
	info, err := os.Stat(path)
	if err != nil {
		return -1, err
	}
	return info.Size(), nil
}
