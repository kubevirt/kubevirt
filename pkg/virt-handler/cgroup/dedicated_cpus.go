package cgroup

import (
	"fmt"
	"strings"

	virtutil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

func GetDedicatedCpuCgroupManager(vmi *v1.VirtualMachineInstance) (Manager, error) {
	// Find dedicated cgroup slice and make a manager for it
	isolationRes, err := isolation.NewSocketBasedIsolationDetector(virtutil.VirtShareDir).Detect(vmi)
	if err != nil {
		return nil, err
	}
	dedicatedCgroupPid, dedicatedCgroupPidFound := isolationRes.DedocatedCpuContainerPid()
	if !dedicatedCgroupPidFound {
		return nil, fmt.Errorf("cannot find dedicated cpu container pid for vmi %s/%s", vmi.Namespace, vmi.Name)
	}

	dedicatedCpusCgroupManager, err := NewManagerFromPid(dedicatedCgroupPid)
	if err != nil {
		return dedicatedCpusCgroupManager, err
	}

	return dedicatedCpusCgroupManager, nil
}

func getQemuKvmPid(computeCgroupManager Manager) (int, error) {
	qemuKvmFilter := func(s string) bool { return strings.Contains(s, "qemu-kvm") }
	qemuKvmPids, err := computeCgroupManager.GetCgroupThreadsWithFilter(qemuKvmFilter)

	if err != nil {
		return -1, err
	} else if len(qemuKvmPids) == 0 {
		err := fmt.Errorf("qemu process was not found")
		return -1, err
	} else if len(qemuKvmPids) > 1 {
		err := fmt.Errorf("more than 1 qemu process is found within the compute container")
		return -1, err
	}

	log.Log.V(detailedLogVerbosity).Infof("found qemu-kvm pid: %+v", qemuKvmPids[0])
	return qemuKvmPids[0], nil
}

func getVcpuTids(computeCgroupManager Manager) ([]int, error) {
	vcpusFilter := func(s string) bool { return strings.Contains(s, "CPU ") && strings.Contains(s, "KVM") }
	vcpuTids, err := computeCgroupManager.GetCgroupThreadsWithFilter(vcpusFilter)
	if err != nil {
		return nil, err
	}

	return vcpuTids, nil
}
