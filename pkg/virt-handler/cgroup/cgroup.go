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

package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"kubevirt.io/client-go/log"

	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/devices"

	v1 "kubevirt.io/api/core/v1"

	virtutil "kubevirt.io/kubevirt/pkg/util"
	cgroupconsts "kubevirt.io/kubevirt/pkg/virt-handler/cgroup/constants"
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

	// SetCpuSet returns the cpu set
	SetCpuSet(subcgroup string, cpulist []int) error

	// Create new child cgroup
	CreateChildCgroup(name string, subSystem string) error

	// Attach TID to cgroup
	AttachTID(subSystem string, subCgroup string, tid int) error

	// Get list of threads attached to cgroup
	GetCgroupThreads() ([]int, error)
}

// This is here so that mockgen would create a mock out of it. That way we would have a mocked runc manager.
type runcManager interface {
	runc_cgroups.Manager
}

// If a task is moved into a sub-cgroup, we want the manager to
// reference the root cgroup, not the sub-cgroup.
// Currently the only sub-cgroup create is named "housekeeping".

func managerPath(taskPath string) string {
	retPath := taskPath
	s := strings.Split(taskPath, "/")
	if s[len(s)-1] == "housekeeping" {
		fStr := "/" + strings.Join(s[1:len(s)-1], "/") + "/"
		retPath = fStr
	}
	return retPath
}

// newManagerFromPid initializes a new cgroup manager from VMI's pid.
// The pid is expected to VMI's pid from the host's viewpoint.
func newManagerFromPid(pid int, deviceRules []*devices.Rule) (manager Manager, err error) {
	const isRootless = false
	var version CgroupVersion

	procCgroupBasePath := filepath.Join(cgroupconsts.ProcMountPoint, strconv.Itoa(pid), cgroupconsts.CgroupStr)
	controllerPaths, err := runc_cgroups.ParseCgroupFile(procCgroupBasePath)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize new cgroup manager. err: %v", err)
	}

	config := &configs.Cgroup{
		Path: cgroupconsts.HostCgroupBasePath,
		Resources: &configs.Resources{
			Devices: deviceRules,
		},
		Rootless: isRootless,
	}

	if runc_cgroups.IsCgroup2UnifiedMode() {
		version = V2
		slicePath := filepath.Join(cgroupconsts.CgroupBasePath, controllerPaths[""])
		slicePath = managerPath(slicePath)
		manager, err = newV2Manager(config, slicePath)
	} else {
		version = V1
		for subsystem, path := range controllerPaths {
			if path == "" {
				continue
			}
			path = managerPath(path)
			controllerPaths[subsystem] = filepath.Join("/", subsystem, path)
		}

		manager, err = newV1Manager(config, controllerPaths)
	}

	if err != nil {
		log.Log.Errorf("error occurred while initialized a new cgroup %s manager: %v", version, err)
	} else {
		log.Log.Infof("initialized a new cgroup %s manager successfully. controllerPaths: %v, procCgroupBasePath: %s", version, controllerPaths, procCgroupBasePath)
	}

	return manager, err
}

func NewManagerFromVM(vmi *v1.VirtualMachineInstance, host string) (Manager, error) {
	isolationRes, err := detectVMIsolation(vmi, "")
	if err != nil {
		return nil, err
	}

	vmiDeviceRules, err := generateDeviceRulesForVMI(vmi, isolationRes, host)
	if err != nil {
		return nil, err
	}

	return newManagerFromPid(isolationRes.Pid(), vmiDeviceRules)
}

// GetGlobalCpuSetPath returns the CPU set of the main cgroup slice
func GetGlobalCpuSetPath() string {
	if runc_cgroups.IsCgroup2UnifiedMode() {
		return filepath.Join(cgroupconsts.CgroupBasePath, "cpuset.cpus.effective")
	}
	return filepath.Join(cgroupconsts.CgroupBasePath, "cpuset", "cpuset.cpus")
}

func getCpuSetPath(manager Manager, cpusetFile string) (string, error) {
	cpuSubsystemPath, err := manager.GetBasePathToHostSubsystem("cpuset")
	if err != nil {
		return "", err
	}

	cpuset, err := os.ReadFile(filepath.Join(cpuSubsystemPath, cpusetFile))
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
