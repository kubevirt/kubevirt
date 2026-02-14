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
	vmiMigrationVAPName        = "vmimigration-policy.kubevirt.io"
	vmiMigrationVAPBindingName = "vmimigration-policy-binding"

	VMIMigrationErrVMINameRequired                = "spec.vmiName is required"
	VMIMigrationErrPriorityFeatureGateDisabled    = "MigrationPriorityQueue feature gate is not enabled in kubevirt resource"
	VMIMigrationErrDecentralizedFeatureGateDisabled = "DecentralizedLiveMigration feature gate is not enabled in kubevirt resource"
	VMIMigrationErrMigrationIDMismatch            = "spec.sendTo.migrationID must match spec.receive.migrationID when both are set"
	VMIMigrationErrSpecImmutable                  = "spec is immutable after creation"
)

func NewVMIMigrationValidatingAdmissionPolicy(namespace string) *admissionregistrationv1.ValidatingAdmissionPolicy {
	return &admissionregistrationv1.ValidatingAdmissionPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "ValidatingAdmissionPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: vmiMigrationVAPName,
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
								Resources:   []string{"virtualmachineinstancemigrations"},
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
					Expression: "request.operation != 'CREATE' || has(object.spec.vmiName) && object.spec.vmiName != ''",
					Message:    VMIMigrationErrVMINameRequired,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				// Migration priority feature gate check
				{
					Expression: `request.operation != 'CREATE' || !has(object.spec.priority) || params.spec.configuration.developerConfiguration.featureGates.exists(fg, fg == 'MigrationPriorityQueue')`,
					Message:    VMIMigrationErrPriorityFeatureGateDisabled,
					Reason:     pointer.P(metav1.StatusReasonForbidden),
				},
				// Decentralized migration feature gate check
				{
					Expression: `request.operation != 'CREATE' || (!has(object.spec.sendTo) && !has(object.spec.receive)) || params.spec.configuration.developerConfiguration.featureGates.exists(fg, fg == 'DecentralizedLiveMigration')`,
					Message:    VMIMigrationErrDecentralizedFeatureGateDisabled,
					Reason:     pointer.P(metav1.StatusReasonForbidden),
				},
				// Migration ID match when both sendTo and receive are set
				{
					Expression: "request.operation != 'CREATE' || !has(object.spec.sendTo) || !has(object.spec.receive) || object.spec.sendTo.migrationID == object.spec.receive.migrationID",
					Message:    VMIMigrationErrMigrationIDMismatch,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				// UPDATE validations
				{
					Expression: "request.operation != 'UPDATE' || object.spec == oldObject.spec",
					Message:    VMIMigrationErrSpecImmutable,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
			},
		},
	}
}

func NewVMIMigrationValidatingAdmissionPolicyBinding(namespace string) *admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	return &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "ValidatingAdmissionPolicyBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: vmiMigrationVAPBindingName,
			Labels: map[string]string{
				v1.AppLabel:       "virt-api",
				v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName: vmiMigrationVAPName,
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
