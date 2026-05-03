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

var _ = Describe("VirtualMachineInstanceMigration VAP equivalence", func() {
	const namespace = "kubevirt"

	var (
		vap     *admissionregistrationv1.ValidatingAdmissionPolicy
		binding *admissionregistrationv1.ValidatingAdmissionPolicyBinding
	)

	BeforeEach(func() {
		vap = components.NewVMIMigrationValidatingAdmissionPolicy(namespace)
		binding = components.NewVMIMigrationValidatingAdmissionPolicyBinding(namespace)
	})

	Context("VAP structure validation", func() {
		It("should have correct metadata", func() {
			Expect(vap.Name).To(Equal("vmimigration-policy.kubevirt.io"))
			Expect(binding.Spec.PolicyName).To(Equal(vap.Name))
		})

		It("should target VirtualMachineInstanceMigration resources", func() {
			Expect(vap.Spec.MatchConstraints.ResourceRules).To(HaveLen(1))
			rule := vap.Spec.MatchConstraints.ResourceRules[0]
			Expect(rule.Rule.APIGroups).To(ContainElement("kubevirt.io"))
			Expect(rule.Rule.Resources).To(ContainElement("virtualmachineinstancemigrations"))
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
		It("should have 5 validations migrated from webhook", func() {
			// Total validations in webhooks: 11 (split across CREATE and UPDATE)
			// Migrated to VAP: 5
			// Remaining in webhook: 6 (require API calls or UserInfo)
			Expect(vap.Spec.Validations).To(HaveLen(5))
		})

		It("should cover VMI name required validation", func() {
			// Webhook: migration-create-admitter.go:210-221
			// spec.vmiName must not be empty
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMIMigrationErrVMINameRequired {
					Expect(v.Expression).To(ContainSubstring("object.spec.vmiName"))
					Expect(v.Expression).To(ContainSubstring("CREATE"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should cover migration priority feature gate validation", func() {
			// Webhook: migration-create-admitter.go:112-121
			// If priority is set, MigrationPriorityQueue feature gate must be enabled
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMIMigrationErrPriorityFeatureGateDisabled {
					Expect(v.Expression).To(ContainSubstring("object.spec.priority"))
					Expect(v.Expression).To(ContainSubstring("MigrationPriorityQueue"))
					Expect(v.Expression).To(ContainSubstring("featureGates"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should cover decentralized migration feature gate validation", func() {
			// Webhook: migration-create-admitter.go:160-168
			// If sendTo or receive is set, DecentralizedLiveMigration feature gate must be enabled
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMIMigrationErrDecentralizedFeatureGateDisabled {
					Expect(v.Expression).To(ContainSubstring("object.spec.sendTo"))
					Expect(v.Expression).To(ContainSubstring("object.spec.receive"))
					Expect(v.Expression).To(ContainSubstring("DecentralizedLiveMigration"))
					Expect(v.Expression).To(ContainSubstring("featureGates"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should cover migration ID match validation", func() {
			// Webhook: migration-create-admitter.go:170-175
			// If both sendTo and receive are set, migrationIDs must match
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMIMigrationErrMigrationIDMismatch {
					Expect(v.Expression).To(ContainSubstring("object.spec.sendTo.migrationID"))
					Expect(v.Expression).To(ContainSubstring("object.spec.receive.migrationID"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should cover spec immutability validation", func() {
			// Webhook: migration-update-admitter.go:78-85
			// spec cannot change on UPDATE
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMIMigrationErrSpecImmutable {
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
		It("should document the priority permission check remains in webhook", func() {
			// Webhook: migration-create-admitter.go:123-131
			//
			// If priority is set, only virt-controller service account can set it.
			// This check validates:
			// - ar.Request.UserInfo.Username matches virt-controller service account
			//
			// CEL cannot access UserInfo (the authenticated user making the request).
			// The request.userInfo field is not available in CEL expressions for
			// ValidatingAdmissionPolicy.
			//
			// This authorization check MUST remain in the webhook.
		})

		It("should document the VMI exists check remains in webhook", func() {
			// Webhook: migration-create-admitter.go:134-140
			//
			// Validates that the referenced VMI exists:
			// vmi, err := virtClient.VirtualMachineInstances(namespace).Get(...)
			//
			// This requires an API call to fetch the VMI object.
			// CEL cannot make API calls to check if resources exist.
			//
			// This validation MUST remain in the webhook.
		})

		It("should document the VMI finalized check remains in webhook", func() {
			// Webhook: migration-create-admitter.go:143-145
			//
			// Validates that the VMI is not in a finalized state:
			// if vmi.IsFinal() { reject }
			//
			// This requires:
			// 1. Fetching the VMI object (API call)
			// 2. Calling vmi.IsFinal() which checks vmi.Status.Phase
			//
			// CEL cannot make API calls or access other resources.
			//
			// This validation MUST remain in the webhook.
		})

		It("should document the VMI migratable check remains in webhook", func() {
			// Webhook: migration-create-admitter.go:148-151
			// Function: isMigratable() - lines 59-72
			//
			// Validates that the VMI can be migrated by checking:
			// 1. VMI status conditions for VirtualMachineInstanceIsMigratable
			// 2. If condition is False, check the reason
			// 3. Special handling for decentralized migrations with non-migratable disks
			//
			// This requires:
			// - Fetching the VMI object (API call)
			// - Iterating over vmi.Status.Conditions
			// - Complex conditional logic based on condition.Reason
			// - Cross-referencing with migration.IsDecentralized()
			//
			// CEL cannot:
			// - Make API calls to fetch other resources
			// - Perform complex conditional logic on external resource status
			//
			// This validation MUST remain in the webhook.
		})

		It("should document the migration conflict check remains in webhook", func() {
			// Webhook: migration-create-admitter.go:155-158
			// Function: ensureNoMigrationConflict() - lines 74-95
			//
			// Validates that no other migration is in progress for the same VMI:
			// 1. Lists all migrations with label selector matching VMI name
			// 2. Checks if any migration is not in Succeeded or Failed state
			// 3. Rejects if an in-flight migration is found
			//
			// This requires:
			// - API call to list VirtualMachineInstanceMigrations
			// - Label selector query construction
			// - Iteration over results
			// - Status phase checking
			//
			// CEL cannot:
			// - List other resources
			// - Perform label selector queries
			// - Access other objects in the cluster
			//
			// This validation MUST remain in the webhook.
		})

		It("should document the selector label immutability check remains in webhook", func() {
			// Webhook: migration-update-admitter.go:88-91
			// Function: ensureSelectorLabelSafe() - lines 38-64
			//
			// Validates that the migration selector label cannot be changed
			// on an in-flight migration:
			// 1. Checks if migration is not Succeeded or Failed
			// 2. Validates selector label hasn't been removed or modified
			//
			// This validation logic is complex:
			// - Conditional check based on status.phase (not in spec)
			// - Compares old and new labels
			// - Multiple edge cases (label added, removed, changed)
			//
			// While CEL could theoretically check labels, the conditional
			// logic based on status.phase makes this complex. The webhook
			// provides clear, maintainable logic for this edge case.
			//
			// This validation should remain in the webhook for clarity
			// and maintainability.
		})
	})

	Context("Validation split summary", func() {
		It("should have correct validation distribution", func() {
			// Total validations in webhooks:     11
			//
			// Migrated to VAP (CEL):              5
			//   1. VMI name required
			//   2. Migration priority feature gate
			//   3. Decentralized migration feature gate
			//   4. Migration ID match
			//   5. Spec immutability
			//
			// Remaining in webhook:               6
			//   1. Priority permission check (requires UserInfo)
			//   2. VMI exists check (requires API call)
			//   3. VMI finalized check (requires API call)
			//   4. VMI migratable check (requires API call + complex logic)
			//   5. Migration conflict check (requires list API call)
			//   6. Selector label immutability (complex status-dependent logic)

			Expect(vap.Spec.Validations).To(HaveLen(5), "VAP should have 5 CEL validations")
		})

		It("should document that webhook remains critical for migration validation", func() {
			// Like VirtualMachineRestore and VirtualMachineInstanceReplicaSet,
			// VirtualMachineInstanceMigration has significant validation logic
			// that requires API calls and access to other resources.
			//
			// Migration validation is inherently complex because it must:
			// - Verify the target VMI exists and is in a valid state
			// - Check VMI conditions and status
			// - Ensure no conflicts with other in-flight migrations
			// - Enforce permissions based on the requesting user
			//
			// These requirements cannot be satisfied by CEL alone.
			//
			// The VAP provides ~45% coverage (5/11 validations), which is
			// valuable for early rejection of malformed requests, but the
			// webhook remains critical for comprehensive migration validation.
			//
			// Therefore, the webhook cannot be deprecated for
			// VirtualMachineInstanceMigration.
		})
	})
})
