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
	"k8s.io/apimachinery/pkg/util/sets"

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
			Healthy:      false,
		},
		// Two identical devices
		{
			Vendor:       4321,
			Product:      8765,
			Bus:          4,
			DeviceNumber: 7,
			BCD:          0,
			DevicePath:   "/dev/bus/usb/004/007",
			Healthy:      false,
		},
		{
			Vendor:       4321,
			Product:      8765,
			Bus:          2,
			DeviceNumber: 10,
			BCD:          0,
			DevicePath:   "/dev/bus/usb/002/010",
			Healthy:      false,
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
		for _, u := range usbs {
			u.Healthy = false
		}
		workDir = GinkgoT().TempDir()
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
		dpi = NewUSBDevicePlugin(resourceName1, workDir, devices, newPermissionManager())
		dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
		dpi.socketPath = filepath.Join(workDir, "kubevirt-test.sock")
		createFile(dpi.socketPath)
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

		By("waiting for initial healthchecks to send Healthy message for each device")
		Eventually(dpi.healthUpdate, 5*time.Second).Should(Receive())
		for i := range dpi.devs {
			Expect(dpi.getDevHealthByIndex(i)).To(Equal(pluginapi.Healthy))
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
		Eventually(dpi.healthUpdate, 5*time.Second).Should(Receive())
		for i := range dpi.devs {
			Expect(dpi.getDevHealthByIndex(i)).To(Equal(pluginapi.Healthy))
		}

		By("Removing (fake) device nodes trigger health updates")
		usbDevicePath1 := filepath.Join(workDir, usbs[0].DevicePath)
		usbDevicePath2 := filepath.Join(workDir, usbs[1].DevicePath)
		Expect(os.Remove(usbDevicePath1)).To(Succeed())
		Eventually(dpi.healthUpdate, 5*time.Second).Should(Receive())
		Expect(os.Remove(usbDevicePath2)).To(Succeed())
		Eventually(dpi.healthUpdate, 5*time.Second).Should(Receive())

		By("healthcheck resolve as Unhealthy")
		Expect(dpi.getDevHealthByIndex(0)).To(Equal(pluginapi.Unhealthy))
		Expect(dpi.getDevHealthByIndex(1)).To(Equal(pluginapi.Healthy))

		By("removing and re-adding bus directory to test if watcher on the directory is added back")
		busDir1 := filepath.Dir(usbDevicePath1)
		Expect(os.Remove(busDir1)).To(Succeed())
		Expect(os.MkdirAll(busDir1, 0755)).To(Succeed())

		By("Creating a new (fake) device node 1")
		createFile(usbDevicePath1)

		By("waiting for healthcheck to send Unhealthy message")
		// Since only one of the two devices in the group is healthy, the healthcheck should send Unhealthy
		Eventually(dpi.healthUpdate, 5*time.Second).Should(Receive())
		Expect(dpi.getDevHealthByIndex(0)).To(Equal(pluginapi.Unhealthy))
		Expect(dpi.getDevHealthByIndex(1)).To(Equal(pluginapi.Healthy))

		By("Creating a new (fake) device node 2")
		createFile(usbDevicePath2)

		By("waiting for healthcheck to send Healthy message")
		// Since both devices in the group are now healthy, the healthcheck should send Healthy
		Eventually(dpi.healthUpdate, 5*time.Second).Should(Receive())
		Expect(dpi.getDevHealthByIndex(0)).To(Equal(pluginapi.Healthy))
		Expect(dpi.getDevHealthByIndex(1)).To(Equal(pluginapi.Healthy))
	})

	It("Should report group unhealthy when one device in the group is down", func() {
		// dpi.devs[0] is a device group containing usbs[0] and usbs[1]
		deviceID := dpi.devs[0].ID
		devicePath1 := filepath.Join(workDir, usbs[0].DevicePath)
		devicePath2 := filepath.Join(workDir, usbs[1].DevicePath)

		By("reporting 1/2 devices in group healthy")
		healthy, err := dpi.mutateHealthUpdateFunc(deviceID, devicePath1, true)
		Expect(err).ToNot(HaveOccurred())
		Expect(healthy).To(BeFalse(), "group should stay unhealthy while second device is still unreported (unhealthy)")

		By("reporting 2/2 devices in group healthy")
		healthy, err = dpi.mutateHealthUpdateFunc(deviceID, devicePath2, true)
		Expect(err).ToNot(HaveOccurred())
		Expect(healthy).To(BeTrue(), "group should be healthy when all devices in the group are healthy")

		By("reporting 1/2 device in group unhealthy")
		healthy, err = dpi.mutateHealthUpdateFunc(deviceID, devicePath1, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(healthy).To(BeFalse(), "one device down in the group should make the whole group unhealthy")

		By("reporting second device healthy while first device is still unhealthy")
		healthy, err = dpi.mutateHealthUpdateFunc(deviceID, devicePath2, true)
		Expect(err).ToNot(HaveOccurred())
		Expect(healthy).To(BeFalse(), "since first device is still unhealthy, the group should stay unhealthy")
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
		err = dpi.setupMonitoredDevicesFunc(watcher, monitoredDevices)
		Expect(err).ToNot(HaveOccurred())

		for i := range len(usbs) {
			Expect(monitoredDevices).To(HaveKey(filepath.Join(workDir, usbs[i].DevicePath)))
		}
		watcherSet := sets.NewString(watcher.WatchList()...)
		for i := range len(usbs) {
			Expect(watcherSet).To(HaveKey(filepath.Dir(filepath.Join(workDir, usbs[i].DevicePath))))
		}
		Expect(watcherSet).To(HaveKey(filepath.Join(workDir, "/dev/bus/usb")))
	})

	It("Should return error if device directory does not exist", func() {
		badDevices := []*PluginDevices{{ID: "bad", Devices: []*USBDevice{{DevicePath: "/missing/device"}}}}
		badPlugin := NewUSBDevicePlugin(resourceName1, workDir, badDevices, nil)

		watcher, _ := fsnotify.NewWatcher()
		defer watcher.Close()

		err := badPlugin.setupMonitoredDevicesFunc(watcher, make(map[string]string))
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
