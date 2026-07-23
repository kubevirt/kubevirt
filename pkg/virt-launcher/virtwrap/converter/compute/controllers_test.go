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

package compute_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
)

var _ = Describe("Controllers Domain Configurator", func() {
	const (
		usbNeeded                   = true
		pciHole64DisablingSupported = true
	)

	DescribeTable("should configure USB and SCSI controllers", func(
		vmi *v1.VirtualMachineInstance,
		isUSBNeeded bool,
		autoThreads int,
		expectedControllers []api.Controller,
	) {
		var domain api.Domain

		Expect(compute.NewControllersDomainConfigurator(
			compute.ControllersWithUSBNeeded(isUSBNeeded),
			compute.ControllersWithSCSIModel("test-model"),
			compute.ControllersWithSCSIIOThreads(uint(autoThreads)),
			compute.ControllersWithUseLaunchSecuritySEV(false),
			compute.ControllersWithUseLaunchSecurityPV(false),
			compute.ControllersWithVirtioSerialModel("virtio-test-model"),
		).Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					Controllers: expectedControllers,
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	},
		Entry("when USB is NOT needed and disk hotplug is disabled",
			libvmi.New(withHotplugDisabled()),
			!usbNeeded,
			0,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when USB is needed and disk hotplug is disabled",
			libvmi.New(withHotplugDisabled()),
			usbNeeded,
			0,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "qemu-xhci"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when USB is NOT needed and disk hotplug is enabled",
			libvmi.New(),
			!usbNeeded,
			0,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "test-model"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when USB is needed and disk hotplug is enabled",
			libvmi.New(),
			usbNeeded,
			0,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "qemu-xhci"},
				{Type: "scsi", Index: "0", Model: "test-model"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when VMI has SCSI disk and disk hotplug is disabled",
			libvmi.New(withHotplugDisabled(), libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			!usbNeeded,
			0,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "test-model"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when VMI has SCSI disk and disk hotplug is enabled",
			libvmi.New(libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			!usbNeeded,
			0,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "test-model"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when VMI has SCSI disk and USB is needed",
			libvmi.New(libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			usbNeeded,
			0,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "qemu-xhci"},
				{Type: "scsi", Index: "0", Model: "test-model"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when VMI has SCSI disk with dedicatedIOThread and Virtio disk, VMI has 4 shared IO threads",
			libvmi.New(
				libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI, libvmi.WithDedicatedIOThreads(true)),
				libvmi.WithDisk("virtio-disk", v1.DiskBusVirtio),
			),
			!usbNeeded,
			4,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "test-model", Driver: &api.ControllerDriver{
					Queues: new(uint(1)), IOThread: new(uint(2)),
				}},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when VMI has SCSI disk with dedicatedIOThread and Virtio disks, VMI has 2 shared IO threads, should roll over controller thread",
			libvmi.New(
				libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI, libvmi.WithDedicatedIOThreads(true)),
				libvmi.WithDisk("virtio-disk1", v1.DiskBusVirtio),
				libvmi.WithDisk("virtio-disk2", v1.DiskBusVirtio),
			),
			!usbNeeded,
			2,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "test-model", Driver: &api.ControllerDriver{
					Queues: new(uint(1)), IOThread: new(uint(1)),
				}},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when VMI has multiple SCSI disks with dedicatedIOThread, VMI has 4 shared IO threads",
			libvmi.New(
				libvmi.WithDisk("scsi-disk1", v1.DiskBusSCSI, libvmi.WithDedicatedIOThreads(true)),
				libvmi.WithDisk("scsi-disk2", v1.DiskBusSCSI, libvmi.WithDedicatedIOThreads(true)),
			),
			!usbNeeded,
			4,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "test-model", Driver: &api.ControllerDriver{
					Queues: new(uint(1)), IOThread: new(uint(1)),
				}},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when VMI has SCSI disk with dedicatedIOThread and VMI has no IOThreads",
			libvmi.New(libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			!usbNeeded,
			0,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "test-model"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when VMI has SCSI disk without dedicatedIOThread and VMI has IOThreads",
			libvmi.New(libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			!usbNeeded,
			4,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "test-model"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
	)

	DescribeTable("should configure PCI controller based on arch support and annotation", func(
		vmi *v1.VirtualMachineInstance,
		supportPCIHole64Disabling bool,
		expectedControllers []api.Controller,
	) {
		var domain api.Domain

		configurator := compute.NewControllersDomainConfigurator(
			compute.ControllersWithUSBNeeded(!usbNeeded),
			compute.ControllersWithSCSIModel("test-model"),
			compute.ControllersWithSCSIIOThreads(0),
			compute.ControllersWithUseLaunchSecuritySEV(false),
			compute.ControllersWithUseLaunchSecurityPV(false),
			compute.ControllersWithSupportPCIHole64Disabling(supportPCIHole64Disabling),
			compute.ControllersWithVirtioSerialModel("virtio-test-model"),
		)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		Expect(domain).To(Equal(newDomainWithControllers(expectedControllers)))
	},
		Entry("when arch does not support PCIHole64 disabling, annotation not set",
			libvmi.New(),
			!pciHole64DisablingSupported,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "test-model"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when arch does not support PCIHole64 disabling, annotation set",
			libvmi.New(libvmi.WithAnnotation(v1.DisablePCIHole64, "true")),
			!pciHole64DisablingSupported,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "test-model"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when arch supports PCIHole64 disabling, annotation not set",
			libvmi.New(),
			pciHole64DisablingSupported,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "test-model"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when arch supports PCIHole64 disabling, annotation set to false",
			libvmi.New(libvmi.WithAnnotation(v1.DisablePCIHole64, "false")),
			pciHole64DisablingSupported,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "test-model"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when arch supports PCIHole64 disabling and annotation is true",
			libvmi.New(libvmi.WithAnnotation(v1.DisablePCIHole64, "true")),
			pciHole64DisablingSupported,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "test-model"},
				{Type: "pci", Index: "0", Model: "pcie-root", PCIHole64: &api.PCIHole64{Value: 0, Unit: "KiB"}},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
	)

	DescribeTable("should set IOMMU on SCSI and virtio-serial controllers when launch security is active",
		func(useLaunchSecuritySEV, useLaunchSecurityPV bool, vmi *v1.VirtualMachineInstance, autoThreads int, expectedDomain api.Domain) {
			var domain api.Domain

			Expect(compute.NewControllersDomainConfigurator(
				compute.ControllersWithUSBNeeded(false),
				compute.ControllersWithSCSIModel("test-model"),
				compute.ControllersWithSCSIIOThreads(uint(autoThreads)),
				compute.ControllersWithUseLaunchSecuritySEV(useLaunchSecuritySEV),
				compute.ControllersWithUseLaunchSecurityPV(useLaunchSecurityPV),
				compute.ControllersWithVirtioSerialModel("virtio-test-model"),
			).Configure(vmi, &domain)).To(Succeed())

			Expect(domain).To(Equal(expectedDomain))
		},
		Entry("when SEV is active, SCSI and virtio-serial get IOMMU driver",
			true, false,
			libvmi.New(),
			0,
			newDomainWithControllers([]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{
					Type: "scsi", Index: "0", Model: "test-model",
					Driver: &api.ControllerDriver{IOMMU: "on"},
				},
				{
					Type: "virtio-serial", Index: "0", Model: "virtio-test-model",
					Driver: &api.ControllerDriver{IOMMU: "on"},
				},
			})),
		Entry("when PV is active, SCSI and virtio-serial get IOMMU driver",
			false, true,
			libvmi.New(),
			0,
			newDomainWithControllers([]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{
					Type: "scsi", Index: "0", Model: "test-model",
					Driver: &api.ControllerDriver{IOMMU: "on"},
				},
				{
					Type: "virtio-serial", Index: "0", Model: "virtio-test-model",
					Driver: &api.ControllerDriver{IOMMU: "on"},
				},
			})),
		Entry("when SEV is active with SCSI IOThreads, IOMMU is preserved alongside IOThread and Queues",
			true, false,
			libvmi.New(
				libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI, libvmi.WithDedicatedIOThreads(true)),
				libvmi.WithDisk("virtio-disk", v1.DiskBusVirtio),
			),
			4,
			newDomainWithControllers([]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{
					Type: "scsi", Index: "0", Model: "test-model",
					Driver: &api.ControllerDriver{
						IOMMU:    "on",
						Queues:   new(uint(1)),
						IOThread: new(uint(2)),
					},
				},
				{
					Type: "virtio-serial", Index: "0", Model: "virtio-test-model",
					Driver: &api.ControllerDriver{IOMMU: "on"},
				},
			})),
	)

	DescribeTable("should configure virtio-serial controller based on serial console setting", func(
		vmiOpts []libvmi.Option,
		expectedControllers []api.Controller,
	) {
		var domain api.Domain

		// withHotplugDisabled() is applied to all entries to prevent the SCSI controller
		// from being added (hotplug enabled triggers SCSI), keeping the test focused on the
		// virtio-serial controller only.
		vmi := libvmi.New(append([]libvmi.Option{withHotplugDisabled()}, vmiOpts...)...)

		configurator := compute.NewControllersDomainConfigurator(
			compute.ControllersWithUSBNeeded(!usbNeeded),
			compute.ControllersWithVirtioSerialModel("virtio-test-model"),
			compute.ControllersWithUseLaunchSecuritySEV(false),
			compute.ControllersWithUseLaunchSecurityPV(false),
		)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		Expect(domain).To(Equal(newDomainWithControllers(expectedControllers)))
	},
		Entry("when serial console is enabled by default (nil)",
			nil,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when serial console is explicitly enabled",
			[]libvmi.Option{libvmi.WithAutoattachSerialConsole(true)},
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when serial console is disabled",
			[]libvmi.Option{libvmi.WithAutoattachSerialConsole(false)},
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
			}),
	)
})

func newDomainWithControllers(controllers []api.Controller) api.Domain {
	return api.Domain{
		Spec: api.DomainSpec{
			Devices: api.Devices{
				Controllers: controllers,
			},
		},
	}
}

func withHotplugDisabled() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.DisableHotplug = true
	}
}
