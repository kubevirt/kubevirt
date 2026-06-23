/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package kvm

import (
	"fmt"

	cgroups "github.com/opencontainers/cgroups"

	ps "github.com/mitchellh/go-ps"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type fakeProcess struct {
	pid  int
	ppid int
	comm string
}

func (f *fakeProcess) Pid() int           { return f.pid }
func (f *fakeProcess) PPid() int          { return f.ppid }
func (f *fakeProcess) Executable() string { return f.comm }

type fakeCgroupManager struct {
	threads      []int
	childCgroups []string
	cpuSetCalls  map[string][]int
	attachedTIDs map[string][]int
	createErr    error
	threadsErr   error
	setCpuErr    error
	attachErr    error
}

func newFakeCgroupManager(threads []int) *fakeCgroupManager {
	return &fakeCgroupManager{
		threads:      threads,
		cpuSetCalls:  make(map[string][]int),
		attachedTIDs: make(map[string][]int),
	}
}

func (f *fakeCgroupManager) Set(_ *cgroups.Resources) error { return nil }
func (f *fakeCgroupManager) GetBasePathToHostSubsystem(_ string) (string, error) {
	return "", nil
}
func (f *fakeCgroupManager) GetCgroupVersion() cgroup.CgroupVersion {
	return cgroup.CgroupVersion("2")
}
func (f *fakeCgroupManager) GetCpuSet() (string, error) { return "0-3", nil }
func (f *fakeCgroupManager) SetCpuSet(subcgroup string, cpulist []int) error {
	if f.setCpuErr != nil {
		return f.setCpuErr
	}
	f.cpuSetCalls[subcgroup] = cpulist
	return nil
}
func (f *fakeCgroupManager) CreateChildCgroup(name string, _ string) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.childCgroups = append(f.childCgroups, name)
	return nil
}
func (f *fakeCgroupManager) AttachTID(_ string, subCgroup string, tid int) error {
	if f.attachErr != nil {
		return f.attachErr
	}
	f.attachedTIDs[subCgroup] = append(f.attachedTIDs[subCgroup], tid)
	return nil
}
func (f *fakeCgroupManager) GetCgroupThreads() ([]int, error) {
	return f.threads, f.threadsErr
}

func makeDomain(emulatorPin string, vhostCPUSet string) *api.Domain {
	d := &api.Domain{}
	if emulatorPin != "" {
		d.Spec.CPUTune = &api.CPUTune{
			EmulatorPin: &api.CPUEmulatorPin{CPUSet: emulatorPin},
		}
	}
	d.Spec.Metadata.KubeVirt.VhostCPUSet = vhostCPUSet
	return d
}

func makeVMI(isolateEmulator, isolateVhost bool) *v1.VirtualMachineInstance {
	vmi := &v1.VirtualMachineInstance{}
	vmi.Name = "test-vmi"
	vmi.Namespace = "default"
	vmi.Spec.Domain.CPU = &v1.CPU{
		DedicatedCPUPlacement: true,
		IsolateEmulatorThread: isolateEmulator,
	}
	if isolateVhost {
		policy := v1.VhostThreadPolicyShared
		vmi.Spec.Domain.CPU.VhostThreadPolicy = &policy
	}
	return vmi
}

var _ = Describe("configureHousekeepingCgroup with vhost isolation", func() {
	var (
		runtime         *KvmVirtRuntime
		origFindProcess func(int) (ps.Process, error)
		processMap      map[int]*fakeProcess
	)

	BeforeEach(func() {
		runtime = &KvmVirtRuntime{
			logger: log.DefaultLogger(),
		}
		origFindProcess = findProcess
		processMap = make(map[int]*fakeProcess)

		findProcess = func(pid int) (ps.Process, error) {
			if p, ok := processMap[pid]; ok {
				return p, nil
			}
			return nil, nil
		}
	})

	AfterEach(func() {
		findProcess = origFindProcess
	})

	It("should move vhost thread to vhost cgroup and others to housekeeping", func() {
		processMap[100] = &fakeProcess{pid: 100, comm: "CPU 0/KVM"}
		processMap[101] = &fakeProcess{pid: 101, comm: "CPU 1/KVM"}
		processMap[200] = &fakeProcess{pid: 200, comm: "vhost-99"}
		processMap[300] = &fakeProcess{pid: 300, comm: "worker"}

		mgr := newFakeCgroupManager([]int{100, 101, 200, 300})
		vmi := makeVMI(true, true)
		domain := makeDomain("3", "4")

		err := runtime.configureHousekeepingCgroup(vmi, mgr, domain)
		Expect(err).ToNot(HaveOccurred())

		Expect(mgr.childCgroups).To(ContainElement("vhost"))
		Expect(mgr.cpuSetCalls["vhost"]).To(Equal([]int{4}))
		Expect(mgr.attachedTIDs["vhost"]).To(ConsistOf(200))
		Expect(mgr.attachedTIDs["housekeeping"]).To(ConsistOf(300))
	})

	It("should succeed (no-op) when vhost thread not yet in cgroup", func() {
		processMap[100] = &fakeProcess{pid: 100, comm: "CPU 0/KVM"}
		processMap[300] = &fakeProcess{pid: 300, comm: "worker"}

		mgr := newFakeCgroupManager([]int{100, 300})
		vmi := makeVMI(true, true)
		domain := makeDomain("3", "4")

		err := runtime.configureHousekeepingCgroup(vmi, mgr, domain)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should succeed (no-op for vhost) when VhostCPUSet is empty", func() {
		processMap[100] = &fakeProcess{pid: 100, comm: "CPU 0/KVM"}
		processMap[200] = &fakeProcess{pid: 200, comm: "vhost-99"}
		processMap[300] = &fakeProcess{pid: 300, comm: "worker"}

		mgr := newFakeCgroupManager([]int{100, 200, 300})
		vmi := makeVMI(true, true)
		domain := makeDomain("3", "")

		err := runtime.configureHousekeepingCgroup(vmi, mgr, domain)
		Expect(err).ToNot(HaveOccurred())

		Expect(mgr.attachedTIDs["vhost"]).To(BeEmpty())
		Expect(mgr.attachedTIDs["housekeeping"]).ToNot(ContainElement(200))
	})

	It("should not isolate vhost when VhostThreadPolicy is nil", func() {
		processMap[100] = &fakeProcess{pid: 100, comm: "CPU 0/KVM"}
		processMap[200] = &fakeProcess{pid: 200, comm: "vhost-99"}
		processMap[300] = &fakeProcess{pid: 300, comm: "worker"}

		mgr := newFakeCgroupManager([]int{100, 200, 300})
		vmi := makeVMI(true, false)
		domain := makeDomain("3", "")

		err := runtime.configureHousekeepingCgroup(vmi, mgr, domain)
		Expect(err).ToNot(HaveOccurred())

		Expect(mgr.childCgroups).ToNot(ContainElement("vhost"))
		Expect(mgr.attachedTIDs["housekeeping"]).To(ConsistOf(200, 300))
	})

	It("should succeed even when AttachTID to vhost cgroup fails (next sync retries)", func() {
		processMap[100] = &fakeProcess{pid: 100, comm: "CPU 0/KVM"}
		processMap[200] = &fakeProcess{pid: 200, comm: "vhost-99"}

		mgr := newFakeCgroupManager([]int{100, 200})
		mgr.attachErr = fmt.Errorf("permission denied")
		vmi := makeVMI(true, true)
		domain := makeDomain("3", "4")

		err := runtime.configureHousekeepingCgroup(vmi, mgr, domain)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should skip vhost thread from housekeeping even when vhost cgroup attach fails", func() {
		processMap[100] = &fakeProcess{pid: 100, comm: "CPU 0/KVM"}
		processMap[200] = &fakeProcess{pid: 200, comm: "vhost-99"}
		processMap[300] = &fakeProcess{pid: 300, comm: "worker"}

		vmi := makeVMI(true, true)
		domain := makeDomain("3", "4")

		customMgr := &selectiveFailCgroupManager{
			fakeCgroupManager: *newFakeCgroupManager([]int{100, 200, 300}),
			failSubcgroup:     "vhost",
		}

		err := runtime.configureHousekeepingCgroup(vmi, customMgr, domain)
		Expect(err).ToNot(HaveOccurred())

		Expect(customMgr.attachedTIDs["housekeeping"]).To(ConsistOf(300))
		Expect(customMgr.attachedTIDs["housekeeping"]).ToNot(ContainElement(200))
	})

	It("should return nil early when domain is nil", func() {
		mgr := newFakeCgroupManager(nil)
		vmi := makeVMI(true, true)

		err := runtime.configureHousekeepingCgroup(vmi, mgr, nil)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return nil early when EmulatorPin is nil", func() {
		mgr := newFakeCgroupManager(nil)
		vmi := makeVMI(true, true)
		domain := makeDomain("", "4")

		err := runtime.configureHousekeepingCgroup(vmi, mgr, domain)
		Expect(err).ToNot(HaveOccurred())
	})
})

type selectiveFailCgroupManager struct {
	fakeCgroupManager
	failSubcgroup string
}

func (s *selectiveFailCgroupManager) AttachTID(subSystem string, subCgroup string, tid int) error {
	if subCgroup == s.failSubcgroup {
		return fmt.Errorf("attach to %s failed", subCgroup)
	}
	s.attachedTIDs[subCgroup] = append(s.attachedTIDs[subCgroup], tid)
	return nil
}
