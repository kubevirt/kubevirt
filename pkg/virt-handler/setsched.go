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
 */

//go:build (linux && amd64) || (linux && arm64) || (linux && s390x)

package virthandler

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

// schedParam represents the Linux sched_param struct:
//
//	struct sched_param {
//	   int sched_priority;
//	};
//
// Ref: https://github.com/torvalds/linux/blob/c2bf05db6c78f53ca5cd4b48f3b9b71f78d215f1/include/uapi/linux/sched/types.h#L7-L9
type schedParam struct {
	priority int
}

type policy uint32

const (
	// schedFIFO represents the Linux SCHED_FIFO scheduling policy ID:
	//
	// #define SCHED_FIFO		1
	//
	// Ref: https://github.com/torvalds/linux/blob/c2bf05db6c78f53ca5cd4b48f3b9b71f78d215f1/include/uapi/linux/sched.h#L115
	schedFIFO policy = 1
)

func schedSetScheduler(pid int, policy policy, param schedParam) error {
	_, _, e1 := unix.Syscall(unix.SYS_SCHED_SETSCHEDULER, uintptr(pid), uintptr(policy), uintptr(unsafe.Pointer(&param)))
	if e1 != 0 {
		return e1
	}
	return nil
}
