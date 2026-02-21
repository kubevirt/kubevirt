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
	}
}

func (k *KvmVirtRuntime) AdjustResources(vmi *v1.VirtualMachineInstance, config *v1.KubeVirtConfiguration) error {
	if !util.IsVFIOVMI(vmi) && !vmi.IsRealtimeEnabled() && !util.IsSEVVMI(vmi) {
		return nil
	}

	isolationResult, err := k.podIsolationDetector.Detect(vmi)
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
		targetProcess, err = findVirtqemudProcess(isolationResult)
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
	memlockSize := k.GetMemoryOverhead(vmi, runtime.GOARCH, config.AdditionalGuestMemoryOverheadRatio)
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

	if err := SetProcessMemoryLockRLimit(qemuProcessID, memlockSize.Value()); err != nil {
		return fmt.Errorf("failed to set process %d memlock rlimit to %d: %v", qemuProcessID, memlockSize.Value(), err)
	}
	log.Log.V(5).Object(vmi).Infof("set process %+v memlock rlimits to: Cur: %[2]d Max:%[2]d",
		targetProcess, memlockSize.Value())

	return nil
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

func (k *KvmVirtRuntime) affinePitThread(vmi *v1.VirtualMachineInstance) error {
	res, err := k.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}
	var Mask unix.CPUSet
	Mask.Zero()
	qemuprocess, err := getQEMUProcess(res)
	if err != nil {
		return err
	}
	qemupid := qemuprocess.Pid()
	if qemupid == -1 {
		return nil
	}

	pitpid, err := kvmPitPid(res)
	if err != nil {
		return err
	}
	if pitpid == -1 {
		return nil
	}
	if vmi.IsRealtimeEnabled() {
		param := schedParam{priority: 2}
		err = schedSetScheduler(pitpid, schedFIFO, param)
		if err != nil {
			return fmt.Errorf("failed to set FIFO scheduling and priority 2 for thread %d: %w", pitpid, err)
		}
	}
	vcpus, err := getVCPUThreadIDs(qemupid)
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

func kvmPitPid(r isolation.IsolationResult) (int, error) {
	qemuprocess, err := getQEMUProcess(r)
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

var qemuProcessExecutablePrefixes = []string{"qemu-system", "qemu-kvm"}

// findIsolatedQemuProcess Returns the first occurrence of the QEMU process whose parent is PID"
func findIsolatedQemuProcess(processes []ps.Process, pid int) (ps.Process, error) {
	processes = childProcesses(processes, pid)
	for _, execPrefix := range qemuProcessExecutablePrefixes {
		if qemuProcess := lookupProcessByExecutablePrefix(processes, execPrefix); qemuProcess != nil {
			return qemuProcess, nil
		}
	}

	return nil, fmt.Errorf("no QEMU process found under process %d child processes", pid)
}

func findVirtqemudProcess(res isolation.IsolationResult) (ps.Process, error) {
	processes, err := ps.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to get all processes: %v", err)
	}
	launcherPid := res.Pid()

	for _, process := range processes {
		// consider all processes that are virt-launcher children
		if process.PPid() != launcherPid {
			continue
		}

		// virtqemud process sets the memory lock limit before fork/exec-ing into qemu
		if process.Executable() != "virtqemud" {
			continue
		}

		return process, nil
	}

	return nil, nil
}

// getQEMUProcess encapsulates and exposes the logic to retrieve the QEMU process ID
func getQEMUProcess(r isolation.IsolationResult) (ps.Process, error) {
	processes, err := ps.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to get all processes: %v", err)
	}
	qemuProcess, err := findIsolatedQemuProcess(processes, r.PPid())
	if err != nil {
		return nil, err
	}
	return qemuProcess, nil
}
