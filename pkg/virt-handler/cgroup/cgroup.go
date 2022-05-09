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
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"kubevirt.io/client-go/log"

	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/configs"

	v1 "kubevirt.io/api/core/v1"
	virtutil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

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
	GetCpuSet() (string, error)
}

// NewManagerFromPid initializes a new cgroup manager from VMI's pid.
// The pid is expected to VMI's pid from the host's viewpoint.
func NewManagerFromPid(pid int) (manager Manager, err error) {
	const isRootless = false
	var version CgroupVersion

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
		version = V2
		slicePath := filepath.Join(cgroupBasePath, controllerPaths[""])
		manager, err = newV2Manager(config, slicePath, isRootless)
	} else {
		version = V1
		for subsystem, path := range controllerPaths {
			if path == "" {
				continue
			}
			controllerPaths[subsystem] = filepath.Join("/", subsystem, path)
		}

		manager, err = newV1Manager(config, controllerPaths, isRootless)
	}

	if err != nil {
		log.Log.Errorf("error occurred while initialized a new cgroup %s manager: %v", version, err)
	} else {
		log.Log.Infof("initialized a new cgroup %s manager successfully. controllerPaths: %v, procCgroupBasePath: %s", version, controllerPaths, procCgroupBasePath)
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

// GetGlobalCpuSetPath returns the CPU set of the main cgroup slice
func GetGlobalCpuSetPath() string {
	if runc_cgroups.IsCgroup2UnifiedMode() {
		return filepath.Join(cgroupBasePath, "cpuset.cpus.effective")
	}
	return filepath.Join(cgroupBasePath, "cpuset", "cpuset.cpus")
}

func getCpuSetPath(manager Manager, cpusetFile string) (string, error) {
	cpuSubsystemPath, err := manager.GetBasePathToHostSubsystem("cpuset")
	if err != nil {
		return "", err
	}

	cpuset, err := ioutil.ReadFile(filepath.Join(cpuSubsystemPath, cpusetFile))
	if err != nil {
		return "", err
	}

	cpusetStr := strings.TrimSpace(string(cpuset))
	return cpusetStr, nil
}

// detectVMIsolation detects VM's IsolationResult, which can then be useful for receiving information such as PID.
// Socket is optional and makes the execution faster
func detectVMIsolation(vm *v1.VirtualMachineInstance, socket string) (isolationRes isolation.IsolationResult, err error) {
	const detectionErrFormat = "cannot detect vm \"%s\", err: %v"
	detector := isolation.NewSocketBasedIsolationDetector(virtutil.VirtShareDir)

	if socket == "" {
		isolationRes, err = detector.Detect(vm)
	} else {
		isolationRes, err = detector.DetectForSocket(vm, socket)
	}

	if err != nil {
		return nil, fmt.Errorf(detectionErrFormat, vm.Name, err)
	}

	return isolationRes, nil
}
