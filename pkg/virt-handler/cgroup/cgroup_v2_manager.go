package cgroup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	runc_fs "github.com/opencontainers/runc/libcontainer/cgroups/fs2"
	runc_configs "github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/devices"

	"kubevirt.io/kubevirt/pkg/util"
)

var rulesPerPid = make(map[string][]*devices.Rule)

type cgroupV2File string

const (
	subtreeControl cgroupV2File = "cgroup.subtree_control"
	cgroupType     cgroupV2File = "cgroup.type"
)

type cgroupV2Type string

const (
	domain         cgroupV2Type = "domain"
	threaded       cgroupV2Type = "threaded"
	domainThreaded cgroupV2Type = "domainThreaded"
	domainInvalid  cgroupV2Type = "domainInvalid"
)

type cgroupV2SubtreeCtrlAction string

const (
	subtreeCtrlAdd    cgroupV2SubtreeCtrlAction = "+"
	subtreeCtrlRemove cgroupV2SubtreeCtrlAction = "-"
)

type v2Manager struct {
	runc_cgroups.Manager
	dirPath        string
	isRootless     bool
	execVirtChroot execVirtChrootFunc
}

func newV2Manager(dirPath string) (Manager, error) {
	config := getDeafulCgroupConfig()

	runcManager, err := runc_fs.NewManager(config, dirPath)
	if err != nil {
		return nil, err
	}

	return newCustomizedV2Manager(runcManager, config.Rootless, execVirtChrootCgroups)
}

func newCustomizedV2Manager(runcManager runc_cgroups.Manager, isRootless bool, execVirtChroot execVirtChrootFunc) (Manager, error) {
	manager := v2Manager{
		runcManager,
		runcManager.GetPaths()[""],
		isRootless,
		execVirtChroot,
	}

	return &manager, nil
}

func getConcreteManagerStructV2(m Manager) (*v2Manager, error) {
	v2Struct, ok := m.(*v2Manager)
	if !ok {
		return nil, fmt.Errorf(castingToConcreteTypeFailedErrFmt, V2)
	}
	return v2Struct, nil
}

func (v *v2Manager) GetBasePathToHostSubsystem(_ string) (string, error) {
	return v.dirPath, nil
}

func (v *v2Manager) Set(r *runc_configs.Resources) error {
	// We want to keep given resources untouched
	resourcesToSet := *r

	//Add default rules
	resourcesToSet.Devices = append(resourcesToSet.Devices, GenerateDefaultDeviceRules()...)

	rulesToSet, err := addCurrentRules(rulesPerPid[v.dirPath], resourcesToSet.Devices)
	if err != nil {
		return err
	}
	rulesPerPid[v.dirPath] = rulesToSet
	resourcesToSet.Devices = rulesToSet

	err = v.execVirtChroot(&resourcesToSet, map[string]string{"": v.dirPath}, v.isRootless, v.GetCgroupVersion())
	return err
}

func (v *v2Manager) GetCgroupVersion() CgroupVersion {
	return V2
}

func (v *v2Manager) GetCpuSet() ([]int, error) {
	return getCpuSetPath(v, "cpuset.cpus.effective")
}

func (v *v2Manager) mutateSubtreeControl(subSystems string, action cgroupV2SubtreeCtrlAction) error {
	return runc_cgroups.WriteFile(v.dirPath, string(subtreeControl), fmt.Sprintf("%s%s", string(action), subSystems))
}

func (v *v2Manager) CreateChildCgroup(name string, subSystems ...string) (Manager, error) {
	newGroupPath := filepath.Join(v.dirPath, name)
	log.Log.V(detailedLogVerbosity).Infof("Creating new child cgroup at %s", newGroupPath)

	if _, err := os.Stat(newGroupPath); !errors.Is(err, os.ErrNotExist) {
		log.Log.V(detailedLogVerbosity).Infof(cgroupAlreadyExistsErrFmt, newGroupPath)
		return NewManagerFromPath(map[string]string{"": newGroupPath})
	}

	// Remove unnecessary subsystems from subtree control. This is crucial in order to make the cgroup threaded
	curSubtreeSubsystems, err := runc_cgroups.ReadFile(v.dirPath, string(subtreeControl))
	if err != nil {
		return nil, err
	}

	for _, curSubtreeSubsystem := range strings.Split(curSubtreeSubsystems, " ") {
		if curSubtreeSubsystem == "" {
			continue
		}

		for _, subSystem := range subSystems {
			if curSubtreeSubsystem == subSystem {
				continue
			}
		}

		err := v.mutateSubtreeControl(curSubtreeSubsystem, subtreeCtrlRemove)
		if err != nil {
			return nil, err
		}
	}

	// Configure the given subsystems to be inherited by the new cgroup
	for _, subSystem := range subSystems {
		err := v.mutateSubtreeControl(subSystem, subtreeCtrlAdd)
		if err != nil {
			return nil, err
		}
	}

	// Create a new cgroup directory
	err = util.MkdirAllWithNosec(newGroupPath)
	if err != nil {
		return nil, fmt.Errorf("failed creating cgroup directory %s: %v", newGroupPath, err)
	}

	newManager, err := NewManagerFromPath(map[string]string{"": newGroupPath})
	if err != nil {
		return newManager, err
	}

	return newManager, nil
}

func (v *v2Manager) attachTask(id int, taskType TaskType) error {
	var targetFile string
	switch taskType {
	case Thread:
		targetFile = v2ThreadsFilename
	case Process:
		targetFile = v2ProcsFilename
	default:
		return fmt.Errorf("task type %v is not valid", taskType)
	}

	return attachTask(id, v.dirPath, targetFile)
}

func (v *v2Manager) GetCgroupThreadsWithFilter(filter func(string) bool) ([]int, error) {
	return getCgroupThreadsHelper(v, v2ThreadsFilename, filter)
}

func (v *v2Manager) GetCgroupThreads() ([]int, error) {
	return v.GetCgroupThreadsWithFilter(nil)
}

func (v *v2Manager) SetCpuSet(cpulist []int) error {
	return setCpuSetHelper(v, cpulist)
}

func (v *v2Manager) MakeThreaded() error {
	// Ideally, this implementation needs to reside in runc's repository.
	// An issue is opened to track that: https://github.com/opencontainers/runc/issues/3690.

	const (
		cgTypeFile   = "cgroup.type"
		typeThreaded = "threaded"
	)

	cgroupType, err := runc_cgroups.ReadFile(v.dirPath, cgTypeFile)
	if err != nil {
		return err
	}
	cgroupType = strings.TrimSpace(cgroupType)

	if cgroupType == typeThreaded {
		log.Log.V(detailedLogVerbosity).Infof("cgroup %s already threaded", v.dirPath)
		return nil
	}

	err = runc_cgroups.WriteFile(v.dirPath, cgTypeFile, typeThreaded)
	if err != nil {
		return err
	}

	cgroupType, err = runc_cgroups.ReadFile(v.dirPath, cgTypeFile)
	if err != nil {
		return err
	}
	cgroupType = strings.TrimSpace(cgroupType)

	if cgroupType != typeThreaded {
		return fmt.Errorf("could not change cgroup type (%s) to %s", cgroupType, typeThreaded)
	}

	return nil
}

func (v *v2Manager) HandleDedicatedCpus(vmi *v1.VirtualMachineInstance) error {
	if !vmi.IsCPUDedicated() {
		return fmt.Errorf(vmiNotDedicatedErrFmt, vmi.Name)
	}

	dedicatedCpusCgroupManager, err := GetDedicatedCpuCgroupManager(vmi)
	if err != nil {
		return err
	}

	qemuKvmPid, err := getQemuKvmPid(v)
	if err != nil {
		return err
	}

	if dedicatedCpusCgroupManagerConcrete, err := getConcreteManagerStructV2(dedicatedCpusCgroupManager); err == nil {
		err = dedicatedCpusCgroupManagerConcrete.attachTask(qemuKvmPid, Process)
		if err != nil {
			return err
		}
	} else {
		return err
	}

	housekeepingCgroupManager, err := dedicatedCpusCgroupManager.CreateChildCgroup(V2housekeepingContainerName, CgroupSubsystemCpuset)
	if err != nil {
		return err
	}

	err = housekeepingCgroupManager.MakeThreaded()
	if err != nil {
		return err
	}

	vcpuTids, err := getVcpuTids(v)
	if err != nil {
		return err
	}

	for _, vcpuTid := range vcpuTids {
		if housekeepingCgroupManagerConcrete, err := getConcreteManagerStructV2(housekeepingCgroupManager); err == nil {
			err = housekeepingCgroupManagerConcrete.attachTask(vcpuTid, Thread)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	housekeepingCpuset, err := housekeepingCgroupManager.GetCpuSet()
	if err != nil {
		return err
	}

	dedicatedCpuset, err := dedicatedCpusCgroupManager.GetCpuSet()
	if err != nil {
		return err
	}

	if len(dedicatedCpuset) < 2 {
		return fmt.Errorf("cpuset is expected to be at least of length 2 (for 1 vCPU and 1 extra code): %v", err)
	}

	if len(housekeepingCpuset) == 1 {
		log.Log.V(detailedLogVerbosity).Infof("housekeeping cpuset already configured")
		return nil
	}

	housekeepingCore := dedicatedCpuset[len(dedicatedCpuset)-1:]
	log.Log.V(detailedLogVerbosity).Infof("housekeeping core: %d", housekeepingCore[0])
	err = housekeepingCgroupManager.SetCpuSet(housekeepingCore)
	if err != nil {
		return err
	}

	log.Log.V(detailedLogVerbosity).Infof(handledDedicatedCpusSuccessfully, vmi.Name)

	return nil
}
