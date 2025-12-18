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
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	kvapi "kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	archconverter "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/arch"

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

	Context("with memballoon device", func() {
		var (
			vmi *v1.VirtualMachineInstance
			c   *converter.ConverterContext
		)

		BeforeEach(func() {
			vmi = kvapi.NewMinimalVMI("testvmi")
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		})

		DescribeTable("should set freePageReporting attribute of memballooning device, accordingly to the context value", func(freePageReporting bool, expectedValue string) {
			c = &converter.ConverterContext{
				Architecture:      archconverter.NewConverter(runtime.GOARCH),
				FreePageReporting: freePageReporting,
				AllowEmulation:    true,
			}
			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(BeNil())

			Expect(domain.Spec.Devices).ToNot(BeNil())
			Expect(domain.Spec.Devices.Ballooning).ToNot(BeNil())
			Expect(domain.Spec.Devices.Ballooning.FreePageReporting).To(BeEquivalentTo(expectedValue))
		},
			Entry("when true", true, "on"),
			Entry("when false", false, "off"),
		)

		DescribeTable("should unconditionally set Ballooning device", func(isAutoattachMemballoon *bool) {
			vmi.Spec.Domain.Devices.AutoattachMemBalloon = isAutoattachMemballoon
			c = &converter.ConverterContext{
				Architecture:   archconverter.NewConverter(runtime.GOARCH),
				AllowEmulation: true,
			}

			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(BeNil())
			Expect(domain.Spec.Devices).ToNot(BeNil())
			Expect(domain.Spec.Devices.Ballooning).ToNot(BeNil(), "Ballooning device should be unconditionally present in domain")
		},
			Entry("when AutoattachMemballoon is false", pointer.P(false)),
			Entry("when AutoattachMemballoon is not defined", nil),
		)
	})
})

func vmiToDomain(vmi *v1.VirtualMachineInstance, c *converter.ConverterContext) *api.Domain {
	domain := &api.Domain{}
	ExpectWithOffset(1, converter.Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, domain, c)).To(Succeed())
	api.NewDefaulter(c.Architecture.GetArchitecture()).SetObjectDefaults_Domain(domain)
	return domain
}
