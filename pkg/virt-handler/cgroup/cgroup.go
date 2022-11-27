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

	v1 "kubevirt.io/api/core/v1"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

type TaskType int

const (
	Thread TaskType = iota
	Process
)

// Manager is the only interface to use in order to inspect, update or define cgroup properties.
// This interface is agnostic to cgroups version (supports v1 and v2) and is completely transparent from the
// users perspective. To achieve this "runc"'s cgroup manager is being levitated. This package's implementation
// guide-line is to have the thinnest glue layer possible in order to have all runc's capabilities without extra effort.
// This interface can, of course, extend runc and introduce new functionalities that are specific to Kubevirt's use.
type Manager interface {
	Set(r *configs.Resources) error

	// GetBasePathToHostSubsystem returns the path to the specified subsystem
	// from the host's viewpoint.
	GetBasePathToHostSubsystem(subsystem string) (string, error)

	// GetCgroupVersion returns the current cgroup version (i.e. v1 or v2)
	GetCgroupVersion() CgroupVersion

	// GetCpuSet returns the cpu set
	GetCpuSet() ([]int, error)

	// SetCpuSet returns the cpu set
	SetCpuSet(cpulist []int) error

	// Create new child cgroup
	CreateChildCgroup(name string, subSystems ...string) (Manager, error)

	// GetCgroupThreads returns a list TIDs of the treads that are attached to that cgroup.
	// Ideally, this implementation needs to reside in runc's repository.
	// An issue is opened to track that: https://github.com/opencontainers/runc/issues/3616.
	GetCgroupThreads() ([]int, error)

	GetCgroupThreadsWithFilter(func(string) bool) ([]int, error)

	MakeThreaded() error

	// HandleDedicatedCpus is expected to be called from the VMI's compute container manager, for example,
	// a manager initizlied by NewManagerFromVM()
	HandleDedicatedCpus(vmi *v1.VirtualMachineInstance) error
}

// This is here so that mockgen would create a mock out of it. That way we would have a mocked runc manager.
type runcManager interface {
	runc_cgroups.Manager
}

// NewManagerFromPath returns a new manager that corresponds to the provided cgroup paths.
// Note that for cgroups v2 the map is expected to include only one value which is the unified cgroup
// path. The key is expected to be an empty string ("").
func NewManagerFromPath(controllerPaths map[string]string) (manager Manager, err error) {
	var version CgroupVersion

	if runc_cgroups.IsCgroup2UnifiedMode() {
		version = V2
		controllerPaths = formatCgroupPaths(controllerPaths)
		slicePath := controllerPaths[""]

		manager, err = newV2Manager(slicePath)
	} else {
		version = V1
		controllerPaths = formatCgroupPaths(controllerPaths)

		manager, err = newV1Manager(controllerPaths)
	}

	if err != nil {
		log.Log.Errorf("error occurred while initialized a new cgroup %s manager: %v", version, err)
	} else {
		log.Log.Infof("initialized a new cgroup %s manager successfully. controllerPaths: %v", version, controllerPaths)
	}

	return manager, err
}

func NewManagerFromPid(pid int) (manager Manager, err error) {
	procCgroupBasePath := filepath.Join(procMountPoint, strconv.Itoa(pid), cgroupStr)

	controllerPaths, err := runc_cgroups.ParseCgroupFile(procCgroupBasePath)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize new cgroup manager. err: %v", err)
	}

	return NewManagerFromPath(controllerPaths)
}

// NewManagerFromVM returns a manager which corresponds to the VM's compute container's cgroup.
func NewManagerFromVM(vmi *v1.VirtualMachineInstance) (Manager, error) {
	isolationRes, err := detectVMIsolation(vmi, "")
	if err != nil {
		return nil, err
	}

	virtLauncherPid := isolationRes.Pid()
	log.Log.Infof("creating new cgroup for vmi %s, virt-launcher's pid: %d", vmi.Name, virtLauncherPid)

	return NewManagerFromPid(virtLauncherPid)
}
