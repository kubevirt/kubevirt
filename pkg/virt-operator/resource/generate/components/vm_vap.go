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

package components

import (
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	vmVAPName        = "vm-policy.kubevirt.io"
	vmVAPBindingName = "vm-policy-binding"

	VMErrTemplateRequired                   = "spec.template is required"
	VMErrRunningAndRunStrategyMutuallyExclusive = "spec.running and spec.runStrategy are mutually exclusive"
	VMErrRunStrategyRequired                = "spec.runStrategy must be specified (spec.running is deprecated)"
	VMErrRunStrategyInvalid                 = "spec.runStrategy must be one of: Halted, Manual, Always, RerunOnFailure, Once, or WaitAsReceiver"
	VMErrWaitAsReceiverFeatureGateDisabled  = "DecentralizedLiveMigration feature gate is not enabled in kubevirt resource"
	VMErrDataVolumeNameRequired             = "dataVolumeTemplates[*].name must not be empty"
	VMErrDataVolumePVCOrStorageRequired     = "dataVolumeTemplates[*] must have either spec.pvc or spec.storage (not both)"
	VMErrDataVolumeNotReferenced            = "dataVolumeTemplates[*] must be referenced in spec.template.spec.volumes"
	VMErrCPUSocketsExceedMaxSockets         = "spec.template.spec.domain.cpu.sockets must not exceed maxSockets"
)

// NewVMValidatingAdmissionPolicy creates a ValidatingAdmissionPolicy for VirtualMachine.
//
// IMPORTANT: VirtualMachine has complex validation logic that partially overlaps with
// VirtualMachineInstance validation. This VAP provides basic structural validation only.
// The webhook remains CRITICAL for comprehensive VM validation.
//
// What is migrated to CEL (8 validations):
// 1. Template required
// 2. Running/RunStrategy mutual exclusivity
// 3. RunStrategy required
// 4. RunStrategy valid values
// 5. WaitAsReceiver feature gate check
// 6. DataVolumeTemplate name required
// 7. DataVolumeTemplate PVC/Storage mutual exclusivity
// 8. CPU sockets validation (if rollout strategy is live update)
//
// What MUST remain in webhook:
// 1. VMI metadata validation (requires authorization for reserved labels)
// 2. VMI spec validation (~2100 lines) - see VMI VAP for details
// 3. DataVolumeTemplate reference validation (requires iteration over volumes)
// 4. Volume requests validation (requires API calls, migration checks)
// 5. Snapshot/Restore status validation (requires oldObject comparison)
// 6. DataVolume namespace validation (requires oldObject comparison)
// 7. Network validation (delegates to external validator)
// 8. Instancetype/Preference (requires API calls)
// 9. Live update memory validation (complex validation logic)
// 10. Feature gate validation for VMI spec (requires cluster config)
//
// The webhook will NEVER be deprecated for VirtualMachine due to its
// dependency on VirtualMachineInstance validation and complex runtime checks.
func NewVMValidatingAdmissionPolicy(namespace string) *admissionregistrationv1.ValidatingAdmissionPolicy {
	return &admissionregistrationv1.ValidatingAdmissionPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "ValidatingAdmissionPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: vmVAPName,
			Labels: map[string]string{
				v1.AppLabel:       "virt-api",
				v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicySpec{
			FailurePolicy: pointer.P(admissionregistrationv1.Fail),
			MatchConstraints: &admissionregistrationv1.MatchResources{
				ResourceRules: []admissionregistrationv1.NamedRuleWithOperations{
					{
						RuleWithOperations: admissionregistrationv1.RuleWithOperations{
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{"kubevirt.io"},
								APIVersions: []string{"v1", "v1alpha3"},
								Resources:   []string{"virtualmachines"},
							},
						},
					},
				},
			},
			ParamKind: &admissionregistrationv1.ParamKind{
				APIVersion: "kubevirt.io/v1",
				Kind:       "KubeVirt",
			},
			Validations: []admissionregistrationv1.Validation{
				// Template required
				{
					Expression: "has(object.spec.template)",
					Message:    VMErrTemplateRequired,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				// Running and RunStrategy are mutually exclusive
				{
					Expression: "!has(object.spec.running) || !has(object.spec.runStrategy)",
					Message:    VMErrRunningAndRunStrategyMutuallyExclusive,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				// One of Running or RunStrategy must be specified
				{
					Expression: "has(object.spec.running) || has(object.spec.runStrategy)",
					Message:    VMErrRunStrategyRequired,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				// RunStrategy must be valid
				{
					Expression: `!has(object.spec.runStrategy) || object.spec.runStrategy in ['Halted', 'Manual', 'Always', 'RerunOnFailure', 'Once', 'WaitAsReceiver']`,
					Message:    VMErrRunStrategyInvalid,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				// WaitAsReceiver requires DecentralizedLiveMigration feature gate
				{
					Expression: `!has(object.spec.runStrategy) || object.spec.runStrategy != 'WaitAsReceiver' || params.spec.configuration.developerConfiguration.featureGates.exists(fg, fg == 'DecentralizedLiveMigration')`,
					Message:    VMErrWaitAsReceiverFeatureGateDisabled,
					Reason:     pointer.P(metav1.StatusReasonForbidden),
				},
				// DataVolumeTemplate name required
				{
					Expression: "!has(object.spec.dataVolumeTemplates) || object.spec.dataVolumeTemplates.all(dv, has(dv.metadata.name) && dv.metadata.name != '')",
					Message:    VMErrDataVolumeNameRequired,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				// DataVolumeTemplate must have either PVC or Storage (not both)
				{
					Expression: `!has(object.spec.dataVolumeTemplates) || object.spec.dataVolumeTemplates.all(dv,
						(has(dv.spec.pvc) && !has(dv.spec.storage)) ||
						(!has(dv.spec.pvc) && has(dv.spec.storage))
					)`,
					Message: VMErrDataVolumePVCOrStorageRequired,
					Reason:  pointer.P(metav1.StatusReasonInvalid),
				},
				// CPU sockets validation (when live update is enabled)
				// Note: We cannot check if rollout strategy is live update in CEL,
				// so this validation is optimistic - it validates IF maxSockets is set
				{
					Expression: `!has(object.spec.template.spec.domain.cpu) ||
						!has(object.spec.template.spec.domain.cpu.maxSockets) ||
						!has(object.spec.template.spec.domain.cpu.sockets) ||
						object.spec.template.spec.domain.cpu.sockets <= object.spec.template.spec.domain.cpu.maxSockets`,
					Message: VMErrCPUSocketsExceedMaxSockets,
					Reason:  pointer.P(metav1.StatusReasonInvalid),
				},
			},
		},
	}
}

func NewVMValidatingAdmissionPolicyBinding(namespace string) *admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	return &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "ValidatingAdmissionPolicyBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: vmVAPBindingName,
			Labels: map[string]string{
				v1.AppLabel:       "virt-api",
				v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName: vmVAPName,
			ParamRef: &admissionregistrationv1.ParamRef{
				Name:      "kubevirt",
				Namespace: namespace,
			},
			ValidationActions: []admissionregistrationv1.ValidationAction{
				admissionregistrationv1.Deny,
			},
		},
	}
}
