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
 * Copyright The KubeVirt Authors.
 *
 */

package selinux

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/opencontainers/selinux/go-selinux"
)

// SELinux capability check to avoid false positives on partially-enabled systems.
// Some custom kernels may mount selinuxfs while the SELinux LSM is not actually
// enabled in the kernel cmdline, so reading /proc/<pid>/attr/current returns
// EOPNOTSUPP. A successful read of our own process label is treated as proof that
// SELinux is functional, consistent with ContextExecutor.isSELinuxEnabled.
func IsSELinuxFunctional() bool {
	_, err := selinux.FileLabel(fmt.Sprintf("/proc/%d/attr/current", os.Getpid()))
	return err == nil
}

type Executor interface {
	NewSELinux() (SELinux, bool, error)
	FileLabel(filepath string) (string, error)
	SetExecLabel(label string) error
	LockOSThread()
	UnlockOSThread()
	CloseOnExec(fd int)
	Run(cmd *exec.Cmd) error
}

type SELinuxExecutor struct {
}

func (se SELinuxExecutor) NewSELinux() (SELinux, bool, error) {
	return NewSELinux()
}

func (se SELinuxExecutor) FileLabel(filepath string) (string, error) {
	return selinux.FileLabel(filepath)
}

func (se SELinuxExecutor) SetExecLabel(label string) error {
	return selinux.SetExecLabel(label)
}

func (se SELinuxExecutor) LockOSThread() {
	runtime.LockOSThread()
}

func (se SELinuxExecutor) UnlockOSThread() {
	runtime.UnlockOSThread()
}

func (se SELinuxExecutor) CloseOnExec(fd int) {
	syscall.CloseOnExec(fd)
}

func (se SELinuxExecutor) Run(cmd *exec.Cmd) error {
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed running command %s, err: %v, output: %s", cmd.String(), err, output)
	}
	return nil
}
