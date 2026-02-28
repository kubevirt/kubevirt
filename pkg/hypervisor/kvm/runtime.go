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
	"strconv"
	"strings"

	"github.com/mitchellh/go-ps"
	"golang.org/x/sys/unix"
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
	if vmi.IsCPUDedicated() && vmi.Spec.Domain.CPU.IsolateEmulatorThread {
		err := k.configureHousekeepingCgroup(vmi, cgroupManager, domain)
		if err != nil {
			return err
		}
	}

	// Configure vcpu scheduler for realtime workloads and affine PIT thread for dedicated CPU
	if vmi.IsRealtimeEnabled() && !vmi.IsRunning() && !vmi.IsFinal() {
		k.logger.Object(vmi).Info("Configuring vcpus for real time workloads")
		if err := k.configureVCPUScheduler(vmi); err != nil {
			return err
		}
	}
	if vmi.IsCPUDedicated() && !vmi.IsRunning() && !vmi.IsFinal() {
		k.logger.V(3).Object(vmi).Info("Affining PIT thread")
		if err := k.affinePitThread(vmi); err != nil {
			return err
		}
	}
	return nil
}

func (k *KvmVirtRuntime) affinePitThread(vmi *v1.VirtualMachineInstance) error {
	res, err := k.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}
	var Mask unix.CPUSet
	Mask.Zero()
	qemuprocess, err := GetQEMUProcess(res)
	if err != nil {
		return err
	}
	qemupid := qemuprocess.Pid()
	if qemupid == -1 {
		return nil
	}

	pitpid, err := KvmPitPid(res)
	if err != nil {
		return err
	}
	if pitpid == -1 {
		return nil
	}
	if vmi.IsRealtimeEnabled() {
		param := common.SchedParam{Priority: 2}
		err = common.SchedSetScheduler(pitpid, common.SchedFIFO, param)
		if err != nil {
			return fmt.Errorf("failed to set FIFO scheduling and priority 2 for thread %d: %w", pitpid, err)
		}
	}
	vcpus, err := common.GetVCPUThreadIDs(qemupid, VcpuRegex)
	if err != nil {
		return err
	}
	vpid, ok := vcpus["0"]
	if ok == false {
		return nil
	}
	vcpupid, err := strconv.Atoi(vpid)
	if err != nil {
		return err
	}
	err = unix.SchedGetaffinity(vcpupid, &Mask)
	if err != nil {
		return err
	}
	return unix.SchedSetaffinity(pitpid, &Mask)
}

func (k *KvmVirtRuntime) configureHousekeepingCgroup(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager, domain *api.Domain) error {
	if err := cgroupManager.CreateChildCgroup("housekeeping", "cpuset"); err != nil {
		k.logger.Reason(err).Error("CreateChildCgroup ")
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

	k.logger.V(3).Object(vmi).Infof("housekeeping cpu: %v", hkcpus)

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
			k.logger.Object(vmi).Errorf("Failure to find process: %s", err.Error())
			return err
		}
		if proc == nil {
			return fmt.Errorf("failed to find process with tid: %d", tid)
		}
		comm := proc.Executable()
		if strings.Contains(comm, "CPU ") && strings.Contains(comm, "KVM") {
			continue
		}
		hktids = append(hktids, tid)
	}

	k.logger.V(3).Object(vmi).Infof("hk thread ids: %v", hktids)
	for _, tid := range hktids {
		err = cgroupManager.AttachTID("cpuset", "housekeeping", tid)
		if err != nil {
			k.logger.Object(vmi).Errorf("Error attaching tid %d: %v", tid, err.Error())
			return err
		}
	}

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

func KvmPitPid(r isolation.IsolationResult) (int, error) {
	qemuprocess, err := GetQEMUProcess(r)
	if err != nil {
		return -1, err
	}
	processes, _ := ps.Processes()
	nspid, err := isolation.GetNspid(qemuprocess.Pid())
	if err != nil || nspid == -1 {
		return -1, err
	}
	pitstr := "kvm-pit/" + strconv.Itoa(nspid)

	for _, process := range processes {
		if process.Executable() == pitstr {
			return process.Pid(), nil
		}
	}
	return -1, nil
}
