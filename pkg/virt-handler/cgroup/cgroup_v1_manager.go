package cgroup

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	cgroup_devices "github.com/opencontainers/runc/libcontainer/cgroups/devices"
	"github.com/opencontainers/runc/libcontainer/devices"

	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	runc_fs "github.com/opencontainers/runc/libcontainer/cgroups/fs"
	runc_configs "github.com/opencontainers/runc/libcontainer/configs"

	"kubevirt.io/kubevirt/pkg/util"
)

type v1Manager struct {
	runc_cgroups.Manager
	controllerPaths          map[string]string
	isRootless               bool
	execVirtChroot           execVirtChrootFunc
	getCurrentlyDefinedRules getCurrentlyDefinedRulesFunc
}

func newV1Manager(controllerPaths map[string]string) (Manager, error) {
	config := getDeafulCgroupConfig()

	runcManager, err := runc_fs.NewManager(config, controllerPaths)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize new cgroup manager. err: %v", err)
	}
	return newCustomizedV1Manager(runcManager, config.Rootless, execVirtChrootCgroups, getCurrentlyDefinedRules)
}

func newCustomizedV1Manager(runcManager runc_cgroups.Manager, isRootless bool,
	execVirtChroot execVirtChrootFunc, getCurrentlyDefinedRules getCurrentlyDefinedRulesFunc) (Manager, error) {
	manager := v1Manager{
		runcManager,
		runcManager.GetPaths(),
		isRootless,
		execVirtChroot,
		getCurrentlyDefinedRules,
	}

	return &manager, nil
}

func getConcreteManagerStructV1(m Manager) (*v1Manager, error) {
	v1Struct, ok := m.(*v1Manager)
	if !ok {
		return nil, fmt.Errorf(castingToConcreteTypeFailedErrFmt, V1)
	}
	return v1Struct, nil
}

func (v *v1Manager) GetBasePathToHostSubsystem(subsystem string) (string, error) {
	subsystemPath := v.Path(subsystem)
	if subsystemPath == "" {
		return "", fmt.Errorf("controller %s does not exist", subsystem)
	}
	return filepath.Join(HostCgroupBasePath, subsystemPath), nil
}

func (v *v1Manager) Set(r *runc_configs.Resources) error {
	// We want to keep given resources untouched
	resourcesToSet := *r

	//Add default rules
	resourcesToSet.Devices = append(resourcesToSet.Devices, GenerateDefaultDeviceRules()...)

	// Adding current rules, see addCurrentRules's documentation for more info
	CurrentlyDefinedRules, err := v.getCurrentlyDefinedRules(v.Manager)
	if err != nil {
		return err
	}
	requestedAndCurrentRules, err := addCurrentRules(CurrentlyDefinedRules, resourcesToSet.Devices)
	if err != nil {
		return err
	}

	log.Log.V(loggingVerbosity).Infof("Adding current rules to requested for cgroup %s. Rules added: %d", V1, len(requestedAndCurrentRules)-len(r.Devices))
	resourcesToSet.Devices = requestedAndCurrentRules

	err = v.execVirtChroot(&resourcesToSet, v.controllerPaths, v.isRootless, v.GetCgroupVersion())

	return err
}

func (v *v1Manager) GetCgroupVersion() CgroupVersion {
	return V1
}

func getCurrentlyDefinedRules(runcManager runc_cgroups.Manager) ([]*devices.Rule, error) {
	devicesPath, ok := runcManager.GetPaths()["devices"]
	if !ok {
		return nil, fmt.Errorf("devices subsystem's path is not defined for this manager")
	}
	devicesPath = filepath.Join(HostCgroupBasePath, devicesPath)

	currentRulesStr, err := runc_cgroups.ReadFile(devicesPath, "devices.list")
	if err != nil {
		return nil, fmt.Errorf("error reading current rules: %v", err)
	}

	emulator, err := cgroup_devices.EmulatorFromList(bytes.NewBufferString(currentRulesStr))
	if err != nil {
		return nil, fmt.Errorf("error creating emulator out of current rules: %v", err)
	}

	currentRules, err := emulator.Rules()
	if err != nil {
		return nil, fmt.Errorf("error getting rules from emulator: %v", err)
	}

	return currentRules, nil
}

func (v *v1Manager) GetCpuSet() ([]int, error) {
	return getCpuSetPath(v, "cpuset.cpus")
}

func (v *v1Manager) attachTask(id int, subSystem string, taskType TaskType) error {
	subSystemPath, err := v.GetBasePathToHostSubsystem(subSystem)
	if err != nil {
		return err
	}

	var targetFile string
	switch taskType {
	case Thread:
		targetFile = v1ThreadsFilename
	case Process:
		targetFile = procsFilename
	default:
		return fmt.Errorf("task type %v is not valid", taskType)
	}

	return attachTask(id, subSystemPath, targetFile)
}

func (v *v1Manager) CreateChildCgroup(name string, subSystems ...string) (Manager, error) {
	newControllerPaths := make(map[string]string, len(subSystems))

	for _, subSystem := range subSystems {
		subSysPath, err := v.GetBasePathToHostSubsystem(subSystem)
		if err != nil {
			return nil, err
		}

		newGroupPath := filepath.Join(subSysPath, name)
		if _, err := os.Stat(newGroupPath); !errors.Is(err, os.ErrNotExist) {
			newControllerPaths[subSystem] = newGroupPath
			log.Log.V(detailedLogVerbosity).Infof(cgroupAlreadyExistsErrFmt, newGroupPath)
			continue
		}

		err = util.MkdirAllWithNosec(newGroupPath)
		if err != nil {
			return nil, err
		}

		newControllerPaths[subSystem] = newGroupPath
	}

	return NewManagerFromPath(newControllerPaths)
}

func (v *v1Manager) GetCgroupThreadsWithFilter(filter func(string) bool) ([]int, error) {
	return getCgroupThreadsHelper(v, v1ThreadsFilename, filter)
}

func (v *v1Manager) GetCgroupThreads() ([]int, error) {
	return v.GetCgroupThreadsWithFilter(nil)
}

func (v *v1Manager) GetCgroupProcsWithFilter(filter func(string) bool) ([]int, error) {
	return getCgroupThreadsHelper(v, procsFilename, filter)
}

func (v *v1Manager) GetCgroupProcs() ([]int, error) {
	return v.GetCgroupProcsWithFilter(nil)
}

func (v *v1Manager) SetCpuSet(cpulist []int) error {
	return setCpuSetHelper(v, cpulist)
}

func (v *v1Manager) MakeThreaded() error {
	// cgroup v1 does not have the notion of a "threaded" cgroup.
	return nil
}

func (v *v1Manager) InitializeEmulatorContainer(vmi *v1.VirtualMachineInstance) error {
	err := initEmulatorContainerHierarchy(vmi, CgroupSubsystemCpuset)
	if err != nil {
		return err
	}

	rootManager, ambassadorManager, emulatorManager, vcpuManager, hkManager, err := getEmulatorContainerCgroups(vmi, CgroupSubsystemCpuset)
	if err != nil {
		return err
	}

	err = setDedicatedCpusToEmulatorContainer(v, rootManager, ambassadorManager, emulatorManager, vcpuManager, hkManager, V1)
	if err != nil {
		return err
	}

	// memory must always be set for v1
	log.Log.V(detailedLogVerbosity).Infof("populating cgroup memory to all children")
	rootCgroupPath, err := rootManager.GetBasePathToHostSubsystem(CgroupSubsystemCpuset)
	if err != nil {
		return err
	}

	const memsFilepath = "cpuset.mems"
	rootCgroupMemory, err := runc_cgroups.ReadFile(rootCgroupPath, memsFilepath)
	if err != nil {
		return err
	}

	err = runc_cgroups.WriteFile(filepath.Join(rootCgroupPath, EmulatorContainerCgroupAmbassador), memsFilepath, rootCgroupMemory)
	if err != nil {
		return err
	}

	err = runc_cgroups.WriteFile(filepath.Join(rootCgroupPath, EmulatorContainerCgroupEmulator), memsFilepath, rootCgroupMemory)
	if err != nil {
		return err
	}

	err = runc_cgroups.WriteFile(filepath.Join(rootCgroupPath, EmulatorContainerCgroupEmulator, EmulatorContainerCgroupVcpu), memsFilepath, rootCgroupMemory)
	if err != nil {
		return err
	}

	err = runc_cgroups.WriteFile(filepath.Join(rootCgroupPath, EmulatorContainerCgroupEmulator, EmulatorContainerCgroupHousekeeping), memsFilepath, rootCgroupMemory)
	if err != nil {
		return err
	}

	err = attachTasksToEmulatorContainer(vmi, v, rootManager, ambassadorManager, emulatorManager, vcpuManager, hkManager, v.getAttachProcFunc(CgroupSubsystemCpuset), v.getAttachThreadFunc(CgroupSubsystemCpuset))
	if err != nil {
		return err
	}

	return nil
}

func (v *v1Manager) getAttachProcFunc(subsystem string) attachTaskFunc {
	return func(manager Manager, id int) error {
		if concreteManager, err := getConcreteManagerStructV1(manager); err == nil {
			err = concreteManager.attachTask(id, subsystem, Process)
			if err != nil {
				return err
			}
		} else {
			return err
		}

		return nil
	}
}

func (v *v1Manager) getAttachThreadFunc(subsystem string) attachTaskFunc {
	return func(manager Manager, id int) error {
		if concreteManager, err := getConcreteManagerStructV1(manager); err == nil {
			err = concreteManager.attachTask(id, subsystem, Thread)
			if err != nil {
				return err
			}
		} else {
			return err
		}

		return nil
	}
}
