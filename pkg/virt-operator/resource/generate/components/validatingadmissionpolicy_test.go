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

package components_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

var _ = Describe("Validation Admission Policy", func() {
	Context("ValidatingAdmissionPolicy", func() {
		It("should generate the expected policy", func() {
			const userName = "system:serviceaccount:kubevirt-ns:kubevirt-handler"
			validatingAdmissionPolicy := components.NewHandlerV1ValidatingAdmissionPolicy(userName)

			expectedMatchConditionExpression := fmt.Sprintf("request.userInfo.username == %q", userName)
			Expect(validatingAdmissionPolicy.Spec.MatchConditions[0].Expression).To(Equal(expectedMatchConditionExpression))
			Expect(validatingAdmissionPolicy.Kind).ToNot(BeEmpty())
		})
	})

	Context("ValidatingAdmissionPolicyBinding", func() {
		It("should generate the expected policy binding", func() {
			const userName = "system:serviceaccount:kubevirt-ns:kubevirt-handler"
			validatingAdmissionPolicy := components.NewHandlerV1ValidatingAdmissionPolicy(userName)
			validatingAdmissionPolicyBinding := components.NewHandlerV1ValidatingAdmissionPolicyBinding()

			Expect(validatingAdmissionPolicyBinding.Spec.PolicyName).To(Equal(validatingAdmissionPolicy.Name))
			Expect(validatingAdmissionPolicyBinding.Kind).ToNot(BeEmpty())
		})
	})
})
