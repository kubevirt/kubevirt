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
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"

	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
	"kubevirt.io/kubevirt/pkg/virt-handler/selinux"
)

var _ = Describe("Socket device", func() {
	var workDir string
	var dpi *SocketDevicePlugin
	var stop chan struct{}
	var sockDevPath string
	const socket = "fake-test.sock"
	var mockExec *selinux.MockExecutor
	var mockPermManager *MockPermissionManager
	var mockSelinux *selinux.MockSELinux

	BeforeEach(func() {
		var err error
		workDir := GinkgoT().TempDir()
		Expect(err).ToNot(HaveOccurred())
		sockDevPath = path.Join(workDir, socket)
		createFile(sockDevPath)

		ctrl := gomock.NewController(GinkgoT())
		mockExec = selinux.NewMockExecutor(ctrl)
		mockPermManager = NewMockPermissionManager(ctrl)
		mockSelinux = selinux.NewMockSELinux(ctrl)

		// Default mocks
		mockExec.EXPECT().NewSELinux().Return(mockSelinux, true, nil).AnyTimes()
		mockSelinux.EXPECT().IsPermissive().Return(true).AnyTimes()
		mockPermManager.EXPECT().ChownAtNoFollow(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		dpi, err = NewSocketDevicePlugin("test", workDir, socket, 1, mockExec, mockPermManager)
		Expect(err).ToNot(HaveOccurred())
		dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
		dpi.socketPath = filepath.Join(workDir, "kubevirt-test.sock")
		createFile(dpi.socketPath)
		dpi.done = make(chan struct{})
		stop = make(chan struct{})
		dpi.stop = stop
		DeferCleanup(func() { close(stop) })
	})

	It("Should stop if the device plugin socket file is deleted", func() {

		errChan := make(chan error, 1)
		go func(errChan chan error) {
			errChan <- dpi.healthCheck()
		}(errChan)

		By("waiting for initial healthcheck to send Healthy message")
		Eventually(dpi.health, 5*time.Second).Should(Receive(HaveField("Health", Equal(pluginapi.Healthy))))

		Expect(os.Remove(dpi.socketPath)).To(Succeed())

		Expect(<-errChan).ToNot(HaveOccurred())
	})

	It("Should monitor health of device node", func() {
		By("Confirming that the device begins as unhealthy")
		expectAllDevHealthIs(dpi.devs, pluginapi.Unhealthy)

		By("waiting for initial healthcheck to send Healthy message")
		go dpi.healthCheck()
		Eventually(dpi.health, 5*time.Second).Should(Receive(HaveField("Health", Equal(pluginapi.Healthy))))

		By("Removing a (fake) device node")
		os.Remove(sockDevPath)

		By("waiting for healthcheck to send Unhealthy message")
		Eventually(dpi.health, 5*time.Second).Should(Receive(HaveField("Health", Equal(pluginapi.Unhealthy))))

		By("Creating a new (fake) device node")
		createFile(sockDevPath)

		By("waiting for healthcheck to send Healthy message")
		Eventually(dpi.health, 5*time.Second).Should(Receive(HaveField("Health", Equal(pluginapi.Healthy))))
	})

	It("Should mark device unhealthy on SELinux failure", func() {
		// Create a new DPI with a failing executor
		ctrl := gomock.NewController(GinkgoT())
		failingMockExec := selinux.NewMockExecutor(ctrl)
		// mock NewSELinux to return error
		failingMockExec.EXPECT().NewSELinux().Return(nil, false, fmt.Errorf("selinux error")).AnyTimes()

		// Re-create dpi with failing executor
		var err error
		dpi, err = NewSocketDevicePlugin("test", workDir, socket, 1, failingMockExec, mockPermManager)
		Expect(err).ToNot(HaveOccurred())
		dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
		dpi.socketPath = filepath.Join(workDir, "kubevirt-test.sock")
		dpi.stop = stop

		os.OpenFile(dpi.socketPath, os.O_RDONLY|os.O_CREATE, 0666)

		go dpi.healthCheck()

		By("Confirming that the device reports unhealthy due to SELinux failure")
		Eventually(dpi.health, 5*time.Second).Should(Receive(HaveField("Health", Equal(pluginapi.Unhealthy))))
	})

	It("Should setup watcher for socket device", func() {
		watcher, err := fsnotify.NewWatcher()
		Expect(err).ToNot(HaveOccurred())
		defer watcher.Close()

		monitoredDevices := make(map[string]string)
		err = dpi.SetupMonitoredDevicesFunc(watcher, monitoredDevices)
		Expect(err).ToNot(HaveOccurred())

		Expect(monitoredDevices).To(HaveLen(1))
		Expect(watcher.WatchList()).To(ContainElement(workDir))
	})

	It("Should return error if parent directory cannot be watched", func() {
		ctrl := gomock.NewController(GinkgoT())
		mockExec := selinux.NewMockExecutor(ctrl)

		badDpi, _ := NewSocketDevicePlugin("test", "/nonexistent/dir", "test.sock", 1, mockExec, nil)
		badDpi.deviceRoot = "/nonexistent/dir"

		watcher, _ := fsnotify.NewWatcher()
		defer watcher.Close()

		err := badDpi.SetupMonitoredDevicesFunc(watcher, make(map[string]string))
		Expect(err).To(HaveOccurred())
	})

	It("Should allocate the device", func() {
		allocateRequest := &pluginapi.AllocateRequest{
			ContainerRequests: []*pluginapi.ContainerAllocateRequest{
				{
					DevicesIDs: []string{dpi.devs[0].ID},
				},
			},
		}

		allocateResponse, err := dpi.Allocate(context.Background(), allocateRequest)
		Expect(err).ToNot(HaveOccurred())
		Expect(allocateResponse.ContainerResponses).To(HaveLen(1))
		Expect(allocateResponse.ContainerResponses[0].Mounts).To(HaveLen(1))
		Expect(allocateResponse.ContainerResponses[0].Mounts[0].HostPath).To(Equal(dpi.deviceRoot))
		Expect(allocateResponse.ContainerResponses[0].Mounts[0].ContainerPath).To(Equal(dpi.deviceRoot))
		Expect(allocateResponse.ContainerResponses[0].Mounts[0].ReadOnly).To(BeFalse())
	})
})

func createFile(path string) {
	// create parent director(y,ies) if it doesn't exist
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	Expect(err).ToNot(HaveOccurred())
	// create file
	fileObj, err := os.Create(path)
	Expect(err).ToNot(HaveOccurred())
	fileObj.Close()
}

func expectAllDevHealthIs(devs []*pluginapi.Device, expectedHealth string) {
	for _, dev := range devs {
		Expect(dev.Health).To(Equal(expectedHealth))
	}
}
