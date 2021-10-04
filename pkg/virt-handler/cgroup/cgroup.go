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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package cgroup

import (
	"fmt"
	"path/filepath"
	"strconv"

	"kubevirt.io/client-go/log"

	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/configs"

	v1 "kubevirt.io/client-go/api/v1"
	virtutil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

var (
	isolationDetector *isolation.PodIsolationDetector
)

// Manager is the only interface to use in order to inspect, update or define cgroup properties.
// This interface is agnostic to cgroups version (supports v1 and v2) and is completely transparent from the
// users perspective. To achieve this "runc"'s cgroup manager is being levitated. This package's implementation
// guide-line is to have the thinnest glue layer possible in order to have all runc's capabilities without extra effort.
// This interface can, of course, extend runc and introduce new functionalities that are specific to Kubevirt's use.
type Manager interface {
	Set(r *configs.Resources) error
	runc_cgroups.Manager

	// GetBasePathToHostSubsystem returns the path to the specified subsystem
	// from the host's viewpoint.
	GetBasePathToHostSubsystem(subsystem string) (string, error)

	// GetCgroupVersion returns the current cgroup version (i.e. v1 or v2)
	GetCgroupVersion() string
}

// NewManagerFromPid initializes a new cgroup manager from VMI's pid.
// The pid is expected to VMI's pid from the host's viewpoint.
func NewManagerFromPid(pid int) (manager Manager, err error) {
	const isRootless = false
	var cgroupVersion string

	procCgroupBasePath := filepath.Join(procMountPoint, strconv.Itoa(pid), cgroupStr)
	controllerPaths, err := runc_cgroups.ParseCgroupFile(procCgroupBasePath)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize new cgroup manager. err: %v", err)
	}

	config := &configs.Cgroup{
		Path:      HostCgroupBasePath,
		Resources: &configs.Resources{},
	}

	if runc_cgroups.IsCgroup2UnifiedMode() {
		cgroupVersion = v2Str
		slicePath := filepath.Join(cgroupBasePath, controllerPaths[""])
		manager, err = newV2Manager(config, slicePath, isRootless, pid)
	} else {
		cgroupVersion = v1Str
		for subsystem, path := range controllerPaths {
			if path == "" {
				continue
			}
			controllerPaths[subsystem] = filepath.Join("/", subsystem, path)
		}

		manager, err = newV1Manager(config, controllerPaths, isRootless)
	}

	if err != nil {
		log.Log.Infof("error occurred while initialized a new cgroup %s manager: %v", cgroupVersion, err)
	} else {
		log.Log.Infof("initialized a new cgroup %s manager successfully", cgroupVersion)
	}

	return manager, err
}

func NewManagerFromVM(vmi *v1.VirtualMachineInstance) (Manager, error) {
	isolationRes, err := detectVMIsolation(vmi, "")
	if err != nil {
		return nil, err
	}

	return NewManagerFromPid(isolationRes.Pid())
}

// NewManagerFromVMAndSocket is similar to NewManagerFromVM but is faster since there is no need
// to search for the socket.
func NewManagerFromVMAndSocket(vmi *v1.VirtualMachineInstance, socket string) (Manager, error) {
	if socket == "" {
		return nil, fmt.Errorf("socket has to be a non-empty string")
	}

	isolationRes, err := detectVMIsolation(vmi, socket)
	if err != nil {
		return nil, err
	}

	return NewManagerFromPid(isolationRes.Pid())
}

func GetCpuSetPath() string {
	if runc_cgroups.IsCgroup2UnifiedMode() {
		return filepath.Join(cgroupBasePath, "cpuset.cpus.effective")
	}
	return filepath.Join(cgroupBasePath, "cpuset", "cpuset.cpus")
}

func initIsolationDetectorIfNil() {
	if isolationDetector != nil {
		return
	}

	detector := isolation.NewSocketBasedIsolationDetector(virtutil.VirtShareDir)
	isolationDetector = &detector
}

// detectVMIsolation detects VM's isolation. Socket is optional and makes the execution faster
func detectVMIsolation(vm *v1.VirtualMachineInstance, socket string) (isolationRes isolation.IsolationResult, err error) {
	const detectionErrFormat = "cannot detect vm \"%s\", err: %v"
	initIsolationDetectorIfNil()

	if socket == "" {
		isolationRes, err = (*isolationDetector).Detect(vm)
	} else {
		isolationRes, err = (*isolationDetector).DetectForSocket(vm, socket)
	}

	if err != nil {
		return nil, fmt.Errorf(detectionErrFormat, vm.Name, err)
	}

	return isolationRes, nil
}
