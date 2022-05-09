package webhooks

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocpv1 "github.com/openshift/api/config/v1"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("TLS Security Profile", func() {
	type loadSecurityProfileCase struct {
		config                *v1.KubeVirtSpec
		expectedCiphers       []string
		expectedMinTLSVersion ocpv1.TLSProtocolVersion
	}

	testCustomTLSProfileSpec := ocpv1.TLSProfileSpec{
		Ciphers:       []string{"ECDHE-ECDSA-CHACHA20-POLY1305", "ECDHE-RSA-CHACHA20-POLY1305", "ECDHE-RSA-AES128-GCM-SHA256", "ECDHE-ECDSA-AES128-GCM-SHA256"},
		MinTLSVersion: "VersionTLS11",
	}

	DescribeTable("SecurityProfileSpec should be loaded correctly",
		func(c loadSecurityProfileCase) {
			ciphers, minTLSVersion := SelectCipherSuitesAndMinTLSVersion(c.config.TLSSecurityProfile)
			Expect(ciphers).To(Equal(CipherSuitesIDs(c.expectedCiphers)))
			Expect(minTLSVersion).To(Equal(TlsVersion(c.expectedMinTLSVersion)))
		},
		Entry("when TLSSecurityProfile is nil", loadSecurityProfileCase{
			config:                &v1.KubeVirtSpec{},
			expectedCiphers:       ocpv1.TLSProfiles[ocpv1.TLSProfileIntermediateType].Ciphers,
			expectedMinTLSVersion: ocpv1.TLSProfiles[ocpv1.TLSProfileIntermediateType].MinTLSVersion,
		}),
		Entry("when Old Security Profile is selected", loadSecurityProfileCase{
			config: &v1.KubeVirtSpec{
				TLSSecurityProfile: &ocpv1.TLSSecurityProfile{
					Type: ocpv1.TLSProfileOldType,
					Old:  &ocpv1.OldTLSProfile{},
				},
			},
			expectedCiphers:       ocpv1.TLSProfiles[ocpv1.TLSProfileOldType].Ciphers,
			expectedMinTLSVersion: ocpv1.TLSProfiles[ocpv1.TLSProfileOldType].MinTLSVersion,
		}),
		Entry("when Intermediate Security Profile is selected", loadSecurityProfileCase{
			config: &v1.KubeVirtSpec{
				TLSSecurityProfile: &ocpv1.TLSSecurityProfile{
					Type:         ocpv1.TLSProfileIntermediateType,
					Intermediate: &ocpv1.IntermediateTLSProfile{},
				},
			},
			expectedCiphers:       ocpv1.TLSProfiles[ocpv1.TLSProfileIntermediateType].Ciphers,
			expectedMinTLSVersion: ocpv1.TLSProfiles[ocpv1.TLSProfileIntermediateType].MinTLSVersion,
		}),
		Entry("when Custom Security Profile is selected", loadSecurityProfileCase{
			config: &v1.KubeVirtSpec{
				TLSSecurityProfile: &ocpv1.TLSSecurityProfile{
					Type: ocpv1.TLSProfileCustomType,
					Custom: &ocpv1.CustomTLSProfile{
						TLSProfileSpec: testCustomTLSProfileSpec,
					},
				},
			},
			expectedCiphers:       testCustomTLSProfileSpec.Ciphers,
			expectedMinTLSVersion: testCustomTLSProfileSpec.MinTLSVersion,
		}),
	)
})
