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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute/video"
)

var _ = Describe("Graphics Domain Configurator", func() {
	Context("AutoattachGraphicsDevice", func() {
		DescribeTable("should not configure video and VNC when AutoattachGraphicsDevice is false", func(
			configureVideoDevice compute.ConfigureVideoDeviceFunc,
		) {
			vmi := libvmi.New(libvmi.WithAutoattachGraphicsDevice(false))

			domain := api.Domain{}
			configurator := compute.NewGraphicsDomainConfigurator(configureVideoDevice, false, false)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			Expect(domain).To(Equal(api.Domain{}))
		},
			Entry("on amd64", video.ConfigureAMD64Device),
			Entry("on arm64", video.ConfigureVirtioDevice),
			Entry("on s390x", video.ConfigureVirtioDevice),
		)

		DescribeTable("should configure video and VNC", func(
			configureVideoDevice compute.ConfigureVideoDeviceFunc,
			autoAttach *bool,
			expectedVideo api.Video,
		) {
			vmi := libvmi.New(libvmi.WithUID("test-uid"))
			vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = autoAttach

			domain := api.Domain{}
			configurator := compute.NewGraphicsDomainConfigurator(configureVideoDevice, false, false)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Video: []api.Video{expectedVideo},
						Graphics: []api.Graphics{
							{
								Type: "vnc",
								Listen: &api.GraphicsListen{
									Type:   "socket",
									Socket: "/var/run/kubevirt-private/test-uid/virt-vnc",
								},
							},
						},
					},
				},
			}

			Expect(domain).To(Equal(expectedDomain))
		},
			Entry("amd64 when AutoattachGraphicsDevice is true", video.ConfigureAMD64Device, new(true), newExpectedAMD64VideoDevice()),
			Entry("arm64 when AutoattachGraphicsDevice is true", video.ConfigureVirtioDevice, new(true), newExpectedARM64VideoDevice()),
			Entry("s390x when AutoattachGraphicsDevice is true", video.ConfigureVirtioDevice, new(true), newExpectedS390XVideoDevice()),
			Entry("amd64 when AutoattachGraphicsDevice is nil", video.ConfigureAMD64Device, nil, newExpectedAMD64VideoDevice()),
			Entry("arm64 when AutoattachGraphicsDevice is nil", video.ConfigureVirtioDevice, nil, newExpectedARM64VideoDevice()),
			Entry("s390x when AutoattachGraphicsDevice is nil", video.ConfigureVirtioDevice, nil, newExpectedS390XVideoDevice()),
		)
	})

	Context("Cross-architecture emulation", func() {
		It("should disable graphics when cross-arch emulation is enabled", func() {
			vmi := libvmi.New()
			domain := api.Domain{}
			configurator := compute.NewGraphicsDomainConfigurator(video.ConfigureAMD64Device, false, true)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			Expect(domain.Spec.Devices.Graphics).To(BeNil())
			Expect(domain.Spec.Devices.Video).To(BeNil())
		})
	})

	Context("Video device configuration", func() {
		DescribeTable("Should use user-specified video type when provided", func(
			configureVideoDevice compute.ConfigureVideoDeviceFunc,
			bochsForEFI bool,
		) {
			vmi := libvmi.New(libvmi.WithVideo("virtio"))
			var domain api.Domain

			configurator := compute.NewGraphicsDomainConfigurator(configureVideoDevice, bochsForEFI, false)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Video: []api.Video{
							{
								Model: api.VideoModel{
									Type:  "virtio",
									Heads: new(uint(1)),
									VRam:  new(uint(16384)),
								},
							},
						},
						Graphics: []api.Graphics{
							{
								Type: "vnc",
								Listen: &api.GraphicsListen{
									Type:   "socket",
									Socket: fmt.Sprintf("/var/run/kubevirt-private/%s/virt-vnc", vmi.ObjectMeta.UID),
								},
							},
						},
					},
				},
			}

			Expect(domain).To(Equal(expectedDomain))
		},
			Entry("on amd64 with bochsForEFI", video.ConfigureAMD64Device, true),
			Entry("on amd64 without bochsForEFI", video.ConfigureAMD64Device, false),
			Entry("on arm64 with bochsForEFI", video.ConfigureVirtioDevice, true),
			Entry("on arm64 without bochsForEFI", video.ConfigureVirtioDevice, false),
			Entry("on s390x with bochsForEFI", video.ConfigureVirtioDevice, true),
			Entry("on s390x without bochsForEFI", video.ConfigureVirtioDevice, false),
		)

		DescribeTable("amd64 defaults to VGA with VRAM", func(vmi *v1.VirtualMachineInstance, bochsForEFI bool) {
			var domain api.Domain

			configurator := compute.NewGraphicsDomainConfigurator(video.ConfigureAMD64Device, bochsForEFI, false)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Video: []api.Video{
							{
								Model: api.VideoModel{
									Type:  "vga",
									Heads: new(uint(1)),
									VRam:  new(uint(16384)),
								},
							},
						},
						Graphics: []api.Graphics{
							{
								Type: "vnc",
								Listen: &api.GraphicsListen{
									Type:   "socket",
									Socket: fmt.Sprintf("/var/run/kubevirt-private/%s/virt-vnc", vmi.ObjectMeta.UID),
								},
							},
						},
					},
				},
			}

			Expect(domain).To(Equal(expectedDomain))
		},
			Entry("non-EFI with bochsForEFI disabled", libvmi.New(), false),
			Entry("non-EFI with bochsForEFI enabled", libvmi.New(), true),
			Entry("EFI with bochsForEFI disabled", libvmi.New(libvmi.WithUefi(true)), false),
		)

		It("amd64 EFI with Bochs enabled uses Bochs", func() {
			vmi := libvmi.New(libvmi.WithUefi(true))
			var domain api.Domain

			configurator := compute.NewGraphicsDomainConfigurator(video.ConfigureAMD64Device, true, false)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Video: []api.Video{
							{
								Model: api.VideoModel{
									Type:  "bochs",
									Heads: new(uint(1)),
								},
							},
						},
						Graphics: []api.Graphics{
							{
								Type: "vnc",
								Listen: &api.GraphicsListen{
									Type:   "socket",
									Socket: fmt.Sprintf("/var/run/kubevirt-private/%s/virt-vnc", vmi.ObjectMeta.UID),
								},
							},
						},
					},
				},
			}

			Expect(domain).To(Equal(expectedDomain))
		})
	})
})

func newExpectedAMD64VideoDevice() api.Video {
	return api.Video{
		Model: api.VideoModel{
			Type:  "vga",
			Heads: new(uint(1)),
			VRam:  new(uint(16384)),
		},
	}
}

func newExpectedARM64VideoDevice() api.Video {
	return api.Video{
		Model: api.VideoModel{
			Type:  v1.VirtIO,
			Heads: new(uint(1)),
		},
	}
}

func newExpectedS390XVideoDevice() api.Video {
	return api.Video{
		Model: api.VideoModel{
			Type:  v1.VirtIO,
			Heads: new(uint(1)),
		},
	}
}
