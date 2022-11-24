package cgroup

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"golang.org/x/sys/unix"

	"github.com/opencontainers/runc/libcontainer/devices"

	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	runc_configs "github.com/opencontainers/runc/libcontainer/configs"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

type CgroupVersion string

const (
	cgroupStr = "cgroup"

	procMountPoint = "/proc"

	HostRootPath       = procMountPoint + "/1/root"
	cgroupBasePath     = "/sys/fs/" + cgroupStr
	HostCgroupBasePath = HostRootPath + cgroupBasePath
)

// Templates for logging / error messages
const (
	V1 CgroupVersion = "v1"
	V2 CgroupVersion = "v2"

	loggingVerbosity = 2
)

var (
	defaultDeviceRules []*devices.Rule
)

type execVirtChrootFunc func(r *runc_configs.Resources, subsystemPaths map[string]string, rootless bool, version CgroupVersion) error
type getCurrentlyDefinedRulesFunc func(runcManager runc_cgroups.Manager) ([]*devices.Rule, error)

// addCurrentRules gets a slice of rules as a parameter and returns a new slice that contains all given rules
// and all of the rules that are currently set. This way rules that are already defined won't be deleted by this
// current request. Every old rule that is part of the new request will be overridden.
//
// For example, if the following rules are defined:
// 1) {Minor: 111, Major: 111, Allow: true}
// 2) {Minor: 222, Major: 222, Allow: true}
//
// And we get a request to enable the following rule: {Minor: 222, Major: 222, Allow: false}
// Than we expect rule (1) to stay unchanged.
func addCurrentRules(currentRules, newRules []*devices.Rule) ([]*devices.Rule, error) {
	if currentRules == nil {
		return newRules, nil
	}
	if newRules == nil {
		return nil, fmt.Errorf("new rules cannot be nil")
	}

	isCurrentRulePartOfRequestedRules := func(rule *devices.Rule, rulesSlice []*devices.Rule) bool {
		for _, ruleInSlice := range rulesSlice {
			if rule.Type == ruleInSlice.Type && rule.Minor == ruleInSlice.Minor && rule.Major == ruleInSlice.Major {
				return true
			}
		}
		return false
	}

	for _, currentRule := range currentRules {
		if !isCurrentRulePartOfRequestedRules(currentRule, newRules) {
			newRules = append(newRules, currentRule)
		}
	}

	return newRules, nil
}

func generateDeviceRulesForVMI(vmi *v1.VirtualMachineInstance, isolationRes isolation.IsolationResult) ([]*devices.Rule, error) {
	mountRoot, err := isolationRes.MountRoot()
	if err != nil {
		return nil, err
	}

	var vmiDeviceRules []*devices.Rule
	for _, volume := range vmi.Spec.Volumes {
		if volume.VolumeSource.PersistentVolumeClaim != nil {
			if volume.VolumeSource.PersistentVolumeClaim.Hotpluggable {
				continue
			}
		} else if volume.VolumeSource.DataVolume != nil {
			if volume.VolumeSource.DataVolume.Hotpluggable {
				continue
			}
		} else if volume.VolumeSource.Ephemeral != nil {
		} else {
			continue
		}
		path, err := safepath.JoinNoFollow(mountRoot, filepath.Join("dev", volume.Name))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("failed to resolve path for volume %s: %v", volume.Name, err)
		}
		fileInfo, err := safepath.StatAtNoFollow(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("failed to stat path %s: %v", path, err)
		}
		if (fileInfo.Mode() & os.ModeDevice) != 0 {
			info := fileInfo.Sys().(*syscall.Stat_t)
			deviceRule := &devices.Rule{
				Type:        devices.BlockDevice,
				Major:       int64(unix.Major(info.Rdev)),
				Minor:       int64(unix.Minor(info.Rdev)),
				Permissions: "rwm",
				Allow:       true,
			}
			log.Log.V(loggingVerbosity).Infof("device rule for volume %s: %v", volume.Name, deviceRule)
			vmiDeviceRules = append(vmiDeviceRules, deviceRule)
		}
	}
	return vmiDeviceRules, nil
}

func GenerateDefaultDeviceRules() []*devices.Rule {
	if len(defaultDeviceRules) > 0 {
		// To avoid re-computing default device rules
		return defaultDeviceRules
	}

	const toAllow = true

	var permissions devices.Permissions
	if cgroups.IsCgroup2UnifiedMode() {
		permissions = "rwm"
	} else {
		permissions = "rw"
	}

	defaultRules := []*devices.Rule{
		{ // /dev/ptmx (PTY master multiplex)
			Type:        devices.CharDevice,
			Major:       5,
			Minor:       2,
			Permissions: permissions,
			Allow:       toAllow,
		},
		{ // /dev/null (Null device)
			Type:        devices.CharDevice,
			Major:       1,
			Minor:       3,
			Permissions: permissions,
			Allow:       toAllow,
		},
		{ // /dev/kvm (hardware virtualization extensions)
			Type:        devices.CharDevice,
			Major:       10,
			Minor:       232,
			Permissions: permissions,
			Allow:       toAllow,
		},
		{ // /dev/net/tun (TAP/TUN network device)
			Type:        devices.CharDevice,
			Major:       10,
			Minor:       200,
			Permissions: permissions,
			Allow:       toAllow,
		},
		{ // /dev/vhost-net
			Type:        devices.CharDevice,
			Major:       10,
			Minor:       238,
			Permissions: permissions,
			Allow:       toAllow,
		},
	}

	// Add PTY slaves. See this for more info:
	// https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/Documentation/admin-guide/devices.txt?h=v5.14#n2084
	const ptyFirstMajor int64 = 136
	const ptyMajors int64 = 16

	for i := int64(0); i < ptyMajors; i++ {
		defaultRules = append(defaultRules,
			&devices.Rule{
				Type:        devices.CharDevice,
				Major:       ptyFirstMajor + i,
				Minor:       -1,
				Permissions: permissions,
				Allow:       toAllow,
			})
	}

	defaultDeviceRules = defaultRules

	return defaultRules
}

// execVirtChrootCgroups executes virt-chroot cgroups command to apply changes via virt-chroot.
// This is needed since high privileges are needed and root is needed to change.
func execVirtChrootCgroups(r *runc_configs.Resources, subsystemPaths map[string]string, rootless bool, version CgroupVersion) error {
	marshalledRules, err := json.Marshal(*r)
	if err != nil {
		return fmt.Errorf("failed to marshall resources. err: %v resources: %+v", err, *r)
	}

	marshalledPaths, err := json.Marshal(subsystemPaths)
	if err != nil {
		return fmt.Errorf("failed to marshall paths. err: %v resources: %+v", err, marshalledPaths)
	}

	args := []string{
		"set-cgroups-resources",
		"--subsystem-paths", base64.StdEncoding.EncodeToString(marshalledPaths),
		"--resources", base64.StdEncoding.EncodeToString(marshalledRules),
		fmt.Sprintf("--rootless=%t", rootless),
		fmt.Sprintf("--isV2=%t", version == V2),
	}

	cmd := exec.Command("virt-chroot", args...)

	log.Log.V(loggingVerbosity).Infof("setting resources for cgroup %s: %+v", version, *r)
	log.Log.V(loggingVerbosity).Infof("applying resources with virt-chroot. Full command: %s", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed running command %s, err: %v, output: %s", cmd.String(), err, output)
	}
	return nil
}

func getCgroupThreadsHelper(manager Manager, fname string) ([]int, error) {
	tIds := make([]int, 0, 10)

	subSysPath, err := manager.GetBasePathToHostSubsystem("cpuset")
	if err != nil {
		return nil, err
	}

	fh, err := os.Open(filepath.Join(subSysPath, fname))
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		line := scanner.Text()
		intVal, err := strconv.Atoi(line)
		if err != nil {
			log.Log.Errorf("error converting %s: %v", line, err)
			return nil, err
		}
		tIds = append(tIds, intVal)
	}
	if err := scanner.Err(); err != nil {
		log.Log.Errorf("error reading %s: %v", fname, err)
		return nil, err
	}
	return tIds, nil
}

// set cpus "cpusList" on the allowed CPUs. Optionally on a subcgroup of
// the pods control group (if subcgroup != nil).
func setCpuSetHelper(manager Manager, subCgroup string, cpusList []int) error {
	subSysPath, err := manager.GetBasePathToHostSubsystem("cpuset")
	if err != nil {
		return err
	}

	if subCgroup != "" {
		subSysPath = filepath.Join(subSysPath, subCgroup)
	}

	wVal := strings.Trim(strings.Replace(fmt.Sprint(cpusList), " ", ",", -1), "[]")

	return runc_cgroups.WriteFile(subSysPath, "cpuset.cpus", wVal)
}
