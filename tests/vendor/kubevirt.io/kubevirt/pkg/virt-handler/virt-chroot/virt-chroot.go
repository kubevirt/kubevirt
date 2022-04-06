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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package virt_chroot

import "os/exec"

const (
	binaryPath     = "/usr/bin/virt-chroot"
	mountNamespace = "/proc/1/ns/mnt"
)

func getBaseArgs() []string {
	return []string{"--mount", mountNamespace}
}

func GetChrootBinaryPath() string {
	return binaryPath
}

func GetChrootMountNamespace() string {
	return mountNamespace
}

func MountChroot(sourcePath, targetPath string, ro bool) *exec.Cmd {
	args := append(getBaseArgs(), "mount", "-o")
	optionArgs := "bind"

	if ro {
		optionArgs = "ro," + optionArgs
	}

	args = append(args, optionArgs, sourcePath, targetPath)
	return exec.Command(binaryPath, args...)
}

func UmountChroot(path string) *exec.Cmd {
	args := append(getBaseArgs(), "umount", path)
	return exec.Command(binaryPath, args...)
}

func CreateMDEVType(mdevType string, parentID string, uuid string) *exec.Cmd {
	args := append(getBaseArgs(), "create-mdev")
	args = append(args, "--type", mdevType, "--parent", parentID, "--uuid", uuid)
	return exec.Command(binaryPath, args...)
}

func RemoveMDEVType(mdevUUID string) *exec.Cmd {
	args := append(getBaseArgs(), "remove-mdev")
	args = append(args, "--uuid", mdevUUID)
	return exec.Command(binaryPath, args...)
}

// For general purposes
func ExecChroot(args ...string) *exec.Cmd {
	return exec.Command(binaryPath, args...)
}
