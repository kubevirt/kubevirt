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

package dra

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("ResourceClaims", func() {
	const directClaimName = "direct-claim"
	const templateClaimName = "template-claim"
	DescribeTable("should convert VMI resourceClaims to Pod resourceClaims",
		func(resourceClaims []v1.VirtualMachineInstanceResourceClaim, expected []k8sv1.PodResourceClaim) {
			Expect(ToPodResourceClaims(resourceClaims)).To(Equal(expected))
		},
		Entry("nil resourceClaims",
			nil,
			nil,
		),
		Entry("empty resourceClaims",
			[]v1.VirtualMachineInstanceResourceClaim{},
			nil,
		),
		Entry("direct and template resourceClaims",
			[]v1.VirtualMachineInstanceResourceClaim{
				{
					Name:              directClaimName,
					ResourceClaimName: ptr.To("resource-claim"),
				},
				{
					Name:                      templateClaimName,
					ResourceClaimTemplateName: ptr.To("resource-claim-template"),
				},
			},
			[]k8sv1.PodResourceClaim{
				{
					Name:              directClaimName,
					ResourceClaimName: ptr.To("resource-claim"),
				},
				{
					Name:                      templateClaimName,
					ResourceClaimTemplateName: ptr.To("resource-claim-template"),
				},
			},
		),
	)
})
