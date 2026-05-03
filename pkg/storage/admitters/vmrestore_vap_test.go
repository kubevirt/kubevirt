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

var _ = Describe("VirtualMachineRestore VAP equivalence", func() {
	const namespace = "kubevirt"

	var (
		vap     *admissionregistrationv1.ValidatingAdmissionPolicy
		binding *admissionregistrationv1.ValidatingAdmissionPolicyBinding
	)

	BeforeEach(func() {
		vap = components.NewVMRestoreValidatingAdmissionPolicy(namespace)
		binding = components.NewVMRestoreValidatingAdmissionPolicyBinding(namespace)
	})

	Context("VAP structure validation", func() {
		It("should have correct metadata", func() {
			Expect(vap.Name).To(Equal("vmrestore-policy.snapshot.kubevirt.io"))
			Expect(binding.Spec.PolicyName).To(Equal(vap.Name))
		})

		It("should target VirtualMachineRestore resources", func() {
			Expect(vap.Spec.MatchConstraints.ResourceRules).To(HaveLen(1))
			rule := vap.Spec.MatchConstraints.ResourceRules[0]
			Expect(rule.Rule.APIGroups).To(ContainElement("snapshot.kubevirt.io"))
			Expect(rule.Rule.Resources).To(ContainElement("virtualmachinerestores"))
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

	Context("Validation coverage", func() {
		It("should have 10 CEL validations migrated from webhook", func() {
			// VirtualMachineRestore webhook has 12 total validations.
			// 10 are migrated to VAP, 2 must remain in webhook (require API calls).
			Expect(vap.Spec.Validations).To(HaveLen(10))
		})

		It("should cover feature gate validation", func() {
			// Webhook: vmrestore.go:68-70
			// CREATE only: reject if Snapshot feature gate not enabled
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMRestoreErrFeatureGateDisabled {
					Expect(v.Expression).To(ContainSubstring("CREATE"))
					Expect(v.Expression).To(ContainSubstring("featureGates"))
					Expect(v.Expression).To(ContainSubstring("Snapshot"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should cover apiGroup validations", func() {
			// Webhook: vmrestore.go:85-92 (missing), 94-134 (invalid)
			messages := []string{}
			for _, v := range vap.Spec.Validations {
				messages = append(messages, v.Message)
			}
			Expect(messages).To(ContainElement(components.VMRestoreErrMissingAPIGroup))
			Expect(messages).To(ContainElement(components.VMRestoreErrInvalidAPIGroup))
		})

		It("should cover kind validation", func() {
			// Webhook: vmrestore.go:96-125
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMRestoreErrInvalidKind {
					Expect(v.Expression).To(ContainSubstring("VirtualMachine"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should cover volume override validations", func() {
			// Webhook: vmrestore.go:270-299
			// - volumeName required (278-287)
			// - at least one field required (289-295)
			messages := []string{}
			for _, v := range vap.Spec.Validations {
				messages = append(messages, v.Message)
			}
			Expect(messages).To(ContainElement(components.VMRestoreErrVolumeOverrideVolumeName))
			Expect(messages).To(ContainElement(components.VMRestoreErrVolumeOverrideAtLeastOneField))
		})

		It("should cover volume restore policy enum validation", func() {
			// Webhook: vmrestore.go:301-326
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMRestoreErrInvalidVolumeRestorePolicy {
					Expect(v.Expression).To(ContainSubstring("InPlace"))
					Expect(v.Expression).To(ContainSubstring("RandomizeNames"))
					Expect(v.Expression).To(ContainSubstring("PrefixTargetName"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should cover volume ownership policy enum validation", func() {
			// Webhook: vmrestore.go:328-352
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMRestoreErrInvalidVolumeOwnershipPolicy {
					Expect(v.Expression).To(ContainSubstring("Vm"))
					Expect(v.Expression).To(ContainSubstring("None"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should cover basic patches validation", func() {
			// Webhook: vmrestore.go:233-268
			// VAP can validate basic path restrictions
			// Complex JSON parsing remains in webhook
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMRestoreErrInvalidPatchPath {
					Expect(v.Expression).To(ContainSubstring("/spec/"))
					Expect(v.Expression).To(ContainSubstring("/metadata/labels/"))
					Expect(v.Expression).To(ContainSubstring("/metadata/annotations/"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should cover spec immutability", func() {
			// Webhook: vmrestore.go:162-170
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMRestoreErrSpecImmutable {
					Expect(v.Expression).To(ContainSubstring("UPDATE"))
					Expect(v.Expression).To(ContainSubstring("object.spec == oldObject.spec"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})
	})

	Context("Webhook-only validations (cannot migrate to CEL)", func() {
		It("should document the restore-in-progress check remains in webhook", func() {
			// Webhook: vmrestore.go:137-153
			//
			// This validation requires VMRestoreInformer lookup to check if another
			// restore for the same target is already in progress. CEL cannot access
			// informers or perform list operations, so this validation must remain
			// in the webhook.
			//
			// The webhook iterates through existing VMRestores and checks:
			// - Same target (equality.Semantic.DeepEqual(r.Spec.Target, vmRestore.Spec.Target))
			// - Not complete (!*r.Status.Complete)
			//
			// This is a runtime state check that requires API server queries.
		})

		It("should document the backend storage check remains in webhook", func() {
			// Webhook: vmrestore.go:204-228 (validateTargetVM)
			//
			// This validation requires multiple API calls:
			// 1. Fetch VirtualMachineSnapshot
			// 2. Fetch VirtualMachineSnapshotContent
			// 3. Fetch target VirtualMachine
			// 4. Check if source VM has backend storage (persistent TPM/EFI)
			// 5. Compare source UID with target UID
			//
			// If restoring to a different VM and source has backend storage,
			// the restore is rejected.
			//
			// This complex logic involves API calls and custom business logic
			// (backendstorage.IsBackendStorageNeeded) that cannot be expressed in CEL.
		})

		It("should document that complex patch validation remains in webhook", func() {
			// Webhook: vmrestore.go:233-268 (validatePatches)
			//
			// The webhook performs complex JSON patch parsing:
			// 1. Splits patch string by comma
			// 2. Extracts key-value pairs
			// 3. Validates JSON structure (expects exactly one ":" per pair)
			// 4. Checks if "path" key targets allowed fields
			//
			// The VAP covers basic path validation (checking if patch contains
			// "/spec/", "/metadata/labels/", or "/metadata/annotations/").
			//
			// However, the detailed JSON parsing and validation logic is too
			// complex for CEL and must remain in the webhook for comprehensive
			// patch validation.
		})
	})

	Context("Validation split summary", func() {
		It("should have correct validation distribution", func() {
			// Total validations in webhook: 12
			// Migrated to VAP (CEL):        10
			//   1. Feature gate check
			//   2. APIGroup required
			//   3. APIGroup value
			//   4. Kind value
			//   5. Volume override - volumeName required
			//   6. Volume override - at least one field
			//   7. Volume restore policy enum
			//   8. Volume ownership policy enum
			//   9. Patches basic validation
			//  10. Spec immutability
			//
			// Remaining in webhook:          2
			//   1. Restore in progress check (requires informer)
			//   2. Backend storage check (requires API calls)
			//
			// Note: Complex patch parsing also remains in webhook for comprehensive validation

			Expect(vap.Spec.Validations).To(HaveLen(10), "VAP should have 10 CEL validations")
		})
	})
})
