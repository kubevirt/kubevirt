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

var _ = Describe("VirtualMachineInstanceReplicaSet VAP equivalence", func() {
	var (
		vap     *admissionregistrationv1.ValidatingAdmissionPolicy
		binding *admissionregistrationv1.ValidatingAdmissionPolicyBinding
	)

	BeforeEach(func() {
		vap = components.NewVMIRSValidatingAdmissionPolicy()
		binding = components.NewVMIRSValidatingAdmissionPolicyBinding()
	})

	Context("VAP structure validation", func() {
		It("should have correct metadata", func() {
			Expect(vap.Name).To(Equal("vmirs-policy.kubevirt.io"))
			Expect(binding.Spec.PolicyName).To(Equal(vap.Name))
		})

		It("should target VirtualMachineInstanceReplicaSet resources", func() {
			Expect(vap.Spec.MatchConstraints.ResourceRules).To(HaveLen(1))
			rule := vap.Spec.MatchConstraints.ResourceRules[0]
			Expect(rule.Rule.APIGroups).To(ContainElement("kubevirt.io"))
			Expect(rule.Rule.Resources).To(ContainElement("virtualmachineinstancereplicasets"))
			Expect(rule.Operations).To(HaveLen(2))
			Expect(rule.Operations[0]).To(Equal(admissionregistrationv1.Create))
			Expect(rule.Operations[1]).To(Equal(admissionregistrationv1.Update))
		})

		It("should not require params since there are no config-dependent validations", func() {
			Expect(vap.Spec.ParamKind).To(BeNil())
			Expect(binding.Spec.ParamRef).To(BeNil())
		})
	})

	Context("Validation coverage", func() {
		It("should have 3 simple structural validations migrated from webhook", func() {
			// VMIRS webhook has 5 total validation categories.
			// 3 are simple structural checks migrated to VAP.
			// 2 are complex validations that must remain in webhook.
			Expect(vap.Spec.Validations).To(HaveLen(3))
		})

		It("should cover template required validation", func() {
			// Webhook: vmirs-admitter.go:84-90
			// If spec.template is nil, reject with "missing virtual machine template"
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMIRSErrTemplateRequired {
					Expect(v.Expression).To(Equal("has(object.spec.template)"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should cover selector required validation", func() {
			// Webhook: vmirs-admitter.go:93-99
			// Selector must be present and valid
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMIRSErrSelectorRequired {
					Expect(v.Expression).To(Equal("has(object.spec.selector)"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should cover selector matches labels validation", func() {
			// Webhook: vmirs-admitter.go:100-106
			// selector.Matches(labels.Set(spec.Template.ObjectMeta.Labels))
			//
			// CEL expression validates that all matchLabels in selector
			// exist in template.metadata.labels with the same values
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMIRSErrSelectorMatchesLabels {
					Expect(v.Expression).To(ContainSubstring("object.spec.selector.matchLabels"))
					Expect(v.Expression).To(ContainSubstring("object.spec.template.metadata.labels"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})
	})

	Context("Webhook-only validations (cannot migrate to CEL)", func() {
		It("should document the VMI spec validation remains in webhook", func() {
			// Webhook: vmirs-admitter.go:91
			// causes = append(causes, ValidateVirtualMachineInstanceSpec(...))
			//
			// This calls ValidateVirtualMachineInstanceSpec which is defined in
			// vmi-create-admitter.go (~2100 lines of complex validation logic).
			//
			// This function validates:
			// - Domain spec (CPU, memory, devices, features, firmware, clock, etc.)
			// - Volume specs (all volume types and their constraints)
			// - Network specs (interfaces, networks, binding methods)
			// - Affinity and scheduling constraints
			// - Resource requirements and limits
			// - Feature gates and compatibility
			// - Disk and volume matching
			// - Boot order validation
			// - And many more complex business logic rules
			//
			// This is far too complex to express in CEL and involves:
			// - Deep nested object validation
			// - Cross-field dependencies
			// - Complex conditional logic
			// - Feature gate checks
			// - Resource compatibility validation
			//
			// This validation MUST remain in the webhook.
		})

		It("should document the feature gate validation remains in webhook", func() {
			// Webhook: vmirs-admitter.go:64-69 (CREATE only)
			// causes = append(causes, featuregate.ValidateFeatureGates(devCfg.FeatureGates, &vmirs.Spec.Template.Spec)...)
			//
			// This validates that:
			// 1. Features used in VMI spec are enabled via feature gates
			// 2. Deprecated features are flagged
			// 3. Feature combinations are valid
			// 4. Feature-specific validations are enforced
			//
			// The featuregate.ValidateFeatureGates function performs complex
			// analysis of the VMI spec against the enabled feature gates.
			//
			// This cannot be expressed in CEL because:
			// - It requires introspecting the VMI spec deeply
			// - It has conditional logic based on which features are used
			// - It accesses cluster config (feature gates)
			// - It performs complex compatibility checks
			//
			// This validation MUST remain in the webhook.
		})

		It("should document that matchExpressions in selector are not fully validated", func() {
			// The VAP validates basic matchLabels selector matching.
			//
			// However, Kubernetes label selectors also support matchExpressions
			// which allow more complex matching (In, NotIn, Exists, DoesNotExist).
			//
			// Example:
			// selector:
			//   matchExpressions:
			//   - key: app
			//     operator: In
			//     values: [web, api]
			//
			// The webhook uses metav1.LabelSelectorAsSelector() which fully
			// validates and evaluates matchExpressions.
			//
			// CEL cannot easily replicate the complex label selector semantics,
			// especially matchExpressions with different operators.
			//
			// So the webhook provides full selector validation, while VAP
			// provides basic matchLabels validation for early rejection of
			// obviously incorrect configurations.
		})
	})

	Context("Validation split summary", func() {
		It("should have correct validation distribution", func() {
			// Total validation categories in webhook: 5
			//
			// Migrated to VAP (CEL):               3
			//   1. Template required
			//   2. Selector required
			//   3. Selector matches template labels (basic matchLabels only)
			//
			// Remaining in webhook:                2
			//   1. VMI spec validation (~2100 lines of complex logic)
			//   2. Feature gate validation (complex feature analysis)
			//
			// Note: The webhook also provides complete label selector validation
			// including matchExpressions, while VAP only covers basic matchLabels.

			Expect(vap.Spec.Validations).To(HaveLen(3), "VAP should have 3 basic structural validations")
		})

		It("should document that webhook remains critical for VMIRS validation", func() {
			// Unlike previous resources where most validations migrated to VAP,
			// VirtualMachineInstanceReplicaSet requires the webhook for the
			// majority of its validation logic.
			//
			// The webhook validates the entire VirtualMachineInstance spec,
			// which is the core of a VMIRS. The VAP only validates the
			// ReplicaSet-specific structural requirements (template, selector).
			//
			// This is an example where VAP provides early rejection of
			// structurally invalid objects, but the webhook performs the
			// comprehensive validation of the actual workload specification.
			//
			// Therefore, the webhook cannot be deprecated for VMIRS even after
			// VAP is deployed.
		})
	})
})
