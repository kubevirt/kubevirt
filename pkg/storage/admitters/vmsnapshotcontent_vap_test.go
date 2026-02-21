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

package admitters

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

var _ = Describe("VirtualMachineSnapshotContent VAP validation", func() {
	var (
		vap     *admissionregistrationv1.ValidatingAdmissionPolicy
		binding *admissionregistrationv1.ValidatingAdmissionPolicyBinding
	)

	BeforeEach(func() {
		vap = components.NewVMSnapshotContentValidatingAdmissionPolicy()
		binding = components.NewVMSnapshotContentValidatingAdmissionPolicyBinding()
	})

	Context("VAP structure validation", func() {
		It("should have correct metadata", func() {
			Expect(vap.Name).To(Equal("vmsnapshotcontent-policy.snapshot.kubevirt.io"))
			Expect(binding.Spec.PolicyName).To(Equal(vap.Name))
		})

		It("should target VirtualMachineSnapshotContent resources", func() {
			Expect(vap.Spec.MatchConstraints.ResourceRules).To(HaveLen(1))
			rule := vap.Spec.MatchConstraints.ResourceRules[0]
			Expect(rule.Rule.APIGroups).To(ContainElement("snapshot.kubevirt.io"))
			Expect(rule.Rule.Resources).To(ContainElement("virtualmachinesnapshotcontents"))
			Expect(rule.Operations).To(HaveLen(1))
			Expect(rule.Operations[0]).To(Equal(admissionregistrationv1.Update))
		})

		It("should not require params since there are no config-dependent validations", func() {
			Expect(vap.Spec.ParamKind).To(BeNil())
			Expect(binding.Spec.ParamRef).To(BeNil())
		})
	})

	Context("Validation rules", func() {
		It("should have validation for spec immutability", func() {
			Expect(vap.Spec.Validations).To(HaveLen(1))
			validation := vap.Spec.Validations[0]
			Expect(validation.Message).To(Equal(components.VMSnapshotContentErrSpecImmutable))
			Expect(validation.Expression).To(Equal("object.spec == oldObject.spec"))
		})
	})

	Context("Resource design", func() {
		It("should validate UPDATE operations only", func() {
			// VirtualMachineSnapshotContent is created by the snapshot controller,
			// not directly by users. Therefore, we only validate UPDATE operations
			// to ensure the snapshot data (spec) remains immutable once created.
			//
			// This prevents accidental or malicious modifications to snapshot content
			// which could compromise data integrity.

			rule := vap.Spec.MatchConstraints.ResourceRules[0]
			Expect(rule.Operations).To(ConsistOf(admissionregistrationv1.Update))
			Expect(rule.Operations).NotTo(ContainElement(admissionregistrationv1.Create))
		})

		It("should enforce spec immutability to protect snapshot data", func() {
			// VirtualMachineSnapshotContent.spec contains the actual snapshot data
			// including VM spec and volume backup information. Once created, this
			// data must not be modified to maintain snapshot consistency and integrity.
			//
			// The CEL expression "object.spec == oldObject.spec" ensures that any
			// UPDATE operation cannot modify the spec field.

			validation := vap.Spec.Validations[0]
			Expect(validation.Expression).To(Equal("object.spec == oldObject.spec"))
			Expect(validation.Message).To(ContainSubstring("immutable"))
		})
	})

	Context("Comparison with similar resources", func() {
		It("should follow the same immutability pattern as VirtualMachineSnapshot", func() {
			// Both VirtualMachineSnapshot and VirtualMachineSnapshotContent enforce
			// spec immutability, which is a common pattern for snapshot resources.
			// This test documents the consistency across snapshot-related resources.

			vmSnapshotVAP := components.NewVMSnapshotValidatingAdmissionPolicy("kubevirt")

			// Find the spec immutability validation in VirtualMachineSnapshot VAP
			var vmSnapshotImmutabilityValidation *admissionregistrationv1.Validation
			for i := range vmSnapshotVAP.Spec.Validations {
				if vmSnapshotVAP.Spec.Validations[i].Message == components.VMSnapshotErrSpecImmutable {
					vmSnapshotImmutabilityValidation = &vmSnapshotVAP.Spec.Validations[i]
					break
				}
			}
			Expect(vmSnapshotImmutabilityValidation).NotTo(BeNil())

			// Both should use the same CEL expression for spec immutability
			contentImmutabilityValidation := vap.Spec.Validations[0]
			Expect(contentImmutabilityValidation.Expression).To(ContainSubstring("object.spec == oldObject.spec"))
			Expect(vmSnapshotImmutabilityValidation.Expression).To(ContainSubstring("object.spec == oldObject.spec"))
		})
	})
})
