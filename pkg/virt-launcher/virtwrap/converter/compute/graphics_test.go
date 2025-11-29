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
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
)

var _ = Describe("Graphics Domain Configurator", func() {
	Context("AutoattachGraphicsDevice", func() {

		DescribeTable("should not configure video and VNC when AutoattachGraphicsDevice is false", func(arch string) {
			vmi := libvmi.New(libvmi.WithAutoattachGraphicsDevice(false))

			domain := api.Domain{}
			configurator := compute.NewGraphicsDomainConfigurator(arch, false)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			Expect(domain).To(Equal(api.Domain{}))
		},
			Entry("on amd64", "amd64"),
			Entry("on arm64", "arm64"),
			Entry("on s390x", "s390x"),
		)

		DescribeTable("should configure video and VNC", func(arch string, autoAttach *bool, expectedVideo api.Video) {
			vmi := libvmi.New(libvmi.WithUID("test-uid"))
			vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = autoAttach

			domain := api.Domain{}
			configurator := compute.NewGraphicsDomainConfigurator(arch, false)
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
			Entry("amd64 when AutoattachGraphicsDevice is true", "amd64", pointer.P(true), newExpectedAMD64VideoDevice()),
			Entry("arm64 when AutoattachGraphicsDevice is true", "arm64", pointer.P(true), newExpectedARM64VideoDevice()),
			Entry("s390x when AutoattachGraphicsDevice is true", "s390x", pointer.P(true), newExpectedS390XVideoDevice()),
			Entry("amd64 when AutoattachGraphicsDevice is nil", "amd64", nil, newExpectedAMD64VideoDevice()),
			Entry("arm64 when AutoattachGraphicsDevice is nil", "arm64", nil, newExpectedARM64VideoDevice()),
			Entry("s390x when AutoattachGraphicsDevice is nil", "s390x", nil, newExpectedS390XVideoDevice()),
		)
	})

	Context("Video device configuration", func() {
		DescribeTable("Should use user-specified video type when provided", func(arch string, bochsForEFI bool) {
			vmi := libvmi.New(libvmi.WithVideo("virtio"))
			var domain api.Domain

			configurator := compute.NewGraphicsDomainConfigurator(arch, bochsForEFI)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Video: []api.Video{
							{
								Model: api.VideoModel{
									Type:  "virtio",
									Heads: pointer.P(uint(1)),
									VRam:  pointer.P(uint(16384)),
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
			Entry("on amd64 with bochsForEFI", "amd64", true),
			Entry("on amd64 without bochsForEFI", "amd64", false),
			Entry("on arm64 with bochsForEFI", "arm64", true),
			Entry("on arm64 without bochsForEFI", "arm64", false),
			Entry("on s390x with bochsForEFI", "s390x", true),
			Entry("on s390x without bochsForEFI", "s390x", false),
		)

		DescribeTable("amd64 defaults to VGA with VRAM", func(vmi *v1.VirtualMachineInstance, bochsForEFI bool) {
			var domain api.Domain

			configurator := compute.NewGraphicsDomainConfigurator("amd64", bochsForEFI)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Video: []api.Video{
							{
								Model: api.VideoModel{
									Type:  "vga",
									Heads: pointer.P(uint(1)),
									VRam:  pointer.P(uint(16384)),
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

			configurator := compute.NewGraphicsDomainConfigurator("amd64", true)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Video: []api.Video{
							{
								Model: api.VideoModel{
									Type:  "bochs",
									Heads: pointer.P(uint(1)),
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
			Heads: pointer.P(uint(1)),
			VRam:  pointer.P(uint(16384)),
		},
	}
}

func newExpectedARM64VideoDevice() api.Video {
	return api.Video{
		Model: api.VideoModel{
			Type:  v1.VirtIO,
			Heads: pointer.P(uint(1)),
		},
	}
}

func newExpectedS390XVideoDevice() api.Video {
	return api.Video{
		Model: api.VideoModel{
			Type:  v1.VirtIO,
			Heads: pointer.P(uint(1)),
		},
	}
}
