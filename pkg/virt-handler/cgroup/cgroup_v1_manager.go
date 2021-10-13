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
}

func newV1Manager(config *runc_configs.Cgroup, controllerPaths map[string]string, rootless bool) (Manager, error) {
	runcManager := runc_fs.NewManager(config, controllerPaths, rootless)
	manager := v1Manager{
		runcManager,
	}

	return &manager, nil
}

func (v *v1Manager) GetBasePathToHostSubsystem(subsystem string) (string, error) {
	if v.Path(subsystem) == "" {
		return "", fmt.Errorf("controller %s does not exist", subsystem)
	}
	return filepath.Join(HostCgroupBasePath, subsystem), nil
}

func (v *v1Manager) Set(r *runc_configs.Resources) error {
	// We want to keep given resources untouched
	resourcesToSet := *r

	// Adding current rules, see addCurrentRules's documentation for more info
	CurrentlyDefinedRules, err := v.getCurrentlyDefinedRules()
	if err != nil {
		return err
	}
	requestedAndCurrentRules, err := addCurrentRules(CurrentlyDefinedRules, r.Devices)
	if err != nil {
		return err
	}

	log.Log.V(loggingVerbosity).Infof("Adding current rules to requested for cgroup %s. Rules added: %d", v1Str, len(requestedAndCurrentRules)-len(r.Devices))
	resourcesToSet.Devices = requestedAndCurrentRules

	err = RunWithChroot(HostCgroupBasePath, func() error {
		err := v.Manager.Set(&resourcesToSet)
		return err
	})

	return logAndReturnErrorWithSprintfIfNotNil(err, errApplyingNonDeviceRules, err)
}

func (v *v1Manager) GetCgroupVersion() string {
	return v1Str
}

func (v *v1Manager) getCurrentlyDefinedRules() ([]*devices.Rule, error) {
	devicesPath, ok := v.GetPaths()["devices"]
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
