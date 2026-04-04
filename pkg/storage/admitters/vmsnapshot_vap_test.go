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

var _ = Describe("VirtualMachineSnapshot VAP equivalence", func() {
	const namespace = "kubevirt"

	var (
		vap     *admissionregistrationv1.ValidatingAdmissionPolicy
		binding *admissionregistrationv1.ValidatingAdmissionPolicyBinding
	)

	BeforeEach(func() {
		vap = components.NewVMSnapshotValidatingAdmissionPolicy(namespace)
		binding = components.NewVMSnapshotValidatingAdmissionPolicyBinding(namespace)
	})

	Context("VAP structure validation", func() {
		It("should have correct metadata", func() {
			Expect(vap.Name).To(Equal("vmsnapshot-policy.snapshot.kubevirt.io"))
			Expect(binding.Spec.PolicyName).To(Equal(vap.Name))
		})

		It("should target VirtualMachineSnapshot resources", func() {
			Expect(vap.Spec.MatchConstraints.ResourceRules).To(HaveLen(1))
			rule := vap.Spec.MatchConstraints.ResourceRules[0]
			Expect(rule.Rule.APIGroups).To(ContainElement("snapshot.kubevirt.io"))
			Expect(rule.Rule.Resources).To(ContainElement("virtualmachinesnapshots"))
			Expect(rule.Operations).To(HaveLen(2))
			Expect(rule.Operations[0]).To(Equal(admissionregistrationv1.Create))
			Expect(rule.Operations[1]).To(Equal(admissionregistrationv1.Update))
		})

		It("should reference KubeVirt as param kind", func() {
			Expect(vap.Spec.ParamKind).NotTo(BeNil())
			Expect(vap.Spec.ParamKind.Kind).To(Equal("KubeVirt"))
			Expect(vap.Spec.ParamKind.APIVersion).To(Equal("kubevirt.io/v1"))
		})

		It("should bind to kubevirt singleton", func() {
			Expect(binding.Spec.ParamRef).NotTo(BeNil())
			Expect(binding.Spec.ParamRef.Name).To(Equal("kubevirt"))
			Expect(binding.Spec.ParamRef.Namespace).To(Equal(namespace))
		})
	})

	Context("Validation rules", func() {
		It("should have validation for feature gate", func() {
			var featureGateValidation *admissionregistrationv1.Validation
			for i := range vap.Spec.Validations {
				if vap.Spec.Validations[i].Message == components.VMSnapshotErrFeatureGateDisabled {
					featureGateValidation = &vap.Spec.Validations[i]
					break
				}
			}
			Expect(featureGateValidation).NotTo(BeNil())
			Expect(featureGateValidation.Expression).To(ContainSubstring("request.operation != 'CREATE'"))
			Expect(featureGateValidation.Expression).To(ContainSubstring("featureGates"))
			Expect(featureGateValidation.Expression).To(ContainSubstring("Snapshot"))
		})

		It("should have validation for missing apiGroup", func() {
			var apiGroupValidation *admissionregistrationv1.Validation
			for i := range vap.Spec.Validations {
				if vap.Spec.Validations[i].Message == components.VMSnapshotErrMissingAPIGroup {
					apiGroupValidation = &vap.Spec.Validations[i]
					break
				}
			}
			Expect(apiGroupValidation).NotTo(BeNil())
			Expect(apiGroupValidation.Expression).To(ContainSubstring("has(object.spec.source.apiGroup)"))
		})

		It("should have validation for invalid apiGroup", func() {
			var apiGroupValidation *admissionregistrationv1.Validation
			for i := range vap.Spec.Validations {
				if vap.Spec.Validations[i].Message == components.VMSnapshotErrInvalidAPIGroup {
					apiGroupValidation = &vap.Spec.Validations[i]
					break
				}
			}
			Expect(apiGroupValidation).NotTo(BeNil())
			Expect(apiGroupValidation.Expression).To(ContainSubstring("kubevirt.io"))
		})

		It("should have validation for invalid kind", func() {
			var kindValidation *admissionregistrationv1.Validation
			for i := range vap.Spec.Validations {
				if vap.Spec.Validations[i].Message == components.VMSnapshotErrInvalidKind {
					kindValidation = &vap.Spec.Validations[i]
					break
				}
			}
			Expect(kindValidation).NotTo(BeNil())
			Expect(kindValidation.Expression).To(ContainSubstring("VirtualMachine"))
		})

		It("should have validation for spec immutability", func() {
			var immutabilityValidation *admissionregistrationv1.Validation
			for i := range vap.Spec.Validations {
				if vap.Spec.Validations[i].Message == components.VMSnapshotErrSpecImmutable {
					immutabilityValidation = &vap.Spec.Validations[i]
					break
				}
			}
			Expect(immutabilityValidation).NotTo(BeNil())
			Expect(immutabilityValidation.Expression).To(ContainSubstring("request.operation != 'UPDATE'"))
			Expect(immutabilityValidation.Expression).To(ContainSubstring("object.spec == oldObject.spec"))
		})
	})

	Context("Test case coverage", func() {
		It("should cover the same cases as webhook - missing apiGroup", func() {
			// This test verifies that the VAP has equivalent logic to the webhook test
			// "should reject missing apigroup" from vmsnapshot_test.go:110
			//
			// Webhook logic (vmsnapshot.go:79-88):
			//   if vmSnapshot.Spec.Source.APIGroup == nil { reject with "missing apiGroup" }
			//
			// VAP expression should be:
			//   request.operation != 'CREATE' || has(object.spec.source.apiGroup)
			//
			// This rejects CREATE when apiGroup is not present

			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMSnapshotErrMissingAPIGroup {
					Expect(v.Expression).To(Equal("request.operation != 'CREATE' || has(object.spec.source.apiGroup)"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "VAP should have missing apiGroup validation")
		})

		It("should cover the same cases as webhook - invalid apiGroup", func() {
			// This test verifies that the VAP has equivalent logic to the webhook test
			// "should reject invalid apiGroup" from vmsnapshot_test.go:244
			//
			// Webhook logic (vmsnapshot.go:90-109):
			//   switch *vmSnapshot.Spec.Source.APIGroup {
			//   case core.GroupName:  // "kubevirt.io"
			//     ...
			//   default:
			//     reject with "invalid apiGroup"
			//   }

			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMSnapshotErrInvalidAPIGroup {
					Expect(v.Expression).To(ContainSubstring("kubevirt.io"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "VAP should have invalid apiGroup validation")
		})

		It("should cover the same cases as webhook - invalid kind", func() {
			// This test verifies that the VAP has equivalent logic to the webhook test
			// "should reject invalid kind" from vmsnapshot_test.go:224
			//
			// Webhook logic (vmsnapshot.go:92-100):
			//   if vmSnapshot.Spec.Source.Kind != "VirtualMachine" {
			//     reject with "invalid kind"
			//   }

			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMSnapshotErrInvalidKind {
					Expect(v.Expression).To(ContainSubstring("VirtualMachine"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "VAP should have invalid kind validation")
		})

		It("should cover the same cases as webhook - spec immutability", func() {
			// This test verifies that the VAP has equivalent logic to the webhook test
			// "should reject spec update" from vmsnapshot_test.go:138
			//
			// Webhook logic (vmsnapshot.go:118-126):
			//   if !equality.Semantic.DeepEqual(prevObj.Spec, vmSnapshot.Spec) {
			//     reject with "spec in immutable after creation"
			//   }

			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMSnapshotErrSpecImmutable {
					Expect(v.Expression).To(ContainSubstring("UPDATE"))
					Expect(v.Expression).To(ContainSubstring("object.spec == oldObject.spec"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "VAP should have spec immutability validation")
		})

		It("should cover the same cases as webhook - feature gate check", func() {
			// This test verifies that the VAP has equivalent logic to the webhook test
			// "should reject anything" when feature gate disabled from vmsnapshot_test.go:54
			//
			// Webhook logic (vmsnapshot.go:62-64):
			//   if ar.Request.Operation == admissionv1.Create && !admitter.Config.SnapshotEnabled() {
			//     return error("snapshot feature gate not enabled")
			//   }

			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMSnapshotErrFeatureGateDisabled {
					Expect(v.Expression).To(ContainSubstring("CREATE"))
					Expect(v.Expression).To(ContainSubstring("featureGates"))
					Expect(v.Expression).To(ContainSubstring("Snapshot"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "VAP should have feature gate validation")
		})
	})
})
