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

package device_manager

import (
	"os"
	"path"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"

	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
	"kubevirt.io/kubevirt/pkg/virt-handler/selinux"
)

var _ = Describe("Socket device", func() {
	var dpi *SocketDevicePlugin
	var sockDevPath string
	const socket = "fake-test.sock"

	BeforeEach(func() {
		var err error
		workDir := GinkgoT().TempDir()
		Expect(err).ToNot(HaveOccurred())
		sockDevPath = path.Join(workDir, socket)
		createFile(sockDevPath)

		mockExec, mockPermManager := socketDeviceMocks()
		dpi, err = NewSocketDevicePlugin("test", workDir, socket, 1, mockExec, mockPermManager, false)
		Expect(err).ToNot(HaveOccurred())
		dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
		dpi.socketPath = filepath.Join(workDir, "kubevirt-test.sock")
		createFile(dpi.socketPath)
		dpi.done = make(chan struct{})
		stop := make(chan struct{})
		dpi.stop = stop
		DeferCleanup(func() { close(stop) })
	})

	It("Should stop if the device plugin socket file is deleted", func() {
		errChan := make(chan error, 1)
		go func(errChan chan error) {
			errChan <- dpi.healthCheck()
		}(errChan)
		Consistently(func() string {
			return dpi.devs[0].Health
		}, 500*time.Millisecond, 100*time.Millisecond).Should(Equal(pluginapi.Healthy))
		Expect(os.Remove(dpi.socketPath)).To(Succeed())

		Expect(<-errChan).ToNot(HaveOccurred())
	})

	It("Should monitor health of device node", func() {
		go dpi.healthCheck()
		Expect(dpi.devs[0].Health).To(Equal(pluginapi.Healthy))

		By("Removing a (fake) device node")
		os.Remove(sockDevPath)

		By("waiting for healthcheck to send Unhealthy message")
		Eventually(func() string {
			return (<-dpi.health).Health
		}, 5*time.Second).Should(Equal(pluginapi.Unhealthy))

		By("Creating a new (fake) device node")
		createFile(sockDevPath)

		By("waiting for healthcheck to send Healthy message")
		Eventually(func() string {
			return (<-dpi.health).Health
		}, 5*time.Second).Should(Equal(pluginapi.Healthy))
	})
})

var _ = Describe("Optional socket device", func() {
	var dpi *SocketDevicePlugin
	var sockDevPath string
	const socket = "fake-test.sock"

	BeforeEach(func() {
		workDir := GinkgoT().TempDir()
		sockDevPath = path.Join(workDir, socket)
		createFile(sockDevPath)

		mockExec, mockPermManager := socketDeviceMocks()
		dpi = NewOptionalSocketDevicePlugin("test", workDir, socket, 1, mockExec, mockPermManager, false)
		Expect(dpi).ToNot(BeNil())
		dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
		dpi.socketPath = filepath.Join(workDir, "kubevirt-test.sock")
		createFile(dpi.socketPath)
		dpi.done = make(chan struct{})
		stop := make(chan struct{})
		dpi.stop = stop
		DeferCleanup(func() { close(stop) })
	})

	It("Should have healthChecks disabled", func() {
		Expect(dpi.healthChecks).To(BeFalse())
	})

	It("Should stop if the device plugin socket file is deleted", func() {
		errChan := make(chan error, 1)
		go func(errChan chan error) {
			errChan <- dpi.healthCheck()
		}(errChan)
		Consistently(func() string {
			return dpi.devs[0].Health
		}, 500*time.Millisecond, 100*time.Millisecond).Should(Equal(pluginapi.Healthy))
		Expect(os.Remove(dpi.socketPath)).To(Succeed())

		Expect(<-errChan).ToNot(HaveOccurred())
	})

	It("Should stay healthy when device socket is removed", func() {
		go dpi.healthCheck()
		Expect(dpi.devs[0].Health).To(Equal(pluginapi.Healthy))

		By("Removing a (fake) device node")
		Expect(os.Remove(sockDevPath)).To(Succeed())

		By("device should remain healthy since health checks are disabled")
		Consistently(func() string {
			return dpi.devs[0].Health
		}, 500*time.Millisecond, 100*time.Millisecond).Should(Equal(pluginapi.Healthy))

		By("no health messages should be sent on the channel")
		Consistently(func() bool {
			select {
			case <-dpi.health:
				return false
			default:
				return true
			}
		}, 500*time.Millisecond, 100*time.Millisecond).Should(BeTrue())
	})
})

func socketDeviceMocks() (selinux.Executor, PermissionManager) {
	ctrl := gomock.NewController(GinkgoT())
	mockExec := selinux.NewMockExecutor(ctrl)
	mockPermManager := NewMockPermissionManager(ctrl)
	mockSelinux := selinux.NewMockSELinux(ctrl)
	mockExec.EXPECT().NewSELinux().Return(mockSelinux, true, nil).AnyTimes()
	mockSelinux.EXPECT().IsPermissive().Return(true).AnyTimes()
	mockPermManager.EXPECT().ChownAtNoFollow(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	return mockExec, mockPermManager
}

func createFile(path string) {
	fileObj, err := os.Create(path)
	Expect(err).ToNot(HaveOccurred())
	fileObj.Close()
}
