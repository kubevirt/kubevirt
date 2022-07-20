package cgroup

import (
	"bytes"
	"fmt"
	"path/filepath"

	"kubevirt.io/client-go/log"

	cgroup_devices "github.com/opencontainers/runc/libcontainer/cgroups/devices"
	"github.com/opencontainers/runc/libcontainer/devices"

	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	runc_fs "github.com/opencontainers/runc/libcontainer/cgroups/fs"
	runc_configs "github.com/opencontainers/runc/libcontainer/configs"
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
	return filepath.Join(HostCgroupBasePath, subsystemPath), nil
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
	devicesPath = filepath.Join(HostCgroupBasePath, devicesPath)

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
