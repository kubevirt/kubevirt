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
		usbNeeded                 = true
		pciHole64DisablingSupport = true
	)

	DescribeTable("should configure controllers",
		func(vmi *v1.VirtualMachineInstance, isUSBNeeded bool, autoThreads int, supportPCIHole64Disabling bool, expectedControllers []api.Controller) {
			var domain api.Domain
			Expect(compute.NewControllersDomainConfigurator(
				compute.ControllersWithUSBNeeded(isUSBNeeded),
				compute.ControllersWithSCSIModel("scsi-test-model"),
				compute.ControllersWithSCSIIOThreads(uint(autoThreads)),
				compute.ControllersWithControllerDriver(nil),
				compute.ControllersWithSupportPCIHole64Disabling(supportPCIHole64Disabling),
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
		Entry("when USB NOT needed, hotplug disabled, PCIHole64 disabling NOT supported",
			libvmi.New(withHotplugDisabled()),
			!usbNeeded, 0, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
			}),
		Entry("when USB needed, hotplug disabled, PCIHole64 disabling NOT supported",
			libvmi.New(withHotplugDisabled()),
			usbNeeded, 0, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "qemu-xhci"},
			}),
		Entry("when USB NOT needed, hotplug enabled, PCIHole64 disabling NOT supported",
			libvmi.New(),
			!usbNeeded, 0, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
			}),
		Entry("when USB needed, hotplug enabled, PCIHole64 disabling NOT supported",
			libvmi.New(),
			usbNeeded, 0, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "qemu-xhci"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
			}),
		Entry("when SCSI disk present, hotplug disabled, PCIHole64 disabling NOT supported",
			libvmi.New(withHotplugDisabled(), libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			!usbNeeded, 0, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
			}),
		Entry("when SCSI disk present, hotplug enabled, PCIHole64 disabling NOT supported",
			libvmi.New(libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			!usbNeeded, 0, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
			}),
		Entry("when SCSI disk present, USB needed, PCIHole64 disabling NOT supported",
			libvmi.New(libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			usbNeeded, 0, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "qemu-xhci"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
			}),
		Entry("when VMI has SCSI disk with dedicatedIOThread and Virtio disk, VMI has 4 shared IO threads",
			libvmi.New(
				libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI, libvmi.WithDedicatedIOThreads(true)),
				libvmi.WithDisk("virtio-disk", v1.DiskBusVirtio),
			),
			!usbNeeded, 4, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model", Driver: &api.ControllerDriver{Queues: pointer.P[uint](1), IOThread: pointer.P[uint](2)}},
			}),
		Entry("when VMI has SCSI disk with dedicatedIOThread and Virtio disks, VMI has 2 shared IO threads, should roll over controller thread",
			libvmi.New(
				libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI, libvmi.WithDedicatedIOThreads(true)),
				libvmi.WithDisk("virtio-disk1", v1.DiskBusVirtio),
				libvmi.WithDisk("virtio-disk2", v1.DiskBusVirtio),
			),
			!usbNeeded, 2, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model", Driver: &api.ControllerDriver{Queues: pointer.P[uint](1), IOThread: pointer.P[uint](1)}},
			}),
		Entry("when VMI has multiple SCSI disks with dedicatedIOThread, VMI has 4 shared IO threads",
			libvmi.New(
				libvmi.WithDisk("scsi-disk1", v1.DiskBusSCSI, libvmi.WithDedicatedIOThreads(true)),
				libvmi.WithDisk("scsi-disk2", v1.DiskBusSCSI, libvmi.WithDedicatedIOThreads(true)),
			),
			!usbNeeded, 4, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model", Driver: &api.ControllerDriver{Queues: pointer.P[uint](1), IOThread: pointer.P[uint](1)}},
			}),
		Entry("when VMI has SCSI disk with dedicatedIOThread and VMI has no IOThreads",
			libvmi.New(libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			!usbNeeded, 0, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
			}),
		Entry("when VMI has SCSI disk without dedicatedIOThread and VMI has IOThreads",
			libvmi.New(libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			!usbNeeded, 4, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
			}),
		Entry("when hotplug disabled, PCIHole64 disabling supported, annotation NOT set",
			libvmi.New(withHotplugDisabled()),
			!usbNeeded, 0, pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
			}),
		Entry("when hotplug disabled, PCIHole64 disabling NOT supported, annotation set",
			libvmi.New(withHotplugDisabled(), withDisablePCIHole64Annotation()),
			!usbNeeded, 0, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
			}),
		Entry("when hotplug disabled, PCIHole64 disabling supported, annotation set",
			libvmi.New(withHotplugDisabled(), withDisablePCIHole64Annotation()),
			!usbNeeded, 0, pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "pci", Index: "0", Model: "pcie-root", PCIHole64: &api.PCIHole64{Value: 0, Unit: "KiB"}},
			}),
		Entry("when hotplug enabled, PCIHole64 disabling supported, annotation set",
			libvmi.New(withDisablePCIHole64Annotation()),
			!usbNeeded, 0, pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
				{Type: "pci", Index: "0", Model: "pcie-root", PCIHole64: &api.PCIHole64{Value: 0, Unit: "KiB"}},
			}),
	)
})

func withHotplugDisabled() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.DisableHotplug = true
	}
}

func withDisablePCIHole64Annotation() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Annotations == nil {
			vmi.Annotations = make(map[string]string)
		}
		vmi.Annotations[v1.DisablePCIHole64] = "true"
	}
}
