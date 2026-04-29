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
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
)

var _ = Describe("Controllers Domain Configurator", func() {

	const (
		usbNeeded                   = true
		pciHole64DisablingSupported = true
	)

	DescribeTable("should configure USB and SCSI controllers", func(vmi *v1.VirtualMachineInstance, isUSBNeeded bool, autoThreads int, expectedControllers []api.Controller) {
		var domain api.Domain

		Expect(compute.NewControllersDomainConfigurator(
			compute.ControllersWithUSBNeeded(isUSBNeeded),
			compute.ControllersWithSCSIModel("test-model"),
			compute.ControllersWithSCSIIOThreads(uint(autoThreads)),
			compute.ControllersWithControllerDriver(nil),
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
				{Type: "scsi", Index: "0", Model: "test-model", Driver: &api.ControllerDriver{Queues: pointer.P[uint](1), IOThread: pointer.P[uint](2)}},
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
				{Type: "scsi", Index: "0", Model: "test-model", Driver: &api.ControllerDriver{Queues: pointer.P[uint](1), IOThread: pointer.P[uint](1)}},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when VMI has multiple SCSI disks with dedicatedIOThread, VMI has 4 shared IO threads",
			libvmi.New(
				libvmi.WithDisk("scsi-disk1", v1.DiskBusSCSI, libvmi.WithDedicatedIOThreads(true)),
				libvmi.WithDisk("scsi-disk1", v1.DiskBusSCSI, libvmi.WithDedicatedIOThreads(true)),
			),
			!usbNeeded,
			4,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "test-model", Driver: &api.ControllerDriver{Queues: pointer.P[uint](1), IOThread: pointer.P[uint](1)}},
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

	DescribeTable("should configure PCI controller based on arch support and annotation", func(vmi *v1.VirtualMachineInstance, supportPCIHole64Disabling bool, expectedControllers []api.Controller) {
		var domain api.Domain

		configurator := compute.NewControllersDomainConfigurator(
			compute.ControllersWithUSBNeeded(!usbNeeded),
			compute.ControllersWithSCSIModel("test-model"),
			compute.ControllersWithSCSIIOThreads(0),
			compute.ControllersWithControllerDriver(nil),
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
