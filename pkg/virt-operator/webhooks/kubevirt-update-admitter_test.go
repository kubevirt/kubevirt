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
	"crypto/tls"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Validating KubeVirtUpdate Admitter", func() {

	test := field.NewPath("test")
	vmProfileField := test.Child("virtualMachineInstanceProfile")

	DescribeTable("validateSeccompConfiguration", func(seccompConfiguration *v1.SeccompConfiguration, expectedFields []string) {
		causes := validateSeccompConfiguration(test, seccompConfiguration)
		Expect(causes).To(HaveLen(len(expectedFields)))
		for _, cause := range causes {
			Expect(cause.Field).To(BeElementOf(expectedFields))
		}

	},
		Entry("don't specifying custom ", &v1.SeccompConfiguration{
			VirtualMachineInstanceProfile: &v1.VirtualMachineInstanceProfile{
				CustomProfile: nil,
			},
		}, []string{vmProfileField.Child("customProfile").String()}),

		Entry("having custom local and runtimeDefault Profile", &v1.SeccompConfiguration{
			VirtualMachineInstanceProfile: &v1.VirtualMachineInstanceProfile{
				CustomProfile: &v1.CustomProfile{
					RuntimeDefaultProfile: true,
					LocalhostProfile:      pointer.String("somethingNotImportant"),
				},
			},
		}, []string{vmProfileField.Child("customProfile", "runtimeDefaultProfile").String(), vmProfileField.Child("customProfile", "localhostProfile").String()}),
	)

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

	Context("with TLSConfiguration", func() {
		DescribeTable("should reject", func(tlsConfiguration *v1.TLSConfiguration, expectedErrorMessage string, indexInField int) {
			causes := validateTLSConfiguration(tlsConfiguration)

			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(Equal(expectedErrorMessage))
			field := "spec.configuration.tlsConfiguration.ciphers"
			if indexInField != -1 {
				field = fmt.Sprintf("%s#%d", field, indexInField)
			}
			Expect(causes[0].Field).To(Equal(field))

		},
			Entry("with unspecified minTLSVersion but non empty ciphers",
				&v1.TLSConfiguration{Ciphers: []string{tls.CipherSuiteName(tls.TLS_AES_256_GCM_SHA384)}},
				"You cannot specify ciphers when spec.configuration.tlsConfiguration.minTLSVersion is empty or VersionTLS13",
				-1,
			),
			Entry("with specified ciphers and minTLSVersion = 1.3",
				&v1.TLSConfiguration{Ciphers: []string{tls.CipherSuiteName(tls.TLS_AES_256_GCM_SHA384)}, MinTLSVersion: v1.VersionTLS13},
				"You cannot specify ciphers when spec.configuration.tlsConfiguration.minTLSVersion is empty or VersionTLS13",
				-1,
			),
			Entry("with unknown cipher in the list",
				&v1.TLSConfiguration{
					MinTLSVersion: v1.VersionTLS12,
					Ciphers:       []string{tls.CipherSuiteName(tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256), "NOT_VALID_CIPHER"},
				},
				"NOT_VALID_CIPHER is not a valid cipher",
				1,
			),
		)
	})

	Context("with memoryOverCommitmentProfile", func() {
		It("should reject when both Custom and Strategy are defined", func() {
			strategy := v1.Baseline
			memoryOverCommitmentProfile := &v1.MemoryOverCommitmentProfile{
				Strategy: &strategy,
				Custom:   &v1.MemoryOverCommitment{},
			}
			causes := validateMemoryOverCommitment(memoryOverCommitmentProfile)

			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(causes[0].Message).To(Equal("It is not possible to specify both Strategy and Custom fields"))
			Expect(causes[0].Field).To(Equal("spec.configuration.memoryOverCommitmentProfile"))

		})
	})

	Context("with AdditionalGuestMemoryOverheadRatio", func() {
		DescribeTable("the ratio must be parsable to float", func(unparsableRatio string) {
			causes := validateGuestToRequestHeadroom(&unparsableRatio)
			Expect(causes).To(HaveLen(1))
		},
			Entry("not a number", "abcdefg"),
			Entry("number with bad formatting", "1.fd3ggx"),
		)

		DescribeTable("the ratio must be larger than 1", func(lessThanOneRatio string) {
			causes := validateGuestToRequestHeadroom(&lessThanOneRatio)
			Expect(causes).ToNot(BeEmpty())
		},
			Entry("0.999", "0.999"),
			Entry("negative number", "-1.3"),
		)

		DescribeTable("valid values", func(validRatio string) {

		},
			Entry("1.0", "1.0"),
			Entry("5", "5"),
			Entry("1.123", "1.123"),
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
