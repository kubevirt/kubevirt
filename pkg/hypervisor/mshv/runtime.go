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

package mshv

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/mitchellh/go-ps"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/hypervisor/common"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type MshvVirtRuntime struct {
	podIsolationDetector isolation.PodIsolationDetector
	logger               *log.FilteredLogger
	MshvHypervisorBackend
}

func NewMshvVirtRuntime(podIsoDetector isolation.PodIsolationDetector, logger *log.FilteredLogger) *MshvVirtRuntime {
	return &MshvVirtRuntime{
		podIsolationDetector: podIsoDetector,
		logger:               logger,
	}
}

func (m *MshvVirtRuntime) AdjustResources(vmi *v1.VirtualMachineInstance, config *v1.KubeVirtConfiguration) error {
	if !util.IsVFIOVMI(vmi) && !vmi.IsRealtimeEnabled() && !util.IsSEVVMI(vmi) {
		return nil
	}

	isolationResult, err := m.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}

	var targetProcess ps.Process
	if vmi.IsRunning() {
		targetProcess, err = getQEMUProcess(isolationResult)
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

	qemuProcessID := targetProcess.Pid()

	// make the best estimate for memory required by libvirt
	memlockSize := m.GetMemoryOverhead(vmi, runtime.GOARCH, config.AdditionalGuestMemoryOverheadRatio)
	// Add max memory assigned to the VM
	var vmiBaseMemory *resource.Quantity

	switch {
	case vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.MaxGuest != nil:
		vmiBaseMemory = vmi.Spec.Domain.Memory.MaxGuest
	case vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Guest != nil:
		vmiBaseMemory = vmi.Spec.Domain.Memory.Guest
	case vmi.Spec.Domain.Resources.Requests.Memory() != nil:
		vmiBaseMemory = vmi.Spec.Domain.Resources.Requests.Memory()
	case vmi.Spec.Domain.Memory != nil:
		vmiBaseMemory = vmi.Spec.Domain.Memory.Guest
	}

	memlockSize.Add(*resource.NewScaledQuantity(vmiBaseMemory.ScaledValue(resource.Kilo), resource.Kilo))

	if err := common.SetProcessMemoryLockRLimit(qemuProcessID, memlockSize.Value()); err != nil {
		return fmt.Errorf("failed to set process %d memlock rlimit to %d: %v", qemuProcessID, memlockSize.Value(), err)
	}
	log.Log.V(5).Object(vmi).Infof("set process %+v memlock rlimits to: Cur: %[2]d Max:%[2]d",
		targetProcess, memlockSize.Value())

	return nil
}

func (m *MshvVirtRuntime) HandleHousekeeping(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager, domain *api.Domain) error {
	if vmi.IsCPUDedicated() && vmi.Spec.Domain.CPU.IsolateEmulatorThread {
		err := m.configureHousekeepingCgroup(vmi, cgroupManager, domain)
		if err != nil {
			return err
		}
	}

	// Configure vcpu scheduler for realtime workloads and affine PIT thread for dedicated CPU
	if vmi.IsRealtimeEnabled() && !vmi.IsRunning() && !vmi.IsFinal() {
		m.logger.Object(vmi).Info("Configuring vcpus for real time workloads")
		if err := m.configureVCPUScheduler(vmi); err != nil {
			return err
		}
	}
	return nil
}

func (m *MshvVirtRuntime) configureHousekeepingCgroup(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager, domain *api.Domain) error {
	if err := cgroupManager.CreateChildCgroup("housekeeping", "cpuset"); err != nil {
		m.logger.Reason(err).Error("CreateChildCgroup ")
		return err
	}

	// bail out if domain does not exist
	if domain == nil {
		return nil
	}

	if domain.Spec.CPUTune == nil || domain.Spec.CPUTune.EmulatorPin == nil {
		return nil
	}

	hkcpus, err := hardware.ParseCPUSetLine(domain.Spec.CPUTune.EmulatorPin.CPUSet, 100)
	if err != nil {
		return err
	}

	m.logger.V(3).Object(vmi).Infof("housekeeping cpu: %v", hkcpus)

	err = cgroupManager.SetCpuSet("housekeeping", hkcpus)
	if err != nil {
		return err
	}

	tids, err := cgroupManager.GetCgroupThreads()
	if err != nil {
		return err
	}
	hktids := make([]int, 0, 10)

	for _, tid := range tids {
		proc, err := ps.FindProcess(tid)
		if err != nil {
			m.logger.Object(vmi).Errorf("Failure to find process: %s", err.Error())
			return err
		}
		if proc == nil {
			return fmt.Errorf("failed to find process with tid: %d", tid)
		}
		comm := proc.Executable()
		if strings.Contains(comm, "CPU ") && strings.Contains(comm, "MSHV") {
			continue
		}
		hktids = append(hktids, tid)
	}

	m.logger.V(3).Object(vmi).Infof("hk thread ids: %v", hktids)
	for _, tid := range hktids {
		err = cgroupManager.AttachTID("cpuset", "housekeeping", tid)
		if err != nil {
			m.logger.Object(vmi).Errorf("Error attaching tid %d: %v", tid, err.Error())
			return err
		}
	}

	return nil
}

var qemuProcessExecutablePrefixes = []string{"qemu-system"}

// getQEMUProcess encapsulates and exposes the logic to retrieve the QEMU process ID
func getQEMUProcess(r isolation.IsolationResult) (ps.Process, error) {
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
