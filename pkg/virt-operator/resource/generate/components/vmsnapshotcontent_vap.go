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
	vmSnapshotContentVAPName        = "vmsnapshotcontent-policy.snapshot.kubevirt.io"
	vmSnapshotContentVAPBindingName = "vmsnapshotcontent-policy-binding"

	VMSnapshotContentErrSpecImmutable = "spec is immutable after creation"
)

func NewVMSnapshotContentValidatingAdmissionPolicy() *admissionregistrationv1.ValidatingAdmissionPolicy {
	return &admissionregistrationv1.ValidatingAdmissionPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "ValidatingAdmissionPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: vmSnapshotContentVAPName,
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
								admissionregistrationv1.Update,
							},
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{"snapshot.kubevirt.io"},
								APIVersions: []string{"v1beta1", "v1alpha1"},
								Resources:   []string{"virtualmachinesnapshotcontents"},
							},
						},
					},
				},
			},
			Validations: []admissionregistrationv1.Validation{
				// UPDATE validation: spec is immutable
				{
					Expression: "object.spec == oldObject.spec",
					Message:    VMSnapshotContentErrSpecImmutable,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
			},
		},
	}
}

func NewVMSnapshotContentValidatingAdmissionPolicyBinding() *admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	return &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "ValidatingAdmissionPolicyBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: vmSnapshotContentVAPBindingName,
			Labels: map[string]string{
				v1.AppLabel:       vmSnapshotAppLabelValue,
				v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName: vmSnapshotContentVAPName,
			ValidationActions: []admissionregistrationv1.ValidationAction{
				admissionregistrationv1.Deny,
			},
		},
	}
}
