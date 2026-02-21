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
					Type: "sev",
					LaunchSecuritySEVCommon: api.LaunchSecuritySEVCommon{
						Policy: "0x5",
					},
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
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
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	})
})

var _ = Describe("configureSEVLaunchSecurity", func() {
	It("should configure basic SEV with default policy", func() {
		sevConfig := &v1.SEV{}

		result := compute.ConfigureSEVLaunchSecurity(sevConfig)

		Expect(result).ToNot(BeNil())
		Expect(result.Type).To(Equal("sev"))
		Expect(result.LaunchSecuritySEVCommon.Policy).To(Equal("0x1")) // Default SEV policy
		Expect(result.LaunchSecuritySEV.DHCert).To(BeEmpty())
		Expect(result.LaunchSecuritySEV.Session).To(BeEmpty())
	})

	It("should configure SEV with custom DHCert and Session", func() {
		sevConfig := &v1.SEV{
			DHCert:  "test-dh-cert",
			Session: "test-session",
		}

		result := compute.ConfigureSEVLaunchSecurity(sevConfig)

		Expect(result).ToNot(BeNil())
		Expect(result.Type).To(Equal("sev"))
		Expect(result.LaunchSecuritySEV.DHCert).To(Equal("test-dh-cert"))
		Expect(result.LaunchSecuritySEV.Session).To(Equal("test-session"))
	})

	It("should configure SEV-ES with encrypted state policy", func() {
		encryptedState := true
		sevConfig := &v1.SEV{
			Policy: &v1.SEVPolicy{
				EncryptedState: &encryptedState,
			},
		}

		result := compute.ConfigureSEVLaunchSecurity(sevConfig)

		Expect(result).ToNot(BeNil())
		Expect(result.Type).To(Equal("sev"))
		Expect(result.Policy).To(Equal("0x5")) // NoDebug + EncryptedState
	})
})

var _ = Describe("configureSEVSNPLaunchSecurity", func() {
	It("should configure basic SEV-SNP with default policy", func() {
		snpConfig := &v1.SEVSNP{}

		result := compute.ConfigureSEVSNPLaunchSecurity(snpConfig)

		Expect(result).ToNot(BeNil())
		Expect(result.Type).To(Equal("sev-snp"))
		Expect(result.Policy).To(Equal("0x30000")) // Default SEV-SNP policy
	})

	It("should configure SEV-SNP with custom policy", func() {
		snpConfig := &v1.SEVSNP{
			Policy: "0x40000",
		}

		result := compute.ConfigureSEVSNPLaunchSecurity(snpConfig)

		Expect(result).ToNot(BeNil())
		Expect(result.Type).To(Equal("sev-snp"))
		Expect(result.Policy).To(Equal("0x40000"))
	})

	It("should configure SEV-SNP with all fields", func() {
		snpConfig := &v1.SEVSNP{
			VCEK:            "test-vcek",
			AuthorKey:       "test-author-key",
			KernelHashes:    "test-kernel-hashes",
			Cbitpos:         "51",
			IdBlock:         "test-id-block",
			IdAuth:          "test-id-auth",
			ReducedPhysBits: "1",
		}

		result := compute.ConfigureSEVSNPLaunchSecurity(snpConfig)

		Expect(result).ToNot(BeNil())
		Expect(result.Type).To(Equal("sev-snp"))
		Expect(result.LaunchSecuritySNP.VCEK).To(Equal("test-vcek"))
		Expect(result.LaunchSecuritySNP.AuthorKey).To(Equal("test-author-key"))
		Expect(result.LaunchSecuritySEVCommon.KernelHashes).To(Equal("test-kernel-hashes"))
		Expect(result.LaunchSecuritySEVCommon.Cbitpos).To(Equal("51"))
		Expect(result.LaunchSecuritySNP.IdBlock).To(Equal("test-id-block"))
		Expect(result.LaunchSecuritySNP.IdAuth).To(Equal("test-id-auth"))
		Expect(result.LaunchSecuritySEVCommon.ReducedPhysBits).To(Equal("1"))
	})

	It("should handle empty string fields correctly", func() {
		snpConfig := &v1.SEVSNP{
			VCEK:         "",
			AuthorKey:    "test-author-key",
			KernelHashes: "",
		}

		result := compute.ConfigureSEVSNPLaunchSecurity(snpConfig)

		Expect(result).ToNot(BeNil())
		Expect(result.LaunchSecuritySNP.VCEK).To(BeEmpty())
		Expect(result.LaunchSecuritySNP.AuthorKey).To(Equal("test-author-key"))
		Expect(result.LaunchSecuritySEVCommon.KernelHashes).To(BeEmpty())
	})
})

var _ = Describe("amd64LaunchSecurity", func() {
	It("should configure SEV when only SEV is specified", func() {
		vmi := libvmi.New()
		vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{
			SEV: &v1.SEV{},
		}

		result := compute.Amd64LaunchSecurity(vmi)

		Expect(result).ToNot(BeNil())
		Expect(result.Type).To(Equal("sev"))

	})

	It("should configure SEV-SNP when only SNP is specified", func() {
		vmi := libvmi.New()
		vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{
			SNP: &v1.SEVSNP{},
		}

		result := compute.Amd64LaunchSecurity(vmi)

		Expect(result).ToNot(BeNil())
		Expect(result.Type).To(Equal("sev-snp"))

	})

	It("should configure TDX when TDX is specified", func() {
		vmi := libvmi.New()
		vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{
			TDX: &v1.TDX{},
		}

		result := compute.Amd64LaunchSecurity(vmi)

		Expect(result).ToNot(BeNil())
		Expect(result.Type).To(Equal("tdx"))
	})
})

func withTDX() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{
			TDX: &v1.TDX{},
		}
	}
}
