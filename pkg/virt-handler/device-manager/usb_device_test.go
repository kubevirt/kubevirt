package device_manager

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
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
		usbDevices := make(map[int][]*USBDevice, 0)
		for _, device := range usbs {
			usbDevices[device.Vendor] = append(usbDevices[device.Vendor], device)
		}
		return &LocalDevices{devices: usbDevices}
	}

	const resourceName1 = "testing.usb/usecase"
	const resourceName2 = "testing.usb/another"

	DescribeTable("with USBHostDevice configuration", func(hostDeviceConfig []v1.USBHostDevice, result map[string][]*PluginDevices) {
		discoverLocalUSBDevicesFunc = findAll
		pdmap := discoverAllowedUSBDevices(hostDeviceConfig)

		Expect(pdmap).To(HaveLen(len(result)), "Expected number of resource names")
		for resourceName, pluginDevices := range pdmap {
			Expect(pluginDevices).To(HaveLen(len(result[resourceName])), "Number of k8s devices")
			for i, dev := range pluginDevices {
				Expect(dev.isHealthy).To(BeTrue())
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
				resourceName1: []*PluginDevices{
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
				resourceName1: []*PluginDevices{
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
				resourceName1: []*PluginDevices{
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
				resourceName1: []*PluginDevices{
					newPluginDevices(resourceName1, 0, []*USBDevice{usbs[0]}),
				},
				resourceName2: []*PluginDevices{
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
				resourceName1: []*PluginDevices{
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
				resourceName1: []*PluginDevices{
					newPluginDevices(resourceName1, 0, []*USBDevice{usbs[0]}),
				},
			},
		),
	)
})

func expectMatch(a, b *USBDevice) {
	Expect(a.Vendor).To(Equal(b.Vendor))
	Expect(a.Product).To(Equal(b.Product))
	Expect(a.Bus).To(Equal(b.Bus))
	Expect(a.DeviceNumber).To(Equal(b.DeviceNumber))
	Expect(a.BCD).To(Equal(b.BCD))
	Expect(a.DevicePath).To(Equal(b.DevicePath))
}
