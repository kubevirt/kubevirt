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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
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

	DescribeTable("should accept the WorkloadUpdateMethods", func(oldSpec, newSpec *v1.KubeVirtSpec) {
		causes := validateWorkloadUpdateMethods(oldSpec, newSpec)
		Expect(causes).To(BeEmpty())
	},
		Entry("should accept LiveMigrate feature gate enabled and workloadUpdateMethod LiveMigrate",
			newKubeVirtSpec(),
			newKubeVirtSpec(
				withFeatureGate(virtconfig.LiveMigrationGate),
				withWorkloadUpdateMethod(v1.WorkloadUpdateMethodLiveMigrate),
			),
		),
		Entry("should accept LiveMigrate feature gate enabled and no workloadUpdateMethod",
			newKubeVirtSpec(),
			newKubeVirtSpec(
				withFeatureGate(virtconfig.LiveMigrationGate),
			),
		),
		Entry("should accept LiveMigrate feature gate disabled and no workloadUpdateMethod",
			newKubeVirtSpec(),
			newKubeVirtSpec(),
		),
		Entry("should accept if the misconfiguration was already present",
			newKubeVirtSpec(
				withWorkloadUpdateMethod(v1.WorkloadUpdateMethodLiveMigrate),
			),
			newKubeVirtSpec(
				withWorkloadUpdateMethod(v1.WorkloadUpdateMethodLiveMigrate),
			),
		),
	)

	It("should reject LiveMigrate feature gate disabled and workloadUpdateMethod LiveMigrate", func() {
		causes := validateWorkloadUpdateMethods(
			newKubeVirtSpec(),
			newKubeVirtSpec(
				withWorkloadUpdateMethod(v1.WorkloadUpdateMethodLiveMigrate),
			))
		Expect(causes).NotTo(BeEmpty())
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
