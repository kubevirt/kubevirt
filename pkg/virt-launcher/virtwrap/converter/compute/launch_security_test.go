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

const QGSTestSocketPath = "/test/qgs.socket"

var _ = Describe("LaunchSecurity Domain Configurator", func() {
	DescribeTable("Should not configure LaunchSecurity when unspecified in the VMI", func(architecture string) {
		vmi := libvmi.New()
		var domain api.Domain

		configurator := compute.NewLaunchSecurityDomainConfigurator(architecture)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())
		Expect(domain).To(Equal(api.Domain{}))
	},
		Entry("on amd64", "amd64"),
		Entry("on arm64", "arm64"),
		Entry("on s390x", "s390x"),
	)

	It("should configure LaunchSecurity when specified in the VMI on amd64", func() {
		vmi := libvmi.New(libvmi.WithSEV(true, true))
		var domain api.Domain

		configurator := compute.NewLaunchSecurityDomainConfigurator("amd64")
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				LaunchSecurity: &api.LaunchSecurity{
					Type:            "sev",
					DHCert:          "",
					Session:         "",
					Cbitpos:         "",
					ReducedPhysBits: "",
					Policy:          "0x5",
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	})

	Context("kernel hashes", func() {
		directKernelBoot := []libvmi.Option{
			libvmi.WithKernelBoot("kernel-image", "/boot/vmlinuz", "/boot/initrd", "console=ttyS0"),
		}
		initrdOnlyKernelBoot := []libvmi.Option{libvmi.WithKernelBootContainer("kernel-image")}

		It("should be enabled for SEV-SNP with a direct kernel boot", func() {
			vmi := libvmi.New(append([]libvmi.Option{libvmi.WithSEV(false, true)}, directKernelBoot...)...)
			var domain api.Domain

			configurator := compute.NewLaunchSecurityDomainConfigurator("amd64")
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			Expect(domain.Spec.LaunchSecurity).NotTo(BeNil())
			Expect(domain.Spec.LaunchSecurity.Type).To(Equal("sev-snp"))
			Expect(domain.Spec.LaunchSecurity.KernelHashes).To(Equal("yes"))
		})

		DescribeTable("should be disabled", func(launchSecurity libvmi.Option, kernelBoot []libvmi.Option, expectedType string) {
			vmi := libvmi.New(append([]libvmi.Option{launchSecurity}, kernelBoot...)...)
			var domain api.Domain

			configurator := compute.NewLaunchSecurityDomainConfigurator("amd64")
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			Expect(domain.Spec.LaunchSecurity).NotTo(BeNil())
			Expect(domain.Spec.LaunchSecurity.Type).To(Equal(expectedType))
			Expect(domain.Spec.LaunchSecurity.KernelHashes).To(BeEmpty())
		},
			// Kernel hashes are a SEV-SNP-only feature here: SEV and SEV-ES keep
			// booting through the stateful firmware and are left untouched.
			Entry("for SEV even with a direct kernel boot",
				libvmi.WithSEV(false, false), directKernelBoot, "sev"),
			Entry("for SEV-ES even with a direct kernel boot",
				libvmi.WithSEV(true, false), directKernelBoot, "sev"),
			Entry("for TDX even with a direct kernel boot",
				withTDX(), directKernelBoot, "tdx"),
			Entry("for SEV-SNP when the kernel boot container has no kernel path",
				libvmi.WithSEV(false, true), initrdOnlyKernelBoot, "sev-snp"),
			Entry("for SEV-SNP when the VMI has no kernel boot",
				libvmi.WithSEV(false, true), nil, "sev-snp"),
		)
	})

	It("should configure LaunchSecurity when specified in the VMI on intel", func() {
		vmi := libvmi.New(withTDX())
		var domain api.Domain

		configurator := compute.NewLaunchSecurityDomainConfigurator("amd64")
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				LaunchSecurity: &api.LaunchSecurity{
					Type: "tdx",
					QuoteGenerationService: &api.QGS{
						Path: QGSTestSocketPath,
					},
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	})

	It("should use QGS path from annotation for TDX", func() {
		vmi := libvmi.New(withTDX())
		var domain api.Domain

		configurator := compute.NewLaunchSecurityDomainConfigurator("amd64")
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		Expect(domain.Spec.LaunchSecurity).NotTo(BeNil())
		Expect(domain.Spec.LaunchSecurity.Type).To(Equal("tdx"))
		Expect(domain.Spec.LaunchSecurity.QuoteGenerationService).NotTo(BeNil())
		Expect(domain.Spec.LaunchSecurity.QuoteGenerationService.Path).To(Equal(QGSTestSocketPath))
	})
})

func withTDX() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{
			TDX: &v1.TDX{},
		}
		if vmi.Annotations == nil {
			vmi.Annotations = map[string]string{}
		}
		vmi.Annotations[v1.QGSSocketPathAnnotation] = QGSTestSocketPath
	}
}
