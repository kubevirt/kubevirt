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
	"net"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-ps"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"golang.org/x/sys/unix"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

var _ = Describe("Unlimited memlock", func() {
	for _, running := range []bool{false, true} {
		Context(fmt.Sprintf("running QEMU: %t", running), func() {
			var (
				vmi          *v1.VirtualMachineInstance
				adjust       func(bool) error
				calls        int
				capturedPID  int
				capturedSize uint64
				limitError   error
			)
			const targetPID = 1001

			BeforeEach(func() {
				vmi = api.NewMinimalVMIWithNS("default", "memlock-test")
				calls, capturedPID, capturedSize = 0, 0, 0
				limitError = nil
				originalSet := setMemlockRLimitFunc
				setMemlockRLimitFunc = func(pid int, size uint64) error {
					calls++
					capturedPID, capturedSize = pid, size
					return limitError
				}
				DeferCleanup(func() { setMemlockRLimitFunc = originalSet })

				if running {
					ctrl := gomock.NewController(GinkgoT())
					detector := NewMockPodIsolationDetector(ctrl)
					result := NewMockIsolationResult(ctrl)
					adjust = func(requiresMemlock bool) error {
						if requiresMemlock {
							detector.EXPECT().Detect(vmi).Return(result, nil)
							result.EXPECT().GetQEMUProcess().Return(ProcessStub{pid: targetPID, binary: "qemu-kvm"}, nil)
						}
						return AdjustQemuProcessMemoryLimits(detector, vmi, nil)
					}
					return
				}

				podUID := types.UID("memlock-pod")
				vmi.Status.ActivePods = map[types.UID]string{podUID: "test-node"}
				cmdclient.SetPodsBaseDir(GinkgoT().TempDir())
				socketPath := cmdclient.SocketFilePathOnHost(string(podUID))
				Expect(os.MkdirAll(filepath.Dir(socketPath), 0700)).To(Succeed())
				listener, err := net.Listen("unix", socketPath)
				Expect(err).NotTo(HaveOccurred())
				finished := make(chan struct{})
				go func() {
					defer close(finished)
					for {
						connection, err := listener.Accept()
						if err != nil {
							return
						}
						connection.Close()
					}
				}()
				DeferCleanup(func() {
					listener.Close()
					<-finished
				})
				originalList := listProcesses
				listProcesses = func() ([]ps.Process, error) {
					return []ps.Process{
						ProcessStub{pid: targetPID + 1, ppid: os.Getpid() + 1, binary: "virtqemud"},
						ProcessStub{pid: targetPID + 2, ppid: os.Getpid(), binary: "other"},
						ProcessStub{pid: targetPID, ppid: os.Getpid(), binary: "virtqemud"},
					}, nil
				}
				DeferCleanup(func() { listProcesses = originalList })
				adjust = func(bool) error {
					return NewSocketBasedIsolationDetector("").AdjustResources(vmi, nil)
				}
			})

			It("does not adjust ordinary VMIs", func() {
				Expect(adjust(false)).To(Succeed())
				Expect(calls).To(BeZero())
			})

			DescribeTable("sets unlimited on the correct process",
				func(configure func(*v1.VirtualMachineInstance)) {
					configure(vmi)
					Expect(adjust(true)).To(Succeed())
					Expect(calls).To(Equal(1))
					Expect(capturedPID).To(Equal(targetPID))
					Expect(capturedSize).To(Equal(uint64(unix.RLIM_INFINITY)))
				},
				Entry("GPU", func(vmi *v1.VirtualMachineInstance) {
					vmi.Spec.Domain.Devices.GPUs = []v1.GPU{{Name: "gpu1"}, {Name: "gpu2"}}
				}),
				Entry("host device", func(vmi *v1.VirtualMachineInstance) {
					vmi.Spec.Domain.Devices.HostDevices = []v1.HostDevice{{Name: "device1"}}
				}),
				Entry("SR-IOV", func(vmi *v1.VirtualMachineInstance) {
					vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "sriov", InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}}}
				}),
				Entry("realtime", func(vmi *v1.VirtualMachineInstance) {
					vmi.Spec.Domain.CPU = &v1.CPU{Realtime: &v1.Realtime{}}
				}),
				Entry("SEV", func(vmi *v1.VirtualMachineInstance) {
					vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{SEV: &v1.SEV{}}
				}),
			)

			It("surfaces failures to change the rlimit", func() {
				vmi.Spec.Domain.Devices.GPUs = []v1.GPU{{Name: "gpu1"}}
				limitError = fmt.Errorf("prlimit denied")
				Expect(adjust(true)).To(MatchError(ContainSubstring("memlock rlimit to unlimited: prlimit denied")))
			})
		})
	}
})
