/*
 * This file is part of the kubevirt project
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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package isolation

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/api"

	"github.com/mitchellh/go-ps"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/unsafepath"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

var _ = Describe("Isolation Detector", func() {

	Context("With an existing socket", func() {

		var socket net.Listener
		var tmpDir string
		var podUID string
		var finished chan struct{} = nil

		podUID = "pid-uid-1234"
		vm := api.NewMinimalVMIWithNS("default", "testvm")
		vm.UID = "1234"
		vm.Status = v1.VirtualMachineInstanceStatus{
			ActivePods: map[types.UID]string{
				types.UID(podUID): "myhost",
			},
		}
		vm.Status.NodeName = "myhost"

		BeforeEach(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "kubevirt")
			Expect(err).ToNot(HaveOccurred())

			cmdclient.SetLegacyBaseDir(tmpDir)
			cmdclient.SetPodsBaseDir(tmpDir)

			os.MkdirAll(filepath.Join(tmpDir, "sockets/"), os.ModePerm)
			socketFile := cmdclient.SocketFilePathOnHost(podUID)
			os.MkdirAll(filepath.Dir(socketFile), os.ModePerm)
			socket, err = net.Listen("unix", socketFile)
			Expect(err).ToNot(HaveOccurred())
			finished = make(chan struct{})
			go func() {
				for {
					conn, err := socket.Accept()
					if err != nil {
						close(finished)
						// closes when socket listener is closed
						return
					}
					conn.Close()
				}
			}()
		})

		AfterEach(func() {
			socket.Close()
			os.RemoveAll(tmpDir)
			if finished != nil {
				<-finished
			}
		})

		It("Should detect the PID of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector(tmpDir).Allowlist([]string{"devices"}).Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Pid()).To(Equal(os.Getpid()))
		})

		It("Should detect the PID namespace of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector(tmpDir).Allowlist([]string{"devices"}).Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.PIDNamespace()).To(Equal(fmt.Sprintf("/proc/%d/ns/pid", os.Getpid())))
		})

		It("Should detect the Parent PID of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector(tmpDir).Allowlist([]string{"devices"}).Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.PPid()).To(Equal(os.Getppid()))
		})

		It("Should detect the Mount root of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector(tmpDir).Allowlist([]string{"devices"}).Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			root, err := result.MountRoot()
			Expect(err).ToNot(HaveOccurred())
			Expect(unsafepath.UnsafeAbsolute(root.Raw())).To(Equal(fmt.Sprintf("/proc/%d/root", os.Getpid())))
		})
	})
})

var _ = Describe("findIsolatedQemuProcess", func() {
	const virtLauncherPid = 1
	fakeProcess1 := ProcessStub{pid: virtLauncherPid, ppid: 0, binary: "fake-process-1"}
	fakeProcess2 := ProcessStub{pid: 26, ppid: virtLauncherPid, binary: "fake-process-2"}
	fakeProcess3 := ProcessStub{pid: 226, ppid: 26, binary: "fake-process-3"}
	virtLauncherProcesses := []ps.Process{
		fakeProcess1,
		fakeProcess2,
		fakeProcess3}

	qemuKvmProc := ProcessStub{pid: 101, ppid: virtLauncherPid, binary: "qemu-kvm"}
	qemuSystemProc := ProcessStub{pid: 101, ppid: virtLauncherPid, binary: "qemu-system-x86_64"}

	DescribeTable("should return QEMU process",
		func(processes []ps.Process, pid int, expectedProcess ps.Process) {
			proc, err := findIsolatedQemuProcess(processes, pid)
			Expect(err).NotTo(HaveOccurred())
			Expect(proc).To(Equal(expectedProcess))
		},
		Entry("when qemu-kvm binary running",
			append(virtLauncherProcesses, qemuKvmProc),
			virtLauncherPid,
			qemuKvmProc,
		),
		Entry("when qemu-system binary running",
			append(virtLauncherProcesses, qemuSystemProc),
			virtLauncherPid,
			qemuSystemProc,
		),
	)
	It("should fail when no QEMU process exists", func() {
		proc, err := findIsolatedQemuProcess(virtLauncherProcesses, virtLauncherPid)
		Expect(err).To(HaveOccurred())
		Expect(proc).To(BeNil())
	})
})
