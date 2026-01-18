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

	DescribeTable("should configure USB and SCSI controllers", func(vmi *v1.VirtualMachineInstance, isUSBNeeded bool, expectedControllers []api.Controller) {
		var domain api.Domain

		Expect(compute.NewControllersDomainConfigurator(
			compute.ControllersWithUSBNeeded(isUSBNeeded),
			compute.ControllersWithSCSIModel("test-model"),
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
		Entry("when USB is NOT needed and disk hotplug is disabled",
			libvmi.New(withHotplugDisabled()),
			!usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
			}),
		Entry("when USB is needed and disk hotplug is disabled",
			libvmi.New(withHotplugDisabled()),
			usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "qemu-xhci"},
			}),
		Entry("when USB is NOT needed and disk hotplug is enabled",
			libvmi.New(),
			!usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "test-model"},
			}),
		Entry("when USB is needed and disk hotplug is enabled",
			libvmi.New(),
			usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "qemu-xhci"},
				{Type: "scsi", Index: "0", Model: "test-model"},
			}),
		Entry("when VMI has SCSI disk and disk hotplug is disabled",
			libvmi.New(withHotplugDisabled(), libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			!usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "test-model"},
			}),
		Entry("when VMI has SCSI disk and disk hotplug is enabled",
			libvmi.New(libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			!usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "test-model"},
			}),
		Entry("when VMI has SCSI disk and USB is needed",
			libvmi.New(libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			usbNeeded,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "qemu-xhci"},
				{Type: "scsi", Index: "0", Model: "test-model"},
			}),
	)
})

func withHotplugDisabled() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.DisableHotplug = true
	}
}
