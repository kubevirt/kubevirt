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
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"

	v1 "kubevirt.io/api/core/v1"

	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

var _ = Describe("USB Device", func() {
	usbs := []*USBDevice{
		// Unique device
		{
			Vendor:       1234,
			Product:      5678,
			Bus:          3,
			DeviceNumber: 11,
			BCD:          0,
			DevicePath:   "/dev/bus/usb/003/011",
		},
		// Two identical devices
		{
			Vendor:       4321,
			Product:      8765,
			Bus:          4,
			DeviceNumber: 7,
			BCD:          0,
			DevicePath:   "/dev/bus/usb/004/007",
		},
		{
			Vendor:       4321,
			Product:      8765,
			Bus:          2,
			DeviceNumber: 10,
			BCD:          0,
			DevicePath:   "/dev/bus/usb/002/010",
		},
	}

	findAll := func() *LocalDevices {
		usbDevices := make(map[int][]*USBDevice)
		for _, device := range usbs {
			usbDevices[device.Vendor] = append(usbDevices[device.Vendor], device)
		}
		return &LocalDevices{devices: usbDevices}
	}

	const resourceName1 = "testing.usb/usecase"
	const resourceName2 = "testing.usb/another"

	var (
		dpi     *USBDevicePlugin
		stop    chan struct{}
		workDir string
	)

	BeforeEach(func() {
		workDir = GinkgoT().TempDir()
		ctrl := gomock.NewController(GinkgoT())
		permissionManager := NewMockPermissionManager(ctrl)
		permissionManager.EXPECT().ChownAtNoFollow(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		devices := []*PluginDevices{
			newPluginDevices(resourceName1, 0, []*USBDevice{usbs[0], usbs[1]}),
			newPluginDevices(resourceName2, 1, []*USBDevice{usbs[2]}),
		}
		// create dummy devices in the workDir
		for _, device := range devices {
			for _, usb := range device.Devices {
				createFile(filepath.Join(workDir, usb.DevicePath))
			}
		}
		dpi = NewUSBDevicePlugin(resourceName1, workDir, devices, permissionManager)
		dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
		dpi.socketPath = filepath.Join(workDir, "kubevirt-test.sock")
		createFile(dpi.socketPath)
		stop = make(chan struct{})
		dpi.stop = stop
	})

	AfterEach(func() {
		close(stop)
	})

	It("Should stop if the device plugin socket file is deleted", func() {
		os.OpenFile(dpi.socketPath, os.O_RDONLY|os.O_CREATE, 0666)

		errChan := make(chan error, 1)
		healthCheckContext, err := dpi.setupHealthCheckContext()
		Expect(err).ToNot(HaveOccurred())
		go func() {
			errChan <- dpi.healthCheck(healthCheckContext)
		}()

		By("waiting for initial healthchecks to send Healthy message for each device")
		for range dpi.devs {
			Eventually(dpi.health, 5*time.Second).Should(Receive(HaveField("Health", Equal(pluginapi.Healthy))))
		}

		Expect(os.Remove(dpi.socketPath)).To(Succeed())

		Eventually(errChan, 5*time.Second).Should(Receive(Not(HaveOccurred())))
	})

	It("Should monitor health of device node", func() {
		os.OpenFile(dpi.socketPath, os.O_RDONLY|os.O_CREATE, 0666)

		By("Confirming that the device begins as unhealthy")
		expectAllDevHealthIs(dpi.devs, pluginapi.Unhealthy)

		By("waiting for initial healthchecks to send Healthy message")
		healthCheckContext, err := dpi.setupHealthCheckContext()
		Expect(err).ToNot(HaveOccurred())
		go dpi.healthCheck(healthCheckContext)
		for range dpi.devs {
			Eventually(dpi.health, 5*time.Second).Should(Receive(HaveField("Health", Equal(pluginapi.Healthy))))
		}

		By("Removing (fake) device nodes")
		usbDevicePath1 := filepath.Join(workDir, usbs[0].DevicePath)
		usbDevicePath2 := filepath.Join(workDir, usbs[1].DevicePath)
		Expect(os.Remove(usbDevicePath1)).To(Succeed())
		Expect(os.Remove(usbDevicePath2)).To(Succeed())

		By("waiting for healthcheck to send Unhealthy message")
		Eventually(dpi.health, 5*time.Second).Should(Receive(HaveField("Health", Equal(pluginapi.Unhealthy))))

		By("Creating a new (fake) device node 1")
		createFile(usbDevicePath1)

		By("waiting for healthcheck to send Unhealthy message")
		// Since only one of the two devices in the group is healthy, the healthcheck should send Unhealthy
		Eventually(dpi.health, 5*time.Second).Should(Receive(HaveField("Health", Equal(pluginapi.Unhealthy))))

		By("Creating a new (fake) device node 2")
		createFile(usbDevicePath2)

		By("waiting for healthcheck to send Healthy message")
		// Since both devices in the group are now healthy, the healthcheck should send Healthy
		Eventually(dpi.health, 5*time.Second).Should(Receive(HaveField("Health", Equal(pluginapi.Healthy))))
	})

	DescribeTable("with USBHostDevice configuration", func(hostDeviceConfig []v1.USBHostDevice, result map[string][]*PluginDevices) {
		discoverLocalUSBDevicesFunc = findAll
		pdmap := discoverAllowedUSBDevices(hostDeviceConfig)

		Expect(pdmap).To(HaveLen(len(result)), "Expected number of resource names")
		for resourceName, pluginDevices := range pdmap {
			Expect(pluginDevices).To(HaveLen(len(result[resourceName])), "Number of k8s devices")
			for i, dev := range pluginDevices {
				Expect(dev.Devices).To(HaveLen(len(result[resourceName][i].Devices)), "Number of USB devices")
				for j, usbdev := range dev.Devices {
					expectMatch(usbdev, result[resourceName][i].Devices[j])
				}
			}
		}
	},
		Entry("1 resource with 1 selector matching 1 USB device",
			[]v1.USBHostDevice{
				{
					ResourceName: resourceName1,
					Selectors: []v1.USBSelector{
						{
							Vendor:  fmt.Sprintf("%x", usbs[0].Vendor),
							Product: fmt.Sprintf("%x", usbs[0].Product),
						},
					},
				},
			},
			map[string][]*PluginDevices{
				resourceName1: {
					newPluginDevices(resourceName1, 0, []*USBDevice{usbs[0]}),
				},
			},
		),
		Entry("1 resource with 1 selector matching 2 USB devices",
			[]v1.USBHostDevice{
				{
					ResourceName: resourceName1,
					Selectors: []v1.USBSelector{
						{
							Vendor:  fmt.Sprintf("%x", usbs[1].Vendor),
							Product: fmt.Sprintf("%x", usbs[1].Product),
						},
					},
				},
			},
			map[string][]*PluginDevices{
				resourceName1: {
					newPluginDevices(resourceName1, 0, []*USBDevice{usbs[1]}),
					newPluginDevices(resourceName1, 1, []*USBDevice{usbs[2]}),
				},
			},
		),
		Entry("1 resource with 2 selectors matching 2 USB devices, single k8s device",
			[]v1.USBHostDevice{
				{
					ResourceName: resourceName1,
					Selectors: []v1.USBSelector{
						{
							Vendor:  fmt.Sprintf("%x", usbs[0].Vendor),
							Product: fmt.Sprintf("%x", usbs[0].Product),
						},
						{
							Vendor:  fmt.Sprintf("%x", usbs[1].Vendor),
							Product: fmt.Sprintf("%x", usbs[1].Product),
						},
					},
				},
			},
			map[string][]*PluginDevices{
				resourceName1: {
					newPluginDevices(resourceName1, 0, []*USBDevice{usbs[0], usbs[1]}),
				},
			},
		),
		Entry("2 resources with 1 selector each matching 3 USB devices total",
			[]v1.USBHostDevice{
				{
					ResourceName: resourceName1,
					Selectors: []v1.USBSelector{
						{
							Vendor:  fmt.Sprintf("%x", usbs[0].Vendor),
							Product: fmt.Sprintf("%x", usbs[0].Product),
						},
					},
				},
				{
					ResourceName: resourceName2,
					Selectors: []v1.USBSelector{
						{
							Vendor:  fmt.Sprintf("%x", usbs[1].Vendor),
							Product: fmt.Sprintf("%x", usbs[1].Product),
						},
					},
				},
			},
			map[string][]*PluginDevices{
				resourceName1: {
					newPluginDevices(resourceName1, 0, []*USBDevice{usbs[0]}),
				},
				resourceName2: {
					newPluginDevices(resourceName2, 1, []*USBDevice{usbs[1]}),
					newPluginDevices(resourceName2, 2, []*USBDevice{usbs[2]}),
				},
			},
		),
		Entry("1 resource external-provider, 1 resource matching 1 USB device total",
			[]v1.USBHostDevice{
				{
					ResourceName: resourceName1,
					Selectors: []v1.USBSelector{
						{
							Vendor:  fmt.Sprintf("%x", usbs[0].Vendor),
							Product: fmt.Sprintf("%x", usbs[0].Product),
						},
					},
				},
				{
					ResourceName:             resourceName2,
					ExternalResourceProvider: true,
					Selectors: []v1.USBSelector{
						{
							Vendor:  fmt.Sprintf("%x", usbs[1].Vendor),
							Product: fmt.Sprintf("%x", usbs[1].Product),
						},
					},
				},
			},
			map[string][]*PluginDevices{
				resourceName1: {
					newPluginDevices(resourceName1, 0, []*USBDevice{usbs[0]}),
				},
			},
		),
		Entry("Should ignore a config with same USB selector",
			[]v1.USBHostDevice{
				{
					ResourceName: resourceName1,
					Selectors: []v1.USBSelector{
						{
							Vendor:  fmt.Sprintf("%x", usbs[0].Vendor),
							Product: fmt.Sprintf("%x", usbs[0].Product),
						},
					},
				},
				{
					ResourceName: resourceName2,
					Selectors: []v1.USBSelector{
						{
							Vendor:  fmt.Sprintf("%x", usbs[0].Vendor),
							Product: fmt.Sprintf("%x", usbs[0].Product),
						},
						{
							Vendor:  fmt.Sprintf("%x", usbs[1].Vendor),
							Product: fmt.Sprintf("%x", usbs[1].Product),
						},
					},
				},
			},
			map[string][]*PluginDevices{
				resourceName1: {
					newPluginDevices(resourceName1, 0, []*USBDevice{usbs[0]}),
				},
			},
		),
	)
	It("Should return empty when encountering an error", func() {
		originalPath := pathToUSBDevices
		defer func() {
			pathToUSBDevices = originalPath
		}()
		pathToUSBDevices = "/this/path/does/not/exist"

		devices := discoverPluggedUSBDevices()
		Expect(devices.devices).To(BeEmpty())
	})

	It("Should setup watcher for USB devices", func() {
		watcher, err := fsnotify.NewWatcher()
		Expect(err).ToNot(HaveOccurred())
		defer watcher.Close()

		monitoredDevices := make(map[string]string)
		err = dpi.SetupMonitoredDevicesFunc(watcher, monitoredDevices)
		Expect(err).ToNot(HaveOccurred())

		Expect(monitoredDevices).To(HaveLen(3))
		Expect(watcher.WatchList()).To(HaveLen(3))
	})

	It("Should return error if device directory does not exist", func() {
		badDevices := []*PluginDevices{{ID: "bad", Devices: []*USBDevice{{DevicePath: "/missing/device"}}}}
		badPlugin := NewUSBDevicePlugin(resourceName1, workDir, badDevices, nil)

		watcher, _ := fsnotify.NewWatcher()
		defer watcher.Close()

		err := badPlugin.SetupMonitoredDevicesFunc(watcher, make(map[string]string))
		Expect(err).To(MatchError(ContainSubstring("failed to watch device")))
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
		// Devices[0] has 2 USB devices
		Expect(allocateResponse.ContainerResponses[0].Devices).To(HaveLen(2))
		Expect(allocateResponse.ContainerResponses[0].Devices[0].HostPath).To(Equal(usbs[0].DevicePath))
		Expect(allocateResponse.ContainerResponses[0].Devices[0].ContainerPath).To(Equal(usbs[0].DevicePath))
		Expect(allocateResponse.ContainerResponses[0].Devices[0].Permissions).To(Equal("mrw"))
		Expect(allocateResponse.ContainerResponses[0].Devices[1].HostPath).To(Equal(usbs[1].DevicePath))

		Expect(allocateResponse.ContainerResponses[0].Envs).To(HaveLen(1))
		for key, val := range allocateResponse.ContainerResponses[0].Envs {
			Expect(key).To(ContainSubstring("USB_RESOURCE_"))
			// 3:11 and 4:7
			Expect(val).To(ContainSubstring(fmt.Sprintf("%d:%d", usbs[0].Bus, usbs[0].DeviceNumber)))
			Expect(val).To(ContainSubstring(fmt.Sprintf("%d:%d", usbs[1].Bus, usbs[1].DeviceNumber)))
		}
	})
})

func expectMatch(a, b *USBDevice) {
	Expect(a.Vendor).To(Equal(b.Vendor))
	Expect(a.Product).To(Equal(b.Product))
	Expect(a.Bus).To(Equal(b.Bus))
	Expect(a.DeviceNumber).To(Equal(b.DeviceNumber))
	Expect(a.BCD).To(Equal(b.BCD))
	Expect(a.DevicePath).To(Equal(b.DevicePath))
}
