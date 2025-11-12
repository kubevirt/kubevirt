package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"syscall"

	runc_fs "github.com/opencontainers/runc/libcontainer/cgroups/fs"
	runc_fs2 "github.com/opencontainers/runc/libcontainer/cgroups/fs2"
	runc_configs "github.com/opencontainers/runc/libcontainer/configs"

	// Import the cgroups/devices package to register the default cgroups managers.
	_ "github.com/opencontainers/runc/libcontainer/cgroups/devices"

	cgroupconsts "kubevirt.io/kubevirt/pkg/virt-handler/cgroup/constants"
)

func decodeResources(marshalledResourcesHash string) (*runc_configs.Resources, error) {
	var unmarshalledResources runc_configs.Resources

	marshalledResources, err := base64.StdEncoding.DecodeString(marshalledResourcesHash)
	if err != nil {
		return nil, fmt.Errorf("cannot decode marshalled cgroups resources. "+
			"encoded resources: %s. err: %v", marshalledResourcesHash, err)
	}

	err = json.Unmarshal(marshalledResources, &unmarshalledResources)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshall cgroups resources. "+
			"marshalled resources: %s. err: %v", marshalledResources, err)
	}

	return &unmarshalledResources, err
}

func decodePaths(marshalledPathsHash string) (map[string]string, error) {
	var unmarshalledPaths map[string]string

	marshalledPaths, err := base64.StdEncoding.DecodeString(marshalledPathsHash)
	if err != nil {
		return nil, fmt.Errorf("cannot decode marshalled cgroups paths. "+
			"encoded paths: %s. err: %v", marshalledPathsHash, err)
	}

	err = json.Unmarshal(marshalledPaths, &unmarshalledPaths)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshall cgroups paths. "+
			"marshalled paths: %s. err: %v", marshalledPaths, err)
	}

	return unmarshalledPaths, err
}

func setCgroupResources(paths map[string]string, resources *runc_configs.Resources, isRootless bool, isV2 bool) error {
	config := &runc_configs.Cgroup{
		Path:      cgroupconsts.HostCgroupBasePath,
		Resources: resources,
		Rootless:  isRootless,
	}

	var err error

	if isV2 {
		err = setCgroupResourcesV2(paths, resources, config)
	} else {
		err = setCgroupResourcesV1(paths, resources, config)
	}

	if err != nil {
		return fmt.Errorf("cannot set cgroup resources. err: %v", err)
	}

	return nil
}

func setCgroupResourcesV1(paths map[string]string, resources *runc_configs.Resources, config *runc_configs.Cgroup) error {
	return RunWithChroot(cgroupconsts.HostCgroupBasePath, func() error {
		cgroupManager, err := runc_fs.NewManager(config, paths)
		if err != nil {
			return fmt.Errorf("cannot create cgroups v1 manager. err: %v", err)
		}
		return cgroupManager.Set(resources)
	})
}

func setCgroupResourcesV2(paths map[string]string, resources *runc_configs.Resources, config *runc_configs.Cgroup) error {
	cgroupDirPath := paths[""]

	cgroupManager, err := runc_fs2.NewManager(config, cgroupDirPath)
	if err != nil {
		return fmt.Errorf("cannot create cgroups v2 manager. err: %v", err)
	}

	err = cgroupManager.Set(resources)
	return err
}

// RunWithChroot changes the root directory (via "chroot") into newPath, then
// runs toRun function. When the function finishes, changes back the root directory
// to the original one that
func RunWithChroot(newPath string, toRun func() error) error {
	// Ensure no other goroutines are effected by this
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	originalRoot, err := os.Open("/")
	if err != nil {
		return fmt.Errorf("failed to run with chroot - failed to open root directory. error: %v", err)
	}
	defer originalRoot.Close()

	err = syscall.Chroot(newPath)
	if err != nil {
		return fmt.Errorf("failed to chroot into \"%s\". error: %v", newPath, err)
	}

	changeRootToOriginal := func() {
		_ = originalRoot.Chdir()
		_ = syscall.Chroot(".")
	}
	defer changeRootToOriginal()

	err = toRun()
	return err
}
