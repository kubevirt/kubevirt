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
	"runtime"

	"github.com/mitchellh/go-ps"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/hypervisor/common"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type KvmVirtRuntime struct {
	podIsolationDetector isolation.PodIsolationDetector
	logger               *log.FilteredLogger
	KvmHypervisorBackend
}

func NewKvmVirtRuntime(podIsoDetector isolation.PodIsolationDetector, logger *log.FilteredLogger) *KvmVirtRuntime {
	return &KvmVirtRuntime{
		podIsolationDetector: podIsoDetector,
		logger:               logger,
		KvmHypervisorBackend: KvmHypervisorBackend{},
	}
}

func (k *KvmVirtRuntime) AdjustResources(vmi *v1.VirtualMachineInstance, config *v1.KubeVirtConfiguration) error {
	if !util.IsVFIOVMI(vmi) && !vmi.IsRealtimeEnabled() && !util.IsSEVVMI(vmi) && !util.RequiresLockingMemory(vmi) {
		return nil
	}

	isolationResult, err := k.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}

	// If the VMI is running, we adjust the QEMU process
	// otherwise, we adjust the virtqemud process
	var targetProcess ps.Process
	if vmi.IsRunning() {
		targetProcess, err = GetQEMUProcess(isolationResult)
		if err != nil {
			return err
		}
	} else {
		processes, err := ps.Processes()
		if err != nil {
			return fmt.Errorf("failed to get all processes: %v", err)
		}
		targetProcess, err = common.FindVirtqemudProcess(processes, isolationResult.Pid())
		if err != nil {
			return err
		}
		// If the virtqemud process is not found, do nothing
		if targetProcess == nil {
			return nil
		}
	}

	targetProcessID := targetProcess.Pid()

	// make the best estimate for memory required by libvirt
	memlockSize := k.GetMemoryOverhead(vmi, runtime.GOARCH, config.AdditionalGuestMemoryOverheadRatio)
	// Add memory assigned to the VM
	vmiBaseMemory := getVMIBaseMemory(vmi)

	memlockSize.Add(*resource.NewScaledQuantity(vmiBaseMemory.ScaledValue(resource.Kilo), resource.Kilo))

	if err := common.SetProcessMemoryLockRLimit(targetProcessID, memlockSize.Value()); err != nil {
		return fmt.Errorf("failed to set process %d memlock rlimit to %d: %v", targetProcessID, memlockSize.Value(), err)
	}
	log.Log.V(5).Object(vmi).Infof("set process %+v memlock rlimits to: Cur: %[2]d Max:%[2]d",
		targetProcess, memlockSize.Value())

	return nil
}

func getVMIBaseMemory(vmi *v1.VirtualMachineInstance) *resource.Quantity {
	vmiBaseMemory := resource.NewScaledQuantity(0, resource.Kilo)
	switch {
	case vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.MaxGuest != nil:
		vmiBaseMemory = vmi.Spec.Domain.Memory.MaxGuest
	case vmi.Spec.Domain.Resources.Requests.Memory() != nil && !vmi.Spec.Domain.Resources.Requests.Memory().IsZero():
		vmiBaseMemory = vmi.Spec.Domain.Resources.Requests.Memory()
	case vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Guest != nil:
		vmiBaseMemory = vmi.Spec.Domain.Memory.Guest
	}
	return vmiBaseMemory
}

func (k *KvmVirtRuntime) HandleHousekeeping(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager, domain *api.Domain) error {
	return nil
}

var qemuProcessExecutablePrefixes = []string{"qemu-system", "qemu-kvm"}

// GetQEMUProcess encapsulates and exposes the logic to retrieve the QEMU process ID
func GetQEMUProcess(r isolation.IsolationResult) (ps.Process, error) {
	processes, err := ps.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to get all processes: %v", err)
	}
	qemuProcess, err := common.FindIsolatedQemuProcess(qemuProcessExecutablePrefixes, processes, r.PPid())
	if err != nil {
		return nil, err
	}
	return qemuProcess, nil
}
