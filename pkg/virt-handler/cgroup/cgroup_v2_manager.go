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
	"errors"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	runc_fs "github.com/opencontainers/runc/libcontainer/cgroups/fs2"
	runc_configs "github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/devices"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
)

type v2Manager struct {
	runc_cgroups.Manager
	dirPath        string
	isRootless     bool
	deviceRules    []*devices.Rule
	execVirtChroot execVirtChrootFunc
}

func newV2Manager(config *runc_configs.Cgroup, dirPath string) (Manager, error) {
	runcManager, err := runc_fs.NewManager(config, dirPath)
	if err != nil {
		return nil, err
	}

	return newCustomizedV2Manager(runcManager, config.Rootless, config.Resources.Devices, execVirtChrootCgroups)
}

func newCustomizedV2Manager(
	runcManager runc_cgroups.Manager,
	isRootless bool,
	deviceRules []*devices.Rule,
	execVirtChroot execVirtChrootFunc,
) (Manager, error) {
	manager := v2Manager{
		runcManager,
		runcManager.GetPaths()[""],
		isRootless,
		append(deviceRules, GenerateDefaultDeviceRules()...),
		execVirtChroot,
	}

	return &manager, nil
}

func (v *v2Manager) GetBasePathToHostSubsystem(_ string) (string, error) {
	return v.dirPath, nil
}

func (v *v2Manager) Set(r *runc_configs.Resources) error {
	// We want to keep given resources untouched
	resourcesToSet := *r

	rulesToSet, err := addCurrentRules(v.deviceRules, resourcesToSet.Devices)
	if err != nil {
		return err
	}
	v.deviceRules = rulesToSet
	resourcesToSet.Devices = rulesToSet
	for _, rule := range rulesToSet {
		if rule == nil {
			continue
		}
		log.Log.V(5).Infof("cgroupsv2 device allowlist: rule after appending current+new: type: %d permissions: %s allow: %t major: %d minor: %d", rule.Type, rule.Permissions, rule.Allow, rule.Major, rule.Minor)
	}

	subsystemPaths := map[string]string{
		"target": v.dirPath,
	}
	if targetDir, parentPath := filepath.Base(v.dirPath), path.Dir(v.dirPath); targetDir == "container" && strings.HasSuffix(parentPath, ".scope") {
		// This is needed for crun based installations for a brief period of time
		// crun will eventually stop configuring both cgroups
		subsystemPaths["parent"] = parentPath
	}
	log.Log.V(5).Infof("cgroupsv2 device allowlist: paths passed to virt-chroot: %s", subsystemPaths)

	return v.execVirtChroot(&resourcesToSet, subsystemPaths, v.isRootless, v.GetCgroupVersion())
}

func (v *v2Manager) GetCgroupVersion() CgroupVersion {
	return V2
}

func (v *v2Manager) CreateChildCgroup(name string, subSystem string) error {
	subSysPath, err := v.GetBasePathToHostSubsystem(subSystem)
	if err != nil {
		return err
	}

	newGroupPath := filepath.Join(subSysPath, name)
	if _, err = os.Stat(newGroupPath); !errors.Is(err, os.ErrNotExist) {
		return nil
	}

	// Write "+subsystem" to cgroup.subtree_control
	wVal := "+" + subSystem
	err = runc_cgroups.WriteFile(subSysPath, "cgroup.subtree_control", wVal)
	if err != nil {
		return err
	}

	// Create new cgroup directory
	err = util.MkdirAllWithNosec(newGroupPath)
	if err != nil {
		log.Log.Infof("mkdir %s failed", newGroupPath)
		return err
	}

	// Enable threaded cgroup controller
	err = runc_cgroups.WriteFile(newGroupPath, "cgroup.type", "threaded")
	if err != nil {
		return err
	}

	// Write "+subsystem" to newcgroup/cgroup.subtree_control
	wVal = "+" + subSystem
	err = runc_cgroups.WriteFile(newGroupPath, "cgroup.subtree_control", wVal)
	if err != nil {
		return err
	}
	return nil
}

// Attach TID to cgroup. Optionally on a subcgroup of
// the pods control group (if subcgroup != nil).
func (v *v2Manager) AttachTID(subSystem string, subCgroup string, tid int) error {
	cgroupPath, err := v.GetBasePathToHostSubsystem(subSystem)
	if err != nil {
		return err
	}
	if subCgroup != "" {
		cgroupPath = filepath.Join(cgroupPath, subCgroup)
	}

	wVal := strconv.Itoa(tid)

	err = runc_cgroups.WriteFile(cgroupPath, "cgroup.threads", wVal)
	if err != nil {
		return err
	}

	return nil
}

func (v *v2Manager) GetCgroupThreads() ([]int, error) {
	return getCgroupThreadsHelper(v, "cgroup.threads")
}

func (v *v2Manager) SetCpuSet(subcgroup string, cpulist []int) error {
	return setCpuSetHelper(v, subcgroup, cpulist)
}
