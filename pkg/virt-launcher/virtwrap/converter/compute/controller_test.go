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

var _ = Describe("Controller Domain Configurator", func() {
	DescribeTable("should configure controllers based on architecture", func(architecture, expectedUSBModel, expectedSCSIModel string) {
		vmi := libvmi.New()
		var domain api.Domain

		Expect(compute.NewControllerDomainConfigurator(compute.WithArchitecture(architecture)).Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					Controllers: []api.Controller{
						{Type: "usb", Index: "0", Model: expectedUSBModel},
						{Type: "scsi", Index: "0", Model: expectedSCSIModel},
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

	Context("USB Controller", func() {
		Context("on amd64", func() {
			DescribeTable("should enable USB controller", func(vmiMutator func(*v1.VirtualMachineInstance)) {
				vmi := libvmi.New()
				vmi.Spec.Domain.Devices.DisableHotplug = true
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
		DescribeTable("should not add SCSI controller when hotplug is disabled and no SCSI disks", func(architecture, expectedUSBModel string) {
			vmi := libvmi.New()
			vmi.Spec.Domain.Devices.DisableHotplug = true
			var domain api.Domain

			Expect(compute.NewControllerDomainConfigurator(compute.WithArchitecture(architecture)).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Controllers: []api.Controller{
							{Type: "usb", Index: "0", Model: expectedUSBModel},
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		},
			Entry("amd64", "amd64", "none"),
			Entry("arm64", "arm64", "qemu-xhci"),
			Entry("s390x", "s390x", "none"),
		)

		DescribeTable("should add SCSI controller when disk uses SCSI bus", func(architecture, expectedUSBModel, expectedSCSIModel string) {
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

			Expect(compute.NewControllerDomainConfigurator(compute.WithArchitecture(architecture)).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Controllers: []api.Controller{
							{Type: "usb", Index: "0", Model: expectedUSBModel},
							{Type: "scsi", Index: "0", Model: expectedSCSIModel},
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
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		},
			Entry("amd64", "amd64", "none"),
			Entry("arm64", "arm64", "qemu-xhci"),
		)

		Context("on amd64", func() {
			DescribeTable("should configure IOMMU driver when launch security is enabled", func(opt compute.ControllerOption) {
				vmi := libvmi.New()
				var domain api.Domain

				Expect(compute.NewControllerDomainConfigurator(
					compute.WithArchitecture("amd64"),
					opt,
				).Configure(vmi, &domain)).To(Succeed())

				expectedDomain := api.Domain{
					Spec: api.DomainSpec{
						Devices: api.Devices{
							Controllers: []api.Controller{
								{Type: "usb", Index: "0", Model: "none"},
								{Type: "scsi", Index: "0", Model: "virtio-non-transitional", Driver: &api.ControllerDriver{IOMMU: "on"}},
							},
						},
					},
				}
				Expect(domain).To(Equal(expectedDomain))
			},
				Entry("SEV", compute.WithUseLaunchSecuritySEV(true)),
				Entry("PV", compute.WithUseLaunchSecurityPV(true)),
			)
		})
	})
})
