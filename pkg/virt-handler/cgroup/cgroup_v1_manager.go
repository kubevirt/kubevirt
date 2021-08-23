package cgroup

import (
	"fmt"
	"path/filepath"

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
	err := RunWithChroot(HostCgroupBasePath, func() error {
		err := v.Manager.Set(r)
		return err
	})

	return logAndReturnErrorWithSprintfIfNotNil(err, errApplyingOtherRules, err)
}

func (v *v1Manager) GetCgroupVersion() string {
	return "v1"
}
