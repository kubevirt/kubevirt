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

var _ = Describe("Controller Domain Configurator", func() {
	DescribeTable("should configure controllers based on architecture",
		func(arch, usbModel, scsiModel, serialModel string) {
			vmi := libvmi.New()
			var domain api.Domain

			Expect(compute.NewControllerDomainConfigurator(compute.WithArchitecture(arch)).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Controllers: []api.Controller{
							{Type: "usb", Index: "0", Model: usbModel},
							{Type: "scsi", Index: "0", Model: scsiModel},
							{Type: "virtio-serial", Index: "0", Model: serialModel},
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		},
		Entry("amd64", "amd64", "none", "virtio-non-transitional", "virtio-non-transitional"),
		Entry("arm64", "arm64", "qemu-xhci", "virtio-non-transitional", "virtio-non-transitional"),
		Entry("s390x", "s390x", "none", "virtio-scsi", "virtio"),
	)

	Context("USB Controller", func() {
		Context("on amd64", func() {
			DescribeTable("should enable USB controller", func(vmiMutator func(*v1.VirtualMachineInstance)) {
				vmi := libvmi.New()
				vmi.Spec.Domain.Devices.DisableHotplug = true
				vmi.Spec.Domain.Devices.AutoattachSerialConsole = pointer.P(false)
				vmiMutator(vmi)
				var domain api.Domain

				Expect(compute.NewControllerDomainConfigurator(compute.WithArchitecture("amd64")).Configure(vmi, &domain)).To(Succeed())

				expectedDomain := api.Domain{
					Spec: api.DomainSpec{
						Devices: api.Devices{
							Controllers: []api.Controller{
								{Type: "usb", Index: "0", Model: "qemu-xhci"},
							},
						},
					},
				}
				Expect(domain).To(Equal(expectedDomain))
			},
				Entry("when input device uses USB bus", func(vmi *v1.VirtualMachineInstance) {
					vmi.Spec.Domain.Devices.Inputs = []v1.Input{
						{Name: "tablet", Type: "tablet", Bus: "usb"},
					}
				}),
				Entry("when disk uses USB bus", func(vmi *v1.VirtualMachineInstance) {
					vmi.Spec.Domain.Devices.Disks = []v1.Disk{
						{
							Name: "usb-disk",
							DiskDevice: v1.DiskDevice{
								Disk: &v1.DiskTarget{Bus: v1.DiskBusUSB},
							},
						},
					}
				}),
				Entry("when client passthrough is specified", func(vmi *v1.VirtualMachineInstance) {
					vmi.Spec.Domain.Devices.ClientPassthrough = &v1.ClientPassthroughDevices{}
				}),
			)
		})
	})

	Context("SCSI Controller", func() {
		DescribeTable("should add SCSI controller when disk uses SCSI bus",
			func(arch, usbModel, scsiModel, serialModel string) {
				vmi := libvmi.New()
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{
					{
						Name: "scsi-disk",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{Bus: v1.DiskBusSCSI},
						},
					},
				}
				var domain api.Domain

				Expect(compute.NewControllerDomainConfigurator(
					compute.WithArchitecture(arch),
				).Configure(vmi, &domain)).To(Succeed())

				expectedDomain := api.Domain{
					Spec: api.DomainSpec{
						Devices: api.Devices{
							Controllers: []api.Controller{
								{Type: "usb", Index: "0", Model: usbModel},
								{Type: "scsi", Index: "0", Model: scsiModel},
								{Type: "virtio-serial", Index: "0", Model: serialModel},
							},
						},
					},
				}
				Expect(domain).To(Equal(expectedDomain))
			},
			Entry("amd64", "amd64", "none", "virtio-non-transitional", "virtio-non-transitional"),
			Entry("arm64", "arm64", "qemu-xhci", "virtio-non-transitional", "virtio-non-transitional"),
			Entry("s390x", "s390x", "none", "virtio-scsi", "virtio"),
		)

		DescribeTable("should use virtio-transitional when enabled", func(architecture, expectedUSBModel string) {
			vmi := libvmi.New()
			var domain api.Domain

			Expect(compute.NewControllerDomainConfigurator(
				compute.WithArchitecture(architecture),
				compute.WithUseVirtioTransitional(true),
			).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Controllers: []api.Controller{
							{Type: "usb", Index: "0", Model: expectedUSBModel},
							{Type: "scsi", Index: "0", Model: "virtio-transitional"},
							{Type: "virtio-serial", Index: "0", Model: "virtio-transitional"},
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		},
			Entry("amd64", "amd64", "none"),
			Entry("arm64", "arm64", "qemu-xhci"),
		)

		DescribeTable("should configure IOMMU driver when launch security is enabled",
			func(arch, usbModel, scsiModel, serialModel string, opt compute.ControllerOption) {
				vmi := libvmi.New()
				var domain api.Domain

				Expect(compute.NewControllerDomainConfigurator(
					compute.WithArchitecture(arch),
					opt,
				).Configure(vmi, &domain)).To(Succeed())

				iommuDriver := &api.ControllerDriver{IOMMU: "on"}
				expectedDomain := api.Domain{
					Spec: api.DomainSpec{
						Devices: api.Devices{
							Controllers: []api.Controller{
								{Type: "usb", Index: "0", Model: usbModel},
								{Type: "scsi", Index: "0", Model: scsiModel, Driver: iommuDriver},
								{Type: "virtio-serial", Index: "0", Model: serialModel, Driver: iommuDriver},
							},
						},
					},
				}
				Expect(domain).To(Equal(expectedDomain))
			},
			Entry("amd64 with SEV",
				"amd64", "none", "virtio-non-transitional", "virtio-non-transitional",
				compute.WithUseLaunchSecuritySEV(true)),
			Entry("amd64 with PV",
				"amd64", "none", "virtio-non-transitional", "virtio-non-transitional",
				compute.WithUseLaunchSecurityPV(true)),
			Entry("arm64 with SEV",
				"arm64", "qemu-xhci", "virtio-non-transitional", "virtio-non-transitional",
				compute.WithUseLaunchSecuritySEV(true)),
			Entry("arm64 with PV",
				"arm64", "qemu-xhci", "virtio-non-transitional", "virtio-non-transitional",
				compute.WithUseLaunchSecurityPV(true)),
			Entry("s390x with SEV",
				"s390x", "none", "virtio-scsi", "virtio",
				compute.WithUseLaunchSecuritySEV(true)),
			Entry("s390x with PV",
				"s390x", "none", "virtio-scsi", "virtio",
				compute.WithUseLaunchSecurityPV(true)),
		)
	})

	Context("PCI Controller", func() {
		It("should add PCI controller on amd64 when DisablePCIHole64 annotation is set", func() {
			vmi := libvmi.New()
			vmi.Annotations = map[string]string{
				v1.DisablePCIHole64: "true",
			}
			var domain api.Domain

			Expect(compute.NewControllerDomainConfigurator(
				compute.WithArchitecture("amd64"),
			).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Controllers: []api.Controller{
							{Type: "usb", Index: "0", Model: "none"},
							{Type: "scsi", Index: "0", Model: "virtio-non-transitional"},
							{Type: "virtio-serial", Index: "0", Model: "virtio-non-transitional"},
							{
								Type:      "pci",
								Index:     "0",
								Model:     "pcie-root",
								PCIHole64: &api.PCIHole64{Value: 0, Unit: "KiB"},
							},
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		})
	})

	Context("should not add controllers", func() {
		DescribeTable("SCSI when hotplug disabled and no SCSI disks",
			func(arch, usbModel, serialModel string) {
				vmi := libvmi.New()
				vmi.Spec.Domain.Devices.DisableHotplug = true
				var domain api.Domain

				Expect(compute.NewControllerDomainConfigurator(compute.WithArchitecture(arch)).Configure(vmi, &domain)).To(Succeed())

				expectedDomain := api.Domain{
					Spec: api.DomainSpec{
						Devices: api.Devices{
							Controllers: []api.Controller{
								{Type: "usb", Index: "0", Model: usbModel},
								{Type: "virtio-serial", Index: "0", Model: serialModel},
							},
						},
					},
				}
				Expect(domain).To(Equal(expectedDomain))
			},
			Entry("amd64", "amd64", "none", "virtio-non-transitional"),
			Entry("arm64", "arm64", "qemu-xhci", "virtio-non-transitional"),
			Entry("s390x", "s390x", "none", "virtio"),
		)

		DescribeTable("virtio-serial when AutoattachSerialConsole disabled",
			func(arch, usbModel, scsiModel string) {
				vmi := libvmi.New()
				vmi.Spec.Domain.Devices.AutoattachSerialConsole = pointer.P(false)
				var domain api.Domain

				Expect(compute.NewControllerDomainConfigurator(compute.WithArchitecture(arch)).Configure(vmi, &domain)).To(Succeed())

				expectedDomain := api.Domain{
					Spec: api.DomainSpec{
						Devices: api.Devices{
							Controllers: []api.Controller{
								{Type: "usb", Index: "0", Model: usbModel},
								{Type: "scsi", Index: "0", Model: scsiModel},
							},
						},
					},
				}
				Expect(domain).To(Equal(expectedDomain))
			},
			Entry("amd64", "amd64", "none", "virtio-non-transitional"),
			Entry("arm64", "arm64", "qemu-xhci", "virtio-non-transitional"),
			Entry("s390x", "s390x", "none", "virtio-scsi"),
		)

		DescribeTable("PCI when DisablePCIHole64 annotation is not set or on non-amd64",
			func(arch, usbModel, scsiModel, serialModel string) {
				vmi := libvmi.New()
				vmi.Annotations = map[string]string{
					v1.DisablePCIHole64: "true",
				}
				var domain api.Domain

				Expect(compute.NewControllerDomainConfigurator(compute.WithArchitecture(arch)).Configure(vmi, &domain)).To(Succeed())

				expectedDomain := api.Domain{
					Spec: api.DomainSpec{
						Devices: api.Devices{
							Controllers: []api.Controller{
								{Type: "usb", Index: "0", Model: usbModel},
								{Type: "scsi", Index: "0", Model: scsiModel},
								{Type: "virtio-serial", Index: "0", Model: serialModel},
							},
						},
					},
				}
				Expect(domain).To(Equal(expectedDomain))
			},
			Entry("arm64", "arm64", "qemu-xhci", "virtio-non-transitional", "virtio-non-transitional"),
			Entry("s390x", "s390x", "none", "virtio-scsi", "virtio"),
		)
	})
})
