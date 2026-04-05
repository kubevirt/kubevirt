/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package virtchroot

import (
	"os/exec"
	"strconv"

	"kubevirt.io/kubevirt/pkg/virt-handler/selinux"
)

type VirtCHRoot struct{}

const virtChrootBin = "virt-chroot"

func (v VirtCHRoot) AddTapDevice(name string, mtu, queues, ownerID int) error {
	cmd := v.addTapDeviceCmd(name, mtu, queues, ownerID)
	return cmd.Run()
}

func (v VirtCHRoot) AddTapDeviceWithSELinuxLabel(name string, mtu, queues, ownerID, pid int) error {
	cmd := v.addTapDeviceCmd(name, mtu, queues, ownerID)
	return v.runWithSELinuxLabelFromPID(pid, cmd)
}

func (v VirtCHRoot) addTapDeviceCmd(name string, mtu, queues, ownerID int) *exec.Cmd {
	id := strconv.Itoa(ownerID)
	cmdArgs := []string{
		"create-tap",
		"--tap-name", name,
		"--uid", id,
		"--gid", id,
		"--queue-number", strconv.Itoa(queues),
		"--mtu", strconv.Itoa(mtu),
	}
	// #nosec No risk for attacker injection. cmdArgs includes predefined strings
	return exec.Command(virtChrootBin, cmdArgs...)
}

func (v VirtCHRoot) runWithSELinuxLabelFromPID(pid int, cmd *exec.Cmd) error {
	ctxExec, err := selinux.NewContextExecutor(pid, cmd)
	if err != nil {
		return err
	}

	return ctxExec.Execute()
}
