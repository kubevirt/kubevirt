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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package virtchroot

import (
	"fmt"
	"os/exec"
	"strconv"
)

func (v VirtCHRoot) AddMacvtapDeviceWithSELinuxLabel(name, lowerDeviceName, mode string, ownerID, pid int) error {
	mountNS := fmt.Sprintf("/proc/%d/ns/mnt", pid)
	cmd := v.addMacvtapDeviceCmd(name, lowerDeviceName, mode, mountNS, ownerID)
	return runWithSELinuxLabelFromPID(pid, cmd)
}

func (v VirtCHRoot) addMacvtapDeviceCmd(name, lowerDeviceName, mode, mountNS string, ownerID int) *exec.Cmd {
	cmdArgs := []string{
		"--mount", mountNS,
		"create-macvtap",
		"--name", name,
		"--lower-device-name", lowerDeviceName,
		"--mode", mode,
		"--uid", strconv.Itoa(ownerID),
		"--gid", strconv.Itoa(ownerID),
	}
	// #nosec No risk for attacket injection. cmdArgs includes predefined strings
	return exec.Command(virtChrootBin, cmdArgs...)
}
