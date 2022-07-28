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
