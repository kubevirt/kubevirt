/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package selinux

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

import (
	"fmt"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/opencontainers/selinux/go-selinux"
)

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
