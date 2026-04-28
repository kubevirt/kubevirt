/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package isolation

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

import (
	"fmt"
	"net"
	"syscall"
	"time"

	ps "github.com/mitchellh/go-ps"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/safepath"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

// PodIsolationDetector helps detecting cgroups, namespaces and PIDs of Pods from outside of them.
// Different strategies may be applied to do that.
type PodIsolationDetector interface {
	// Detect takes a vm, looks up a socket based the VM and detects pid, cgroups and namespaces of the owner of that socket.
	// It returns an IsolationResult containing all isolation information
	Detect(vm *v1.VirtualMachineInstance) (IsolationResult, error)

	DetectForSocket(socket string) (IsolationResult, error)
}

const isolationDialTimeout = 5

type socketBasedIsolationDetector struct {
}

// NewSocketBasedIsolationDetector takes socketDir and creates a socket based IsolationDetector
// It returns a PodIsolationDetector which detects pid, cgroups and namespaces of the socket owner.
func NewSocketBasedIsolationDetector() PodIsolationDetector {
	return &socketBasedIsolationDetector{}
}

func (s *socketBasedIsolationDetector) Detect(vm *v1.VirtualMachineInstance) (IsolationResult, error) {
	// Look up the socket of the virt-launcher Pod which was created for that VM, and extract the PID from it
	socket, err := cmdclient.FindSocket(vm)
	if err != nil {
		return nil, err
	}

	return s.DetectForSocket(socket)
}

func (s *socketBasedIsolationDetector) DetectForSocket(socket string) (IsolationResult, error) {
	pid, err := s.getPid(socket)
	if err != nil {
		return nil, fmt.Errorf("Could not get owner Pid of socket %s: %w", socket, err)
	}

	ppid, err := getPPid(pid)
	if err != nil {
		return nil, fmt.Errorf("Could not get owner PPid of socket %s: %w", socket, err)
	}

	return NewIsolationResult(pid, ppid), nil
}

func (s *socketBasedIsolationDetector) getPid(socket string) (int, error) {
	safeSocket, err := safepath.NewFileNoFollow(socket)
	if err != nil {
		return -1, err
	}
	defer safeSocket.Close()

	sock, err := net.DialTimeout("unix", safeSocket.SafePath(), time.Duration(isolationDialTimeout)*time.Second)
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
	if process == nil {
		return -1, fmt.Errorf("failed to find process with pid: %d", pid)
	}
	return process.PPid(), nil
}
