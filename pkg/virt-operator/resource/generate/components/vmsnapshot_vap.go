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
	vmSnapshotVAPName        = "vmsnapshot-policy.snapshot.kubevirt.io"
	vmSnapshotVAPBindingName = "vmsnapshot-policy-binding"
	vmSnapshotAppLabelValue  = "virt-api"

	VMSnapshotErrFeatureGateDisabled  = "snapshot feature gate not enabled"
	VMSnapshotErrMissingAPIGroup      = "spec.source.apiGroup is required"
	VMSnapshotErrInvalidAPIGroup      = "spec.source.apiGroup must be 'kubevirt.io'"
	VMSnapshotErrInvalidKind          = "spec.source.kind must be 'VirtualMachine' when apiGroup is 'kubevirt.io'"
	VMSnapshotErrSpecImmutable        = "spec is immutable after creation"
)

func NewVMSnapshotValidatingAdmissionPolicy(namespace string) *admissionregistrationv1.ValidatingAdmissionPolicy {
	return &admissionregistrationv1.ValidatingAdmissionPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "ValidatingAdmissionPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: vmSnapshotVAPName,
			Labels: map[string]string{
				v1.AppLabel:       vmSnapshotAppLabelValue,
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
								APIGroups:   []string{"snapshot.kubevirt.io"},
								APIVersions: []string{"v1beta1", "v1alpha1"},
								Resources:   []string{"virtualmachinesnapshots"},
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
				// CREATE validations
				{
					Expression: "request.operation != 'CREATE' || params.spec.configuration.developerConfiguration.featureGates.exists(fg, fg == 'Snapshot')",
					Message:    VMSnapshotErrFeatureGateDisabled,
					Reason:     pointer.P(metav1.StatusReasonForbidden),
				},
				{
					Expression: "request.operation != 'CREATE' || has(object.spec.source.apiGroup)",
					Message:    VMSnapshotErrMissingAPIGroup,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				{
					Expression: "request.operation != 'CREATE' || !has(object.spec.source.apiGroup) || object.spec.source.apiGroup == 'kubevirt.io'",
					Message:    VMSnapshotErrInvalidAPIGroup,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				{
					Expression: "request.operation != 'CREATE' || !has(object.spec.source.apiGroup) || object.spec.source.apiGroup != 'kubevirt.io' || object.spec.source.kind == 'VirtualMachine'",
					Message:    VMSnapshotErrInvalidKind,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				// UPDATE validations
				{
					Expression: "request.operation != 'UPDATE' || object.spec == oldObject.spec",
					Message:    VMSnapshotErrSpecImmutable,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
			},
		},
	}
}

func NewVMSnapshotValidatingAdmissionPolicyBinding(namespace string) *admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	return &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "ValidatingAdmissionPolicyBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: vmSnapshotVAPBindingName,
			Labels: map[string]string{
				v1.AppLabel:       vmSnapshotAppLabelValue,
				v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName: vmSnapshotVAPName,
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
