/*
 * This file is part of the kubevirt project
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

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

import (
	"fmt"
	"net"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	ps "github.com/mitchellh/go-ps"
	"golang.org/x/sys/unix"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

// PodIsolationDetector helps detecting cgroups, namespaces and PIDs of Pods from outside of them.
// Different strategies may be applied to do that.
type PodIsolationDetector interface {
	// Detect takes a vm, looks up a socket based the VM and detects pid, cgroups and namespaces of the owner of that socket.
	// It returns an IsolationResult containing all isolation information
	Detect(vm *v1.VirtualMachineInstance) (IsolationResult, error)

	DetectForSocket(vm *v1.VirtualMachineInstance, socket string) (IsolationResult, error)

	// Allowlist allows specifying cgroup controller which should be considered to detect the cgroup slice
	// It returns a PodIsolationDetector to allow configuring the PodIsolationDetector via the builder pattern.
	Allowlist(controller []string) PodIsolationDetector

	// Adjust system resources to run the passed VM
	AdjustResources(vm *v1.VirtualMachineInstance, additionalOverheadRatio *string) error
}

const isolationDialTimeout = 5

type socketBasedIsolationDetector struct {
	socketDir  string
	controller []string
}

// NewSocketBasedIsolationDetector takes socketDir and creates a socket based IsolationDetector
// It returns a PodIsolationDetector which detects pid, cgroups and namespaces of the socket owner.
func NewSocketBasedIsolationDetector(socketDir string) PodIsolationDetector {
	return &socketBasedIsolationDetector{
		socketDir:  socketDir,
		controller: []string{"devices"},
	}
}

func (s *socketBasedIsolationDetector) Detect(vm *v1.VirtualMachineInstance) (IsolationResult, error) {
	// Look up the socket of the virt-launcher Pod which was created for that VM, and extract the PID from it
	socket, err := cmdclient.FindSocketOnHost(vm)
	if err != nil {
		return nil, err
	}

	return s.DetectForSocket(vm, socket)
}

func (s *socketBasedIsolationDetector) DetectForSocket(vm *v1.VirtualMachineInstance, socket string) (IsolationResult, error) {
	var pid int
	var ppid int
	var err error

	if pid, err = s.getPid(socket); err != nil {
		log.Log.Object(vm).Reason(err).Errorf("Could not get owner Pid of socket %s", socket)
		return nil, err
	}

	if ppid, err = getPPid(pid); err != nil {
		log.Log.Object(vm).Reason(err).Errorf("Could not get owner PPid of socket %s", socket)
		return nil, err
	}

	return NewIsolationResult(pid, ppid), nil
}

func (s *socketBasedIsolationDetector) Allowlist(controller []string) PodIsolationDetector {
	s.controller = controller
	return s
}

func (s *socketBasedIsolationDetector) AdjustResources(vm *v1.VirtualMachineInstance, additionalOverheadRatio *string) error {
	// only VFIO attached or with lock guest memory domains require MEMLOCK adjustment
	if !util.IsVFIOVMI(vm) && !vm.IsRealtimeEnabled() && !util.IsSEVVMI(vm) {
		return nil
	}

	// bump memlock ulimit for virtqemud
	res, err := s.Detect(vm)
	if err != nil {
		return err
	}
	launcherPid := res.Pid()

	processes, err := ps.Processes()
	if err != nil {
		return fmt.Errorf("failed to get all processes: %v", err)
	}

	for _, process := range processes {
		// consider all processes that are virt-launcher children
		if process.PPid() != launcherPid {
			continue
		}

		// virtqemud process sets the memory lock limit before fork/exec-ing into qemu
		if process.Executable() != "virtqemud" {
			continue
		}

		// make the best estimate for memory required by libvirt
		memlockSize := services.GetMemoryOverhead(vm, runtime.GOARCH, additionalOverheadRatio)
		// Add base memory requested for the VM
		vmiMemoryReq := vm.Spec.Domain.Resources.Requests.Memory()
		memlockSize.Add(*resource.NewScaledQuantity(vmiMemoryReq.ScaledValue(resource.Kilo), resource.Kilo))

		err = setProcessMemoryLockRLimit(process.Pid(), memlockSize.Value())
		if err != nil {
			return fmt.Errorf("failed to set process %d memlock rlimit to %d: %v", process.Pid(), memlockSize.Value(), err)
		}
		// we assume a single process should match
		break
	}
	return nil
}

// AdjustQemuProcessMemoryLimits adjusts QEMU process MEMLOCK rlimits that runs inside
// virt-launcher pod on the given VMI according to its spec.
// Only VMI's with VFIO devices (e.g: SRIOV, GPU), SEV or RealTime workloads require QEMU process MEMLOCK adjustment.
func AdjustQemuProcessMemoryLimits(podIsoDetector PodIsolationDetector, vmi *v1.VirtualMachineInstance, additionalOverheadRatio *string) error {
	if !util.IsVFIOVMI(vmi) && !vmi.IsRealtimeEnabled() && !util.IsSEVVMI(vmi) {
		return nil
	}

	isolationResult, err := podIsoDetector.Detect(vmi)
	if err != nil {
		return err
	}

	qemuProcess, err := isolationResult.GetQEMUProcess()
	if err != nil {
		return err
	}
	qemuProcessID := qemuProcess.Pid()
	// make the best estimate for memory required by libvirt
	memlockSize := services.GetMemoryOverhead(vmi, runtime.GOARCH, additionalOverheadRatio)
	// Add base memory requested for the VM
	vmiMemoryReq := vmi.Spec.Domain.Resources.Requests.Memory()
	memlockSize.Add(*resource.NewScaledQuantity(vmiMemoryReq.ScaledValue(resource.Kilo), resource.Kilo))

	if err := setProcessMemoryLockRLimit(qemuProcessID, memlockSize.Value()); err != nil {
		return fmt.Errorf("failed to set process %d memlock rlimit to %d: %v", qemuProcessID, memlockSize.Value(), err)
	}
	log.Log.V(5).Object(vmi).Infof("set process %+v memlock rlimits to: Cur: %[2]d Max:%[2]d",
		qemuProcess, memlockSize.Value())

	return nil
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

// setProcessMemoryLockRLimit Adjusts process MEMLOCK
// soft-limit (current) and hard-limit (max) to the given size.
func setProcessMemoryLockRLimit(pid int, size int64) error {
	// standard golang libraries don't provide API to set runtime limits
	// for other processes, so we have to directly call to kernel
	rlimit := unix.Rlimit{
		Cur: uint64(size),
		Max: uint64(size),
	}
	_, _, errno := unix.RawSyscall6(unix.SYS_PRLIMIT64,
		uintptr(pid),
		uintptr(unix.RLIMIT_MEMLOCK),
		uintptr(unsafe.Pointer(&rlimit)), // #nosec used in unix RawSyscall6
		0, 0, 0)
	if errno != 0 {
		return fmt.Errorf("error setting prlimit: %v", errno)
	}

	return nil
}

func (s *socketBasedIsolationDetector) getPid(socket string) (int, error) {
	sock, err := net.DialTimeout("unix", socket, time.Duration(isolationDialTimeout)*time.Second)
	if err != nil {
		return -1, err
	}
	defer sock.Close()

	ufile, err := sock.(*net.UnixConn).File()
	if err != nil {
		return -1, err
	}
	defer ufile.Close()

	// This is the tricky part, which will give us the PID of the owning socket
	ucreds, err := syscall.GetsockoptUcred(int(ufile.Fd()), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	if err != nil {
		return -1, err
	}

	if int(ucreds.Pid) == 0 {
		return -1, fmt.Errorf("the detected PID is 0. Is the isolation detector running in the host PID namespace?")
	}

	return int(ucreds.Pid), nil
}

func getPPid(pid int) (int, error) {
	process, err := ps.FindProcess(pid)
	if err != nil {
		return -1, err
	}

	return process.PPid(), nil
}
