package cgroup

import (
	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/configs"
)

const (
	ProcMountPoint   = "/proc"
	CgroupMountPoint = "/sys/fs/cgroup"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

// DEFINE INTERFACE HERE

//// ihol3 name
//type DeviceManager interface {
//	SetDeviceRule(path string, rule devices.Rule) error
//}
//
//type cpuManager interface {
//	GetCpuSetPath() string
//}

// ihol3 Change name?
type Manager interface {
	//DeviceManager
	//cpuManager

	cgroups.Manager

	// GetControllersAndPaths ... returns key: controller, value: path.
	//GetControllersAndPaths(pid int) (map[string]string, error)

	// GetControllerPath ...
	//GetControllerPath(controller string) string
}

func NewManager(config *configs.Cgroup, dirPath string, paths map[string]string, rootless bool) (Manager, error) {
	if cgroups.IsCgroup2UnifiedMode() {
		return newV2Manager(config, dirPath, rootless)
	} else {
		return newV1Manager(config, paths, rootless)
	}
}
