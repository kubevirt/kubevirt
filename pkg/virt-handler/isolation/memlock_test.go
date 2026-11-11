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

package isolation

import (
	"fmt"

	"github.com/mitchellh/go-ps"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"golang.org/x/sys/unix"

	v1 "kubevirt.io/api/core/v1"
)

type processStub struct {
	pid    int
	ppid   int
	binary string
}

func (p processStub) Pid() int           { return p.pid }
func (p processStub) PPid() int          { return p.ppid }
func (p processStub) Executable() string { return p.binary }

func blackwellVMI() *v1.VirtualMachineInstance {
	return &v1.VirtualMachineInstance{
		Spec: v1.VirtualMachineInstanceSpec{
			Domain: v1.DomainSpec{
				Devices: v1.Devices{
					HostDevices: []v1.HostDevice{
						{Name: "gpu0", DeviceName: "nvidia.com/GB202GL_RTX_PRO_6000_BLACKWELL_SERVER_EDITION"},
						{Name: "gpu1", DeviceName: "nvidia.com/GB202GL_RTX_PRO_6000_BLACKWELL_SERVER_EDITION"},
					},
				},
			},
		},
	}
}

var _ = Describe("memory lock limits", func() {
	const (
		launcherPid  = 1000
		virtqemudPid = 1001
		qemuPid      = 77
	)

	DescribeTable("should require memory locking only for VFIO, realtime, and SEV",
		func(vmi *v1.VirtualMachineInstance, required bool) {
			Expect(requiresMemoryLock(vmi)).To(Equal(required))
		},
		Entry("ordinary VMI", &v1.VirtualMachineInstance{}, false),
		Entry("two Blackwell host devices", blackwellVMI(), true),
		Entry("GPU", &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{Devices: v1.Devices{GPUs: []v1.GPU{{Name: "gpu1"}}}},
			},
		}, true),
		Entry("SR-IOV", &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{Devices: v1.Devices{Interfaces: []v1.Interface{{
					Name:                   "sriov1",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}},
				}}}},
			},
		}, true),
		Entry("realtime", &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{CPU: &v1.CPU{Realtime: &v1.Realtime{}}},
			},
		}, true),
		Entry("SEV", &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{LaunchSecurity: &v1.LaunchSecurity{SEV: &v1.SEV{}}},
			},
		}, true),
	)

	It("should not change memlock for an ordinary VMI", func() {
		origList := listProcesses
		listProcesses = func() ([]ps.Process, error) {
			return nil, fmt.Errorf("process list should not be read")
		}
		DeferCleanup(func() { listProcesses = origList })

		orig := setMemlockRLimit
		setMemlockRLimit = func(int, uint64) error {
			return fmt.Errorf("memlock should not be changed")
		}
		DeferCleanup(func() { setMemlockRLimit = orig })

		detector := &socketBasedIsolationDetector{}
		Expect(detector.AdjustResources(&v1.VirtualMachineInstance{}, nil)).To(Succeed())
		Expect(AdjustQemuProcessMemoryLimits(nil, &v1.VirtualMachineInstance{}, nil)).To(Succeed())
	})

	It("should set unlimited memlock on the virtqemud child before startup", func() {
		origList := listProcesses
		listProcesses = func() ([]ps.Process, error) {
			return []ps.Process{
				processStub{pid: qemuPid, ppid: launcherPid, binary: "qemu-system-x86_64"},
				processStub{pid: virtqemudPid, ppid: launcherPid, binary: "virtqemud"},
				processStub{pid: 99, ppid: 5, binary: "virtqemud"},
			}, nil
		}
		DeferCleanup(func() { listProcesses = origList })

		var capturedPid int
		var capturedSize uint64
		orig := setMemlockRLimit
		setMemlockRLimit = func(pid int, size uint64) error {
			capturedPid = pid
			capturedSize = size
			return nil
		}
		DeferCleanup(func() { setMemlockRLimit = orig })

		Expect(setVirtqemudMemoryLockUnlimited(launcherPid, blackwellVMI())).To(Succeed())
		Expect(capturedPid).To(Equal(virtqemudPid))
		Expect(capturedSize).To(Equal(uint64(unix.RLIM_INFINITY)))
	})

	It("should leave memlock unchanged when virtqemud is not a child of virt-launcher", func() {
		origList := listProcesses
		listProcesses = func() ([]ps.Process, error) {
			return []ps.Process{
				processStub{pid: virtqemudPid, ppid: 5, binary: "virtqemud"},
			}, nil
		}
		DeferCleanup(func() { listProcesses = origList })

		orig := setMemlockRLimit
		setMemlockRLimit = func(int, uint64) error {
			return fmt.Errorf("memlock should not be changed")
		}
		DeferCleanup(func() { setMemlockRLimit = orig })

		Expect(setVirtqemudMemoryLockUnlimited(launcherPid, blackwellVMI())).To(Succeed())
	})

	It("should return a process-list error from the virtqemud path", func() {
		origList := listProcesses
		listProcesses = func() ([]ps.Process, error) {
			return nil, fmt.Errorf("ps failed")
		}
		DeferCleanup(func() { listProcesses = origList })

		Expect(setVirtqemudMemoryLockUnlimited(launcherPid, blackwellVMI())).To(MatchError(ContainSubstring("ps failed")))
	})

	It("should return the prlimit error from the virtqemud path", func() {
		origList := listProcesses
		listProcesses = func() ([]ps.Process, error) {
			return []ps.Process{
				processStub{pid: virtqemudPid, ppid: launcherPid, binary: "virtqemud"},
			}, nil
		}
		DeferCleanup(func() { listProcesses = origList })

		orig := setMemlockRLimit
		setMemlockRLimit = func(int, uint64) error {
			return fmt.Errorf("operation not permitted")
		}
		DeferCleanup(func() { setMemlockRLimit = orig })

		err := setVirtqemudMemoryLockUnlimited(launcherPid, blackwellVMI())
		Expect(err).To(MatchError(ContainSubstring("operation not permitted")))
		Expect(err.Error()).To(ContainSubstring("unlimited"))
	})

	It("should set unlimited memlock on the selected QEMU process", func() {
		ctrl := gomock.NewController(GinkgoT())
		DeferCleanup(ctrl.Finish)
		detector := NewMockPodIsolationDetector(ctrl)
		result := NewMockIsolationResult(ctrl)
		vmi := blackwellVMI()

		detector.EXPECT().Detect(vmi).Return(result, nil)
		result.EXPECT().GetQEMUProcess().Return(processStub{pid: qemuPid, binary: "qemu-system-x86_64"}, nil)

		var capturedPid int
		var capturedSize uint64
		orig := setMemlockRLimit
		setMemlockRLimit = func(pid int, size uint64) error {
			capturedPid = pid
			capturedSize = size
			return nil
		}
		DeferCleanup(func() { setMemlockRLimit = orig })

		Expect(AdjustQemuProcessMemoryLimits(detector, vmi, nil)).To(Succeed())
		Expect(capturedPid).To(Equal(qemuPid))
		Expect(capturedSize).To(Equal(uint64(unix.RLIM_INFINITY)))
	})

	It("should return the prlimit error from the QEMU path", func() {
		ctrl := gomock.NewController(GinkgoT())
		DeferCleanup(ctrl.Finish)
		detector := NewMockPodIsolationDetector(ctrl)
		result := NewMockIsolationResult(ctrl)
		vmi := blackwellVMI()

		detector.EXPECT().Detect(vmi).Return(result, nil)
		result.EXPECT().GetQEMUProcess().Return(processStub{pid: qemuPid, binary: "qemu-system-x86_64"}, nil)

		orig := setMemlockRLimit
		setMemlockRLimit = func(int, uint64) error {
			return fmt.Errorf("operation not permitted")
		}
		DeferCleanup(func() { setMemlockRLimit = orig })

		err := AdjustQemuProcessMemoryLimits(detector, vmi, nil)
		Expect(err).To(MatchError(ContainSubstring("operation not permitted")))
		Expect(err.Error()).To(ContainSubstring("unlimited"))
	})
})
