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
				compute.BalloonWithArchitecture("amd64"),
				compute.BalloonWithUseVirtioTransitional(false),
				compute.BalloonWithUseLaunchSecuritySEV(true),
				compute.BalloonWithUseLaunchSecurityPV(false),
				compute.BalloonWithFreePageReporting(false),
				compute.BalloonWithMemBalloonStatsPeriod(0),
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
				compute.BalloonWithArchitecture("amd64"),
				compute.BalloonWithUseVirtioTransitional(false),
				compute.BalloonWithUseLaunchSecuritySEV(false),
				compute.BalloonWithUseLaunchSecurityPV(true),
				compute.BalloonWithFreePageReporting(false),
				compute.BalloonWithMemBalloonStatsPeriod(0),
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
})
