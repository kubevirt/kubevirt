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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package webhooks

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocpv1 "github.com/openshift/api/config/v1"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Validating KubeVirtUpdate Admitter", func() {

	DescribeTable("test validateCustomizeComponents", func(cc v1.CustomizeComponents, expectedCauses int) {
		causes := validateCustomizeComponents(cc)
		Expect(causes).To(HaveLen(expectedCauses))
	},
		Entry("invalid values rejected", v1.CustomizeComponents{
			Patches: []v1.CustomizeComponentsPatch{
				{
					ResourceName: "virt-api",
					ResourceType: "Deployment",
					Type:         v1.StrategicMergePatchType,
					Patch:        `{"json: "not valid"}`,
				},
			},
		}, 1),
		Entry("empty patch field rejected", v1.CustomizeComponents{
			Patches: []v1.CustomizeComponentsPatch{
				{
					ResourceName: "virt-api",
					ResourceType: "Deployment",
					Type:         v1.StrategicMergePatchType,
					Patch:        "",
				},
			},
		}, 1),
		Entry("valid values accepted", v1.CustomizeComponents{
			Patches: []v1.CustomizeComponentsPatch{
				{
					ResourceName: "virt-api",
					ResourceType: "Deployment",
					Type:         v1.StrategicMergePatchType,
					Patch:        `{}`,
				},
			},
		}, 0),
	)
	Context("with TLSSecurityProfile", func() {
		DescribeTable("should reject with missing type field but", func(securityProfile *ocpv1.TLSSecurityProfile) {
			causes := validateTLSSecurityProfile(
				securityProfile)

			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(Equal("Required field"))
			Expect(causes[0].Field).To(Equal("spec.tlsSecurityProfile.type"))
		},
			Entry("Intermediate field is present", &ocpv1.TLSSecurityProfile{
				Intermediate: &ocpv1.IntermediateTLSProfile{},
			}),
			Entry("Old field is present", &ocpv1.TLSSecurityProfile{
				Old: &ocpv1.OldTLSProfile{},
			}),
			Entry("Modern field is present", &ocpv1.TLSSecurityProfile{
				Modern: &ocpv1.ModernTLSProfile{},
			}),
			Entry("Custom field is present", &ocpv1.TLSSecurityProfile{
				Custom: &ocpv1.CustomTLSProfile{},
			}),
		)

		DescribeTable("should reject with missing fields", func(profileType ocpv1.TLSProfileType, specField string) {
			causes := validateTLSSecurityProfile(
				&ocpv1.TLSSecurityProfile{
					Type: profileType,
				})

			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(Equal(fmt.Sprintf("Required field when using spec.tlsSecurityProfile.type==%s", profileType)))
			Expect(causes[0].Field).To(Equal(fmt.Sprintf("spec.tlsSecurityProfile.%s", specField)))
		},
			Entry("with Intermediate", ocpv1.TLSProfileIntermediateType, "intermediate"),
			Entry("with Old", ocpv1.TLSProfileOldType, "old"),
			Entry("with Modern", ocpv1.TLSProfileModernType, "modern"),
			Entry("with Custom", ocpv1.TLSProfileCustomType, "custom"),
		)

		DescribeTable("should reject with Custom type", func(tlsSecuritySpec ocpv1.TLSProfileSpec, specField, expectedErrorMessage string) {
			causes := validateTLSSecurityProfile(
				&ocpv1.TLSSecurityProfile{
					Type: ocpv1.TLSProfileCustomType,
					Custom: &ocpv1.CustomTLSProfile{
						TLSProfileSpec: tlsSecuritySpec,
					},
				})

			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(Equal(expectedErrorMessage))
			Expect(causes[0].Field).To(Equal(fmt.Sprintf("spec.tlsSecurityProfile.custom.%s", specField)))

		},
			Entry("with unspecified minTLSVersion",
				ocpv1.TLSProfileSpec{Ciphers: []string{"ECDHE-ECDSA-CHACHA20-POLY1305"}},
				"minTLSVersion",
				"Required field when using spec.tlsSecurityProfile.type==Custom",
			),
			Entry("with unspecified ciphers and minTLSVersion < 1.3",
				ocpv1.TLSProfileSpec{MinTLSVersion: "VersionTLS11"},
				"ciphers",
				"You must specify at least one cipher",
			),
			Entry("with specified ciphers and minTLSVersion = 1.3",
				ocpv1.TLSProfileSpec{MinTLSVersion: "VersionTLS13", Ciphers: []string{"ECDHE-ECDSA-CHACHA20-POLY1305"}},
				"ciphers",
				"You cannot specify ciphers when spec.tlsSecurityProfile.custom.minTLSVersion is VersionTLS13",
			),
		)
	})
})

type kubevirtSpecOption func(*v1.KubeVirtSpec)

func newKubeVirtSpec(opts ...kubevirtSpecOption) *v1.KubeVirtSpec {
	kvSpec := &v1.KubeVirtSpec{
		Configuration: v1.KubeVirtConfiguration{
			DeveloperConfiguration: &v1.DeveloperConfiguration{},
		},
	}

	for _, kvOptFunc := range opts {
		kvOptFunc(kvSpec)
	}
	return kvSpec
}

func withFeatureGate(featureGate string) kubevirtSpecOption {
	return func(kvSpec *v1.KubeVirtSpec) {
		kvSpec.Configuration.DeveloperConfiguration.FeatureGates = append(kvSpec.Configuration.DeveloperConfiguration.FeatureGates, featureGate)
	}
}

func withWorkloadUpdateMethod(method v1.WorkloadUpdateMethod) kubevirtSpecOption {
	return func(kvSpec *v1.KubeVirtSpec) {
		kvSpec.WorkloadUpdateStrategy.WorkloadUpdateMethods = append(kvSpec.WorkloadUpdateStrategy.WorkloadUpdateMethods, method)
	}
}
