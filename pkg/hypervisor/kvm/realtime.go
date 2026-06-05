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

package kvm

import (
	"fmt"
	"regexp"
	"strconv"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hypervisor/common"
)

var (
	// parse thread comm value expression
	VcpuRegex = regexp.MustCompile(`^CPU (\d+)/KVM\n$`) // These threads follow this naming pattern as their command value (/proc/{pid}/task/{taskid}/comm)
	// QEMU uses threads to represent vCPUs.

)

// configureRealTimeVCPUs parses the realtime mask value and configured the selected vcpus
// for real time workloads by setting the scheduler to FIFO and process priority equal to 1.
func (c *KvmVirtRuntime) configureVCPUScheduler(vmi *v1.VirtualMachineInstance) error {
	res, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}
	qemuProcess, err := GetQEMUProcess(res)
	if err != nil {
		return err
	}
	vcpus, err := common.GetVCPUThreadIDs(qemuProcess.Pid(), VcpuRegex)
	if err != nil {
		return err
	}
	mask, err := common.ParseCPUMask(vmi.Spec.Domain.CPU.Realtime.Mask)
	if err != nil {
		return err
	}
	for vcpuID, threadID := range vcpus {
		if mask.IsEnabled(vcpuID) {
			param := common.SchedParam{Priority: 1}
			tid, err := strconv.Atoi(threadID)
			if err != nil {
				return err
			}
			err = common.SchedSetScheduler(tid, common.SchedFIFO, param)
			if err != nil {
				return fmt.Errorf("failed to set FIFO scheduling and priority 1 for thread %d: %w", tid, err)
			}
		}
	}
	return nil
}
