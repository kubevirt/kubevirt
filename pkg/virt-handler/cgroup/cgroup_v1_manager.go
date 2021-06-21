package cgroup

import (
	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	runc_fs "github.com/opencontainers/runc/libcontainer/cgroups/fs"
	runc_configs "github.com/opencontainers/runc/libcontainer/configs"
)

type v1Manager struct {
	runc_cgroups.Manager
}

func newV1Manager(config *runc_configs.Cgroup, paths map[string]string, rootless bool) (Manager, error) {
	runcManager := runc_fs.NewManager(config, paths, rootless)
	manager := v1Manager{
		runcManager,
	}

	return manager, nil
}

func (v v1Manager) GetBasePathToHostController(controller string) (string, error) {
	return getBasePathToHostController(controller)
}
