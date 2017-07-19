package isolation

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
)

type PodIsolationDetector interface {
	Detect(vm *v1.VM) (IsolationResult, error)
}

type IsolationResult interface {
	Controller() []string
	Slice() string
	Pid() int
	PidNS() string
}

type socketBasedIsolationDetector struct {
	socketDir string
}

func NewSocketBasedIsolationDetector(socketDir string) PodIsolationDetector {
	return &socketBasedIsolationDetector{socketDir: socketDir}
}

func SocketFromNamespaceName(baseDir string, namespace string, name string) string {
	return filepath.Clean(baseDir) + "/" + namespace + "/" + name + "/sock"
}

func (s *socketBasedIsolationDetector) Detect(vm *v1.VM) (IsolationResult, error) {
	var pid int
	var slice string
	var err error
	var controller []string

	socket := SocketFromNamespaceName(s.socketDir, vm.ObjectMeta.Namespace, vm.ObjectMeta.Name)
	if pid, err = s.getPid(socket); err != nil {
		logging.DefaultLogger().Object(vm).Error().Reason(err).V(3).Msgf("Could not get owner Pid of socket %s", socket)
		return nil, err

	}
	if controller, slice, err = s.getSlice(pid); err != nil {
		logging.DefaultLogger().Object(vm).Error().Reason(err).V(3).Msgf("Could not get cgroup slice for Pid %d", pid)
		return nil, err
	}

	return &isolationResult{pid: pid, slice: slice, controller: controller}, nil
}

type isolationResult struct {
	pid        int
	slice      string
	controller []string
}

func (r *isolationResult) Slice() string {
	return r.slice
}

func (r *isolationResult) PidNS() string {
	return fmt.Sprintf("/proc/%d/ns/pid", r.pid)
}

func (r *isolationResult) Pid() int {
	return r.pid
}

func (r *isolationResult) Controller() []string {
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
	ucreds, err := syscall.GetsockoptUcred(int(ufile.Fd()), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	if err != nil {
		return -1, err
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
		// Skip the systemd entry, if it is present.
		if cgEntry[1] == "name=systemd" {
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
	return
}
