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
	vmiVAPName        = "vmi-policy.kubevirt.io"
	vmiVAPBindingName = "vmi-policy-binding"

	VMIErrSpecImmutable = "spec is immutable except via the VirtualMachine resource or hotplug operations"
)

// NewVMIValidatingAdmissionPolicy creates a minimal ValidatingAdmissionPolicy for VirtualMachineInstance.
//
// IMPORTANT: VirtualMachineInstance has extremely complex validation logic (~2100+ lines)
// that cannot be migrated to CEL. This VAP provides only basic immutability protection.
// The webhook remains CRITICAL and REQUIRED for comprehensive VMI validation.
//
// Why comprehensive CEL validation is not feasible for VMI:
// 1. CREATE validation (~2100 lines): ValidateVirtualMachineInstanceSpec validates:
//    - Domain spec (CPU, memory, devices, features, firmware, clock, resources)
//    - Volume specs (all volume types with complex constraints)
//    - Network specs (interfaces, networks, binding methods)
//    - Affinity and scheduling constraints
//    - Feature gates and compatibility checks
//    - Cross-field dependencies throughout the spec
//    - Complex conditional logic based on feature enablement
//
// 2. UPDATE validation requires authorization (UserInfo):
//    - Only KubeVirt service accounts can modify spec (via sub-resource)
//    - Node restriction checks for virt-handler
//    - Reserved label protection
//    - Hotplug validation (CPU, memory, volumes)
//
// The webhook will NEVER be deprecated for VirtualMachineInstance.
func NewVMIValidatingAdmissionPolicy() *admissionregistrationv1.ValidatingAdmissionPolicy {
	return &admissionregistrationv1.ValidatingAdmissionPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "ValidatingAdmissionPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: vmiVAPName,
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
								admissionregistrationv1.Update,
							},
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{"kubevirt.io"},
								APIVersions: []string{"v1", "v1alpha3"},
								Resources:   []string{"virtualmachineinstances"},
							},
						},
					},
				},
			},
			Validations: []admissionregistrationv1.Validation{
				// Basic spec immutability for non-KubeVirt users
				// Note: This is a PARTIAL check. The webhook performs comprehensive validation
				// including authorization checks, hotplug validation, and reserved label protection.
				{
					Expression: "object.spec == oldObject.spec",
					Message:    VMIErrSpecImmutable,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
			},
		},
	}
}

func NewVMIValidatingAdmissionPolicyBinding() *admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	return &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "ValidatingAdmissionPolicyBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: vmiVAPBindingName,
			Labels: map[string]string{
				v1.AppLabel:       "virt-api",
				v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName: vmiVAPName,
			ValidationActions: []admissionregistrationv1.ValidationAction{
				admissionregistrationv1.Deny,
			},
		},
	}
}
