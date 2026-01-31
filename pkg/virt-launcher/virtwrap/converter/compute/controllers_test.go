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

	const usbNeeded = true

	DescribeTable("should configure controllers", func(vmi *v1.VirtualMachineInstance, isUSBNeeded bool, expectedControllers []api.Controller) {
		var domain api.Domain

		Expect(compute.NewControllersDomainConfigurator(
			compute.ControllersWithUSBNeeded(isUSBNeeded),
			compute.ControllersWithSCSIModel("scsi-test-model"),
			compute.ControllersWithVirtioModel("virtio-test-model"),
			compute.ControllersWithControllerDriver(nil),
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
		Entry("when USB is NOT needed, hotplug disabled, serial console disabled",
			libvmi.New(libvmi.WithoutSerialConsole(), withHotplugDisabled()),
			!usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
			}),
		Entry("when USB is needed, hotplug disabled, serial console disabled",
			libvmi.New(libvmi.WithoutSerialConsole(), withHotplugDisabled()),
			usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "qemu-xhci"},
			}),
		Entry("when USB is NOT needed, hotplug enabled, serial console disabled",
			libvmi.New(libvmi.WithoutSerialConsole()),
			!usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
			}),
		Entry("when USB is needed, hotplug enabled, serial console disabled",
			libvmi.New(libvmi.WithoutSerialConsole()),
			usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "qemu-xhci"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
			}),
		Entry("when VMI has SCSI disk, hotplug disabled, serial console disabled",
			libvmi.New(libvmi.WithoutSerialConsole(), withHotplugDisabled(), libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			!usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
			}),
		Entry("when VMI has SCSI disk, hotplug enabled, serial console disabled",
			libvmi.New(libvmi.WithoutSerialConsole(), libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			!usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
			}),
		Entry("when USB is NOT needed, hotplug disabled, serial console enabled",
			libvmi.New(withHotplugDisabled()),
			!usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when USB is needed, hotplug disabled, serial console enabled",
			libvmi.New(withHotplugDisabled()),
			usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "qemu-xhci"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when USB is NOT needed, hotplug enabled, serial console enabled",
			libvmi.New(),
			!usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when USB is needed, hotplug enabled, serial console enabled",
			libvmi.New(),
			usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "qemu-xhci"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when VMI has SCSI disk, hotplug disabled, serial console enabled",
			libvmi.New(withHotplugDisabled(), libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			!usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
		Entry("when VMI has SCSI disk, hotplug enabled, serial console enabled",
			libvmi.New(libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			!usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
				{Type: "virtio-serial", Index: "0", Model: "virtio-test-model"},
			}),
	)
})

func withHotplugDisabled() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.DisableHotplug = true
	}
}
