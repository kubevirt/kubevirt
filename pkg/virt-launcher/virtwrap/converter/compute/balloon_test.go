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

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
)

var _ = Describe("Balloon Domain Configurator", func() {
	const (
		useSEV = true
		usePV  = true
	)

	DescribeTable("IOMMU driver configuration", func(useSEV, usePV bool, expectedDriver *api.MemBalloonDriver) {
		vmi := libvmi.New()
		var domain api.Domain

		configurator := compute.NewBalloonDomainConfigurator(
			compute.BalloonWithUseLaunchSecuritySEV(useSEV),
			compute.BalloonWithUseLaunchSecurityPV(usePV),
			compute.BalloonWithFreePageReporting(false),
			compute.BalloonWithMemBalloonStatsPeriod(0),
			compute.BalloonWithVirtioModel("virtio-non-transitional"),
		)

		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := newDomainWithBallooning(
			api.MemBalloon{
				Model:             "virtio-non-transitional",
				Driver:            expectedDriver,
				FreePageReporting: "off",
			},
		)
		Expect(domain).To(Equal(expectedDomain))
	},
		Entry("neither SEV nor PV", !useSEV, !usePV, nil),
		Entry("SEV enabled", useSEV, !usePV, &api.MemBalloonDriver{IOMMU: "on"}),
		Entry("PV enabled", !useSEV, usePV, &api.MemBalloonDriver{IOMMU: "on"}),
		Entry("Both SEV and PV enabled", useSEV, usePV, &api.MemBalloonDriver{IOMMU: "on"}),
	)

	DescribeTable("free page reporting", func(enabled bool, expected string) {
		vmi := libvmi.New()
		var domain api.Domain

		configurator := compute.NewBalloonDomainConfigurator(
			compute.BalloonWithUseLaunchSecuritySEV(false),
			compute.BalloonWithUseLaunchSecurityPV(false),
			compute.BalloonWithFreePageReporting(enabled),
			compute.BalloonWithMemBalloonStatsPeriod(0),
			compute.BalloonWithVirtioModel("virtio-non-transitional"),
		)

		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := newDomainWithBallooning(api.MemBalloon{
			Model:             "virtio-non-transitional",
			FreePageReporting: expected,
		})
		Expect(domain).To(Equal(expectedDomain))
	},
		Entry("enabled", true, "on"),
		Entry("disabled", false, "off"),
	)

	DescribeTable("memballoon stats period", func(period uint, expectedStats *api.Stats) {
		vmi := libvmi.New()
		var domain api.Domain

		configurator := compute.NewBalloonDomainConfigurator(
			compute.BalloonWithUseLaunchSecuritySEV(false),
			compute.BalloonWithUseLaunchSecurityPV(false),
			compute.BalloonWithFreePageReporting(false),
			compute.BalloonWithMemBalloonStatsPeriod(period),
			compute.BalloonWithVirtioModel("virtio-non-transitional"),
		)

		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := newDomainWithBallooning(api.MemBalloon{
			Model:             "virtio-non-transitional",
			FreePageReporting: "off",
			Stats:             expectedStats,
		})
		Expect(domain).To(Equal(expectedDomain))
	},
		Entry("zero period", uint(0), nil),
		Entry("non-zero period", uint(5), &api.Stats{Period: 5}),
	)

	It("should configure memballoon with model none when AutoattachMemBalloon is false", func() {
		vmi := libvmi.New(libvmi.WithAutoattachMemBalloon(false))
		var domain api.Domain

		configurator := compute.NewBalloonDomainConfigurator(
			compute.BalloonWithUseLaunchSecuritySEV(false),
			compute.BalloonWithUseLaunchSecurityPV(false),
			compute.BalloonWithFreePageReporting(true),
			compute.BalloonWithMemBalloonStatsPeriod(10),
			compute.BalloonWithVirtioModel("virtio-non-transitional"),
		)

		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := newDomainWithBallooning(
			api.MemBalloon{
				Model: "none",
			},
		)
		Expect(domain).To(Equal(expectedDomain))
	})
})

func newDomainWithBallooning(ballooning api.MemBalloon) api.Domain {
	return api.Domain{
		Spec: api.DomainSpec{
			Devices: api.Devices{
				Ballooning: &ballooning,
			},
		},
	}
}
