//go:build !linux

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

import "fmt"

var ProcessMadviseCollapseFunc = processMadviseCollapseDefault

func ProcessMadviseCollapse(pid int, addr uintptr, length uint64) (int, error) {
	return ProcessMadviseCollapseFunc(pid, addr, length)
}

func processMadviseCollapseDefault(pid int, addr uintptr, length uint64) (int, error) {
	return 0, fmt.Errorf("process_madvise(MADV_COLLAPSE) is unsupported on this platform")
}
