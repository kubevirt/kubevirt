package cgroup

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"

	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/cgroups/cgroups/fscommon"
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

func (v *v1Manager) GetBasePathToHostController(controller string) (string, error) {
	return getBasePathToHostController(controller)
}

func (v *v1Manager) Set(r *runc_configs.Resources) error {
	// handle devices separately
	for _, deviceRule := range r.Devices {
		if err := v.SetDeviceRule(deviceRule); err != nil {
			return fmt.Errorf("error occured while applying device rule: %v", err)
		} else {
			log.Log.Infof("hotplug [SET] - setting device rule: %v", deviceRule)
		}
	}

	resourcesWithoutDevices := *r
	resourcesWithoutDevices.Devices = nil

	log.Log.Infof("hotplug [SET] - setting though libcontainer...")
	err := v.Manager.Set(&resourcesWithoutDevices)
	log.Log.Infof("hotplug [SET] - err: %v", err)
	return err
}

// ihol3 doc that this will be deprecated once libcontainer's "transition" is not broken...
func (v *v1Manager) SetDeviceRule(rule *devices.Rule) error {
	path, ok := v.GetPaths()["devices"]
	path = filepath.Join("/sys/fs/cgroup/devices/", path)
	if !ok {
		//ihol3
	}
	log.Log.Infof("hotplug [SetDeviceRule]: path == %v", path)

	// Generate two emulators, one for the target state of the cgroup and one
	// for the requested state by the user.
	target, err := loadEmulator(path)
	if err != nil {
		return err
	}

	log.Log.Infof("hotplug [SetDeviceRule]: target == %v", target)
	target.Apply(*rule)
	log.Log.Infof("hotplug [SetDeviceRule]: target after applying == %v", target)
	log.Log.Infof("hotplug [SetDeviceRule]: new rule == %v", *rule)
	file := "devices.deny"
	if rule.Allow {
		file = "devices.allow"
	}
	content, err := fscommon.ReadFile(path, "devices.list")
	log.Log.Infof("hotplug [SetDeviceRule]: ReadFile - err: %v, Content: %s", err, content)

	if err := fscommon.WriteFile(path, file, rule.CgroupString()); err != nil {
		return err
	}
	log.Log.Infof("hotplug [SetDeviceRule]: WriteFile - ERR: %v", err)
	log.Log.Infof("hotplug [SetDeviceRule]: WriteFile - Rule: %s", rule.CgroupString())

	//Compute the minimal set of transition rules needed to achieve the
	//requested state.
	//transitionRules, err := transition(target, target)
	//if err != nil {
	//	return err
	//}
	//
	//log.Log.Infof("hotplug [SetDeviceRule]: len(transitionRules) == %v", len(transitionRules))
	//log.Log.Infof("hotplug [SetDeviceRule]: transitionRules == %v", transitionRules)
	//for _, rule := range transitionRules {
	//	file := "devices.deny"
	//	if rule.Allow {
	//		file = "devices.allow"
	//	}
	//	log.Log.Infof("hotplug [SetDeviceRule]: path == %v, file == %v, rule.CgroupString() == %v, ", path, file, rule.CgroupString())
	//	if err := fscommon.WriteFile(path, file, rule.CgroupString()); err != nil {
	//		return err
	//	}
	//}
	//
	//if len(transitionRules) == 0 {
	//	log.Log.Infof("hotplug [SetDeviceRule]: len(transitionRules) == 0")
	//
	//	if err := fscommon.WriteFile(path, file, rule.CgroupString()); err != nil {
	//		return err
	//	}
	//	log.Log.Infof("hotplug [SetDeviceRule]: WriteFile - ERR: %v", err)
	//	log.Log.Infof("hotplug [SetDeviceRule]: WriteFile - Rule: %s", rule.CgroupString())
	//}

	content, err = fscommon.ReadFile(path, "devices.list")
	log.Log.Infof("hotplug [SetDeviceRule]: ReadFile - err: %v, Content: %s", err, content)

	//Final safety check -- ensure that the resulting state is what was
	//requested. This is only really correct for white-lists, but for
	//black-lists we can at least check that the cgroup is in the right mode.
	//
	//This safety-check is skipped for the unit tests because we cannot
	//currently mock devices.list correctly.
	currentAfter, err := loadEmulator(path)
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

func loadEmulator(path string) (*cgroupdevices.Emulator, error) {
	list, err := fscommon.ReadFile(path, "devices.list")
	if err != nil {
		return nil, err
	}
	return cgroupdevices.EmulatorFromList(bytes.NewBufferString(list))
}

//func buildEmulator(rules []*devices.Rule) (*cgroupdevices.Emulator, error) {
//	// This defaults to a white-list -- which is what we want!
//	emu := &cgroupdevices.Emulator{}
//	for _, rule := range rules {
//		if err := emu.Apply(*rule); err != nil {
//			return nil, err
//		}
//	}
//	return emu, nil
//}
