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

package common

import (
	"fmt"

	"github.com/mitchellh/go-ps"
)

// FindIsolatedQemuProcess Returns the first occurrence of the QEMU process whose parent is PID"
func FindIsolatedQemuProcess(qemuProcessExecutablePrefixes []string, processes []ps.Process, pid int) (ps.Process, error) {
	processes = childProcesses(processes, pid)
	for _, execPrefix := range qemuProcessExecutablePrefixes {
		if qemuProcess := lookupProcessByExecutablePrefix(processes, execPrefix); qemuProcess != nil {
			return qemuProcess, nil
		}
	}

	return nil, fmt.Errorf("no QEMU process found under process %d child processes", pid)
}

func FindVirtqemudProcess(processes []ps.Process, launcherPid int) (ps.Process, error) {
	for _, process := range processes {
		// consider all processes that are virt-launcher children
		if process.PPid() != launcherPid {
			continue
		}

		// virtqemud process sets the memory lock limit before fork/exec-ing into qemu
		if process.Executable() != "virtqemud" {
			continue
		}

		return process, nil
	}

	return nil, nil
}
