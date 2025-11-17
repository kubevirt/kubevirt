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

var _ = Describe("Graphics Domain Configurator", func() {

	Context("AutoattachGraphicsDevice", func() {

		DescribeTable("should configure both video and VNC socket based on AutoattachGraphicsDevice", func(autoAttach *bool) {
			vmi := libvmi.New(libvmi.WithUID("test-uid"))
			vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = autoAttach

			domain := api.Domain{}
			configurator := compute.NewGraphicsDomainConfigurator("amd64", false)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			if autoAttach == nil || *autoAttach {
				Expect(domain.Spec.Devices.Video).ToNot(BeEmpty())
				Expect(domain.Spec.Devices.Graphics).To(HaveLen(1))
				Expect(domain.Spec.Devices.Graphics[0].Type).To(Equal("vnc"))
				Expect(domain.Spec.Devices.Graphics[0].Listen).ToNot(BeNil())
				Expect(domain.Spec.Devices.Graphics[0].Listen.Type).To(Equal("socket"))
				Expect(domain.Spec.Devices.Graphics[0].Listen.Socket).To(Equal("/var/run/kubevirt-private/test-uid/virt-vnc"))
			} else {
				Expect(domain.Spec.Devices.Video).To(BeEmpty())
				Expect(domain.Spec.Devices.Graphics).To(BeEmpty())
			}
		},
			Entry("when false, should not configure Video and VNC", pointer.P(false)),
			Entry("when true, should configure Video and VNC", pointer.P(true)),
			Entry("when nil, should configure Video and VNC", nil),
		)
	})

	Context("Video device configuration", func() {
		It("Should use user-specified video type when provided", func() {
			vmi := libvmi.New(libvmi.WithVideo("virtio"))
			var domain api.Domain

			configurator := compute.NewGraphicsDomainConfigurator("amd64", false)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			Expect(domain.Spec.Devices.Video).To(HaveLen(1))
			Expect(domain.Spec.Devices.Video[0].Model.Type).To(Equal("virtio"))
			Expect(domain.Spec.Devices.Video[0].Model.Heads).To(Equal(pointer.P(uint(1))))
			Expect(domain.Spec.Devices.Video[0].Model.VRam).To(Equal(pointer.P(uint(16384))))
		})

		Context("AMD64 architecture", func() {
			It("Should use VGA for non-EFI systems", func() {
				vmi := libvmi.New()
				var domain api.Domain

				configurator := compute.NewGraphicsDomainConfigurator("amd64", false)
				Expect(configurator.Configure(vmi, &domain)).To(Succeed())

				Expect(domain.Spec.Devices.Video).To(HaveLen(1))
				Expect(domain.Spec.Devices.Video[0].Model.Type).To(Equal("vga"))
				Expect(domain.Spec.Devices.Video[0].Model.Heads).To(Equal(pointer.P(uint(1))))
				Expect(domain.Spec.Devices.Video[0].Model.VRam).To(Equal(pointer.P(uint(16384))))
			})

			DescribeTable("should configure video based on EFI and BochsForEFIGuests", func(isEFI, bochsForEFI bool, expectedType string, expectVRam bool) {
				vmi := libvmi.New(libvmi.WithUefi(isEFI))
				var domain api.Domain

				configurator := compute.NewGraphicsDomainConfigurator("amd64", bochsForEFI)
				Expect(configurator.Configure(vmi, &domain)).To(Succeed())

				Expect(domain.Spec.Devices.Video).To(HaveLen(1))
				Expect(domain.Spec.Devices.Video[0].Model.Type).To(Equal(expectedType))
				Expect(domain.Spec.Devices.Video[0].Model.Heads).To(Equal(pointer.P(uint(1))))
				if expectVRam {
					Expect(domain.Spec.Devices.Video[0].Model.VRam).To(Equal(pointer.P(uint(16384))))
				} else {
					Expect(domain.Spec.Devices.Video[0].Model.VRam).To(BeNil())
				}
			},
				Entry("EFI with BochsForEFIGuests=false uses VGA", true, false, "vga", true),
				Entry("EFI with BochsForEFIGuests=true uses Bochs", true, true, "bochs", false),
			)
		})

		DescribeTable("should configure architecture-specific video devices", func(arch, expectedType string) {
			vmi := libvmi.New()
			var domain api.Domain

			configurator := compute.NewGraphicsDomainConfigurator(arch, false)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			Expect(domain.Spec.Devices.Video).To(HaveLen(1))
			Expect(domain.Spec.Devices.Video[0].Model.Type).To(Equal(expectedType))
			Expect(domain.Spec.Devices.Video[0].Model.Heads).To(Equal(pointer.P(uint(1))))
		},
			Entry("arm64 uses virtio", "arm64", v1.VirtIO),
			Entry("s390x uses virtio", "s390x", v1.VirtIO),
		)

		It("Should return error for unsupported architecture", func() {
			vmi := libvmi.New()
			var domain api.Domain

			configurator := compute.NewGraphicsDomainConfigurator("test-arch", false)

			err := configurator.Configure(vmi, &domain)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported architecture: test-arch"))
		})
	})

	Context("VNC socket connection", func() {
		It("Should configure VNC socket with correct path format", func() {
			vmi := libvmi.New()
			vmi.ObjectMeta.UID = "test-uid-123"
			var domain api.Domain

			configurator := compute.NewGraphicsDomainConfigurator("amd64", false)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			Expect(domain.Spec.Devices.Graphics).To(HaveLen(1))
			Expect(domain.Spec.Devices.Graphics[0].Type).To(Equal("vnc"))
			Expect(domain.Spec.Devices.Graphics[0].Listen).ToNot(BeNil())
			Expect(domain.Spec.Devices.Graphics[0].Listen.Type).To(Equal("socket"))
			Expect(domain.Spec.Devices.Graphics[0].Listen.Socket).To(Equal("/var/run/kubevirt-private/test-uid-123/virt-vnc"))
		})

		DescribeTable("should configure VNC socket for all architectures", func(arch string) {
			vmi := libvmi.New()
			vmi.ObjectMeta.UID = "test-uid-789"
			var domain api.Domain

			configurator := compute.NewGraphicsDomainConfigurator(arch, false)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			Expect(domain.Spec.Devices.Graphics).To(HaveLen(1))
			Expect(domain.Spec.Devices.Graphics[0].Type).To(Equal("vnc"))
			Expect(domain.Spec.Devices.Graphics[0].Listen).ToNot(BeNil())
			Expect(domain.Spec.Devices.Graphics[0].Listen.Type).To(Equal("socket"))
			Expect(domain.Spec.Devices.Graphics[0].Listen.Socket).To(Equal("/var/run/kubevirt-private/test-uid-789/virt-vnc"))
		},
			Entry("amd64", "amd64"),
			Entry("arm64", "arm64"),
			Entry("s390x", "s390x"),
		)
	})
})
