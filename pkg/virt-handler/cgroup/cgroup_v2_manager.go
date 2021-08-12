package cgroup

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"

	realselinux "github.com/opencontainers/selinux/go-selinux"

	"kubevirt.io/kubevirt/pkg/virt-handler/selinux"

	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	runc_fs "github.com/opencontainers/runc/libcontainer/cgroups/fs2"
	runc_configs "github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/devices"

	"kubevirt.io/client-go/log"
)

type v2Manager struct {
	runc_cgroups.Manager
	pid        int
	dirPath    string
	isRootless bool
}

func newV2Manager(config *runc_configs.Cgroup, dirPath string, rootless bool, pid int) (Manager, error) {
	runcManager, err := runc_fs.NewManager(config, dirPath, rootless)
	manager := v2Manager{
		runcManager,
		pid,
		dirPath,
		rootless,
	}

	return &manager, err
}

func (v *v2Manager) GetBasePathToHostSubsystem(_ string) (string, error) {
	return HostCgroupBasePath, nil
}

func (v *v2Manager) Set(r *runc_configs.Resources) error {
	if err := v.setDevices(r.Devices); err != nil {
		log.Log.Infof("hotplug [SETv2] - setting device rules. err: %v", err)
		return err
	}

	resourcesWithoutDevices := getNewResourcesWithoutDevices(r)
	if !reflect.DeepEqual(resourcesWithoutDevices, runc_configs.Resources{}) {
		return v.Manager.Set(&resourcesWithoutDevices)
	}

	return nil
}

func (v *v2Manager) setDevices(deviceRules []*devices.Rule) error {
	marshalledRules, err := json.Marshal(deviceRules)
	if err != nil {
		return err
	}

	args := []string{
		"set-cgroupsv2-devices",
		"--pid", fmt.Sprintf("%d", int32(v.pid)),
		"--path", v.dirPath,
		"--rules", base64.StdEncoding.EncodeToString(marshalledRules),
		fmt.Sprintf("--rootless=%t", v.isRootless),
	}

	// #nosec
	cmd := exec.Command("virt-chroot", args...)
	log.Log.Infof("hotplug [SETv2] - args: %v", args)
	curLabel, err := realselinux.CurrentLabel()
	log.Log.Infof("hotplug [SETv2] - curLabel label: %v, err: %v", curLabel, err)
	//finalCmd, err := selinux.NewContextExecutorWithType(cmd, 12345, containerRuntimeLabel)
	finalCmd, err := selinux.NewContextExecutor(v.pid, cmd)
	//output, err := cmd.CombinedOutput()
	//if err != nil {
	//	return fmt.Errorf("failed running ><> command %s, err: %v, output: %s", cmd.String(), err, output)
	//} else {
	//	log.Log.Infof("hotplug [Run] ><> - err: %v, output: %s", cmd.String(), err, output)
	//}

	//finalCmd, err := selinux.NewContextExecutor(cmd, os.Getpid())
	if err != nil {
		// ihol3
		log.Log.Infof("hotplug [SETv2] - NewContextExecutorWithType err - %v", err)
	}

	if err = finalCmd.Execute(); err != nil {
		log.Log.Infof("hotplug [SETv2] - finalCmd.Execute() err - %v", err)
	}

	return nil
}
