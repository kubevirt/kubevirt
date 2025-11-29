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

var _ = Describe("RNG Domain Configurator", func() {
	DescribeTable("Should not configure RNG device when RNG is unspecified in VMI", func(architecture string) {
		vmi := libvmi.New()
		var domain api.Domain

		configurator := compute.NewRNGDomainConfigurator(
			compute.RNGWithArchitecture(architecture),
			compute.RNGWithUseVirtioTransitional(false),
			compute.RNGWithUseLaunchSecuritySEV(false),
			compute.RNGWithUseLaunchSecurityPV(false),
		)

		Expect(configurator.Configure(vmi, &domain)).To(Succeed())
		Expect(domain).To(Equal(api.Domain{}))
	},
		Entry("On amd64", "amd64"),
		Entry("On arm64", "arm64"),
		Entry("On s390x", "s390x"),
	)

	Context("amd46 with SEV", func() {
		It("Should set IOMMU attribute of the RngDriver", func() {
			vmi := libvmi.New(
				libvmi.WithRng(),
				withLaunchSecurity(v1.LaunchSecurity{SEV: &v1.SEV{}}),
			)
			var domain api.Domain

			configurator := compute.NewRNGDomainConfigurator(
				compute.RNGWithArchitecture("amd64"),
				compute.RNGWithUseVirtioTransitional(false),
				compute.RNGWithUseLaunchSecuritySEV(true),
				compute.RNGWithUseLaunchSecurityPV(false),
			)

			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Rng: &api.Rng{
							Model: "virtio-non-transitional",
							Backend: &api.RngBackend{
								Model:  "random",
								Source: "/dev/urandom",
							},
							Address: nil,
							Driver: &api.RngDriver{
								IOMMU: "on",
							},
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		})
	})

	Context("s390x with LaunchSecurity", func() {
		It("Should set IOMMU attribute of the RngDriver", func() {
			vmi := libvmi.New(
				libvmi.WithRng(),
				withLaunchSecurity(v1.LaunchSecurity{}),
			)
			var domain api.Domain

			configurator := compute.NewRNGDomainConfigurator(
				compute.RNGWithArchitecture("s390x"),
				compute.RNGWithUseVirtioTransitional(false),
				compute.RNGWithUseLaunchSecuritySEV(false),
				compute.RNGWithUseLaunchSecurityPV(true),
			)

			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Rng: &api.Rng{
							Model: "virtio",
							Backend: &api.RngBackend{
								Model:  "random",
								Source: "/dev/urandom",
							},
							Address: nil,
							Driver: &api.RngDriver{
								IOMMU: "on",
							},
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		})
	})
})

func withLaunchSecurity(launchSecurity v1.LaunchSecurity) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.LaunchSecurity = &launchSecurity
	}
}
