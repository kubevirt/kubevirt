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

package isolation

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/mitchellh/go-ps"
	"golang.org/x/sys/unix"
)

// childProcesses given a list of processes, it returns the ones that are children
// of the given PID.
func childProcesses(processes []ps.Process, pid int) []ps.Process {
	var childProcesses []ps.Process
	for _, process := range processes {
		if process.PPid() == pid {
			childProcesses = append(childProcesses, process)
		}
	}

	return childProcesses
}

// lookupProcessByExecutablePrefix given list of processes, it return the first occurrence
// of a process with the given executable prefix.
func lookupProcessByExecutablePrefix(processes []ps.Process, execPrefix string) ps.Process {
	if execPrefix == "" {
		return nil
	}
	for _, process := range processes {
		if strings.HasPrefix(process.Executable(), execPrefix) {
			return process
		}
	}

	return nil
}

// SetProcessMemoryLockRLimit Adjusts process MEMLOCK
// soft-limit (current) and hard-limit (max) to the given size.
func SetProcessMemoryLockRLimit(pid int, size int64) error {
	// standard golang libraries don't provide API to set runtime limits
	// for other processes, so we have to directly call to kernel
	rlimit := unix.Rlimit{
		Cur: uint64(size),
		Max: uint64(size),
	}
	_, _, errno := unix.RawSyscall6(unix.SYS_PRLIMIT64,
		uintptr(pid),
		uintptr(unix.RLIMIT_MEMLOCK),
		uintptr(unsafe.Pointer(&rlimit)), // #nosec used in unix RawSyscall6
		0, 0, 0)
	if errno != 0 {
		return fmt.Errorf("error setting prlimit: %v", errno)
	}

	return nil
}

func getPPid(pid int) (int, error) {
	process, err := ps.FindProcess(pid)
	if err != nil {
		return -1, err
	}
	if process == nil {
		return -1, fmt.Errorf("failed to find process with pid: %d", pid)
	}
	return process.PPid(), nil
}
