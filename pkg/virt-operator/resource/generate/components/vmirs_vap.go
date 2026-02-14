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
	vmirsVAPName        = "vmirs-policy.kubevirt.io"
	vmirsVAPBindingName = "vmirs-policy-binding"

	VMIRSErrTemplateRequired      = "spec.template is required"
	VMIRSErrSelectorRequired      = "spec.selector is required"
	VMIRSErrSelectorMatchesLabels = "spec.selector must match spec.template.metadata.labels"
)

func NewVMIRSValidatingAdmissionPolicy() *admissionregistrationv1.ValidatingAdmissionPolicy {
	return &admissionregistrationv1.ValidatingAdmissionPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "ValidatingAdmissionPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: vmirsVAPName,
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
								Resources:   []string{"virtualmachineinstancereplicasets"},
							},
						},
					},
				},
			},
			Validations: []admissionregistrationv1.Validation{
				// Template required
				{
					Expression: "has(object.spec.template)",
					Message:    VMIRSErrTemplateRequired,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				// Selector required
				{
					Expression: "has(object.spec.selector)",
					Message:    VMIRSErrSelectorRequired,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				// Selector matches template labels
				// This checks if all matchLabels in selector exist in template.metadata.labels with same values
				{
					Expression: `!has(object.spec.selector.matchLabels) || !has(object.spec.template.metadata.labels) || object.spec.selector.matchLabels.all(key, key in object.spec.template.metadata.labels && object.spec.template.metadata.labels[key] == object.spec.selector.matchLabels[key])`,
					Message:    VMIRSErrSelectorMatchesLabels,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
			},
		},
	}
}

func NewVMIRSValidatingAdmissionPolicyBinding() *admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	return &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "ValidatingAdmissionPolicyBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: vmirsVAPBindingName,
			Labels: map[string]string{
				v1.AppLabel:       "virt-api",
				v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName: vmirsVAPName,
			ValidationActions: []admissionregistrationv1.ValidationAction{
				admissionregistrationv1.Deny,
			},
		},
	}
}
