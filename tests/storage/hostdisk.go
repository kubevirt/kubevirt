package storage

import (
	"fmt"
	"path/filepath"

	k8sv1 "k8s.io/api/core/v1"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/libpod"
)

func CreateHostDiskImage(diskPath string) *k8sv1.Pod {
	hostPathType := k8sv1.HostPathDirectoryOrCreate
	dir := filepath.Dir(diskPath)

	command := fmt.Sprintf(`dd if=/dev/zero of=%s bs=1 count=0 seek=1G && ls -l %s`, diskPath, dir)
	if !checks.HasFeature(virtconfig.Root) {
		command = command + fmt.Sprintf(" && chown 107:107 %s", diskPath)
	}

	args := []string{command}
	pod := libpod.RenderHostPathPod("hostdisk-create-job", dir, hostPathType, k8sv1.MountPropagationNone, []string{"/bin/bash", "-c"}, args)

	return pod
}
