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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

var _ = Describe("VirtualMachineInstance Validation Admission Policy", func() {
	Context("ValidatingAdmissionPolicy", func() {
		It("should generate a minimal policy", func() {
			vap := components.NewVMIValidatingAdmissionPolicy()

			Expect(vap.Name).To(Equal("vmi-policy.kubevirt.io"))
			Expect(vap.Kind).To(Equal("ValidatingAdmissionPolicy"))

			// VMI has minimal CEL validation - only 1 basic check
			Expect(vap.Spec.Validations).To(HaveLen(1))

			// Verify resource rules - UPDATE only, no CREATE validation in VAP
			Expect(vap.Spec.MatchConstraints.ResourceRules).To(HaveLen(1))
			rule := vap.Spec.MatchConstraints.ResourceRules[0]
			Expect(rule.Operations).To(HaveLen(1))
			Expect(rule.Operations[0]).To(Equal(admissionregistrationv1.Update))
			Expect(rule.Rule.APIGroups).To(ContainElement("kubevirt.io"))
			Expect(rule.Rule.Resources).To(ContainElement("virtualmachineinstances"))
		})

		It("should not have param kind since webhook handles all complex validation", func() {
			vap := components.NewVMIValidatingAdmissionPolicy()
			Expect(vap.Spec.ParamKind).To(BeNil())
		})

		It("should only validate UPDATE operations", func() {
			vap := components.NewVMIValidatingAdmissionPolicy()

			// CREATE validation (~2100 lines) cannot be migrated to CEL
			// Only UPDATE has a simple immutability check
			rule := vap.Spec.MatchConstraints.ResourceRules[0]
			Expect(rule.Operations).To(ConsistOf(admissionregistrationv1.Update))
			Expect(rule.Operations).NotTo(ContainElement(admissionregistrationv1.Create))
		})
	})

	Context("ValidatingAdmissionPolicyBinding", func() {
		It("should generate the expected policy binding", func() {
			vap := components.NewVMIValidatingAdmissionPolicy()
			binding := components.NewVMIValidatingAdmissionPolicyBinding()

			Expect(binding.Name).To(Equal("vmi-policy-binding"))
			Expect(binding.Kind).To(Equal("ValidatingAdmissionPolicyBinding"))
			Expect(binding.Spec.PolicyName).To(Equal(vap.Name))
			Expect(binding.Spec.ValidationActions).To(HaveLen(1))
			Expect(binding.Spec.ValidationActions[0]).To(Equal(admissionregistrationv1.Deny))
		})

		It("should not have param ref since policy has no param kind", func() {
			binding := components.NewVMIValidatingAdmissionPolicyBinding()
			Expect(binding.Spec.ParamRef).To(BeNil())
		})
	})
})
