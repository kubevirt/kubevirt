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
		usbNeeded                 = true
		pciHole64DisablingSupport = true
	)

	DescribeTable("should configure controllers",
		func(vmi *v1.VirtualMachineInstance, isUSBNeeded, supportPCIHole64Disabling bool, expectedControllers []api.Controller) {
			var domain api.Domain

			Expect(compute.NewControllersDomainConfigurator(
				compute.ControllersWithUSBNeeded(isUSBNeeded),
				compute.ControllersWithSCSIModel("scsi-test-model"),
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
			!usbNeeded, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
			}),
		Entry("when USB needed, hotplug disabled, PCIHole64 disabling NOT supported",
			libvmi.New(withHotplugDisabled()),
			usbNeeded, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "qemu-xhci"},
			}),
		Entry("when USB NOT needed, hotplug enabled, PCIHole64 disabling NOT supported",
			libvmi.New(),
			!usbNeeded, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
			}),
		Entry("when USB needed, hotplug enabled, PCIHole64 disabling NOT supported",
			libvmi.New(),
			usbNeeded, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "qemu-xhci"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
			}),
		Entry("when SCSI disk present, hotplug disabled, PCIHole64 disabling NOT supported",
			libvmi.New(withHotplugDisabled(), libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			!usbNeeded, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
			}),
		Entry("when SCSI disk present, hotplug enabled, PCIHole64 disabling NOT supported",
			libvmi.New(libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			!usbNeeded, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
			}),
		Entry("when SCSI disk present, USB needed, PCIHole64 disabling NOT supported",
			libvmi.New(libvmi.WithDisk("scsi-disk", v1.DiskBusSCSI)),
			usbNeeded, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "qemu-xhci"},
				{Type: "scsi", Index: "0", Model: "scsi-test-model"},
			}),
		Entry("when hotplug disabled, PCIHole64 disabling supported, annotation NOT set",
			libvmi.New(withHotplugDisabled()),
			!usbNeeded, pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
			}),
		Entry("when hotplug disabled, PCIHole64 disabling NOT supported, annotation set",
			libvmi.New(withHotplugDisabled(), withDisablePCIHole64Annotation()),
			!usbNeeded, !pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
			}),
		Entry("when hotplug disabled, PCIHole64 disabling supported, annotation set",
			libvmi.New(withHotplugDisabled(), withDisablePCIHole64Annotation()),
			!usbNeeded, pciHole64DisablingSupport,
			[]api.Controller{
				{Type: "usb", Index: "0", Model: "none"},
				{Type: "pci", Index: "0", Model: "pcie-root", PCIHole64: &api.PCIHole64{Value: 0, Unit: "KiB"}},
			}),
		Entry("when hotplug enabled, PCIHole64 disabling supported, annotation set",
			libvmi.New(withDisablePCIHole64Annotation()),
			!usbNeeded, pciHole64DisablingSupport,
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
