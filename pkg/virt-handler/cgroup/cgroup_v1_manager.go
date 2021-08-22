package cgroup

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"

	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	cgroupdevices "github.com/opencontainers/runc/libcontainer/cgroups/devices"
	runc_fs "github.com/opencontainers/runc/libcontainer/cgroups/fs"
	runc_configs "github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/devices"

	"kubevirt.io/client-go/log"
)

type v1Manager struct {
	runc_cgroups.Manager
}

func newV1Manager(config *runc_configs.Cgroup, paths map[string]string, rootless bool) (Manager, error) {
	runcManager := runc_fs.NewManager(config, paths, rootless)
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
	// handle devices separately
	for _, deviceRule := range r.Devices {
		applyDeviceRule := func() error { return v.SetDeviceRule(deviceRule) }
		log.Log.Infof(settingDeviceRule, v.GetCgroupVersion(), deviceRule)
		err := RunWithChroot(HostRootPath, applyDeviceRule)
		return logAndReturnErrorWithSprintfIfNotNil(err, errApplyingDeviceRule, err)
	}

	resourcesWithoutDevices := getNewResourcesWithoutDevices(r)
	if areResourcesEmpty(&resourcesWithoutDevices) {
		return nil
	}

	err := v.Manager.Set(&resourcesWithoutDevices)
	return logAndReturnErrorWithSprintfIfNotNil(err, errApplyingOtherRules, err)
}

// SetDeviceRule sets a new cgroup device rule.
//
// This function overrides runc's logic as their code is currently broken. In their code, they use a "transition"
// function which supposed to calculate the minimum delta of rules to apply in order to support the given rule.
// This function however always returns an empty delta.
//
// The following issue has been opened to rnuc: https://github.com/opencontainers/runc/issues/3141
// TODO: when this issue is resolved, this function needs to be entirely deleted and we should use runc's logic instead
func (v *v1Manager) SetDeviceRule(rule *devices.Rule) error {
	const loggingVerbosity = 3
	loadEmulator := func(path string) (*cgroupdevices.Emulator, error) {
		list, err := runc_cgroups.ReadFile(path, "devices.list")
		if err != nil {
			return nil, err
		}
		return cgroupdevices.EmulatorFromList(bytes.NewBufferString(list))
	}

	devicesPath, ok := v.GetPaths()["devices"]
	if !ok {
		return fmt.Errorf("devices subsystem's path is not defined for this manager")
	}

	devicesPath = filepath.Join(cgroupBasePath, "devices", devicesPath)
	log.Log.V(loggingVerbosity).Infof("setting device rule (%v) for path: %s", *rule, devicesPath)

	// Generate two emulators, one for the target state of the cgroup and one
	// for the requested state by the user.
	expectedEmulator, err := loadEmulator(devicesPath)
	if err != nil {
		return err
	}
	_ = expectedEmulator.Apply(*rule)

	file := "devices.deny"
	if rule.Allow {
		file = "devices.allow"
	}

	// This is the main workaround here - we write the new rule directly into cgroup without calculating the
	// shortest delta.
	if err := runc_cgroups.WriteFile(devicesPath, file, rule.CgroupString()); err != nil {
		return err
	}
	log.Log.V(loggingVerbosity).Infof("writing device rule into cgroup. rule: %s, err: %v", rule.CgroupString(), err)

	//Final safety check -- ensure that the resulting state is what was
	//requested. This is only really correct for white-lists, but for
	//black-lists we can at least check that the cgroup is in the right mode.
	resultEmulator, err := loadEmulator(devicesPath)
	if err != nil {
		return err
	}

	log.Log.V(loggingVerbosity).Errorf("error - expected the result cgroups state does not match."+
		"expected: %v, result: %v", expectedEmulator, resultEmulator)
	if !expectedEmulator.IsBlacklist() && !reflect.DeepEqual(resultEmulator, expectedEmulator) {
		return fmt.Errorf("resulting devices cgroup doesn't precisely match target")
	} else if expectedEmulator.IsBlacklist() != resultEmulator.IsBlacklist() {
		return fmt.Errorf("resulting devices cgroup doesn't match target mode")
	}

	return nil
}

func (v *v1Manager) GetCgroupVersion() string {
	return "v1"
}
