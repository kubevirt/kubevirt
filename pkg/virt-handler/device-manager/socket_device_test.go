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
	"errors"
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
	var deviceDir string
	var sockDevPath string
	const socket = "fake-test.sock"

	BeforeEach(func() {
		var err error
		workDir := GinkgoT().TempDir()
		// The device socket lives in a directory of its own, as it does on a
		// host where it is provided out of a service's runtime directory. The
		// kubelet device-plugin directory is unrelated to it, so that the two
		// watches the health check sets up are covered independently.
		deviceDir = filepath.Join(workDir, "qgs")
		Expect(os.Mkdir(deviceDir, 0o755)).To(Succeed())
		sockDevPath = filepath.Join(deviceDir, socket)
		createFile(sockDevPath)
		kubeletDir := filepath.Join(workDir, "device-plugins")
		Expect(os.Mkdir(kubeletDir, 0o755)).To(Succeed())

		mockExec, mockPermManager := socketDeviceMocks()
		dpi, err = NewSocketDevicePlugin("test", deviceDir, socket, 1, mockExec, mockPermManager, false)
		Expect(err).ToNot(HaveOccurred())
		dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
		dpi.socketPath = filepath.Join(kubeletDir, "kubevirt-test.sock")
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

		Eventually(errChan, 5*time.Second).Should(Receive(BeNil()))
	})

	DescribeTable("Should monitor health of device node", func(remove, recreate func()) {
		go func() {
			defer GinkgoRecover()
			Expect(dpi.healthCheck()).ToNot(HaveOccurred())
		}()

		By("waiting for the health check to be watching an intact device")
		Expect(dpi.devs[0].Health).To(Equal(pluginapi.Healthy))
		Consistently(dpi.health, 500*time.Millisecond, 100*time.Millisecond).ShouldNot(Receive())

		By("Removing the (fake) device node")
		remove()

		By("waiting for healthcheck to send Unhealthy message")
		Eventually(dpi.health, 5*time.Second).Should(Receive(HaveField("Health", pluginapi.Unhealthy)))

		By("Creating a new (fake) device node")
		recreate()

		By("waiting for healthcheck to send Healthy message")
		Eventually(dpi.health, 5*time.Second).Should(Receive(HaveField("Health", pluginapi.Healthy)))
	},
		Entry("when the device node itself is removed",
			func() { Expect(os.Remove(sockDevPath)).To(Succeed()) },
			func() { createFile(sockDevPath) },
		),
		Entry("when the directory holding it is removed, as systemd does for a service's RuntimeDirectory",
			func() { Expect(os.RemoveAll(deviceDir)).To(Succeed()) },
			func() {
				Expect(os.Mkdir(deviceDir, 0o755)).To(Succeed())
				createFile(sockDevPath)
			},
		),
	)

	It("Should degrade to Unhealthy and recover when the device directory is missing on start", func() {
		Expect(os.RemoveAll(deviceDir)).To(Succeed())

		go func() {
			defer GinkgoRecover()
			Expect(dpi.healthCheck()).ToNot(HaveOccurred())
		}()

		By("degrading to Unhealthy instead of erroring out")
		Eventually(dpi.health, 5*time.Second).Should(Receive(HaveField("Health", pluginapi.Unhealthy)))

		By("recovering once the directory and its socket are recreated")
		Expect(os.Mkdir(deviceDir, 0o755)).To(Succeed())
		createFile(sockDevPath)
		Eventually(dpi.health, 5*time.Second).Should(Receive(HaveField("Health", pluginapi.Healthy)))
	})

	It("Should reconcile stale health on start, as it survives a plugin restart", func() {
		dpi.healthy = false

		go func() {
			defer GinkgoRecover()
			Expect(dpi.healthCheck()).ToNot(HaveOccurred())
		}()

		Eventually(dpi.health, 5*time.Second).Should(Receive(HaveField("Health", pluginapi.Healthy)))
	})

	It("Should error out when a device's permissions cannot be applied, so the plugin is restarted with backoff", func() {
		failingPermManager := NewMockPermissionManager(gomock.NewController(GinkgoT()))
		failingPermManager.EXPECT().ChownAtNoFollow(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(errors.New("no permission to give")).AnyTimes()
		dpi.p = failingPermManager

		By("Removing the (fake) device node before the health check starts")
		Expect(os.Remove(sockDevPath)).To(Succeed())

		errChan := make(chan error, 1)
		go func(errChan chan error) {
			errChan <- dpi.healthCheck()
		}(errChan)
		Eventually(dpi.health, 5*time.Second).Should(Receive(HaveField("Health", pluginapi.Unhealthy)))

		By("Creating a new (fake) device node whose permissions cannot be set")
		createFile(sockDevPath)

		By("erroring out instead of advertising the device")
		Eventually(errChan, 5*time.Second).Should(Receive(MatchError(ContainSubstring("no permission to give"))))
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

		Eventually(errChan, 5*time.Second).Should(Receive(BeNil()))
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
