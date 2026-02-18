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

var _ = Describe("Input Device Configurator", func() {
	Context("User-specified input devices", func() {
		DescribeTable("should configure user-specified input devices", func(arch string, bus v1.InputBus, expectedModel string) {
			vmi := libvmi.New(libvmi.WithTablet("my-tablet", bus), libvmi.WithAutoattachGraphicsDevice(false))
			var domain api.Domain

			configurator := compute.NewInputDeviceDomainConfigurator(arch)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Inputs: []api.Input{
							{
								Type:  "tablet",
								Bus:   bus,
								Alias: api.NewUserDefinedAlias("my-tablet"),
								Model: expectedModel,
							},
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		},
			Entry("with USB bus on amd64", "amd64", v1.InputBusUSB, ""),
			Entry("with USB bus on arm64", "arm64", v1.InputBusUSB, ""),
			Entry("with USB bus on s390x", "s390x", v1.InputBusUSB, ""),
			Entry("with virtio bus on amd64", "amd64", v1.InputBusVirtio, v1.VirtIO),
			Entry("with virtio bus on arm64", "arm64", v1.InputBusVirtio, v1.VirtIO),
			Entry("with virtio bus on s390x", "s390x", v1.InputBusVirtio, v1.VirtIO),
		)

		DescribeTable("should return error for invalid input devices", func(bus v1.InputBus, deviceType v1.InputType, expectedError string) {
			vmi := libvmi.New()
			vmi.Spec.Domain.Devices.Inputs = []v1.Input{
				{
					Name: "invalid",
					Type: deviceType,
					Bus:  bus,
				},
			}
			var domain api.Domain

			configurator := compute.NewInputDeviceDomainConfigurator("amd64")
			err := configurator.Configure(vmi, &domain)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(expectedError))
		},
			Entry("unsupported bus", v1.InputBus("ps2"), v1.InputTypeTablet, "unsupported bus"),
			Entry("unsupported type", v1.InputBusUSB, v1.InputType("keyboard"), "unsupported type"),
		)
	})

	Context("Architecture-specific input devices", func() {
		DescribeTable("should add architecture-specific input devices when AutoattachGraphicsDevice is nil or true",
			func(arch string, autoattachGraphicsDevice *bool, expectedInputDevices []api.Input) {
				vmi := libvmi.New()
				vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = autoattachGraphicsDevice
				var domain api.Domain

				configurator := compute.NewInputDeviceDomainConfigurator(arch)
				Expect(configurator.Configure(vmi, &domain)).To(Succeed())

				expectedDomain := api.Domain{Spec: api.DomainSpec{Devices: api.Devices{Inputs: expectedInputDevices}}}
				Expect(domain).To(Equal(expectedDomain))
			},
			Entry("amd64 adds no input devices when nil", "amd64", nil, nil),
			Entry("amd64 adds no input devices when true", "amd64", pointer.P(true), nil),
			Entry("arm64 adds tablet and keyboard when nil", "arm64", nil, []api.Input{
				{Type: "tablet", Bus: "usb"},
				{Type: "keyboard", Bus: "usb"},
			}),
			Entry("arm64 adds tablet and keyboard when true", "arm64", pointer.P(true), []api.Input{
				{Type: "tablet", Bus: "usb"},
				{Type: "keyboard", Bus: "usb"},
			}),
			Entry("s390x adds virtio keyboard when nil", "s390x", nil, []api.Input{
				{Type: "keyboard", Bus: "virtio"},
			}),
			Entry("s390x adds virtio keyboard when true", "s390x", pointer.P(true), []api.Input{
				{Type: "keyboard", Bus: "virtio"},
			}),
		)

		DescribeTable("should not add architecture-specific input devices when AutoattachGraphicsDevice is false",
			func(arch string) {
				vmi := libvmi.New()
				vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = pointer.P(false)
				var domain api.Domain

				configurator := compute.NewInputDeviceDomainConfigurator(arch)
				Expect(configurator.Configure(vmi, &domain)).To(Succeed())

				Expect(domain).To(Equal(api.Domain{}))
			},
			Entry("amd64", "amd64"),
			Entry("arm64", "arm64"),
			Entry("s390x", "s390x"),
		)
	})
})
