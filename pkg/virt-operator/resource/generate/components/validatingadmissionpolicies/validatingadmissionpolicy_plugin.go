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

package validatingadmissionpolicies

import (
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
)

const (
	pluginValidationPolicyName        = "kubevirt-plugin-validation-policy"
	pluginValidationPolicyBindingName = "kubevirt-plugin-validation-binding"
	pluginValidationAppLabelValue     = "kubevirt-plugin-validation"

	pluginWarningPolicyName        = "kubevirt-plugin-warning-policy"
	pluginWarningPolicyBindingName = "kubevirt-plugin-warning-binding"
	pluginWarningAppLabelValue     = "kubevirt-plugin-warning"
)

func NewPluginValidatingAdmissionPolicy() *admissionregistrationv1.ValidatingAdmissionPolicy {
	return &admissionregistrationv1.ValidatingAdmissionPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ValidatingAdmissionPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pluginValidationPolicyName,
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicySpec{
			FailurePolicy: new(admissionregistrationv1.Fail),
			MatchConstraints: &admissionregistrationv1.MatchResources{
				ResourceRules: []admissionregistrationv1.NamedRuleWithOperations{
					{
						RuleWithOperations: admissionregistrationv1.RuleWithOperations{
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{"plugin.kubevirt.io"},
								APIVersions: []string{"v1alpha1"},
								Resources:   []string{"plugins"},
							},
						},
					},
				},
			},
			Validations: []admissionregistrationv1.Validation{
				{
					Expression: `!has(object.spec.domainHooks) || object.spec.domainHooks.all(dh, has(dh.cel) != has(dh.sidecar))`,
					Message:    "exactly one of cel or sidecar must be specified for each domain hook",
				},
				{
					Expression: `!has(object.spec.domainHooks) || object.spec.domainHooks.all(dh, !has(dh.failureStrategy) || dh.failureStrategy in ['Fail', 'Ignore'])`,
					Message:    "failureStrategy must be either 'Fail' or 'Ignore'",
				},
				{
					Expression: `!has(object.spec.nodeHooks) || object.spec.nodeHooks.all(nh, !has(nh.failureStrategy) || nh.failureStrategy in ['Fail', 'Ignore'])`,
					Message:    "failureStrategy must be either 'Fail' or 'Ignore'",
				},
				{
					Expression: `!has(object.spec.nodeHooks) || object.spec.nodeHooks.all(nh, nh.permittedHooks.all(p, p in ['PreVMStart', 'PostVMStart', 'PreVMStop', 'PostVMStop', 'PreMigrationSource', 'PostMigrationSource', 'PreMigrationTarget', 'PostMigrationTarget']))`,
					Message:    "permittedHooks must only contain valid node hook points",
				},
				{
					Expression: `!has(object.spec.failureStrategy) || object.spec.failureStrategy in ['Fail', 'Ignore']`,
					Message:    "failureStrategy must be either 'Fail' or 'Ignore'",
				},
			},
		},
	}
}

func NewPluginWarningAdmissionPolicy() *admissionregistrationv1.ValidatingAdmissionPolicy {
	return &admissionregistrationv1.ValidatingAdmissionPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ValidatingAdmissionPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pluginWarningPolicyName,
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicySpec{
			FailurePolicy: new(admissionregistrationv1.Ignore),
			MatchConstraints: &admissionregistrationv1.MatchResources{
				ResourceRules: []admissionregistrationv1.NamedRuleWithOperations{
					{
						RuleWithOperations: admissionregistrationv1.RuleWithOperations{
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{"plugin.kubevirt.io"},
								APIVersions: []string{"v1alpha1"},
								Resources:   []string{"plugins"},
							},
						},
					},
				},
			},
			Variables: []admissionregistrationv1.Variable{
				{
					Name:       "hasDomainHooks",
					Expression: `has(object.spec.domainHooks) && size(object.spec.domainHooks) > 0`,
				},
				{
					Name:       "hasNodeHooks",
					Expression: `has(object.spec.nodeHooks) && size(object.spec.nodeHooks) > 0`,
				},
				{
					Name:       "hasAdmissionRefs",
					Expression: `(has(object.spec.mutatingAdmissionPolicies) && size(object.spec.mutatingAdmissionPolicies) > 0) || (has(object.spec.validatingAdmissionPolicies) && size(object.spec.validatingAdmissionPolicies) > 0) || (has(object.spec.mutatingAdmissionWebhooks) && size(object.spec.mutatingAdmissionWebhooks) > 0) || (has(object.spec.validatingAdmissionWebhooks) && size(object.spec.validatingAdmissionWebhooks) > 0)`,
				},
			},
			Validations: []admissionregistrationv1.Validation{
				{
					Expression: `variables.hasDomainHooks || variables.hasNodeHooks || variables.hasAdmissionRefs`,
					Message:    "no hooks or admission references are defined; this plugin will have no effect",
				},
			},
		},
	}
}

func NewPluginValidatingAdmissionPolicyBinding() *admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	return &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ValidatingAdmissionPolicyBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pluginValidationPolicyBindingName,
			Labels: map[string]string{
				v1.AppLabel:       pluginValidationAppLabelValue,
				v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName: pluginValidationPolicyName,
			ValidationActions: []admissionregistrationv1.ValidationAction{
				admissionregistrationv1.Deny,
			},
			MatchResources: &admissionregistrationv1.MatchResources{
				ResourceRules: []admissionregistrationv1.NamedRuleWithOperations{
					{
						RuleWithOperations: admissionregistrationv1.RuleWithOperations{
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{"plugin.kubevirt.io"},
								APIVersions: []string{"v1alpha1"},
								Resources:   []string{"plugins"},
							},
						},
					},
				},
			},
		},
	}
}

func NewPluginWarningAdmissionPolicyBinding() *admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	return &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ValidatingAdmissionPolicyBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pluginWarningPolicyBindingName,
			Labels: map[string]string{
				v1.AppLabel:       pluginWarningAppLabelValue,
				v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName: pluginWarningPolicyName,
			ValidationActions: []admissionregistrationv1.ValidationAction{
				admissionregistrationv1.Warn,
			},
			MatchResources: &admissionregistrationv1.MatchResources{
				ResourceRules: []admissionregistrationv1.NamedRuleWithOperations{
					{
						RuleWithOperations: admissionregistrationv1.RuleWithOperations{
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{"plugin.kubevirt.io"},
								APIVersions: []string{"v1alpha1"},
								Resources:   []string{"plugins"},
							},
						},
					},
				},
			},
		},
	}
}
