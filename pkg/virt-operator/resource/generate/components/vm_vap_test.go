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

var _ = Describe("VirtualMachine Validation Admission Policy", func() {
	const namespace = "kubevirt"

	Context("ValidatingAdmissionPolicy", func() {
		It("should generate the expected policy", func() {
			vap := components.NewVMValidatingAdmissionPolicy(namespace)

			Expect(vap.Name).To(Equal("vm-policy.kubevirt.io"))
			Expect(vap.Kind).To(Equal("ValidatingAdmissionPolicy"))
			Expect(vap.Spec.ParamKind).NotTo(BeNil())
			Expect(vap.Spec.ParamKind.APIVersion).To(Equal("kubevirt.io/v1"))
			Expect(vap.Spec.ParamKind.Kind).To(Equal("KubeVirt"))

			// VM has 8 validations that can be migrated to CEL
			Expect(vap.Spec.Validations).To(HaveLen(8))

			// Verify resource rules
			Expect(vap.Spec.MatchConstraints.ResourceRules).To(HaveLen(1))
			rule := vap.Spec.MatchConstraints.ResourceRules[0]
			Expect(rule.Operations).To(HaveLen(2))
			Expect(rule.Operations[0]).To(Equal(admissionregistrationv1.Create))
			Expect(rule.Operations[1]).To(Equal(admissionregistrationv1.Update))
			Expect(rule.Rule.APIGroups).To(ContainElement("kubevirt.io"))
			Expect(rule.Rule.Resources).To(ContainElement("virtualmachines"))

			// Verify validation messages
			messages := make([]string, len(vap.Spec.Validations))
			for i, v := range vap.Spec.Validations {
				messages[i] = v.Message
			}
			Expect(messages).To(ContainElements(
				components.VMErrTemplateRequired,
				components.VMErrRunningAndRunStrategyMutuallyExclusive,
				components.VMErrRunStrategyRequired,
				components.VMErrRunStrategyInvalid,
				components.VMErrWaitAsReceiverFeatureGateDisabled,
				components.VMErrDataVolumeNameRequired,
				components.VMErrDataVolumePVCOrStorageRequired,
				components.VMErrCPUSocketsExceedMaxSockets,
			))
		})

		It("should validate both CREATE and UPDATE operations", func() {
			vap := components.NewVMValidatingAdmissionPolicy(namespace)

			rule := vap.Spec.MatchConstraints.ResourceRules[0]
			Expect(rule.Operations).To(ConsistOf(
				admissionregistrationv1.Create,
				admissionregistrationv1.Update,
			))
		})
	})

	Context("ValidatingAdmissionPolicyBinding", func() {
		It("should generate the expected policy binding", func() {
			vap := components.NewVMValidatingAdmissionPolicy(namespace)
			binding := components.NewVMValidatingAdmissionPolicyBinding(namespace)

			Expect(binding.Name).To(Equal("vm-policy-binding"))
			Expect(binding.Kind).To(Equal("ValidatingAdmissionPolicyBinding"))
			Expect(binding.Spec.PolicyName).To(Equal(vap.Name))
			Expect(binding.Spec.ParamRef).NotTo(BeNil())
			Expect(binding.Spec.ParamRef.Name).To(Equal("kubevirt"))
			Expect(binding.Spec.ParamRef.Namespace).To(Equal(namespace))
			Expect(binding.Spec.ValidationActions).To(HaveLen(1))
			Expect(binding.Spec.ValidationActions[0]).To(Equal(admissionregistrationv1.Deny))
		})
	})
})
