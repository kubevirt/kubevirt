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

package isolation

import (
	"strings"

	"github.com/mitchellh/go-ps"
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
