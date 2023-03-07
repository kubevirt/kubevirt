package cgroup

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"

	virtutil "kubevirt.io/kubevirt/pkg/util"

	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

var noIdsFoundErr = fmt.Errorf("cannot find any ids")

func getEmulatorContainerManager(vmi *v1.VirtualMachineInstance, subsystems ...string) (Manager, error) {
	isolationRes, err := isolation.NewSocketBasedIsolationDetector(virtutil.VirtShareDir).Detect(vmi)
	if err != nil {
		return nil, err
	}

	emulatorAmbassadorPid, emulatorAmbassadorPidFound := isolationRes.EmulatorAmbassadorPid()
	if !emulatorAmbassadorPidFound {
		return nil, fmt.Errorf("cannot find emulator container pid for vmi %s/%s", vmi.Namespace, vmi.Name)
	}

	emulatorContainerManager, err := NewManagerFromPid(emulatorAmbassadorPid)
	if err != nil {
		return emulatorContainerManager, err
	}

	// if the ambassador process had already been moved, we should provide the parent cgroup instead
	controllerPaths := map[string]string{}
	for _, subsystem := range subsystems {
		path, err := emulatorContainerManager.GetBasePathToHostSubsystem(subsystem)
		if err != nil {
			return nil, fmt.Errorf("cannot find base path for abmassador cgroup with subsystem %s: %v", subsystem, err)
		}

		if filepath.Base(path) == EmulatorContainerCgroupAmbassador {
			controllerPaths[subsystem] = filepath.Dir(path)
		}
	}

	if len(controllerPaths) > 0 {
		log.Log.V(detailedLogVerbosity).Infof("ambassador process already moved, creating its parent cgroup instead")
		return NewManagerFromPath(controllerPaths)
	}

	return emulatorContainerManager, nil
}

func getQemuKvmPid(computeCgroupManager Manager) ([]int, error) {
	qemuKvmFilter := func(s string) bool {
		for _, qemuExecutableName := range isolation.QemuProcessExecutables {
			if strings.Contains(s, qemuExecutableName) {
				return true
			}
		}
		return false
	}
	qemuKvmPids, err := computeCgroupManager.GetCgroupProcsWithFilter(qemuKvmFilter)

	if err != nil {
		return nil, err
	} else if len(qemuKvmPids) == 0 {
		return nil, fmt.Errorf("qemu pid cannot be found: %w", noIdsFoundErr)
	} else if len(qemuKvmPids) > 1 {
		err := fmt.Errorf("more than 1 qemu process is found within the compute container")
		return nil, err
	}

	log.Log.V(detailedLogVerbosity).Infof("found qemu-kvm pid: %+v", qemuKvmPids[0])
	return qemuKvmPids, nil
}

func getVcpuTids(computeCgroupManager Manager) ([]int, error) {
	vcpusFilter := func(s string) bool { return strings.Contains(s, "CPU ") && strings.Contains(s, "KVM") }
	vcpuTids, err := computeCgroupManager.GetCgroupThreadsWithFilter(vcpusFilter)
	if err != nil {
		return nil, err
	}

	if len(vcpuTids) == 0 {
		return vcpuTids, fmt.Errorf("could not found any vCPU TIDs: %w", noIdsFoundErr)
	}

	return vcpuTids, nil
}

func getEmulatorAmbassadorPid(rootEmulatorManager Manager) ([]int, error) {
	ambassadorProcFilter := func(s string) bool { return strings.Contains(s, "emulator-") }

	ambassadorPids, err := rootEmulatorManager.GetCgroupProcsWithFilter(ambassadorProcFilter)

	if err != nil {
		return nil, err
	} else if len(ambassadorPids) == 0 {
		return nil, fmt.Errorf("ambassador pid cannot be found: %w", noIdsFoundErr)
	} else if len(ambassadorPids) > 1 {
		err := fmt.Errorf("more than 1 ambassador process is found within the compute container")
		return nil, err
	}

	return ambassadorPids, nil
}

func attachToCgroup(sourceCgroup, targetCgroup Manager, attachFunc attachTaskFunc, getIdsFunc getManagerIdsFunc, taskType TaskType) error {
	ids, err := getIdsFunc(sourceCgroup)

	if errors.Is(err, noIdsFoundErr) {
		log.Log.V(detailedLogVerbosity).Infof("pids not found with error: %v", err)
		return nil
	}

	if err != nil {
		return err
	}

	for _, id := range ids {
		err = attachFunc(targetCgroup, id)
		if err != nil {
			return err
		}
	}

	err = verifyIdInCgroup(targetCgroup, ids, taskType)
	if err != nil {
		return err
	}

	return nil
}

func verifyIdInCgroup(manager Manager, ids []int, taskType TaskType) error {
	var targetIds []int
	var err error

	switch taskType {
	case Process:
		targetIds, err = manager.GetCgroupProcs()
	case Thread:
		targetIds, err = manager.GetCgroupThreads()
	default:
		return fmt.Errorf("task type %d is unknown", taskType)
	}

	if err != nil {
		return err
	}

	targetIdsMap := map[int]struct{}{}
	for _, targetId := range targetIds {
		targetIdsMap[targetId] = struct{}{}
	}

	for _, id := range ids {
		if _, exists := targetIdsMap[id]; !exists {
			cgroupPath, _ := manager.GetBasePathToHostSubsystem(CgroupSubsystemCpuset)
			return fmt.Errorf("cannot find id %d in target cgroup %s", id, cgroupPath)
		}
	}

	return nil
}

func initEmulatorContainerHierarchy(vmi *v1.VirtualMachineInstance, subsystems ...string) error {
	if !vmi.IsCPUDedicated() {
		return fmt.Errorf(vmiNotDedicatedErrFmt, vmi.Name)
	}

	log.Log.V(detailedLogVerbosity).Infof("initializing emulator container for vmi %s/%s", vmi.Namespace, vmi.Name)

	emulatorContainerRootManager, err := getEmulatorContainerManager(vmi, subsystems...)
	if err != nil {
		return err
	}

	log.Log.V(detailedLogVerbosity).Infof("initializing cgroup hierarchy")

	_, err = emulatorContainerRootManager.CreateChildCgroup(EmulatorContainerCgroupAmbassador, subsystems...)
	if err != nil {
		return err
	}

	emulatorCgroupManager, err := emulatorContainerRootManager.CreateChildCgroup(EmulatorContainerCgroupEmulator, subsystems...)
	if err != nil {
		return err
	}

	_, err = emulatorCgroupManager.CreateChildCgroup(EmulatorContainerCgroupVcpu, subsystems...)
	if err != nil {
		return err
	}

	_, err = emulatorCgroupManager.CreateChildCgroup(EmulatorContainerCgroupHousekeeping, subsystems...)
	if err != nil {
		return err
	}

	log.Log.V(detailedLogVerbosity).Infof(initializedEmulatorContainerSuccessfully, vmi.Namespace, vmi.Name)

	return nil
}

// This function assumes the cgroup hierarchy is already initialized
func getEmulatorContainerCgroups(vmi *v1.VirtualMachineInstance, subsystems ...string) (rootManager, ambassadorManager, emulatorManager, vcpuManager, hkManager Manager, err error) {
	ambassadorControllerPaths := make(map[string]string, len(subsystems))
	emulatorControllerPaths := make(map[string]string, len(subsystems))
	vcpusControllerPaths := make(map[string]string, len(subsystems))
	hkControllerPaths := make(map[string]string, len(subsystems))

	rootManager, err = getEmulatorContainerManager(vmi, subsystems...)
	if err != nil {
		return
	}

	doesPathExists := func(path string) error {
		exists, err := diskutils.FileExists(path)
		if exists {
			return nil
		}

		if !exists || err != nil {
			if err == nil {
				err = fmt.Errorf("file or folder does not exist")
			}
			return fmt.Errorf("cannot find cgroup path in emulator container: %s. err: %v", path, err)
		}

		return nil
	}

	var basePath string
	for _, subsystem := range subsystems {
		basePath, err = rootManager.GetBasePathToHostSubsystem(subsystem)
		if err != nil {
			return
		}

		log.Log.V(detailedLogVerbosity).Infof("validating that emulator container hierarchy is set up correctly")

		ambassadorPath := filepath.Join(basePath, EmulatorContainerCgroupAmbassador)
		if err = doesPathExists(ambassadorPath); err != nil {
			return
		}

		emulatorPath := filepath.Join(basePath, EmulatorContainerCgroupEmulator)
		if err = doesPathExists(emulatorPath); err != nil {
			return
		}

		vcpuPath := filepath.Join(basePath, EmulatorContainerCgroupEmulator, EmulatorContainerCgroupVcpu)
		if err = doesPathExists(vcpuPath); err != nil {
			return
		}

		hkPath := filepath.Join(basePath, EmulatorContainerCgroupEmulator, EmulatorContainerCgroupHousekeeping)
		if err = doesPathExists(hkPath); err != nil {
			return
		}

		log.Log.V(detailedLogVerbosity).Infof("emulator container hierarchy exists")

		ambassadorControllerPaths[subsystem] = ambassadorPath
		emulatorControllerPaths[subsystem] = emulatorPath
		vcpusControllerPaths[subsystem] = vcpuPath
		hkControllerPaths[subsystem] = hkPath
	}

	if ambassadorManager, err = NewManagerFromPath(ambassadorControllerPaths); err != nil {
		return
	}
	if emulatorManager, err = NewManagerFromPath(emulatorControllerPaths); err != nil {
		return
	}
	if vcpuManager, err = NewManagerFromPath(vcpusControllerPaths); err != nil {
		return
	}
	if hkManager, err = NewManagerFromPath(hkControllerPaths); err != nil {
		return
	}

	return
}

func setDedicatedCpusToEmulatorContainer(computeManager, rootManager, ambassadorManager, emulatorManager, vcpuManager, hkManager Manager, cgroupVersion CgroupVersion) error {
	// Get all relevant cpusets
	rootCpuset, err := rootManager.GetCpuSet()
	if err != nil {
		return err
	}

	dedicatedCpuset := closeIntSlice(rootCpuset)

	sharedCpuset, err := computeManager.GetCpuSet()
	if err != nil {
		return err
	}

	// the private method is used since need to know the allocated CPUs (stored in "cpuset.cpus")
	// and no the "effective" amount (""cpuset.cpus.effective").
	vcpuCpuset, err := getCpuSetPath(vcpuManager, "cpuset.cpus")
	if err != nil {
		return err
	}

	log.Log.V(detailedLogVerbosity).Infof("emulator root cpuset: %v", rootManager)
	log.Log.V(detailedLogVerbosity).Infof("vcpu cpuset: %v", vcpuCpuset)

	if isSubsetSlice(sharedCpuset, rootCpuset) {
		// means that the root cgroup's cpuset is already mutated.
		// therefore, we'll try to get the dedicated cpuset from vcpu cgroup, if set already
		if len(vcpuCpuset) > 0 {
			log.Log.V(detailedLogVerbosity).Infof("dedicated cpuset is updated from vcpu cpuset")
			dedicatedCpuset = vcpuCpuset
		}
	}

	assignDedicatedCpusForVcpus := func() error {
		if len(vcpuCpuset) == 0 {
			log.Log.V(detailedLogVerbosity).Infof("assigning dedicated cpuset to vcpu cgroup")
			return vcpuManager.SetCpuSet(dedicatedCpuset)
		}
		return nil
	}

	if cgroupVersion == V2 {
		err = assignDedicatedCpusForVcpus()
		if err != nil {
			return err
		}
	}

	sharedAndDedicatedCpuset := mergeIntSlices(sharedCpuset, dedicatedCpuset)

	// Assign cpusets to cgroups
	if !areIntSlicesEqual(rootCpuset, sharedAndDedicatedCpuset) {
		log.Log.V(detailedLogVerbosity).Infof("assigning shared and dedicated cpuset to emulator container root cgroup")
		err = rootManager.SetCpuSet(sharedAndDedicatedCpuset)
		if err != nil {
			return err
		}
	}

	log.Log.V(detailedLogVerbosity).Infof("assigning shared cpuset to ambassador cgroup")
	err = ambassadorManager.SetCpuSet(sharedCpuset)
	if err != nil {
		return err
	}

	log.Log.V(detailedLogVerbosity).Infof("assigning shared and dedicated cpuset to emulator cgroup")
	err = emulatorManager.SetCpuSet(sharedAndDedicatedCpuset)
	if err != nil {
		return err
	}

	log.Log.V(detailedLogVerbosity).Infof("assigning shared cpuset to housekeeping cgroup")
	err = hkManager.SetCpuSet(sharedCpuset)
	if err != nil {
		return err
	}

	if cgroupVersion == V1 {
		err = assignDedicatedCpusForVcpus()
		if err != nil {
			return err
		}
	}

	log.Log.V(detailedLogVerbosity).Infof(handledDedicatedCpusSuccessfully)

	return nil
}

func attachTasksToEmulatorContainer(vmi *v1.VirtualMachineInstance, computeManager, rootManager, ambassadorManager, emulatorManager, vcpuManager, hkManager Manager, attachProc, attachThread attachTaskFunc) error {
	log.Log.V(detailedLogVerbosity).Infof("attaching ambassador")
	err := attachToCgroup(rootManager, ambassadorManager, attachProc, getEmulatorAmbassadorPid, Process)
	if err != nil {
		return err
	}

	log.Log.V(detailedLogVerbosity).Infof("attaching emulator process")
	err = attachToCgroup(computeManager, emulatorManager, attachProc, getQemuKvmPid, Process)
	if err != nil {
		return err
	}

	log.Log.V(detailedLogVerbosity).Infof("making cgroups threaded")
	err = vcpuManager.MakeThreaded()
	if err != nil {
		return err
	}

	err = hkManager.MakeThreaded()
	if err != nil {
		return err
	}

	log.Log.V(detailedLogVerbosity).Infof("attaching threads")
	err = attachToCgroup(emulatorManager, vcpuManager, attachThread, getVcpuTids, Thread)
	if err != nil {
		return err
	}

	err = attachToCgroup(emulatorManager, hkManager, attachThread, func(manager Manager) ([]int, error) { return manager.GetCgroupThreads() }, Thread)
	if err != nil {
		return err
	}

	vcpuTids, err := vcpuManager.GetCgroupThreads()
	if err != nil {
		log.Log.Warningf("could not find vcpu count for vmi %s/%s", vmi.Namespace, vmi.Name)
	}
	if len(vcpuTids) != int(vmi.Spec.Domain.CPU.Cores) {
		log.Log.Warningf("number of vCPU threads found (%d) is different than the CPU number (%d) on vmi %s/%s", len(vcpuTids), int(vmi.Spec.Domain.CPU.Cores), vmi.Namespace, vmi.Name)
	}

	log.Log.V(detailedLogVerbosity).Infof(attachedEmulatorTasksSuccessfully)

	return nil
}
