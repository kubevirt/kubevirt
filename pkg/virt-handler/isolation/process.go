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
	"fmt"
	"strings"

	"kubevirt.io/client-go/log"

	ps "github.com/shirou/gopsutil/v4/process"
)

type processType interface {
	Pid() int32
	Ppid() (int32, error)
	Name() (string, error)
}

var _ processType = &processImpl{}

type processImpl struct {
	process *ps.Process
}

func (p *processImpl) Pid() int32 {
	return p.process.Pid
}

func (p *processImpl) Ppid() (int32, error) {
	return p.process.Ppid()
}

func (p *processImpl) Name() (string, error) {
	return p.process.Name()
}

func gopsutilProcSliceToInternal(gopsutilProcs []*ps.Process) []processType {
	var ret []processType

	for _, gopsutilProc := range gopsutilProcs {
		ret = append(ret, &processImpl{process: gopsutilProc})
	}

	return ret
}

func internalProcSliceToGopsutil(procs []processType) []*ps.Process {
	var ret []*ps.Process

	for _, proc := range procs {
		ret = append(ret, proc.(*processImpl).process)
	}

	return ret
}

func childProcesses(processes []*ps.Process, pid int) []*ps.Process {
	childProcs := childProcessesAux(gopsutilProcSliceToInternal(processes), pid)
	return internalProcSliceToGopsutil(childProcs)
}

// childProcesses given a list of processes, it returns the ones that are children
// of the given PID.
func childProcessesAux(processes []processType, pid int) []processType {
	var childProcesses []processType

	for _, process := range processes {
		processPpid, err := process.Ppid()
		if err != nil {
			processName, _ := process.Name()
			log.Log.V(5).Reason(err).Infof("cannot find parent PID for process %s with PID %d", processName, process.Pid())
			continue
		}

		if int(processPpid) == pid {
			childProcesses = append(childProcesses, process)
		}
	}

	return childProcesses
}

func lookupProcessByExecutablePrefix(processes []*ps.Process, execPrefix string) *ps.Process {
	proc := lookupProcessByExecutablePrefixAux(gopsutilProcSliceToInternal(processes), execPrefix)
	return proc.(*processImpl).process
}

// lookupProcessByExecutablePrefix given list of processes, it return the first occurrence
// of a process with the given executable prefix.
func lookupProcessByExecutablePrefixAux(processes []processType, execPrefix string) processType {
	if execPrefix == "" {
		return nil
	}
	for _, process := range processes {
		processName, err := process.Name()
		if err != nil {
			log.Log.V(5).Reason(err).Infof("cannot find process name for process with PID %d", process.Pid())
			continue
		}

		if strings.HasPrefix(processName, execPrefix) {
			return process
		}
	}

	return nil
}

var qemuProcessExecutablePrefixes = []string{"qemu-system", "qemu-kvm"}

func findIsolatedQemuProcess(processes []*ps.Process, pid int) (*ps.Process, error) {
	proc, err := findIsolatedQemuProcessAux(gopsutilProcSliceToInternal(processes), pid)
	return proc.(*processImpl).process, err
}

// findIsolatedQemuProcess Returns the first occurrence of the QEMU process whose parent is PID"
func findIsolatedQemuProcessAux(processes []processType, pid int) (processType, error) {
	processes = childProcessesAux(processes, pid)
	for _, execPrefix := range qemuProcessExecutablePrefixes {
		if qemuProcess := lookupProcessByExecutablePrefixAux(processes, execPrefix); qemuProcess != nil {
			return qemuProcess, nil
		}
	}

	return nil, fmt.Errorf("no QEMU process found under process %d child processes", pid)
}
