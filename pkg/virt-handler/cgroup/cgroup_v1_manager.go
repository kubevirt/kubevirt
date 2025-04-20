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
 */

package cgroup

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"kubevirt.io/client-go/log"

	cgroup_devices "github.com/opencontainers/runc/libcontainer/cgroups/devices"
	"github.com/opencontainers/runc/libcontainer/devices"

	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	runc_fs "github.com/opencontainers/runc/libcontainer/cgroups/fs"
	runc_configs "github.com/opencontainers/runc/libcontainer/configs"

	"kubevirt.io/kubevirt/pkg/util"
	cgroupconsts "kubevirt.io/kubevirt/pkg/virt-handler/cgroup/constants"
)

type v1Manager struct {
	runc_cgroups.Manager
	controllerPaths          map[string]string
	isRootless               bool
	execVirtChroot           execVirtChrootFunc
	getCurrentlyDefinedRules getCurrentlyDefinedRulesFunc
}

func newV1Manager(config *runc_configs.Cgroup, controllerPaths map[string]string) (Manager, error) {
	runcManager, err := runc_fs.NewManager(config, controllerPaths)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize new cgroup manager. err: %v", err)
	}
	return newCustomizedV1Manager(runcManager, config.Rootless, execVirtChrootCgroups, getCurrentlyDefinedRules)
}

func newCustomizedV1Manager(runcManager runc_cgroups.Manager, isRootless bool,
	execVirtChroot execVirtChrootFunc, getCurrentlyDefinedRules getCurrentlyDefinedRulesFunc) (Manager, error) {
	manager := v1Manager{
		runcManager,
		runcManager.GetPaths(),
		isRootless,
		execVirtChroot,
		getCurrentlyDefinedRules,
	}

	return &manager, nil
}

func (v *v1Manager) GetBasePathToHostSubsystem(subsystem string) (string, error) {
	subsystemPath := v.Path(subsystem)
	if subsystemPath == "" {
		return "", fmt.Errorf("controller %s does not exist", subsystem)
	}
	return filepath.Join(cgroupconsts.HostCgroupBasePath, subsystemPath), nil
}

func (v *v1Manager) Set(r *runc_configs.Resources) error {
	// We want to keep given resources untouched
	resourcesToSet := *r

	//Add default rules
	resourcesToSet.Devices = append(resourcesToSet.Devices, GenerateDefaultDeviceRules()...)

	// Adding current rules, see addCurrentRules's documentation for more info
	CurrentlyDefinedRules, err := v.getCurrentlyDefinedRules(v.Manager)
	if err != nil {
		return err
	}
	requestedAndCurrentRules, err := addCurrentRules(CurrentlyDefinedRules, resourcesToSet.Devices)
	if err != nil {
		return err
	}

	log.Log.V(loggingVerbosity).Infof("Adding current rules to requested for cgroup %s. Rules added: %d", V1, len(requestedAndCurrentRules)-len(r.Devices))
	resourcesToSet.Devices = requestedAndCurrentRules

	err = v.execVirtChroot(&resourcesToSet, v.controllerPaths, v.isRootless, v.GetCgroupVersion())

	return err
}

func (v *v1Manager) GetCgroupVersion() CgroupVersion {
	return V1
}

func getCurrentlyDefinedRules(runcManager runc_cgroups.Manager) ([]*devices.Rule, error) {
	devicesPath, ok := runcManager.GetPaths()["devices"]
	if !ok {
		return nil, fmt.Errorf("devices subsystem's path is not defined for this manager")
	}
	devicesPath = filepath.Join(cgroupconsts.HostCgroupBasePath, devicesPath)

	currentRulesStr, err := runc_cgroups.ReadFile(devicesPath, "devices.list")
	if err != nil {
		return nil, fmt.Errorf("error reading current rules: %v", err)
	}

	emulator, err := cgroup_devices.EmulatorFromList(bytes.NewBufferString(currentRulesStr))
	if err != nil {
		return nil, fmt.Errorf("error creating emulator out of current rules: %v", err)
	}

	currentRules, err := emulator.Rules()
	if err != nil {
		return nil, fmt.Errorf("error getting rules from emulator: %v", err)
	}

	return currentRules, nil
}

func (v *v1Manager) GetCpuSet() (string, error) {
	return getCpuSetPath(v, "cpuset.cpus")
}

func rw_filecontents(fReadPath string, fWritePath string) (err error) {
	rFile, err := os.Open(fReadPath)
	if err != nil {
		return fmt.Errorf("Open failed: %s (%v)", fReadPath, err)
	}
	defer rFile.Close()

	wFile, err := os.OpenFile(fWritePath, os.O_RDWR, 0755)
	if err != nil {
		return fmt.Errorf("OpenFile failed: %s (%v)", fWritePath, err)
	}
	defer wFile.Close()

	count, err := io.Copy(wFile, rFile)
	if err != nil {
		return fmt.Errorf("Copy filed: %s -> %s (%v), count=%d", fReadPath, fWritePath, err, count)
	}

	return nil
}

// Attach TID to cgroup. Optionally on a subcgroup of
// the pods control group (if subcgroup != nil).
func (v *v1Manager) AttachTID(subSystem string, subCgroup string, tid int) error {
	cgroupPath, err := v.GetBasePathToHostSubsystem(subSystem)
	if err != nil {
		return err
	}
	if subCgroup != "" {
		cgroupPath = filepath.Join(cgroupPath, subCgroup)
	}

	wVal := strconv.Itoa(tid)

	err = runc_cgroups.WriteFile(cgroupPath, "tasks", wVal)
	if err != nil {
		return err
	}

	return nil
}

func init_cgroup(groupPath string, newCgroupName string, subSystem string) (err error) {
	newGroupPath := filepath.Join(groupPath, newCgroupName)
	if _, err := os.Stat(newGroupPath); !errors.Is(err, os.ErrNotExist) {
		return nil
	}
	err = util.MkdirAllWithNosec(newGroupPath)
	if err != nil {
		log.Log.Infof("mkdir %s failed", newGroupPath)
		return err
	}
	if subSystem == "cpuset" {
		for _, fName := range []string{"cpuset.mems", "cpuset.cpus"} {
			rPath := filepath.Join(groupPath, fName)
			wPath := filepath.Join(newGroupPath, fName)

			err = rw_filecontents(rPath, wPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (v *v1Manager) CreateChildCgroup(name string, subSystem string) error {
	subSysPath, err := v.GetBasePathToHostSubsystem(subSystem)
	if err != nil {
		return err
	}
	err = init_cgroup(subSysPath, name, subSystem)
	if err != nil {
		log.Log.Infof("cannot create child cgroup. err: %v", err)
		return err
	}

	return nil
}

func (v *v1Manager) GetCgroupThreads() ([]int, error) {
	return getCgroupThreadsHelper(v, "tasks")
}

func (v *v1Manager) SetCpuSet(subcgroup string, cpulist []int) error {
	return setCpuSetHelper(v, subcgroup, cpulist)
}
