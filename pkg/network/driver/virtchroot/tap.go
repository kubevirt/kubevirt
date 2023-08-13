/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2022 Red Hat, Inc.
 *
 */

package virtchroot

import (
	"os/exec"
	"strconv"

	"kubevirt.io/kubevirt/pkg/virt-handler/selinux"
)

type VirtCHRoot struct{}

const virtChrootBin = "virt-chroot"

func (v VirtCHRoot) AddTapDevice(name string, mtu int, queues int, ownerID int) error {
	cmd := v.addTapDeviceCmd(name, mtu, queues, ownerID)
	return cmd.Run()
}

func (v VirtCHRoot) AddTapDeviceWithSELinuxLabel(name string, mtu int, queues int, ownerID int, pid int) error {
	cmd := v.addTapDeviceCmd(name, mtu, queues, ownerID)
	return v.runWithSELinuxLabelFromPID(pid, cmd)
}

func (v VirtCHRoot) addTapDeviceCmd(name string, mtu int, queues int, ownerID int) *exec.Cmd {
	id := strconv.Itoa(ownerID)
	cmdArgs := []string{
		"create-tap",
		"--tap-name", name,
		"--uid", id,
		"--gid", id,
		"--queue-number", strconv.Itoa(queues),
		"--mtu", strconv.Itoa(mtu),
	}
	// #nosec No risk for attacket injection. cmdArgs includes predefined strings
	return exec.Command(virtChrootBin, cmdArgs...)
}

func (v VirtCHRoot) runWithSELinuxLabelFromPID(pid int, cmd *exec.Cmd) error {
	ctxExec, err := selinux.NewContextExecutor(pid, cmd)
	if err != nil {
		return err
	}

	return ctxExec.Execute()
}
