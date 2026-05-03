package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"syscall"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/asm"
	"github.com/cilium/ebpf/link"
	runc_cgroups "github.com/opencontainers/cgroups"
	_ "github.com/opencontainers/cgroups/devices"
	runc_fs "github.com/opencontainers/cgroups/fs"
	runc_fs2 "github.com/opencontainers/cgroups/fs2"
	"golang.org/x/sys/unix"

	cgroupconsts "kubevirt.io/kubevirt/pkg/virt-handler/cgroup/constants"
)

func decodeResources(marshalledResourcesHash string) (*runc_cgroups.Resources, error) {
	var unmarshalledResources runc_cgroups.Resources

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

func setCgroupResources(paths map[string]string, resources *runc_cgroups.Resources, isRootless bool, isV2 bool) error {
	config := &runc_cgroups.Cgroup{
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

func setCgroupResourcesV1(paths map[string]string, resources *runc_cgroups.Resources, config *runc_cgroups.Cgroup) error {
	return RunWithChroot(cgroupconsts.HostCgroupBasePath, func() error {
		cgroupManager, err := runc_fs.NewManager(config, paths)
		if err != nil {
			return fmt.Errorf("cannot create cgroups v1 manager. err: %v", err)
		}
		return cgroupManager.Set(resources)
	})
}

func setCgroupResourcesV2(paths map[string]string, resources *runc_cgroups.Resources, config *runc_cgroups.Cgroup) error {
	for _, path := range paths {
		if !resources.SkipDevices {
			if err := attachDummyCgroupDeviceProg(path); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to attach dummy BPF program to %s: %v\n", path, err)
			}
		}
		mgr, err := runc_fs2.NewManager(config, path)
		if err != nil {
			return fmt.Errorf("cannot create cgroups v2 manager. err: %v", err)
		}
		err = mgr.Set(resources)
		if err != nil {
			return err
		}
	}

	return nil
}

// attachDummyCgroupDeviceProg attaches a no-op allow-all BPF_CGROUP_DEVICE
// program to the cgroup. This is a workaround for a cilium/ebpf bug where
// ReplaceProgram (BPF_F_REPLACE) silently fails to replace the existing
// program, causing two programs to be attached with AND logic.
//
// The opencontainers/cgroups library only uses the broken replace path when
// exactly 1 program is attached. By adding a second program, we force it to
// use the safe "attach new, detach all old" fallback instead.
//
// Remove when cilium/ebpf fixes BPF_F_REPLACE in RawAttachProgram.
// Upstream issue: https://github.com/cilium/ebpf/issues/XXXX
func attachDummyCgroupDeviceProg(cgroupPath string) error {
	prog, err := ebpf.NewProgram(&ebpf.ProgramSpec{
		Type:    ebpf.CGroupDevice,
		License: "MIT",
		Instructions: asm.Instructions{
			asm.Mov.Imm(asm.R0, 1),
			asm.Return(),
		},
	})
	if err != nil {
		return err
	}
	defer prog.Close()

	dirFD, err := unix.Open(cgroupPath, unix.O_DIRECTORY|unix.O_RDONLY, 0o600)
	if err != nil {
		return err
	}
	defer unix.Close(dirFD)

	return link.RawAttachProgram(link.RawAttachProgramOptions{
		Target:  dirFD,
		Program: prog,
		Attach:  ebpf.AttachCGroupDevice,
		Flags:   unix.BPF_F_ALLOW_MULTI,
	})
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
