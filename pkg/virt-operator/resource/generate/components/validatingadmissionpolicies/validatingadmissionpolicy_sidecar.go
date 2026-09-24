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

	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	sidecarSubPathPolicyName        = "kubevirt-plugin-sidecar-subpath-policy"
	sidecarSubPathPolicyBindingName = "kubevirt-plugin-sidecar-subpath-binding"
	sidecarSubPathAppLabelValue     = "kubevirt-plugin-sidecar-subpath"
)

func NewSidecarSubPathValidatingAdmissionPolicy() *admissionregistrationv1.ValidatingAdmissionPolicy {
	return &admissionregistrationv1.ValidatingAdmissionPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ValidatingAdmissionPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: sidecarSubPathPolicyName,
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicySpec{
			FailurePolicy: pointer.P(admissionregistrationv1.Fail),
			MatchConstraints: &admissionregistrationv1.MatchResources{
				ObjectSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						v1.AppLabel: "virt-launcher",
					},
				},
				ResourceRules: []admissionregistrationv1.NamedRuleWithOperations{
					{
						RuleWithOperations: admissionregistrationv1.RuleWithOperations{
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{""},
								APIVersions: []string{"v1"},
								Resources:   []string{"pods", "pods/ephemeralcontainers"},
							},
						},
					},
				},
			},
			Variables: []admissionregistrationv1.Variable{
				{
					Name: "containersHaveSubPath",
					Expression: `object.spec.containers.all(c,
    c.name == "compute" ||
    !has(c.volumeMounts) ||
    !c.volumeMounts.exists(vm, vm.name == "kubevirt-plugin-sockets") ||
    c.volumeMounts.filter(vm, vm.name == "kubevirt-plugin-sockets")
        .all(vm, has(vm.subPath) && vm.subPath != ""))`,
				},
				{
					Name: "initContainersHaveSubPath",
					Expression: `!has(object.spec.initContainers) ||
    object.spec.initContainers.all(c,
        !has(c.volumeMounts) ||
        !c.volumeMounts.exists(vm, vm.name == "kubevirt-plugin-sockets") ||
        c.volumeMounts.filter(vm, vm.name == "kubevirt-plugin-sockets")
            .all(vm, has(vm.subPath) && vm.subPath != ""))`,
				},
				{
					Name: "ephemeralContainersHaveSubPath",
					Expression: `!has(object.spec.ephemeralContainers) ||
    object.spec.ephemeralContainers.all(c,
        !has(c.volumeMounts) ||
        !c.volumeMounts.exists(vm, vm.name == "kubevirt-plugin-sockets") ||
        c.volumeMounts.filter(vm, vm.name == "kubevirt-plugin-sockets")
            .all(vm, has(vm.subPath) && vm.subPath != ""))`,
				},
			},
			Validations: []admissionregistrationv1.Validation{
				{
					Expression: `variables.containersHaveSubPath && variables.initContainersHaveSubPath && variables.ephemeralContainersHaveSubPath`,
					Message:    "sidecar containers must use subPath when mounting the plugin socket volume",
				},
			},
		},
	}
}

func NewSidecarSubPathValidatingAdmissionPolicyBinding() *admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	return &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ValidatingAdmissionPolicyBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: sidecarSubPathPolicyBindingName,
			Labels: map[string]string{
				v1.AppLabel:       sidecarSubPathAppLabelValue,
				v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName: sidecarSubPathPolicyName,
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
								APIGroups:   []string{""},
								APIVersions: []string{"v1"},
								Resources:   []string{"pods", "pods/ephemeralcontainers"},
							},
						},
					},
				},
			},
		},
	}
}
