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

	pluginSocketPathPolicyName        = "kubevirt-plugin-socket-path-policy"
	pluginSocketPathPolicyBindingName = "kubevirt-plugin-socket-path-binding"
	pluginSocketPathAppLabelValue     = "kubevirt-plugin-socket-path"
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
			MatchConditions: []admissionregistrationv1.MatchCondition{
				{
					Name:       "is-virt-launcher-pod",
					Expression: `has(object.metadata.labels) && "kubevirt.io" in object.metadata.labels && object.metadata.labels["kubevirt.io"] == "virt-launcher"`,
				},
			},
			Variables: []admissionregistrationv1.Variable{
				{
					Name: "containersHaveSubPath",
					Expression: `object.spec.containers.all(c,
    c.name == "compute" ||
    !c.volumeMounts.exists(vm, vm.name == "kubevirt-plugin-sockets") ||
    c.volumeMounts.filter(vm, vm.name == "kubevirt-plugin-sockets")
        .all(vm, has(vm.subPath) && vm.subPath != ""))`,
				},
				{
					Name: "initContainersHaveSubPath",
					Expression: `!has(object.spec.initContainers) ||
    object.spec.initContainers.all(c,
        !c.volumeMounts.exists(vm, vm.name == "kubevirt-plugin-sockets") ||
        c.volumeMounts.filter(vm, vm.name == "kubevirt-plugin-sockets")
            .all(vm, has(vm.subPath) && vm.subPath != ""))`,
				},
				{
					Name: "ephemeralContainersHaveSubPath",
					Expression: `!has(object.spec.ephemeralContainers) ||
    object.spec.ephemeralContainers.all(c,
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

func NewPluginSocketPathValidatingAdmissionPolicy() *admissionregistrationv1.ValidatingAdmissionPolicy {
	return &admissionregistrationv1.ValidatingAdmissionPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ValidatingAdmissionPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pluginSocketPathPolicyName,
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
					Expression: `!has(object.spec.domainHooks) ||
	object.spec.domainHooks.all(dh,
	    !has(dh.sidecar) || dh.sidecar.socketPath.endsWith(".sock"))`,
					Message: "sidecar socketPath must end with .sock",
				},
				{
					Expression: `!has(object.spec.domainHooks) ||
	object.spec.domainHooks.all(dh,
	    !has(dh.sidecar) || size(dh.sidecar.socketPath) <= 108)`,
					Message: "sidecar socketPath must not exceed 108 characters (Unix socket limit)",
				},
				{
					Expression: `!has(object.spec.domainHooks) ||
	object.spec.domainHooks.all(dh,
	    !has(dh.sidecar) ||
	    dh.sidecar.socketPath.startsWith(
	        "/var/run/kubevirt-plugin/" + object.metadata.name + "/"))`,
					Message: "sidecar socketPath must start with /var/run/kubevirt-plugin/<plugin-name>/",
				},
			},
		},
	}
}

func NewPluginSocketPathValidatingAdmissionPolicyBinding() *admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	return &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ValidatingAdmissionPolicyBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pluginSocketPathPolicyBindingName,
			Labels: map[string]string{
				v1.AppLabel:       pluginSocketPathAppLabelValue,
				v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName: pluginSocketPathPolicyName,
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
