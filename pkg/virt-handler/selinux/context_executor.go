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

package selinux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

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

func NewContextExecutorFromPid(cmd *exec.Cmd, pid int, dismissMCS bool) (*ContextExecutor, error) {
	const emptyLabel = ""
	return newContextExecutor(cmd, pid, emptyLabel, SELinuxExecutor{}, dismissMCS)
}

//
func NewContextExecutorWithType(cmd *exec.Cmd, pid int, seLinuxType string) (*ContextExecutor, error) {
	return newContextExecutor(cmd, pid, seLinuxType, SELinuxExecutor{}, false)
}

func newContextExecutor(cmd *exec.Cmd, pid int, desiredType string, executor Executor, dismissMCS bool) (*ContextExecutor, error) {
	if pid == -1 && desiredType == "" {
		return nil, fmt.Errorf("either pid or label arguments must not be empty")
	}

	ce := &ContextExecutor{
		pid:          pid,
		cmdToExecute: cmd,
		executor:     executor,
	}

	if ce.isSELinuxEnabled() {
		originalLabel, err := ce.getLabelForPID(os.Getpid(), dismissMCS)
		if err != nil {
			return nil, err
		}

		if desiredType == "" {
			if desiredType, err = ce.getLabelForPID(pid, dismissMCS); err != nil {
				return nil, err
			}
		} else {
			const labelSeparator = ":"
			const labelTypeIdx = 2
			splittedCurrentLabel := strings.Split(originalLabel, labelSeparator)
			splittedCurrentLabel[labelTypeIdx] = desiredType
			desiredType = strings.Join(splittedCurrentLabel, labelSeparator)
		}

		ce.desiredLabel = desiredType
		ce.originalLabel = originalLabel
		log.Log.Infof("hotplug [newContextExecutor] setting original (%s) & desired (%s) labels", originalLabel, desiredType)
	}

	return ce, nil
}

func (ce *ContextExecutor) Execute() error {
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

func (ce *ContextExecutor) getLabelForPID(pid int, dismissMCS bool) (string, error) {
	fileLabel, err := ce.executor.FileLabel(fmt.Sprintf("/proc/%d/attr/current", pid))
	if err != nil {
		return "", fmt.Errorf("could not retrieve pid %d selinux label: %v", pid, err)
	}

	const labelSeparator = ":"
	if dismissMCS && strings.Count(fileLabel, labelSeparator) > 3 {

		splittedCurrentLabel := strings.Split(fileLabel, labelSeparator)
		fileLabel = strings.Join(splittedCurrentLabel[:len(splittedCurrentLabel)-1], labelSeparator)

		log.Log.Infof("hotplug [getLabelForPID] NEW LABEL: %s", fileLabel)
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
