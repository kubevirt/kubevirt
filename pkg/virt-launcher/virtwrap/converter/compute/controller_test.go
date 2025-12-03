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
	Context("USB Controller", func() {
		DescribeTable("should configure USB controller based on architecture", func(architecture, expectedModel string) {
			vmi := libvmi.New()
			var domain api.Domain

			Expect(compute.NewControllerDomainConfigurator(compute.WithArchitecture(architecture)).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Controllers: []api.Controller{
							{Type: "usb", Index: "0", Model: expectedModel},
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		},
			Entry("amd64 disables USB by default", "amd64", "none"),
			Entry("arm64 always enables USB", "arm64", "qemu-xhci"),
			Entry("s390x disables USB", "s390x", "none"),
		)

		Context("should configure USB controller model on amd64 architecture", func() {
			It("when input device uses USB bus", func() {
				vmi := libvmi.New()
				vmi.Spec.Domain.Devices.Inputs = []v1.Input{
					{Name: "tablet", Type: "tablet", Bus: "usb"},
				}
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
			})

			It("when disk uses USB bus", func() {
				vmi := libvmi.New()
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{
					{
						Name: "usb-disk",
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: v1.DiskBusUSB,
							},
						},
					},
				}
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
			})

			It("when client passthrough is specified", func() {
				vmi := libvmi.New()
				vmi.Spec.Domain.Devices.ClientPassthrough = &v1.ClientPassthroughDevices{}
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
			})
		})
	})
})
