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
		err := RunWithChroot(HostRootPath, applyDeviceRule)
		if err != nil {
			return fmt.Errorf("error occured while applying device rule: %v", err)
		}
		log.Log.Infof("hotplug [SET] - setting device rule: %v", deviceRule)
	}

	resourcesWithoutDevices := getNewResourcesWithoutDevices(r)

	log.Log.Infof("hotplug [SET] - setting though libcontainer...")
	err := v.Manager.Set(&resourcesWithoutDevices)
	log.Log.Infof("hotplug [SET] - err: %v", err)
	return err
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
	log.Log.Infof("hotplug [SetDeviceRule]: path == %v", devicesPath)

	// Generate two emulators, one for the target state of the cgroup and one
	// for the requested state by the user.
	target, err := loadEmulator(devicesPath)
	if err != nil {
		return err
	}
	_ = target.Apply(*rule)

	log.Log.Infof("hotplug [SetDeviceRule]: new rule == %v", *rule)
	file := "devices.deny"
	if rule.Allow {
		file = "devices.allow"
	}

	content, err := runc_cgroups.ReadFile(devicesPath, "devices.list")
	log.Log.Infof("hotplug [SetDeviceRule]: ReadFile - err: %v, Content: %s", err, content)

	if err := runc_cgroups.WriteFile(devicesPath, file, rule.CgroupString()); err != nil {
		return err
	}
	log.Log.Infof("hotplug [SetDeviceRule]: WriteFile - ERR: %v", err)
	log.Log.Infof("hotplug [SetDeviceRule]: WriteFile - Rule: %s", rule.CgroupString())

	content, err = runc_cgroups.ReadFile(devicesPath, "devices.list")
	log.Log.Infof("hotplug [SetDeviceRule]: ReadFile - err: %v, Content: %s", err, content)

	//Final safety check -- ensure that the resulting state is what was
	//requested. This is only really correct for white-lists, but for
	//black-lists we can at least check that the cgroup is in the right mode.
	currentAfter, err := loadEmulator(devicesPath)
	log.Log.Infof("hotplug [SetDeviceRule]: target after == %v", currentAfter)
	if err != nil {
		return err
	}
	if !target.IsBlacklist() && !reflect.DeepEqual(currentAfter, target) {
		return fmt.Errorf("resulting devices cgroup doesn't precisely match target")
	} else if target.IsBlacklist() != currentAfter.IsBlacklist() {
		return fmt.Errorf("resulting devices cgroup doesn't match target mode")
	}

	return nil
}
