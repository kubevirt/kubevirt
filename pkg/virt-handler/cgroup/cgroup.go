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
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"

	cgroups "github.com/opencontainers/cgroups"
	devices "github.com/opencontainers/cgroups/devices/config"

	v1 "kubevirt.io/api/core/v1"

	cgroupconsts "kubevirt.io/kubevirt/pkg/virt-handler/cgroup/constants"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

// Manager is the only interface to use in order to inspect, update or define cgroup properties.
// This interface is agnostic to cgroups version (supports v1 and v2) and is completely transparent from the
// user's perspective. To achieve this, the opencontainers/cgroups library is being leveraged. This package's
// implementation guide-line is to have the thinnest glue layer possible. This interface can, of course, extend
// the library and introduce new functionalities that are specific to KubeVirt's use.
type Manager interface {
	Set(r *cgroups.Resources) error

	// AllowDevice adds a device to the eBPF device map, allowing access with
	// the given permissions. deviceType is 'b' (block) or 'c' (char).
	AllowDevice(deviceType string, major, minor int64, permissions string) error

	// RemoveDevice removes a device from the eBPF device map. This does not
	// explicitly deny the device — it only removes a previously added dynamic
	// entry, reverting to the base program's decision for that device.
	RemoveDevice(deviceType string, major, minor int64) error

	// ListDevices returns all devices currently in the eBPF device map.
	ListDevices() ([]cgroupconsts.DeviceMapEntry, error)

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

// This is here so that mockgen would create a mock out of it. That way we would have a mocked cgroups manager.
type cgroupsManager interface {
	cgroups.Manager
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

// splicedCgroups tracks cgroup paths that have already had their eBPF device
// map spliced, avoiding redundant virt-chroot forks on every sync loop.
var splicedCgroups sync.Map

// newManagerFromPid initializes a new cgroup manager from VMI's pid.
// The pid is expected to VMI's pid from the host's viewpoint.
func newManagerFromPid(pid int, deviceRules []*devices.Rule, spliceDeviceMap bool) (manager Manager, err error) {
	var version CgroupVersion
	var slicePath string

	procCgroupBasePath := filepath.Join(cgroupconsts.ProcMountPoint, strconv.Itoa(pid), cgroupconsts.CgroupStr)
	controllerPaths, err := cgroups.ParseCgroupFile(procCgroupBasePath)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize new cgroup manager. err: %v", err)
	}

	config := &cgroups.Cgroup{
		Path: cgroupconsts.HostCgroupBasePath,
		Resources: &cgroups.Resources{
			Devices: deviceRules,
		},
	}

	if cgroups.IsCgroup2UnifiedMode() {
		version = V2
		slicePath = filepath.Join(cgroupconsts.CgroupBasePath, controllerPaths[""])
		slicePath = managerPath(slicePath)
		manager, err = newV2Manager(config, slicePath, spliceDeviceMap)
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
		return manager, err
	}
	log.Log.V(5).Infof("initialized cgroup %s manager. controllerPaths: %v, procCgroupBasePath: %s", version, controllerPaths, procCgroupBasePath)

	if v2mgr, ok := manager.(*v2Manager); ok && v2mgr.spliceDeviceMap {
		// Splice the eBPF device map into the cgroup's device filter once per
		// cgroup path. The splice itself (via virt-chroot) is idempotent, but
		// we skip re-running it to avoid forking a process on every sync loop.
		if _, already := splicedCgroups.LoadOrStore(slicePath, true); !already {
			cgroupPaths := []string{slicePath}
			if targetDir, parentPath := filepath.Base(slicePath), path.Dir(slicePath); targetDir == "container" && strings.HasSuffix(parentPath, ".scope") {
				cgroupPaths = append(cgroupPaths, parentPath)
			}
			if err := execVirtChrootSpliceDeviceMap(cgroupPaths, pid); err != nil {
				splicedCgroups.Delete(slicePath)
				return nil, fmt.Errorf("failed to splice eBPF device map: %w", err)
			}
		}
	}

	return manager, nil
}

func NewManagerFromVM(vmi *v1.VirtualMachineInstance, host string, hypervisorDevice string, allowEmulation bool, spliceDeviceMap bool) (Manager, error) {
	isolationRes, err := detectVMIsolation(vmi)
	if err != nil {
		return nil, err
	}

	if spliceDeviceMap {
		return newManagerFromPid(isolationRes.Pid(), nil, true)
	}

	mountRoot, err := isolationRes.MountRoot()
	if err != nil {
		return nil, err
	}

	vmiDeviceRules, err := generateDeviceRulesForVMI(vmi, mountRoot, host, hypervisorDevice, allowEmulation)
	if err != nil {
		return nil, err
	}
	return newManagerFromPid(isolationRes.Pid(), vmiDeviceRules, false)
}

// GetGlobalCpuSetPath returns the CPU set of the main cgroup slice
func GetGlobalCpuSetPath() string {
	if cgroups.IsCgroup2UnifiedMode() {
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
func detectVMIsolation(vm *v1.VirtualMachineInstance) (isolationRes isolation.IsolationResult, err error) {
	const detectionErrFormat = "cannot detect vm \"%s\", err: %v"
	detector := isolation.NewSocketBasedIsolationDetector()

	isolationRes, err = detector.Detect(vm)

	if err != nil {
		return nil, fmt.Errorf(detectionErrFormat, vm.Name, err)
	}

	return isolationRes, nil
}

var miscCapacityPath = path.Join(util.HostRootMount, "/sys/fs/cgroup/misc.capacity")

func GetMiscCapacity(key string) (int, error) {
	f, err := os.Open(miscCapacityPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	// File has lines in the format: "key [capacity]"
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		if parts[0] == key {
			capacity, err := strconv.Atoi(parts[1])
			if err != nil {
				return 0, err
			}
			return capacity, nil
		}
	}
	return 0, fmt.Errorf("key %s not found in misc.capacity", key)
}
