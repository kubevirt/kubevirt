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

package validation_test

import (
	"crypto/tls"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/virt-operator/validation"
)

var _ = Describe("KubeVirt spec semantic validation", func() {
	test := field.NewPath("test")
	vmProfileField := test.Child("virtualMachineInstanceProfile")

	DescribeTable("ValidateVirtTemplateDeployment", func(kvSpec v1.KubeVirtSpec, expectError bool) {
		causes := validation.ValidateVirtTemplateDeployment(&kvSpec.Configuration)
		if expectError {
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(causes[0].Field).To(Equal("spec.configuration.virtTemplateDeployment.enabled"))
		} else {
			Expect(causes).To(BeEmpty())
		}
	},
		Entry("should reject when VirtTemplateDeployment enabled without Template feature gate",
			v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					VirtTemplateDeployment: &v1.VirtTemplateDeployment{
						Enabled: pointer.P(true),
					},
				},
			},
			true,
		),
		Entry("should allow when VirtTemplateDeployment enabled with Template feature gate",
			v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						FeatureGates: []string{featuregate.Template},
					},
					VirtTemplateDeployment: &v1.VirtTemplateDeployment{
						Enabled: pointer.P(true),
					},
				},
			},
			false,
		),
		Entry("should allow when VirtTemplateDeployment is nil",
			v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{},
			},
			false,
		),
		Entry("should allow when VirtTemplateDeployment.Enabled is nil",
			v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					VirtTemplateDeployment: &v1.VirtTemplateDeployment{
						Enabled: nil,
					},
				},
			},
			false,
		),
		Entry("should allow when VirtTemplateDeployment.Enabled is false",
			v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					VirtTemplateDeployment: &v1.VirtTemplateDeployment{
						Enabled: pointer.P(false),
					},
				},
			},
			false,
		),
	)

	DescribeTable("ValidateRoleAggregationStrategy", func(kvSpec v1.KubeVirtSpec, expectError bool) {
		causes := validation.ValidateRoleAggregationStrategy(&kvSpec.Configuration)
		if expectError {
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(causes[0].Field).To(Equal("spec.configuration.roleAggregationStrategy"))
		} else {
			Expect(causes).To(BeEmpty())
		}
	},
		Entry("should reject when RoleAggregationStrategy is Manual without OptOutRoleAggregation feature gate",
			v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					RoleAggregationStrategy: pointer.P(v1.RoleAggregationStrategyManual),
				},
			},
			true,
		),
		Entry("should allow when RoleAggregationStrategy is Manual with OptOutRoleAggregation feature gate",
			v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						FeatureGates: []string{featuregate.OptOutRoleAggregation},
					},
					RoleAggregationStrategy: pointer.P(v1.RoleAggregationStrategyManual),
				},
			},
			false,
		),
		Entry("should allow when RoleAggregationStrategy is nil",
			v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{},
			},
			false,
		),
		Entry("should allow when RoleAggregationStrategy is AggregateToDefault without feature gate",
			v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					RoleAggregationStrategy: pointer.P(v1.RoleAggregationStrategyAggregateToDefault),
				},
			},
			false,
		),
	)

	DescribeTable("ValidateMigrationConfiguration", func(oldConfig, newConfig *v1.KubeVirtConfiguration, expectError bool) {
		causes := validation.ValidateMigrationConfiguration(oldConfig, newConfig)
		if expectError {
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(causes[0].Field).To(Equal("spec.configuration.migrationConfiguration.maxDowntimeMs"))
		} else {
			Expect(causes).To(BeEmpty())
		}
	},
		Entry("should reject when MaxDowntimeMs is newly set without feature gate",
			&v1.KubeVirtConfiguration{},
			&v1.KubeVirtConfiguration{
				MigrationConfiguration: &v1.MigrationConfiguration{MaxDowntimeMs: pointer.P(uint64(900))},
			},
			true,
		),
		Entry("should allow when MaxDowntimeMs is set with feature gate",
			&v1.KubeVirtConfiguration{},
			&v1.KubeVirtConfiguration{
				MigrationConfiguration: &v1.MigrationConfiguration{MaxDowntimeMs: pointer.P(uint64(900))},
				DeveloperConfiguration: &v1.DeveloperConfiguration{
					FeatureGates: []string{featuregate.MigrationStallDetection},
				},
			},
			false,
		),
		Entry("should allow unrelated update when MaxDowntimeMs is unchanged and feature gate is disabled",
			&v1.KubeVirtConfiguration{
				MigrationConfiguration: &v1.MigrationConfiguration{MaxDowntimeMs: pointer.P(uint64(900))},
			},
			&v1.KubeVirtConfiguration{
				MigrationConfiguration: &v1.MigrationConfiguration{MaxDowntimeMs: pointer.P(uint64(900))},
			},
			false,
		),
		Entry("should reject changing MaxDowntimeMs when feature gate is disabled",
			&v1.KubeVirtConfiguration{
				MigrationConfiguration: &v1.MigrationConfiguration{MaxDowntimeMs: pointer.P(uint64(500))},
			},
			&v1.KubeVirtConfiguration{
				MigrationConfiguration: &v1.MigrationConfiguration{MaxDowntimeMs: pointer.P(uint64(900))},
			},
			true,
		),
	)

	DescribeTable("ValidateSeccompConfiguration", func(seccompConfiguration *v1.SeccompConfiguration, expectedFields []string) {
		causes := validation.ValidateSeccompConfiguration(test, seccompConfiguration)
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
					LocalhostProfile:      pointer.P("somethingNotImportant"),
				},
			},
		}, []string{vmProfileField.Child("customProfile", "runtimeDefaultProfile").String(), vmProfileField.Child("customProfile", "localhostProfile").String()}),
	)

	DescribeTable("ValidateCustomizeComponents", func(cc v1.CustomizeComponents, expectedCauses int) {
		causes := validation.ValidateCustomizeComponents(cc)
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
			causes := validation.ValidateTLSConfiguration(tlsConfiguration)

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

	Context("with AdditionalGuestMemoryOverheadRatio", func() {
		DescribeTable("the ratio must be parsable to float", func(unparsableRatio string) {
			causes := validation.ValidateGuestToRequestHeadroom(&unparsableRatio)
			Expect(causes).To(HaveLen(1))
		},
			Entry("not a number", "abcdefg"),
			Entry("number with bad formatting", "1.fd3ggx"),
		)

		DescribeTable("the ratio must be larger than 1", func(lessThanOneRatio string) {
			causes := validation.ValidateGuestToRequestHeadroom(&lessThanOneRatio)
			Expect(causes).ToNot(BeEmpty())
		},
			Entry("0.999", "0.999"),
			Entry("negative number", "-1.3"),
		)

		DescribeTable("valid values", func(validRatio string) {
			causes := validation.ValidateGuestToRequestHeadroom(&validRatio)
			Expect(causes).To(BeEmpty())
		},
			Entry("1.0", "1.0"),
			Entry("5", "5"),
			Entry("1.123", "1.123"),
		)
	})

	Context("ValidateInfraReplicas", func() {
		It("should reject zero replicas", func() {
			causes := validation.ValidateInfraReplicas(pointer.P(uint8(0)))
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		})

		It("should allow nil replicas", func() {
			Expect(validation.ValidateInfraReplicas(nil)).To(BeEmpty())
		})

		It("should allow non-zero replicas", func() {
			Expect(validation.ValidateInfraReplicas(pointer.P(uint8(2)))).To(BeEmpty())
		})
	})

	Context("ValidateFeatureGates", func() {
		It("should reject a gate present in both enabled and disabled lists", func() {
			causes := validation.ValidateFeatureGates(&v1.DeveloperConfiguration{
				FeatureGates:         []string{"ConflictGate", "ValidGate"},
				DisabledFeatureGates: []string{"ConflictGate"},
			})
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeForbidden))
		})

		It("should allow non-conflicting gates", func() {
			causes := validation.ValidateFeatureGates(&v1.DeveloperConfiguration{
				FeatureGates:         []string{"EnabledGate"},
				DisabledFeatureGates: []string{"DisabledGate"},
			})
			Expect(causes).To(BeEmpty())
		})

		It("should allow nil developer configuration", func() {
			Expect(validation.ValidateFeatureGates(nil)).To(BeEmpty())
		})
	})

	Context("ValidateKubeVirtSpec", func() {
		It("should aggregate causes across validators for an invalid spec", func() {
			kv := &v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Infra: &v1.ComponentConfig{Replicas: pointer.P(uint8(0))},
					Configuration: v1.KubeVirtConfiguration{
						DeveloperConfiguration: &v1.DeveloperConfiguration{
							FeatureGates:         []string{"ConflictGate"},
							DisabledFeatureGates: []string{"ConflictGate"},
						},
					},
				},
			}

			causes := validation.ValidateKubeVirtSpec(kv)
			Expect(causes).To(HaveLen(2))
			Expect(validation.CausesToError(causes)).To(HaveOccurred())
		})

		It("should return no causes and a nil error for a valid spec", func() {
			kv := &v1.KubeVirt{}
			causes := validation.ValidateKubeVirtSpec(kv)
			Expect(causes).To(BeEmpty())
			Expect(validation.CausesToError(causes)).ToNot(HaveOccurred())
		})
	})
})
