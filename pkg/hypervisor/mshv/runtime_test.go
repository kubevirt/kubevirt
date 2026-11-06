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

package mshv_test

import (
	"github.com/mitchellh/go-ps"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"golang.org/x/sys/unix"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hypervisor/common"
	"kubevirt.io/kubevirt/pkg/hypervisor/mshv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

type processStub struct {
	pid    int
	ppid   int
	binary string
}

func (p processStub) Pid() int           { return p.pid }
func (p processStub) PPid() int          { return p.ppid }
func (p processStub) Executable() string { return p.binary }

var _ = Describe("AdjustResources", func() {
	var (
		ctrl                  *gomock.Controller
		mockIsolationDetector *isolation.MockPodIsolationDetector
		mockIsolationResult   *isolation.MockIsolationResult
		runtime               *mshv.MshvVirtRuntime
		config                *v1.KubeVirtConfiguration
	)

	const launcherPid = 1000

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockIsolationDetector = isolation.NewMockPodIsolationDetector(ctrl)
		mockIsolationResult = isolation.NewMockIsolationResult(ctrl)
		runtime = mshv.NewMshvVirtRuntime(mockIsolationDetector, nil)
		config = &v1.KubeVirtConfiguration{}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("guard conditions", func() {
		It("should return nil without calling Detect when no features require memlock", func() {
			vmi := libvmi.New(libvmi.WithMemoryRequest("1Gi"))
			Expect(runtime.AdjustResources(vmi, config)).To(Succeed())
		})

		DescribeTable("should call Detect when memlock is required",
			func(vmi *v1.VirtualMachineInstance) {
				mockIsolationDetector.EXPECT().Detect(vmi).Return(mockIsolationResult, nil)
				mockIsolationResult.EXPECT().Pid().Return(launcherPid)

				origListProcesses := mshv.ListProcesses
				mshv.ListProcesses = func() ([]ps.Process, error) {
					return []ps.Process{}, nil
				}
				DeferCleanup(func() { mshv.ListProcesses = origListProcesses })

				Expect(runtime.AdjustResources(vmi, config)).To(Succeed())
			},
			Entry("with VFIO GPU",
				libvmi.New(
					libvmi.WithMemoryRequest("1Gi"),
					libvmi.WithGPU(v1.GPU{Name: "gpu1"}),
				),
			),
			Entry("with realtime",
				libvmi.New(
					libvmi.WithMemoryRequest("1Gi"),
					libvmi.WithRealtimeMask(""),
				),
			),
			Entry("with SEV",
				libvmi.New(
					libvmi.WithMemoryRequest("1Gi"),
					libvmi.WithSEV(false, false),
				),
			),
			Entry("with RequiresLockingMemory",
				libvmi.New(
					libvmi.WithMemoryRequest("1Gi"),
					func(vmi *v1.VirtualMachineInstance) {
						vmi.Spec.Domain.Memory = &v1.Memory{
							ReservedOverhead: &v1.ReservedOverhead{
								MemLock: pointer.P(v1.MemLockRequirement(v1.MemLockRequired)),
							},
						}
					},
				),
			),
		)
	})

	Context("memlock rlimit", func() {
		It("should set unlimited memlock on the virtqemud process", func() {
			vmi := libvmi.New(
				libvmi.WithMemoryRequest("1Gi"),
				libvmi.WithGPU(v1.GPU{Name: "gpu1"}),
			)

			mockIsolationDetector.EXPECT().Detect(vmi).Return(mockIsolationResult, nil)
			mockIsolationResult.EXPECT().Pid().Return(launcherPid)

			virtqemudPid := 1001
			origListProcesses := mshv.ListProcesses
			mshv.ListProcesses = func() ([]ps.Process, error) {
				return []ps.Process{
					processStub{pid: virtqemudPid, ppid: launcherPid, binary: "virtqemud"},
				}, nil
			}
			DeferCleanup(func() { mshv.ListProcesses = origListProcesses })

			var capturedPid int
			var capturedSize uint64
			origMemlock := common.SetMemlockRLimitFunc
			common.SetMemlockRLimitFunc = func(pid int, size uint64) error {
				capturedPid = pid
				capturedSize = size
				return nil
			}
			DeferCleanup(func() { common.SetMemlockRLimitFunc = origMemlock })

			Expect(runtime.AdjustResources(vmi, config)).To(Succeed())
			Expect(capturedPid).To(Equal(virtqemudPid))
			Expect(capturedSize).To(Equal(uint64(unix.RLIM_INFINITY)))
		})
	})
})
