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

var _ = Describe("VirtualMachineInstance VAP equivalence", func() {
	var (
		vap     *admissionregistrationv1.ValidatingAdmissionPolicy
		binding *admissionregistrationv1.ValidatingAdmissionPolicyBinding
	)

	BeforeEach(func() {
		vap = components.NewVMIValidatingAdmissionPolicy()
		binding = components.NewVMIValidatingAdmissionPolicyBinding()
	})

	Context("VAP structure validation", func() {
		It("should have correct metadata", func() {
			Expect(vap.Name).To(Equal("vmi-policy.kubevirt.io"))
			Expect(binding.Spec.PolicyName).To(Equal(vap.Name))
		})

		It("should target VirtualMachineInstance resources", func() {
			Expect(vap.Spec.MatchConstraints.ResourceRules).To(HaveLen(1))
			rule := vap.Spec.MatchConstraints.ResourceRules[0]
			Expect(rule.Rule.APIGroups).To(ContainElement("kubevirt.io"))
			Expect(rule.Rule.Resources).To(ContainElement("virtualmachineinstances"))
		})

		It("should only validate UPDATE operations", func() {
			// CREATE validation (~2100 lines) cannot be migrated to CEL
			rule := vap.Spec.MatchConstraints.ResourceRules[0]
			Expect(rule.Operations).To(HaveLen(1))
			Expect(rule.Operations[0]).To(Equal(admissionregistrationv1.Update))
			Expect(rule.Operations).NotTo(ContainElement(admissionregistrationv1.Create))
		})

		It("should not have param kind since webhook handles all complex validation", func() {
			Expect(vap.Spec.ParamKind).To(BeNil())
		})

		It("should have minimal validations (only basic immutability)", func() {
			// VMI has extremely complex validation that cannot be migrated to CEL
			// Only a basic immutability check is provided in the VAP
			Expect(vap.Spec.Validations).To(HaveLen(1))
		})
	})

	Context("Minimal validation coverage", func() {
		It("should cover basic spec immutability for UPDATE", func() {
			// Webhook: vmi-update-admitter.go (multiple checks)
			// VAP: Basic structural immutability check
			found := false
			for _, v := range vap.Spec.Validations {
				if v.Message == components.VMIErrSpecImmutable {
					Expect(v.Expression).To(Equal("object.spec == oldObject.spec"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})
	})

	Context("CREATE validations (cannot migrate to CEL)", func() {
		It("should document that ~2100 lines of CREATE validation remain in webhook", func() {
			// Webhook: vmi-create-admitter.go:97-2204
			// Function: admitVMICreate() and ValidateVirtualMachineInstanceSpec()
			//
			// The CREATE validation for VirtualMachineInstance is extraordinarily complex
			// and cannot be migrated to CEL. Key validation categories include:
			//
			// 1. Domain Specification Validation (~800 lines):
			//    - CPU configuration (cores, sockets, threads, model, features)
			//    - Memory configuration (guest, hugepages, overcommit)
			//    - Devices validation (disks, interfaces, GPUs, host devices)
			//    - Features validation (ACPI, APIC, hyperv, KVM, SMM)
			//    - Firmware validation (bootloader, BIOS, UEFI, secure boot)
			//    - Clock and timer validation
			//    - Resource requirements (requests, limits, overcommit)
			//    - I/O threads configuration
			//    - Launch security (SEV, TDX)
			//
			// 2. Volume Specification Validation (~500 lines):
			//    - All volume types: containerDisk, ephemeral, cloudInitNoCloud,
			//      cloudInitConfigDrive, persistentVolumeClaim, dataVolume,
			//      emptyDisk, hostDisk, configMap, secret, downwardAPI,
			//      serviceAccount, sysprep, memoryDump
			//    - Volume source validation
			//    - Access mode validation
			//    - Size and capacity validation
			//    - Mutually exclusive volume types
			//    - Volume name uniqueness
			//    - Disk bus type validation
			//
			// 3. Network Specification Validation (~400 lines):
			//    - Network interface configuration
			//    - Network source validation (pod, multus)
			//    - Binding method validation (bridge, masquerade, SRIOV, etc.)
			//    - MAC address validation
			//    - PCI address validation
			//    - Network feature validation (multiqueue, virtio)
			//    - Port validation
			//
			// 4. Affinity and Scheduling (~200 lines):
			//    - Node affinity validation
			//    - Pod affinity/anti-affinity
			//    - Topology spread constraints
			//    - Node selector validation
			//    - Tolerations validation
			//
			// 5. Feature Gates and Compatibility (~200 lines):
			//    - Feature gate checks for experimental features
			//    - Architecture compatibility
			//    - Machine type validation
			//    - Workload type validation
			//    - Instance type validation
			//    - Preference validation
			//
			// 6. Cross-field Dependencies (~100 lines):
			//    - Interface-network mapping validation
			//    - Disk-volume mapping validation
			//    - GPU-PCI device conflicts
			//    - Memory-hugepages consistency
			//    - CPU-NUMA alignment
			//
			// Why CEL cannot handle this:
			// - Complex conditional logic based on feature enablement
			// - Deep nested structure traversal and validation
			// - Cross-field dependency checking across multiple levels
			// - Regex pattern matching for identifiers and formats
			// - Mutually exclusive field combinations
			// - Resource quantity parsing and comparison
			// - PCI address format validation
			// - MAC address format validation
			// - Complex string validation (hostnames, paths, labels)
			// - Feature gate lookups requiring cluster configuration
			// - Architecture-specific validation rules
			//
			// The sheer volume and complexity of these validations makes
			// comprehensive CEL migration infeasible. Each category above
			// would require dozens of CEL expressions, many of which would
			// hit CEL's expression complexity limits or require features
			// that CEL does not support (regex, complex string parsing,
			// recursive validation).
			//
			// This validation MUST remain in the webhook.
		})
	})

	Context("UPDATE validations (cannot migrate to CEL)", func() {
		It("should document that authorization-based validation remains in webhook", func() {
			// Webhook: vmi-update-admitter.go:71-88
			// Function: ValidateVMIUpdate()
			//
			// UPDATE validation for VirtualMachineInstance requires authorization
			// checks that CEL cannot perform:
			//
			// 1. Service Account Authorization:
			//    - Only KubeVirt service accounts can modify VMI spec
			//    - Spec changes are only allowed via sub-resources or by controller
			//    - Checks: ar.Request.UserInfo.Username against known service accounts
			//      - system:serviceaccount:kubevirt:virt-controller
			//      - system:serviceaccount:kubevirt:virt-api
			//
			// 2. Node Restriction for virt-handler:
			//    - virt-handler can only update VMIs it owns (running on its node)
			//    - Validates: ar.Request.UserInfo.Username matches virt-handler SA
			//    - Checks: VMI's status.nodeName matches the requesting node
			//
			// 3. Reserved Label Protection:
			//    - Labels with kubevirt.io/* prefix cannot be modified by users
			//    - Only KubeVirt components can add/modify/remove these labels
			//    - Requires: UserInfo.Username validation
			//
			// 4. Hotplug Validation:
			//    - CPU hotplug validation (requires feature gate + capacity checks)
			//    - Memory hotplug validation (requires feature gate + size validation)
			//    - Volume hotplug validation (requires informer access to check existing volumes)
			//    - Each requires complex state checks and potentially API calls
			//
			// CEL limitations:
			// - Cannot access request.userInfo (the authenticated user)
			// - Cannot make API calls to fetch VMI status or node information
			// - Cannot access informers to check current VMI state
			// - Cannot perform substring matching on service account names
			// - Cannot validate reserved label patterns with complex logic
			//
			// This validation MUST remain in the webhook.
		})

		It("should document that hotplug validation remains in webhook", func() {
			// Webhook: vmi-update-admitter.go:90-234
			// Functions: validateVolumeRequests(), validateCPUMemoryHotplug()
			//
			// Hotplug validation is complex and requires runtime state:
			//
			// 1. Volume Hotplug (lines 123-189):
			//    - Validates volume hotplug requests in status.volumeStatus
			//    - Checks if volumes are being added or removed
			//    - Validates hotplug feature gate is enabled
			//    - Checks volume types are hotplug-compatible
			//    - Validates no duplicate volume names
			//    - Requires informer access to current VMI volumes
			//
			// 2. CPU Hotplug (lines 191-210):
			//    - Validates CPU hotplug requests
			//    - Checks CPUHotplug feature gate is enabled
			//    - Validates new CPU count vs. old CPU count
			//    - Ensures CPU count doesn't exceed maxSockets
			//    - Validates CPU topology (sockets, cores, threads)
			//
			// 3. Memory Hotplug (lines 212-234):
			//    - Validates memory hotplug requests
			//    - Checks MemoryHotplug feature gate is enabled
			//    - Validates new memory size vs. old memory size
			//    - Ensures memory doesn't exceed maxGuest
			//    - Validates memory alignment requirements
			//
			// Why CEL cannot handle this:
			// - Requires comparing current VMI state from informer
			// - Needs feature gate checks (would need params, but auth check conflicts)
			// - Complex capacity and topology calculations
			// - Multi-level nested validation
			// - State-dependent validation (what volumes are currently attached)
			//
			// This validation MUST remain in the webhook.
		})

		It("should document that mutation detection remains in webhook", func() {
			// Webhook: vmi-update-admitter.go:50-69
			// Function: detectVMIMutation()
			//
			// The webhook detects unauthorized mutations by:
			// 1. Comparing oldVMI.Spec with newVMI.Spec
			// 2. Allowing changes only from authorized service accounts
			// 3. Rejecting all other spec mutations
			//
			// While the VAP provides a basic immutability check (object.spec == oldObject.spec),
			// the webhook provides more nuanced handling:
			// - Allows mutations from virt-controller for hotplug operations
			// - Allows mutations from virt-api for certain sub-resource updates
			// - Provides detailed error messages identifying which fields changed
			// - Validates that allowed mutations are within permitted boundaries
			//
			// The VAP's basic check serves as a safety net for unauthorized users,
			// but the webhook's authorization-aware logic is critical for
			// proper KubeVirt operation.
			//
			// This validation MUST remain in the webhook.
		})
	})

	Context("Validation split summary", func() {
		It("should have minimal VAP coverage with webhook handling comprehensive validation", func() {
			// Total validation complexity in webhooks: ~2300 lines
			//
			// Migrated to VAP (CEL):              1
			//   1. Basic spec immutability (UPDATE only, no authorization)
			//
			// Remaining in webhook:               ~2300 lines
			//   CREATE validations (~2100 lines):
			//     - Domain spec validation
			//     - Volume spec validation
			//     - Network spec validation
			//     - Affinity and scheduling validation
			//     - Feature gates and compatibility
			//     - Cross-field dependencies
			//
			//   UPDATE validations (~200 lines):
			//     - Service account authorization
			//     - Node restriction
			//     - Reserved label protection
			//     - Hotplug validation (CPU, memory, volumes)
			//     - Mutation detection
			//
			// Coverage: ~0% (1 basic check out of ~2300 lines)
			//
			// The VAP provides only a minimal safety check. The webhook
			// is absolutely critical and handles all meaningful validation.

			Expect(vap.Spec.Validations).To(HaveLen(1), "VAP should have only 1 basic validation")
		})

		It("should document that the webhook will NEVER be deprecated for VMI", func() {
			// VirtualMachineInstance is the core resource type in KubeVirt.
			// Its validation logic is the most complex of any KubeVirt resource:
			//
			// - ~2100 lines of CREATE validation covering every aspect of
			//   virtual machine configuration
			// - ~200 lines of UPDATE validation with authorization and hotplug
			// - Deeply nested structures with cross-field dependencies
			// - Feature gate checks requiring cluster configuration
			// - Authorization checks requiring UserInfo
			// - Runtime state checks requiring informer access
			//
			// The complexity is not accidental - it reflects the inherent
			// complexity of validating a complete virtual machine specification.
			// This level of validation cannot be expressed in CEL, which is
			// designed for simple structural checks, not comprehensive
			// domain-specific validation.
			//
			// Unlike simpler resources (VirtualMachineSnapshot,
			// VirtualMachineSnapshotContent) where VAPs achieved 100% coverage,
			// or moderately complex resources (VirtualMachineRestore,
			// VirtualMachineInstanceMigration) where VAPs achieved 45-83% coverage,
			// VirtualMachineInstance validation is fundamentally incompatible
			// with the CEL model.
			//
			// The webhook for VirtualMachineInstance will NEVER be deprecated.
			// It is a critical, permanent component of KubeVirt's admission control.
			//
			// The minimal VAP serves only as a basic guard against accidental
			// spec mutations by unauthorized users. All meaningful validation
			// occurs in the webhook.
		})
	})
})
