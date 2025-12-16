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

var _ = Describe("Balloon Domain Configurator", func() {
	Context("amd64 with SEV", func() {
		It("Should set IOMMU attribute of the MemBalloonDriver", func() {
			vmi := libvmi.New(withLaunchSecurity(v1.LaunchSecurity{SEV: &v1.SEV{}}))
			var domain api.Domain

			configurator := compute.NewBalloonDomainConfigurator(
				compute.BalloonWithUseLaunchSecuritySEV(true),
				compute.BalloonWithUseLaunchSecurityPV(false),
				compute.BalloonWithFreePageReporting(false),
				compute.BalloonWithMemBalloonStatsPeriod(0),
				compute.BalloonWithVirtioModel("virtio-non-transitional"),
			)

			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Ballooning: &api.MemBalloon{
							Model:             "virtio-non-transitional",
							Stats:             nil,
							Address:           nil,
							Driver:            &api.MemBalloonDriver{IOMMU: "on"},
							FreePageReporting: "off",
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		})
	})

	Context("s390x with LaunchSecurity", func() {
		It("Should set IOMMU attribute of the MemBalloonDriver", func() {
			vmi := libvmi.New(withLaunchSecurity(v1.LaunchSecurity{SEV: &v1.SEV{}}))
			var domain api.Domain

			configurator := compute.NewBalloonDomainConfigurator(
				compute.BalloonWithUseLaunchSecuritySEV(false),
				compute.BalloonWithUseLaunchSecurityPV(true),
				compute.BalloonWithFreePageReporting(false),
				compute.BalloonWithMemBalloonStatsPeriod(0),
				compute.BalloonWithVirtioModel("virtio"),
			)

			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Ballooning: &api.MemBalloon{
							Model:             "virtio",
							Stats:             nil,
							Address:           nil,
							Driver:            &api.MemBalloonDriver{IOMMU: "on"},
							FreePageReporting: "off",
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		})
	})

	Context("with memballoon stats period", func() {
		It("should configure memballoon with stats period 5 for amd64", func() {
			vmi := libvmi.New()
			var domain api.Domain

			configurator := compute.NewBalloonDomainConfigurator(
				compute.BalloonWithVirtioModel("virtio-non-transitional"),
				compute.BalloonWithFreePageReporting(true),
				compute.BalloonWithMemBalloonStatsPeriod(5),
			)

			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Ballooning: &api.MemBalloon{
							Model:             "virtio-non-transitional",
							FreePageReporting: "on",
							Stats:             &api.Stats{Period: 5},
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should configure memballoon with stats period 5 for arm64", func() {
			vmi := libvmi.New()
			var domain api.Domain

			configurator := compute.NewBalloonDomainConfigurator(
				compute.BalloonWithVirtioModel("virtio-non-transitional"),
				compute.BalloonWithFreePageReporting(true),
				compute.BalloonWithMemBalloonStatsPeriod(5),
			)

			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Ballooning: &api.MemBalloon{
							Model:             "virtio-non-transitional",
							FreePageReporting: "on",
							Stats:             &api.Stats{Period: 5},
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should configure memballoon with stats period 5 for s390x", func() {
			vmi := libvmi.New()
			var domain api.Domain

			configurator := compute.NewBalloonDomainConfigurator(
				compute.BalloonWithVirtioModel("virtio"),
				compute.BalloonWithFreePageReporting(true),
				compute.BalloonWithMemBalloonStatsPeriod(5),
			)

			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Ballooning: &api.MemBalloon{
							Model:             "virtio",
							FreePageReporting: "on",
							Stats:             &api.Stats{Period: 5},
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should configure memballoon with stats period 0 for amd64", func() {
			vmi := libvmi.New()
			var domain api.Domain

			configurator := compute.NewBalloonDomainConfigurator(
				compute.BalloonWithVirtioModel("virtio-non-transitional"),
				compute.BalloonWithFreePageReporting(true),
				compute.BalloonWithMemBalloonStatsPeriod(0),
			)

			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Ballooning: &api.MemBalloon{
							Model:             "virtio-non-transitional",
							FreePageReporting: "on",
							Stats:             nil,
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should configure memballoon with stats period 0 for arm64", func() {
			vmi := libvmi.New()
			var domain api.Domain

			configurator := compute.NewBalloonDomainConfigurator(
				compute.BalloonWithVirtioModel("virtio-non-transitional"),
				compute.BalloonWithFreePageReporting(true),
				compute.BalloonWithMemBalloonStatsPeriod(0),
			)

			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Ballooning: &api.MemBalloon{
							Model:             "virtio-non-transitional",
							FreePageReporting: "on",
							Stats:             nil,
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should configure memballoon with stats period 0 for s390x", func() {
			vmi := libvmi.New()
			var domain api.Domain

			configurator := compute.NewBalloonDomainConfigurator(
				compute.BalloonWithVirtioModel("virtio"),
				compute.BalloonWithFreePageReporting(true),
				compute.BalloonWithMemBalloonStatsPeriod(0),
			)

			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Ballooning: &api.MemBalloon{
							Model:             "virtio",
							FreePageReporting: "on",
							Stats:             nil,
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		})
	})
})
