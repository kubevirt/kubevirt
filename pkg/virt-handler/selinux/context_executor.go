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

import (
	"fmt"
	"os"
	"os/exec"

	"kubevirt.io/client-go/log"
)

const (
	minFDToCloseOnExec = 3
	maxFDToCloseOnExec = 256
)

type ContextExecutor struct {
	cmdToExecute  *exec.Cmd
	desiredLabel  string
	originalLabel string
	pid           int
	executor      Executor
}

func NewContextExecutor(pid int, cmd *exec.Cmd) (*ContextExecutor, error) {
	return newContextExecutor(pid, cmd, SELinuxExecutor{})
}

func newContextExecutor(pid int, cmd *exec.Cmd, executor Executor) (*ContextExecutor, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("pid must be positive")
	}

	ce := &ContextExecutor{
		pid:          pid,
		cmdToExecute: cmd,
		executor:     executor,
	}

	if ce.isSELinuxEnabled() {
		desiredLabel, err := ce.getLabelForPID(pid)
		if err != nil {
			return nil, err
		}
		originalLabel, err := ce.getLabelForPID(os.Getpid())
		if err != nil {
			return nil, err
		}
		ce.desiredLabel = desiredLabel
		ce.originalLabel = originalLabel
	}

	return ce, nil
}

func (ce *ContextExecutor) Execute() error {
	log.Log.Infof("[ContextExecutor]: Executing... Switching from original (%s) to desired (%s) context",
		ce.originalLabel, ce.desiredLabel)

	if ce.isSELinuxEnabled() {
		if err := ce.setDesiredContext(); err != nil {
			return err
		}
		defer ce.resetContext()
	}

	ce.preventFDLeakOntoChild()
	if err := ce.executor.Run(ce.cmdToExecute); err != nil {
		return fmt.Errorf("failed to execute command in launcher namespace %d: %v", ce.pid, err)
	}

	log.Log.Infof("[ContextExecutor]: Execution ended successfully")
	return nil
}

func (ce *ContextExecutor) setDesiredContext() error {
	ce.executor.LockOSThread()
	if err := ce.executor.SetExecLabel(ce.desiredLabel); err != nil {
		return fmt.Errorf("failed to switch selinux context to %s. Reason: %v", ce.desiredLabel, err)
	}
	return nil
}

func (ce *ContextExecutor) resetContext() error {
	defer ce.executor.UnlockOSThread()
	return ce.executor.SetExecLabel(ce.originalLabel)
}

func (ce *ContextExecutor) isSELinuxEnabled() bool {
	_, selinuxEnabled, err := ce.executor.NewSELinux()
	return err == nil && selinuxEnabled
}

func (ce *ContextExecutor) getLabelForPID(pid int) (string, error) {
	fileLabel, err := ce.executor.FileLabel(fmt.Sprintf("/proc/%d/attr/current", pid))
	if err != nil {
		return "", fmt.Errorf("could not retrieve pid %d selinux label: %v", pid, err)
	}
	return fileLabel, nil
}

func (ce *ContextExecutor) preventFDLeakOntoChild() {
	// we want to share the parent process std{in|out|err} - fds 0 through 2.
	// Since the FDs are inherited on fork / exec, we close on exec all others.
	for fd := minFDToCloseOnExec; fd < maxFDToCloseOnExec; fd++ {
		ce.executor.CloseOnExec(fd)
	}
}
