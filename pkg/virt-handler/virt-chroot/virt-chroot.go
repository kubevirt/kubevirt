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

import (
	"os/exec"
	"strings"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/unsafepath"
)

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

func MountChroot(sourcePath, targetPath *safepath.Path, ro bool) *exec.Cmd {
	return UnsafeMountChroot(trimProcPrefix(sourcePath), trimProcPrefix(targetPath), ro)
}

// MountWithMountNamespaceAndRawPath mounts the device using the device's raw
// path instead of its file descriptor. It executes virt-chroot mount in the
// given namespace.
func MountWithMountNamespaceAndRawPath(namespace, source, target, fstype string, options ...string) *exec.Cmd {
	args := []string{"--mount", namespace, "mount", "--type", fstype, "--raw-path"}
	optionArg := strings.Join(options, ",")
	if len(optionArg) > 0 {
		args = append(args, "--options", optionArg)
	}
	args = append(args, source, target)
	return exec.Command(binaryPath, args...)
}

// Deprecated: UnsafeMountChroot is used to connect to code which needs to be refactored
// to handle mounts securely.
func UnsafeMountChroot(sourcePath, targetPath string, ro bool) *exec.Cmd {
	args := append(getBaseArgs(), "mount", "-o")
	optionArgs := "bind"

	if ro {
		optionArgs = "ro," + optionArgs
	}

	args = append(args, optionArgs, sourcePath, targetPath)
	return exec.Command(binaryPath, args...)
}

func UmountChroot(path *safepath.Path) *exec.Cmd {
	return UnsafeUmountChroot(trimProcPrefix(path))
}

// Deprecated: UnsafeUmountChroot is used to connect to code which needs to be refactored
// to handle mounts securely.
func UnsafeUmountChroot(path string) *exec.Cmd {
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

// ExecWithMountNamespace execute virt-chroot exec in the given mount namespace.
func ExecWithMountNamespace(namespace string, args ...string) *exec.Cmd {
	allArgs := []string{"--mount", namespace, "exec", "--"}
	allArgs = append(allArgs, args...)
	return exec.Command(binaryPath, allArgs...)
}

func trimProcPrefix(path *safepath.Path) string {
	return strings.TrimPrefix(unsafepath.UnsafeAbsolute(path.Raw()), "/proc/1/root")
}
