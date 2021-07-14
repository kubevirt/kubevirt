package cgroup

import (
	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	runc_fs "github.com/opencontainers/runc/libcontainer/cgroups/fs2"
	runc_configs "github.com/opencontainers/runc/libcontainer/configs"
)

type v2Manager struct {
	runc_cgroups.Manager
}

func newV2Manager(config *runc_configs.Cgroup, dirPath string, rootless bool) (Manager, error) {
	runcManager, err := runc_fs.NewManager(config, dirPath, rootless)
	manager := v2Manager{
		runcManager,
	}

	return manager, err
}
