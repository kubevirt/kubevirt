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
	"syscall"
	"time"
	"unsafe"

	ps "github.com/mitchellh/go-ps"
	"golang.org/x/sys/unix"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

// PodIsolationDetector helps detecting cgroups, namespaces and PIDs of Pods from outside of them.
// Different strategies may be applied to do that.
type PodIsolationDetector interface {
	// Detect takes a vm, looks up a socket based the VM and detects pid, cgroups and namespaces of the owner of that socket.
	// It returns an IsolationResult containing all isolation information
	Detect(vm *v1.VirtualMachineInstance) (IsolationResult, error)

	DetectForSocket(vm *v1.VirtualMachineInstance, socket string) (IsolationResult, error)

	// Whitelist allows specifying cgroup controller which should be considered to detect the cgroup slice
	// It returns a PodIsolationDetector to allow configuring the PodIsolationDetector via the builder pattern.
	Whitelist(controller []string) PodIsolationDetector

	// Adjust system resources to run the passed VM
	AdjustResources(vm *v1.VirtualMachineInstance) error
}

const isolationDialTimeout = 5

type socketBasedIsolationDetector struct {
	socketDir    string
	controller   []string
	cgroupParser cgroup.Parser
}

// NewSocketBasedIsolationDetector takes socketDir and creates a socket based IsolationDetector
// It returns a PodIsolationDetector which detects pid, cgroups and namespaces of the socket owner.
func NewSocketBasedIsolationDetector(socketDir string, cgroupParser cgroup.Parser) PodIsolationDetector {
	return &socketBasedIsolationDetector{
		socketDir:    socketDir,
		controller:   []string{"devices"},
		cgroupParser: cgroupParser,
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
	var slice string
	var err error
	var controller []string

	if pid, err = s.getPid(socket); err != nil {
		log.Log.Object(vm).Reason(err).Errorf("Could not get owner Pid of socket %s", socket)
		return nil, err
	}

	// Look up the cgroup slice based on the whitelisted controller
	if controller, slice, err = s.getSlice(pid); err != nil {
		log.Log.Object(vm).Reason(err).Errorf("Could not get cgroup slice for Pid %d", pid)
		return nil, err
	}

	return NewIsolationResult(pid, slice, controller), nil
}

func (s *socketBasedIsolationDetector) Whitelist(controller []string) PodIsolationDetector {
	s.controller = controller
	return s
}

func (s *socketBasedIsolationDetector) AdjustResources(vm *v1.VirtualMachineInstance) error {
	// only VFIO attached domains require MEMLOCK adjustment
	if !util.IsVFIOVMI(vm) {
		return nil
	}

	// bump memlock ulimit for libvirtd
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

		// libvirtd process sets the memory lock limit before fork/exec-ing into qemu
		if process.Executable() != "libvirtd" {
			continue
		}

		// make the best estimate for memory required by libvirt
		memlockSize, err := getMemlockSize(vm)
		if err != nil {
			return err
		}
		err = setProcessMemoryLockRLimit(process.Pid(), memlockSize)
		if err != nil {
			return fmt.Errorf("failed to set process %d memlock rlimit to %d: %v", process.Pid(), memlockSize, err)
		}
		// we assume a single process should match
		break
	}
	return nil
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
		return fmt.Errorf("Error setting prlimit: %v", errno)
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
	// This is the tricky part, which will give us the PID of the owning socket
	ucreds, err := syscall.GetsockoptUcred(int(ufile.Fd()), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	if err != nil {
		return -1, err
	}

	if int(ucreds.Pid) == 0 {
		return -1, fmt.Errorf("The detected PID is 0. Is the isolation detector running in the host PID namespace?")
	}

	return int(ucreds.Pid), nil
}

func (s *socketBasedIsolationDetector) getSlice(pid int) (controllers []string, slice string, err error) {
	slices, err := s.cgroupParser.Parse(pid)
	if err != nil {
		return
	}

	// Skip not supported cgroup controller
	for _, c := range s.controller {
		if s, ok := slices[c]; ok {
			// Set and check cgroup slice
			if slice == "" {
				slice = s
			} else if slice != s {
				err = fmt.Errorf("Process is part of more than one slice. Expected %s, found %s", slice, s)
				return
			}
			// Add controller
			controllers = append(controllers, c)
		}
	}

	if slice == "" {
		err = fmt.Errorf("Could not detect slice of whitelisted controllers: %v", s.controller)
	}

	return
}

// consider reusing getMemoryOverhead()
// This is not scientific, but neither what libvirtd does is. See details in:
// https://www.redhat.com/archives/libvirt-users/2019-August/msg00051.html
func getMemlockSize(vm *v1.VirtualMachineInstance) (int64, error) {
	memlockSize := resource.NewQuantity(0, resource.DecimalSI)

	// start with base memory requested for the VM
	vmiMemoryReq := vm.Spec.Domain.Resources.Requests.Memory()
	memlockSize.Add(*resource.NewScaledQuantity(vmiMemoryReq.ScaledValue(resource.Kilo), resource.Kilo))

	// allocate 1Gb for VFIO needs
	memlockSize.Add(resource.MustParse("1G"))

	// add some more memory for NUMA / CPU topology, platform memory alignment and other needs
	memlockSize.Add(resource.MustParse("256M"))

	bytes, ok := memlockSize.AsInt64()
	if !ok {
		return 0, fmt.Errorf("could not calculate memory lock size")
	}
	return bytes, nil
}
