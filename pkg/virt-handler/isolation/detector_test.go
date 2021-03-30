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
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

var _ = Describe("Isolation Detector", func() {

	Context("With an existing socket", func() {

		var socket net.Listener
		var tmpDir string
		var podUID string
		var finished chan struct{} = nil
		var ctrl *gomock.Controller
		var cgroupParser *cgroup.MockParser

		podUID = "pid-uid-1234"
		vm := v1.NewMinimalVMIWithNS("default", "testvm")
		vm.UID = "1234"
		vm.Status = v1.VirtualMachineInstanceStatus{
			ActivePods: map[types.UID]string{
				types.UID(podUID): "myhost",
			},
		}
		vm.Status.NodeName = "myhost"

		BeforeEach(func() {
			var err error
			tmpDir, err = ioutil.TempDir("", "kubevirt")
			Expect(err).ToNot(HaveOccurred())

			cmdclient.SetLegacyBaseDir(tmpDir)
			cmdclient.SetPodsBaseDir(tmpDir)

			os.MkdirAll(tmpDir+"/sockets/", os.ModePerm)
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

			ctrl = gomock.NewController(GinkgoT())
			cgroupParser = cgroup.NewMockParser(ctrl)
			cgroupParser.
				EXPECT().
				Parse(gomock.Eq(os.Getpid())).
				Return(map[string]string{"devices": "/"}, nil).
				AnyTimes()
		})

		AfterEach(func() {
			socket.Close()
			os.RemoveAll(tmpDir)
			if finished != nil {
				<-finished
			}

			ctrl.Finish()
		})

		It("Should detect the PID of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector(tmpDir, cgroupParser).Whitelist([]string{"devices"}).Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Pid()).To(Equal(os.Getpid()))
		})

		It("Should not detect any slice if there is no matching controller", func() {
			_, err := NewSocketBasedIsolationDetector(tmpDir, cgroupParser).Whitelist([]string{"not_existing_slice"}).Detect(vm)
			Expect(err).To(HaveOccurred())
		})

		It("Should detect the 'devices' controller slice of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector(tmpDir, cgroupParser).Whitelist([]string{"devices"}).Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Slice()).To(HavePrefix("/"))
		})

		It("Should detect the PID namespace of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector(tmpDir, cgroupParser).Whitelist([]string{"devices"}).Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.PIDNamespace()).To(Equal(fmt.Sprintf("/proc/%d/ns/pid", os.Getpid())))
		})

		It("Should detect the Mount root of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector(tmpDir, cgroupParser).Whitelist([]string{"devices"}).Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.MountRoot()).To(Equal(fmt.Sprintf("/proc/%d/root", os.Getpid())))
		})

		It("Should detect the Network namespace of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector(tmpDir, cgroupParser).Whitelist([]string{"devices"}).Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.NetNamespace()).To(Equal(fmt.Sprintf("/proc/%d/ns/net", os.Getpid())))
		})

		It("Should detect the root mount info of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector(tmpDir, cgroupParser).Whitelist([]string{"devices"}).Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			mountInfo, err := result.MountInfoRoot()
			Expect(err).ToNot(HaveOccurred())
			Expect(mountInfo.MountPoint).To(Equal("/"))
		})

		It("Should detect the full path of the mount on a node", func() {
			mountInfoTestData := map[string]string{
				"overlay":      "/proc/1/root/var/lib/docker/overlay2/f15d9ce07df72e80d809aa99ab4a171f2f3636f65f0653e75db8ca0befd8ae02/merged",
				"devicemapper": "/proc/1/root/var/lib/docker/devicemapper/mnt/d0990551ba8254871a449b2ff0d9063061ae96a2c195d7a850b62f030eae1710/rootfs",
				"btrfs":        "/proc/1/root/var/lib/containers/storage/btrfs/subvolumes/e9a94e2cde75c54834378d4835d4eda6bebb56b02068b9254780de6f9344ad0e",
			}

			// Restore the overwritten function
			defer func(f func(int) string) { mountInfoFunc = f }(mountInfoFunc)

			for testCase, want := range mountInfoTestData {
				mountInfoFunc = func(pid int) string {
					Expect(pid).To(SatisfyAny(Equal(1), Equal(os.Getpid())))
					base := filepath.Join("testdata", "mountinfo")
					if pid == 1 {
						return filepath.Join(base, fmt.Sprintf("%s_host", testCase))
					}
					return filepath.Join(base, fmt.Sprintf("%s_launcher", testCase))
				}

				mounted, err := NodeIsolationResult().IsMounted("/")
				Expect(err).ToNot(HaveOccurred())
				Expect(mounted).To(BeTrue())
				mounted, err = NodeIsolationResult().IsMounted("???")
				Expect(err).ToNot(HaveOccurred())
				Expect(mounted).To(BeFalse())

				result, err := NewSocketBasedIsolationDetector(tmpDir, cgroupParser).Whitelist([]string{"devices"}).Detect(vm)
				Expect(err).ToNot(HaveOccurred())
				mountInfo, err := result.MountInfoRoot()
				Expect(err).ToNot(HaveOccurred())
				fullPath, err := NodeIsolationResult().FullPath(mountInfo)
				Expect(err).ToNot(HaveOccurred())
				Expect(fullPath).To(Equal(want))
			}
		})
	})
})

var _ = Describe("getMemlockSize", func() {
	vm := v1.NewMinimalVMIWithNS("default", "testvm")

	It("Should return correct number of bytes for memlock limit", func() {
		bytes, err := getMemlockSize(vm)
		Expect(err).ToNot(HaveOccurred())
		// 1Gb (static part for vfio VMs) + 256Mb (estimated overhead) + 8 Mb (VM)
		Expect(int(bytes)).To(Equal(1264389000))
	})
})
