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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package isolation

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

// PodIsolationDetector helps detecting cgroups, namespaces and PIDs of Pods from outside of them.
// Different strategies may be applied to do that.
type PodIsolationDetector interface {
	// Detect takes a vm, looks up a socket based the VM and detects pid, cgroups and namespaces of the owner of that socket.
	// It returns an IsolationResult containing all isolation information
	Detect(vm *v1.VirtualMachineInstance) (*IsolationResult, error)

	// Whitelist allows specifying cgroup controller which should be considered to detect the cgroup slice
	// It returns a PodIsolationDetector to allow configuring the PodIsolationDetector via the builder pattern.
	Whitelist(controller []string) PodIsolationDetector
}

type socketBasedIsolationDetector struct {
	socketDir  string
	controller []string
}

// NewSocketBasedIsolationDetector takes socketDir and creates a socket based IsolationDetector
// It returns a PodIsolationDetector which detects pid, cgroups and namespaces of the socket owner.
func NewSocketBasedIsolationDetector(socketDir string) PodIsolationDetector {
	return &socketBasedIsolationDetector{socketDir: socketDir, controller: []string{"devices"}}
}

func (s *socketBasedIsolationDetector) Whitelist(controller []string) PodIsolationDetector {
	s.controller = controller
	return s
}

func (s *socketBasedIsolationDetector) Detect(vm *v1.VirtualMachineInstance) (*IsolationResult, error) {
	var pid int
	var slice string
	var err error
	var controller []string

	// Look up the socket of the virt-launcher Pod which was created for that VM, and extract the PID from it
	socket := cmdclient.SocketFromUID(s.socketDir, string(vm.UID))
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

func NewIsolationResult(pid int, slice string, controller []string) *IsolationResult {
	return &IsolationResult{pid: pid, slice: slice, controller: controller}
}

type IsolationResult struct {
	pid        int
	slice      string
	controller []string
}

func (r *IsolationResult) Slice() string {
	return r.slice
}

func (r *IsolationResult) PIDNamespace() string {
	return fmt.Sprintf("/proc/%d/ns/pid", r.pid)
}

func (r *IsolationResult) NetNamespace() string {
	return fmt.Sprintf("/proc/%d/ns/net", r.pid)
}

func (r *IsolationResult) MountRoot() string {
	return fmt.Sprintf("/proc/%d/root", r.pid)
}

func (r *IsolationResult) Pid() int {
	return r.pid
}

func (r *IsolationResult) Controller() []string {
	return r.controller
}

func (s *socketBasedIsolationDetector) getPid(socket string) (int, error) {
	sock, err := net.Dial("unix", socket)
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

func (s *socketBasedIsolationDetector) getSlice(pid int) (controller []string, slice string, err error) {
	cgroups, err := os.Open(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return
	}
	defer cgroups.Close()

	scanner := bufio.NewScanner(cgroups)
	for scanner.Scan() {
		cgEntry := strings.Split(scanner.Text(), ":")
		// Check if we have a sane cgroup line
		if len(cgEntry) != 3 {
			err = fmt.Errorf("Could not extract slice from cgroup line: %s", scanner.Text())
			return
		}
		// Skip not supported cgroup controller
		if !sliceContains(s.controller, cgEntry[1]) {
			continue
		}

		// Set and check cgroup slice
		if slice == "" {
			slice = cgEntry[2]
		} else if slice != cgEntry[2] {
			err = fmt.Errorf("Process is part of more than one slice. Expected %s, found %s", slice, cgEntry[2])
			return
		}
		// Add controller
		controller = append(controller, cgEntry[1])
	}
	// Check if we encountered a read error
	if scanner.Err() != nil {
		err = scanner.Err()
		return
	}

	if slice == "" {
		err = fmt.Errorf("Could not detect slice of whitelisted controller: %v", s.controller)
		return
	}
	return
}

func sliceContains(controllers []string, value string) bool {
	for _, c := range controllers {
		if c == value {
			return true
		}
	}
	return false
}

func NodeIsolationResult() *IsolationResult {
	return &IsolationResult{
		pid: 1,
	}
}
