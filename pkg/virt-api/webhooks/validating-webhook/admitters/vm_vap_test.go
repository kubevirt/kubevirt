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

var _ = Describe("VirtualMachine VAP equivalence", func() {
	const namespace = "kubevirt"

	var (
		vap     *admissionregistrationv1.ValidatingAdmissionPolicy
		binding *admissionregistrationv1.ValidatingAdmissionPolicyBinding
	)

	BeforeEach(func() {
		vap = components.NewVMValidatingAdmissionPolicy(namespace)
		binding = components.NewVMValidatingAdmissionPolicyBinding(namespace)
	})

	Context("VAP structure validation", func() {
		It("should have correct metadata", func() {
			Expect(vap.Name).To(Equal("vm-policy.kubevirt.io"))
			Expect(binding.Spec.PolicyName).To(Equal(vap.Name))
		})

		It("should target VirtualMachine resources", func() {
			Expect(vap.Spec.MatchConstraints.ResourceRules).To(HaveLen(1))
			rule := vap.Spec.MatchConstraints.ResourceRules[0]
			Expect(rule.Rule.APIGroups).To(ContainElement("kubevirt.io"))
			Expect(rule.Rule.Resources).To(ContainElement("virtualmachines"))
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
		It("should have 8 validations migrated from webhook", func() {
			// Total validations in webhook: ~18+
			// Migrated to VAP: 8
			// Remaining in webhook: ~10+ (require API calls, complex logic, or runtime checks)
			Expect(vap.Spec.Validations).To(HaveLen(8))
		})

		It("should cover template required validation", func() {
			// Webhook: vms-admitter.go:212-218
			// spec.template must be present
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMErrTemplateRequired {
					Expect(v.Expression).To(ContainSubstring("object.spec.template"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should cover running/runStrategy mutual exclusivity validation", func() {
			// Webhook: vms-admitter.go:231-237
			// Running and RunStrategy are mutually exclusive
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMErrRunningAndRunStrategyMutuallyExclusive {
					Expect(v.Expression).To(ContainSubstring("object.spec.running"))
					Expect(v.Expression).To(ContainSubstring("object.spec.runStrategy"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should cover runStrategy required validation", func() {
			// Webhook: vms-admitter.go:239-245
			// One of Running or RunStrategy must be specified
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMErrRunStrategyRequired {
					Expect(v.Expression).To(ContainSubstring("object.spec.running"))
					Expect(v.Expression).To(ContainSubstring("object.spec.runStrategy"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should cover runStrategy valid values validation", func() {
			// Webhook: vms-admitter.go:259-272
			// RunStrategy must be one of the valid values
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMErrRunStrategyInvalid {
					Expect(v.Expression).To(ContainSubstring("object.spec.runStrategy"))
					Expect(v.Expression).To(ContainSubstring("Halted"))
					Expect(v.Expression).To(ContainSubstring("Manual"))
					Expect(v.Expression).To(ContainSubstring("Always"))
					Expect(v.Expression).To(ContainSubstring("RerunOnFailure"))
					Expect(v.Expression).To(ContainSubstring("Once"))
					Expect(v.Expression).To(ContainSubstring("WaitAsReceiver"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should cover WaitAsReceiver feature gate validation", func() {
			// Webhook: vms-admitter.go:248-257
			// WaitAsReceiver requires DecentralizedLiveMigration feature gate
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMErrWaitAsReceiverFeatureGateDisabled {
					Expect(v.Expression).To(ContainSubstring("WaitAsReceiver"))
					Expect(v.Expression).To(ContainSubstring("DecentralizedLiveMigration"))
					Expect(v.Expression).To(ContainSubstring("featureGates"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should cover DataVolumeTemplate name required validation", func() {
			// Webhook: data-volume-template.go:103-109
			// DataVolume name must not be empty
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMErrDataVolumeNameRequired {
					Expect(v.Expression).To(ContainSubstring("dataVolumeTemplates"))
					Expect(v.Expression).To(ContainSubstring("metadata.name"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should cover DataVolumeTemplate PVC/Storage validation", func() {
			// Webhook: data-volume-template.go:110-123
			// DataVolume must have either PVC or Storage (not both)
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMErrDataVolumePVCOrStorageRequired {
					Expect(v.Expression).To(ContainSubstring("dataVolumeTemplates"))
					Expect(v.Expression).To(ContainSubstring("spec.pvc"))
					Expect(v.Expression).To(ContainSubstring("spec.storage"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should cover CPU sockets validation", func() {
			// Webhook: vms-admitter.go:301-309
			// CPU sockets must not exceed maxSockets
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMErrCPUSocketsExceedMaxSockets {
					Expect(v.Expression).To(ContainSubstring("domain.cpu.sockets"))
					Expect(v.Expression).To(ContainSubstring("maxSockets"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})
	})

	Context("Webhook-only validations (cannot migrate to CEL)", func() {
		It("should document that VMI metadata validation remains in webhook", func() {
			// Webhook: vms-admitter.go:220
			// Function: ValidateVirtualMachineInstanceMetadata()
			//
			// Validates:
			// - Reserved labels (kubevirt.io/* prefix) can only be set by KubeVirt service accounts
			// - Requires: ar.Request.UserInfo.Username validation
			//
			// CEL cannot access request.userInfo, so this authorization check
			// MUST remain in the webhook.
		})

		It("should document that VMI spec validation remains in webhook", func() {
			// Webhook: vms-admitter.go:221
			// Function: ValidateVirtualMachineInstanceSpec()
			//
			// This is the same ~2100 lines of complex validation documented
			// in the VirtualMachineInstance VAP test file.
			//
			// VirtualMachine templates a VirtualMachineInstance, so all VMI
			// validation logic applies here. See vmi_vap_test.go for the
			// comprehensive documentation of why VMI spec validation cannot
			// be migrated to CEL.
			//
			// This validation MUST remain in the webhook.
		})

		It("should document that DataVolumeTemplate reference validation remains in webhook", func() {
			// Webhook: data-volume-template.go:82-97
			// Function: ValidateDataVolumeTemplate()
			//
			// Validates that each DataVolumeTemplate is referenced in
			// spec.template.spec.volumes:
			// - Iterates over all volumes
			// - Checks if volume.volumeSource.persistentVolumeClaim.claimName
			//   or volume.volumeSource.dataVolume.name matches dataVolume.name
			//
			// While CEL could theoretically check this, it would require:
			// - Nested iteration over dataVolumeTemplates and volumes
			// - Complex conditional logic for different volume source types
			// - The expression would be very long and hard to maintain
			//
			// This validation is better suited for the webhook where the
			// logic is clear and maintainable.
		})

		It("should document that DataVolumeTemplate namespace validation remains in webhook", func() {
			// Webhook: data-volume-template.go:38-68
			// Function: validateVirtualMachineDataVolumeTemplateNamespace()
			//
			// Validates:
			// 1. Only validates on CREATE or UPDATE if dataVolumeTemplates changed
			// 2. Requires comparing with oldObject.spec.dataVolumeTemplates
			// 3. Validates embedded DataVolume namespace matches VM namespace
			//
			// This requires:
			// - oldObject comparison (CEL has oldObject but the logic is complex)
			// - Conditional validation based on whether DVTs changed
			// - Access to both ar.Namespace and vm.Namespace
			//
			// While some of this could be done in CEL, the webhook provides
			// clearer logic for this edge case validation.
		})

		It("should document that volume requests validation remains in webhook", func() {
			// Webhook: vms-admitter.go:312-460
			// Function: validateVolumeRequests()
			//
			// Validates hotplug volume requests in status.volumeRequests:
			// 1. Fetches VMI if VM is active (API call)
			// 2. Validates Add/Remove volume request structure
			// 3. Checks for duplicate requests
			// 4. Validates volume configuration
			// 5. Checks for conflicts with existing VM/VMI volumes
			// 6. Simulates applying changes to VMI spec
			// 7. Validates VMI is not migrating (requires VMI status check)
			//
			// This requires:
			// - API call to fetch VirtualMachineInstance
			// - Checking vmi.Status.Ready and vmi.DeletionTimestamp
			// - Complex volume conflict detection
			// - Migration status check (migrationutil.IsMigrating)
			// - Simulating volume changes and re-validating VMI spec
			//
			// CEL cannot:
			// - Make API calls to fetch other resources
			// - Access informers
			// - Perform complex multi-step validation simulations
			//
			// This validation MUST remain in the webhook.
		})

		It("should document that snapshot status validation remains in webhook", func() {
			// Webhook: vm-storage-status.go:61-90
			// Function: validateSnapshotStatus()
			//
			// Validates during UPDATE when status.snapshotInProgress is set:
			// 1. Fetches oldVM from ar.OldObject.Raw
			// 2. Compares old and new volumes (prevents volume changes during snapshot)
			// 3. Compares old and new running spec (prevents running state changes)
			//
			// This requires:
			// - Unmarshaling oldObject
			// - Deep comparison of volume arrays
			// - Conditional logic based on status field
			//
			// While CEL has oldObject, the deep semantic comparison of
			// volume arrays and running spec is complex. The webhook
			// provides clear, maintainable logic using equality.Semantic.DeepEqual.
			//
			// This validation should remain in the webhook.
		})

		It("should document that restore status validation remains in webhook", func() {
			// Webhook: vm-storage-status.go:33-59
			// Function: validateRestoreStatus()
			//
			// Validates during UPDATE when status.restoreInProgress is set:
			// 1. Fetches oldVM from ar.OldObject.Raw
			// 2. Compares old and new spec
			// 3. If spec changed, compares old and new RunStrategy
			// 4. Prevents RunStrategy changes during restore
			//
			// This requires:
			// - Unmarshaling oldObject
			// - Deep comparison of entire spec
			// - Extracting RunStrategy (which could be Running or RunStrategy field)
			// - Conditional logic based on status field
			//
			// While CEL could check spec equality, the RunStrategy extraction
			// logic (handling both Running and RunStrategy fields) adds complexity.
			// The webhook provides clearer logic.
			//
			// This validation should remain in the webhook.
		})

		It("should document that network validation remains in webhook", func() {
			// Webhook: vms-admitter.go:143-146
			// Function: netValidator.ValidateCreation()
			//
			// Delegates to netadmitter.NewValidator which performs:
			// - Network interface configuration validation
			// - Binding method validation
			// - Feature gate checks for network features
			// - Complex cross-field validation
			//
			// This is a complex external validator that would require
			// significant effort to rewrite in CEL.
			//
			// This validation MUST remain in the webhook.
		})

		It("should document that instancetype/preference validation remains in webhook", func() {
			// Webhook: vms-admitter.go:116-133
			// Functions: ApplyToVM(), Check()
			//
			// Validates:
			// 1. Fetches instancetype resource if referenced (API call)
			// 2. Fetches preference resource if referenced (API call)
			// 3. Applies instancetype/preference to VM spec
			// 4. Checks for conflicts between VM spec and preference requirements
			//
			// This requires:
			// - API calls to fetch instancetype/preference resources
			// - Complex merge logic
			// - Conflict detection
			//
			// CEL cannot make API calls or perform complex merge operations.
			//
			// This validation MUST remain in the webhook.
		})

		It("should document that live update memory validation remains in webhook", func() {
			// Webhook: vms-admitter.go:287-294
			// Function: memory.ValidateLiveUpdateMemory()
			//
			// Validates memory hotplug configuration:
			// - Checks guest memory vs maxGuest
			// - Validates memory alignment requirements
			// - Complex validation logic in external package
			//
			// This delegates to an external validation function with
			// complex logic that would be difficult to express in CEL.
			//
			// This validation should remain in the webhook.
		})

		It("should document that feature gate validation remains in webhook", func() {
			// Webhook: vms-admitter.go:138-141
			// Function: featuregate.ValidateFeatureGates()
			//
			// Validates feature gates for VMI spec:
			// - Checks if features used in VMI spec are enabled in cluster config
			// - Complex cross-referencing between spec and cluster config
			// - Multiple feature gates to check across the entire VMI spec
			//
			// While the VAP checks specific feature gates (like WaitAsReceiver),
			// the comprehensive feature gate validation for all VMI features
			// requires complex logic that is better suited for the webhook.
			//
			// This validation should remain in the webhook.
		})
	})

	Context("Validation split summary", func() {
		It("should have moderate VAP coverage with webhook handling complex validation", func() {
			// Total validation categories in webhooks: ~18+
			//
			// Migrated to VAP (CEL):              8
			//   1. Template required
			//   2. Running/RunStrategy mutual exclusivity
			//   3. RunStrategy required
			//   4. RunStrategy valid values
			//   5. WaitAsReceiver feature gate check
			//   6. DataVolumeTemplate name required
			//   7. DataVolumeTemplate PVC/Storage mutual exclusivity
			//   8. CPU sockets validation
			//
			// Remaining in webhook:               ~10+
			//   1. VMI metadata validation (requires UserInfo)
			//   2. VMI spec validation (~2100 lines of complex validation)
			//   3. DataVolumeTemplate reference validation (complex iteration)
			//   4. DataVolumeTemplate namespace validation (requires oldObject)
			//   5. Volume requests validation (requires API calls)
			//   6. Snapshot status validation (requires oldObject comparison)
			//   7. Restore status validation (requires oldObject comparison)
			//   8. Network validation (complex external validator)
			//   9. Instancetype/Preference (requires API calls)
			//   10. Live update memory validation (complex logic)
			//   11. Feature gate validation (comprehensive checks)
			//
			// Coverage: ~44% (8 out of ~18 validation categories)
			//
			// The VAP provides valuable early validation for common
			// structural issues, but the webhook remains critical for
			// comprehensive VM validation.

			Expect(vap.Spec.Validations).To(HaveLen(8), "VAP should have 8 CEL validations")
		})

		It("should document that the webhook will NEVER be deprecated for VM", func() {
			// VirtualMachine is the primary user-facing resource in KubeVirt.
			// It templates VirtualMachineInstance and includes additional
			// complex validation logic for:
			//
			// - VMI spec validation (~2100 lines) - see VMI VAP for details
			// - Volume hotplug operations (requires API calls, runtime checks)
			// - Snapshot/Restore state protection (requires oldObject comparison)
			// - DataVolume template validation (requires API calls)
			// - Network configuration (complex external validator)
			// - Instancetype/Preference application (requires API calls)
			// - Live update features (complex validation logic)
			//
			// The complexity is not accidental - VirtualMachine validation
			// inherits all the complexity of VirtualMachineInstance plus
			// additional runtime state management and resource lifecycle
			// validation.
			//
			// While the VAP achieves ~44% coverage (8/~18 validations),
			// it only covers basic structural checks. All meaningful
			// validation that requires runtime state, API calls, or
			// complex logic remains in the webhook.
			//
			// The webhook for VirtualMachine will NEVER be deprecated.
			// It is a critical, permanent component of KubeVirt's admission control.
		})
	})
})
