//go:build linux

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
	"unsafe"

	"golang.org/x/sys/unix"
)

// ProcessMadviseCollapseFunc collapses memory of another process via process_madvise(MADV_COLLAPSE).
// Exported as a variable so tests can replace the syscall with a stub.
var ProcessMadviseCollapseFunc = processMadviseCollapseDefault

// ProcessMadviseCollapse opens a pidfd for pid and calls process_madvise(MADV_COLLAPSE)
// on [addr, addr+length). Returns bytes processed (kernel return value).
func ProcessMadviseCollapse(pid int, addr uintptr, length uint64) (int, error) {
	return ProcessMadviseCollapseFunc(pid, addr, length)
}

func processMadviseCollapseDefault(pid int, addr uintptr, length uint64) (int, error) {
	fd, err := unix.PidfdOpen(pid, 0)
	if err != nil {
		return 0, fmt.Errorf("pidfd_open(%d): %w", pid, err)
	}
	defer unix.Close(fd)

	iov := unix.Iovec{
		Base: (*byte)(unsafe.Pointer(addr)), // #nosec G103 — kernel interprets as remote VA
		Len:  length,
	}
	n, _, errno := unix.RawSyscall6(
		unix.SYS_PROCESS_MADVISE,
		uintptr(fd),
		uintptr(unsafe.Pointer(&iov)), // #nosec G103 — iovec passed to process_madvise
		1,
		uintptr(unix.MADV_COLLAPSE),
		0,
		0,
	)
	if errno != 0 {
		return 0, fmt.Errorf("process_madvise(MADV_COLLAPSE): %v", errno)
	}
	return int(n), nil
}
