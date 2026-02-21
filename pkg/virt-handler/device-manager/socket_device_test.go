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
	var (
		workDir     string
		dpi         *SocketDevicePlugin
		stop        chan struct{}
		sockDevPath string
	)
	const socket = "fake-test.sock"

	BeforeEach(func() {
		workDir = GinkgoT().TempDir()
		sockDevPath = path.Join(workDir, socket)
		createFile(sockDevPath)

		ctrl := gomock.NewController(GinkgoT())
		mockExec := selinux.NewMockExecutor(ctrl)
		mockSelinux := selinux.NewMockSELinux(ctrl)
		mockPermManager := NewMockpermissionManager(ctrl)
		mockExec.EXPECT().NewSELinux().Return(mockSelinux, true, nil).AnyTimes()
		mockPermManager.EXPECT().ChownAtNoFollow(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockSelinux.EXPECT().IsPermissive().Return(true).AnyTimes()
		dpi = NewSocketDevicePlugin("test", workDir, socket, 1, mockExec, mockPermManager, false)
		dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
		dpi.socketPath = filepath.Join(workDir, "kubevirt-test.sock")
		createFile(dpi.socketPath)
		dpi.done = make(chan struct{})
		stop = make(chan struct{})
		dpi.stop = stop
		dpi.skipDupHealthChecks = false
	})

	AfterEach(func() {
		close(stop)
		dpi.skipDupHealthChecks = true
	})

	It("Should stop if the device plugin socket file is deleted", func() {

		errChan := make(chan error, 1)
		healthCheckContext, err := dpi.setupHealthCheckContext()
		Expect(err).ToNot(HaveOccurred())
		go func() {
			errChan <- dpi.healthCheck(healthCheckContext)
		}()

		By("waiting for initial healthcheck to send Healthy message")
		Eventually(dpi.healthUpdate, 5*time.Second).Should(Receive())
		Expect(dpi.getDevHealthByIndex(0)).To(Equal(pluginapi.Healthy))

		Expect(os.Remove(dpi.socketPath)).To(Succeed())

		Eventually(errChan, 5*time.Second).Should(Receive(Not(HaveOccurred())))
	})

	It("Should monitor health of device node", func() {
		By("Confirming that the device begins as unhealthy")
		expectAllDevHealthIs(dpi.devs, pluginapi.Unhealthy)

		By("waiting for initial healthcheck to send Healthy message")
		healthCheckContext, err := dpi.setupHealthCheckContext()
		Expect(err).ToNot(HaveOccurred())
		go dpi.healthCheck(healthCheckContext)
		Eventually(dpi.healthUpdate, 5*time.Second).Should(Receive())
		Expect(dpi.getDevHealthByIndex(0)).To(Equal(pluginapi.Healthy))

		By("Removing a (fake) device node")
		err = os.Remove(sockDevPath)
		Expect(err).ToNot(HaveOccurred())

		By("waiting for healthcheck to send Unhealthy message")
		Eventually(dpi.healthUpdate, 5*time.Second).Should(Receive())
		Expect(dpi.getDevHealthByIndex(0)).To(Equal(pluginapi.Unhealthy))

		By("Creating a new (fake) device node")
		createFile(sockDevPath)

		By("waiting for healthcheck to send Healthy message")
		Eventually(dpi.healthUpdate, 5*time.Second).Should(Receive())
		Expect(dpi.getDevHealthByIndex(0)).To(Equal(pluginapi.Healthy))
	})

	It("Should mark device unhealthy on SELinux failure", func() {
		// Create a new DPI with a failing executor
		ctrl := gomock.NewController(GinkgoT())
		failingMockExec := selinux.NewMockExecutor(ctrl)
		// mock NewSELinux to return error
		failingMockExec.EXPECT().NewSELinux().Return(nil, false, fmt.Errorf("selinux error")).AnyTimes()

		// Re-create dpi with failing executor
		dpi = NewSocketDevicePlugin("test", workDir, socket, 1, failingMockExec, newPermissionManager(), false)
		dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
		dpi.socketPath = filepath.Join(workDir, "kubevirt-test.sock")
		dpi.stop = stop

		_, err := os.OpenFile(dpi.socketPath, os.O_RDONLY|os.O_CREATE, 0666)
		Expect(err).ToNot(HaveOccurred())

		healthCheckContext, err := dpi.setupHealthCheckContext()
		Expect(err).ToNot(HaveOccurred())
		go dpi.healthCheck(healthCheckContext)

		By("Confirming that the device reports unhealthy due to SELinux failure")
		Eventually(dpi.healthUpdate, 5*time.Second).Should(Receive())
		Expect(dpi.getDevHealthByIndex(0)).To(Equal(pluginapi.Unhealthy))
	})

	It("Should setup watcher for socket device", func() {
		watcher, err := fsnotify.NewWatcher()
		Expect(err).ToNot(HaveOccurred())
		defer watcher.Close()

		monitoredDevices := make(map[string]string)
		err = dpi.setupMonitoredDevicesFunc(watcher, monitoredDevices)
		Expect(err).ToNot(HaveOccurred())

		Expect(monitoredDevices).To(HaveLen(1))
		Expect(watcher.WatchList()).To(ContainElement(workDir))
	})

	It("Should return error if parent directory cannot be watched", func() {
		ctrl := gomock.NewController(GinkgoT())
		mockExec := selinux.NewMockExecutor(ctrl)

		badDpi := NewSocketDevicePlugin("test", "/nonexistent/dir", "test.sock", 1, mockExec, nil, false)

		watcher, _ := fsnotify.NewWatcher()
		defer watcher.Close()

		err := badDpi.setupMonitoredDevicesFunc(watcher, make(map[string]string))
		Expect(err).To(MatchError(ContainSubstring("failed to add the device parent directory")))
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
		socketDir := filepath.Dir(dpi.devicePath)
		Expect(err).ToNot(HaveOccurred())
		Expect(allocateResponse.ContainerResponses).To(HaveLen(1))
		Expect(allocateResponse.ContainerResponses[0].Mounts).To(HaveLen(1))
		Expect(allocateResponse.ContainerResponses[0].Mounts[0].HostPath).To(Equal(socketDir))
		Expect(allocateResponse.ContainerResponses[0].Mounts[0].ContainerPath).To(Equal(socketDir))
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
	err = fileObj.Close()
	Expect(err).ToNot(HaveOccurred())
}

func expectAllDevHealthIs(devs []*pluginapi.Device, expectedHealth string) {
	for _, dev := range devs {
		Expect(dev.Health).To(Equal(expectedHealth))
	}
}
