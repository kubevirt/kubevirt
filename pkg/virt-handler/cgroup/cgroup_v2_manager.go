package cgroup

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"

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
		return logAndReturnErrorWithSprintfIfNotNil(err, errApplyingDeviceRule, err)
	}

	resourcesWithoutDevices := getNewResourcesWithoutDevices(r)
	if areResourcesEmpty(&resourcesWithoutDevices) {
		return nil
	}

	err := v.Manager.Set(&resourcesWithoutDevices)
	return logAndReturnErrorWithSprintfIfNotNil(err, errApplyingOtherRules, err)
}

func (v *v2Manager) setDevices(deviceRules []*devices.Rule) error {
	const loggingVerbosity = 3

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

	cmd := exec.Command("virt-chroot", args...)
	for _, rule := range deviceRules {
		log.Log.V(loggingVerbosity).Infof(settingDeviceRule, v.GetCgroupVersion(), *rule)
	}
	log.Log.V(loggingVerbosity).Infof("applying device rules with virt-chroot. Full command: %s", cmd.String())
	finalCmd, err := selinux.NewContextExecutor(v.pid, cmd)
	if err != nil {
		return logAndReturnErrorWithSprintfIfNotNil(err, "failed creating new context executor. err: %v, pid: %d, cmd: %s", err, v.pid, cmd.String())
	}

	if err = finalCmd.Execute(); err != nil {
		return logAndReturnErrorWithSprintfIfNotNil(err, "failed setting device rule through virt-chroot. "+
			"full command %s, err: %v", cmd.String(), err)
	}

	return nil
}

func (v *v2Manager) GetCgroupVersion() string {
	return "v2"
}
