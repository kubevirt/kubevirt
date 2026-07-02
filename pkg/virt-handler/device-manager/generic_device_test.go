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
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"

	"kubevirt.io/kubevirt/pkg/safepath"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

var _ = Describe("Generic Device", func() {
	var (
		workDir    string
		dpi        *GenericDevicePlugin
		devicePath string
		stop       chan struct{}
	)

	BeforeEach(func() {
		workDir = GinkgoT().TempDir()

		devicePath = filepath.Join(workDir, "foo")
		createFile(devicePath)

		dpi = NewGenericDevicePlugin("foo", workDir, devicePath, 1, "rw", true)
		dpi.socketPath = filepath.Join(workDir, "test.sock")
		dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
		dpi.deviceRoot = "/"
		stop = make(chan struct{})
		dpi.stop = stop
		dpi.skipDupHealthChecks = false
	})

	AfterEach(func() {
		close(stop)
		dpi.skipDupHealthChecks = true
	})

	It("Should stop if the device plugin socket file is deleted", func() {
		os.OpenFile(dpi.socketPath, os.O_RDONLY|os.O_CREATE, 0666)

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

		os.OpenFile(dpi.socketPath, os.O_RDONLY|os.O_CREATE, 0666)

		By("Confirming that the device begins as unhealthy")
		expectAllDevHealthIs(dpi.devs, pluginapi.Unhealthy)

		By("waiting for initial healthcheck to send Healthy message")
		healthCheckContext, err := dpi.setupHealthCheckContext()
		Expect(err).ToNot(HaveOccurred())
		go dpi.healthCheck(healthCheckContext)
		Eventually(dpi.healthUpdate, 5*time.Second).Should(Receive())
		Expect(dpi.getDevHealthByIndex(0)).To(Equal(pluginapi.Healthy))

		By("Removing a (fake) device node")
		os.Remove(devicePath)

		By("waiting for healthcheck to send Unhealthy message")
		Eventually(dpi.healthUpdate, 5*time.Second).Should(Receive())
		Expect(dpi.getDevHealthByIndex(0)).To(Equal(pluginapi.Unhealthy))

		By("Creating a new (fake) device node")
		createFile(devicePath)

		By("waiting for healthcheck to send Healthy message")
		Eventually(dpi.healthUpdate, 5*time.Second).Should(Receive())
		Expect(dpi.getDevHealthByIndex(0)).To(Equal(pluginapi.Healthy))
	})

	It("Should mark device unhealthy if ConfigurePermissions fails", func() {
		os.OpenFile(dpi.socketPath, os.O_RDONLY|os.O_CREATE, 0666)

		// Mock ConfigurePermissions to fail
		dpi.configurePermissions = func(_ *safepath.Path) error {
			return fmt.Errorf("mock permission error")
		}

		By("waiting for initial healthcheck to send Unhealthy message due to permission failure")
		healthCheckContext, err := dpi.setupHealthCheckContext()
		Expect(err).ToNot(HaveOccurred())
		go dpi.healthCheck(healthCheckContext)
		Eventually(dpi.healthUpdate, 5*time.Second).Should(Receive())
		Expect(dpi.getDevHealthByIndex(0)).To(Equal(pluginapi.Unhealthy))
	})

	It("Should setup watcher for device directory", func() {
		watcher, err := fsnotify.NewWatcher()
		Expect(err).ToNot(HaveOccurred())
		defer watcher.Close()

		monitoredDevices := make(map[string]string)
		err = dpi.setupMonitoredDevicesFunc(watcher, monitoredDevices)
		Expect(err).ToNot(HaveOccurred())

		Expect(monitoredDevices).To(HaveLen(1))
		Expect(watcher.WatchList()).To(ContainElement(workDir))
	})

	It("Should return error if device directory cannot be watched", func() {
		badDpi := NewGenericDevicePlugin("foo", workDir, "/nonexistent/device", 1, "rw", true)
		badDpi.deviceRoot = "/"

		watcher, _ := fsnotify.NewWatcher()
		defer watcher.Close()

		err := badDpi.setupMonitoredDevicesFunc(watcher, make(map[string]string))
		Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
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
		Expect(allocateResponse.ContainerResponses[0].Devices).To(HaveLen(1))
		Expect(allocateResponse.ContainerResponses[0].Devices[0].HostPath).To(Equal(devicePath))
		Expect(allocateResponse.ContainerResponses[0].Devices[0].ContainerPath).To(Equal(devicePath))
		Expect(allocateResponse.ContainerResponses[0].Devices[0].Permissions).To(Equal("rw"))
	})
})
