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

package virthandler

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	v1 "kubevirt.io/api/core/v1"
)

type maskType bool

type cpuMask struct {
	mask map[string]maskType
}

const (
	enabled  maskType = true
	disabled maskType = false
)

var (
	// parse CPU Mask expressions
	cpuRangeRegex  = regexp.MustCompile(`^(\d+)-(\d+)$`)
	negateCPURegex = regexp.MustCompile(`^\^(\d+)$`)
	singleCPURegex = regexp.MustCompile(`^(\d+)$`)

	// parse thread comm value expression
	vcpuRegex = regexp.MustCompile(`^CPU (\d+)/KVM\n$`) // These threads follow this naming pattern as their command value (/proc/{pid}/task/{taskid}/comm)
	// QEMU uses threads to represent vCPUs.

)

// configureRealTimeVCPUs parses the realtime mask value and configured the selected vcpus
// for real time workloads by setting the scheduler to FIFO and process priority equal to 1.
func (c *VirtualMachineController) configureVCPUScheduler(vmi *v1.VirtualMachineInstance) error {
	res, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}
	qemuProcess, err := res.GetQEMUProcess()
	if err != nil {
		return err
	}
	vcpus, err := getVCPUThreadIDs(qemuProcess.Pid())
	if err != nil {
		return err
	}
	mask, err := parseCPUMask(vmi.Spec.Domain.CPU.Realtime.Mask)
	if err != nil {
		return err
	}
	for vcpuID, threadID := range vcpus {
		if mask.isEnabled(vcpuID) {
			param := schedParam{priority: 1}
			tid, err := strconv.Atoi(threadID)
			if err != nil {
				return err
			}
			err = schedSetScheduler(tid, schedFIFO, param)
			if err != nil {
				return fmt.Errorf("failed to set FIFO scheduling and priority 1 for thread %d: %w", tid, err)
			}
		}
	}
	return nil
}

func isVCPU(comm []byte) (string, bool) {
	if !vcpuRegex.MatchString(string(comm)) {
		return "", false
	}
	v := vcpuRegex.FindSubmatch(comm)
	return string(v[1]), true
}
func getVCPUThreadIDs(pid int) (map[string]string, error) {

	p := filepath.Join(string(os.PathSeparator), "proc", strconv.Itoa(pid), "task")
	d, err := os.ReadDir(p)
	if err != nil {
		return nil, err
	}
	ret := map[string]string{}
	for _, f := range d {
		if f.IsDir() {
			c, err := os.ReadFile(filepath.Join(p, f.Name(), "comm"))
			if err != nil {
				return nil, err
			}
			if v, ok := isVCPU(c); ok {
				ret[v] = f.Name()
			}
		}
	}
	return ret, nil
}

// parseCPUMask parses the mask and maps the results into a structure that contains which
// CPUs are enabled or disabled for the scheduling and priority changes.
// This implementation reimplements the libvirt parsing logic defined here:
// https://github.com/libvirt/libvirt/blob/56de80cb793aa7aedc45572f8b6ec3fc32c99309/src/util/virbitmap.c#L382
// except that in this case it uses a map[string]maskType instead of a bit array.
func parseCPUMask(mask string) (*cpuMask, error) {

	vcpus := cpuMask{}
	if len(mask) == 0 {
		return &vcpus, nil
	}
	vcpus.mask = make(map[string]maskType)

	masks := strings.Split(mask, ",")
	for _, i := range masks {
		m := strings.TrimSpace(i)
		switch {
		case cpuRangeRegex.MatchString(m):
			match := cpuRangeRegex.FindSubmatch([]byte(m))
			startID, err := strconv.Atoi(string(match[1]))
			if err != nil {
				return nil, err
			}
			endID, err := strconv.Atoi(string(match[2]))
			if err != nil {
				return nil, err
			}
			if startID < 0 {
				return nil, fmt.Errorf("invalid vcpu mask start index `%d`", startID)
			}
			if endID < 0 {
				return nil, fmt.Errorf("invalid vcpu mask end index `%d`", endID)
			}
			if startID > endID {
				return nil, fmt.Errorf("invalid mask range `%d-%d`", startID, endID)
			}
			for id := startID; id <= endID; id++ {
				vid := strconv.Itoa(id)
				if !vcpus.has(vid) {
					vcpus.set(vid, enabled)
				}
			}
		case singleCPURegex.MatchString(m):
			match := singleCPURegex.FindSubmatch([]byte(m))
			vid, err := strconv.Atoi(string(match[1]))
			if err != nil {
				return nil, err
			}
			if vid < 0 {
				return nil, fmt.Errorf("invalid vcpu index `%d`", vid)
			}
			if !vcpus.has(string(match[1])) {
				vcpus.set(string(match[1]), enabled)
			}
		case negateCPURegex.MatchString(m):
			match := negateCPURegex.FindSubmatch([]byte(m))
			vid, err := strconv.Atoi(string(match[1]))
			if err != nil {
				return nil, err
			}
			if vid < 0 {
				return nil, fmt.Errorf("invalid vcpu index `%d`", vid)
			}
			vcpus.set(string(match[1]), disabled)
		default:
			return nil, fmt.Errorf("invalid mask value '%s' in '%s'", i, mask)
		}
	}
	return &vcpus, nil
}

func (c cpuMask) isEnabled(vcpuID string) bool {
	if len(c.mask) == 0 {
		return true
	}
	if t, ok := c.mask[vcpuID]; ok {
		return t == enabled
	}
	return false
}

func (c *cpuMask) has(vcpuID string) bool {
	_, ok := c.mask[vcpuID]
	return ok
}

func (c *cpuMask) set(vcpuID string, mtype maskType) {
	c.mask[vcpuID] = mtype
}
