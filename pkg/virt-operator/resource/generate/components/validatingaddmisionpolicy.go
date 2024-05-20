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
	"fmt"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	validatingAdmissionPolicyBindingName = "kubevirt-node-restriction-binding"
	validatingAdmissionPolicyName        = "kubevirt-node-restriction-policy"
	nodeRestrictionAppLabelValue         = "kubevirt-node-restriction"

	NodeRestrictionErrModifySpec           = "this user cannot modify spec of node"
	NodeRestrictionErrChangeMetadataFields = "this user can only change allowed metadata fields"
	NodeRestrictionErrAddDeleteLabels      = "this user cannot add/delete non kubevirt-owned labels"
	NodeRestrictionErrUpdateLabels         = "this user cannot update non kubevirt-owned labels"
	NodeRestrictionErrAddDeleteAnnotations = "this user cannot add/delete non kubevirt-owned annotations"
	NodeRestrictionErrUpdateAnnotations    = "this user cannot update non kubevirt-owned annotations"
)

func NewHandlerV1ValidatingAdmissionPolicyBinding() *admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	return &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ValidatingAdmissionPolicyBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: validatingAdmissionPolicyBindingName,
			Labels: map[string]string{
				v1.AppLabel:       nodeRestrictionAppLabelValue,
				v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName: validatingAdmissionPolicyName,
			ValidationActions: []admissionregistrationv1.ValidationAction{
				admissionregistrationv1.Deny,
			},
			MatchResources: &admissionregistrationv1.MatchResources{
				ResourceRules: []admissionregistrationv1.NamedRuleWithOperations{
					{
						RuleWithOperations: admissionregistrationv1.RuleWithOperations{
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.OperationAll,
							},
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{""},
								APIVersions: []string{"*"},
								Resources:   []string{"nodes"},
							},
						},
					},
				},
			},
		},
	}
}

func NewHandlerV1ValidatingAdmissionPolicy(virtHandlerServiceAccount string) *admissionregistrationv1.ValidatingAdmissionPolicy {
	return &admissionregistrationv1.ValidatingAdmissionPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ValidatingAdmissionPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: validatingAdmissionPolicyName,
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
								APIGroups:   []string{""},
								APIVersions: []string{"*"},
								Resources:   []string{"nodes"},
							},
						},
					},
				},
			},
			MatchConditions: []admissionregistrationv1.MatchCondition{
				{
					Name:       "virt-handler-user-only",
					Expression: fmt.Sprintf("request.userInfo.username == %q", virtHandlerServiceAccount),
				},
			},
			Variables: []admissionregistrationv1.Variable{
				{
					Name:       "oldNonKubevirtLabels",
					Expression: `oldObject.metadata.labels.filter(k, !k.contains("kubevirt.io") && k != "cpumanager")`,
				},
				{
					Name:       "oldLabels",
					Expression: "oldObject.metadata.labels",
				},
				{
					Name:       "newNonKubevirtLabels",
					Expression: `object.metadata.labels.filter(k, !k.contains("kubevirt.io") && k != "cpumanager")`,
				},
				{
					Name:       "newLabels",
					Expression: "object.metadata.labels",
				},
				{
					Name:       "oldNonKubevirtAnnotations",
					Expression: `oldObject.metadata.annotations.filter(k, !k.contains("kubevirt.io"))`,
				},
				{
					Name:       "newNonKubevirtAnnotations",
					Expression: `object.metadata.annotations.filter(k, !k.contains("kubevirt.io"))`,
				},
				{
					Name:       "oldAnnotations",
					Expression: "oldObject.metadata.annotations",
				},
				{
					Name:       "newAnnotations",
					Expression: "object.metadata.annotations",
				},
			},
			Validations: []admissionregistrationv1.Validation{
				{
					Expression: "object.spec == oldObject.spec",
					Message:    NodeRestrictionErrModifySpec,
				},
				{
					Expression: `oldObject.metadata.filter(k, k != "labels" && k != "annotations" && k != "managedFields" && k != "resourceVersion").all(k, k in object.metadata) && object.metadata.filter(k, k != "labels" && k != "annotations" && k != "managedFields" && k != "resourceVersion").all(k, k in oldObject.metadata && oldObject.metadata[k] == object.metadata[k])`,
					Message:    NodeRestrictionErrChangeMetadataFields,
				},
				{
					Expression: `size(variables.newNonKubevirtLabels) == size(variables.oldNonKubevirtLabels)`,
					Message:    NodeRestrictionErrAddDeleteLabels,
				},
				{
					Expression: `variables.newNonKubevirtLabels.all(k, k in variables.oldNonKubevirtLabels && variables.newLabels[k] == variables.oldLabels[k])`,
					Message:    NodeRestrictionErrUpdateLabels,
				},
				{
					Expression: `size(variables.newNonKubevirtAnnotations) == size(variables.oldNonKubevirtAnnotations)`,
					Message:    NodeRestrictionErrAddDeleteAnnotations,
				},
				{
					Expression: `variables.newNonKubevirtAnnotations.all(k, k in variables.oldNonKubevirtAnnotations && variables.newAnnotations[k] == variables.oldAnnotations[k])`,
					Message:    NodeRestrictionErrUpdateAnnotations,
				},
			},
		},
	}
}
