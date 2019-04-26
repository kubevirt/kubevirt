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
 * Copyright 2019 Red Hat, Inc.
 *
 */
package rbac

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
)

// Used for manifest generation only, not by the operator itself
func GetAllOperator(namespace string) []interface{} {
	return []interface{}{
		newOperatorServiceAccount(namespace),
		NewOperatorClusterRole(),
		newOperatorClusterRoleBinding(namespace),
	}
}

func newOperatorServiceAccount(namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "kubevirt-operator",
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
	}
}

// public, because it's used in manifest-templator
func NewOperatorClusterRole() *rbacv1.ClusterRole {
	// These are permissions needed by the operator itself.
	// For successfully deploying KubeVirt with the operator, you need to add everything
	// that the KubeVirt components' rules use, see below
	// (you can't create rules with permissions you don't have yourself)
	operatorRole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt-operator",
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"kubevirt.io",
				},
				Resources: []string{
					"kubevirts",
				},
				Verbs: []string{
					"get",
					"list",
					"watch",
					"patch",
					"update",
					"patch",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"serviceaccounts",
					"services",
					"endpoints",
				},
				Verbs: []string{
					"get",
					"list",
					"watch",
					"create",
					"update",
					"delete",
					"patch",
				},
			},
			{
				APIGroups: []string{
					"batch",
				},
				Resources: []string{
					"jobs",
				},
				Verbs: []string{
					"get",
					"list",
					"watch",
					"create",
					"delete",
					"patch",
				},
			},
			{
				APIGroups: []string{
					"apps",
				},
				Resources: []string{
					"deployments",
					"daemonsets",
				},
				Verbs: []string{
					"get",
					"list",
					"watch",
					"create",
					"delete",
					"patch",
				},
			},
			{
				APIGroups: []string{
					"rbac.authorization.k8s.io",
				},
				Resources: []string{
					"clusterroles",
					"clusterrolebindings",
					"roles",
					"rolebindings",
				},
				Verbs: []string{
					"get",
					"list",
					"watch",
					"create",
					"delete",
					"patch",
					"update",
				},
			},
			{
				APIGroups: []string{
					"apiextensions.k8s.io",
				},
				Resources: []string{
					"customresourcedefinitions",
				},
				Verbs: []string{
					"get",
					"list",
					"watch",
					"create",
					"delete",
					"patch",
				},
			},
			{
				APIGroups: []string{
					"security.openshift.io",
				},
				Resources: []string{
					"securitycontextconstraints",
				},
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
			},
			{
				APIGroups: []string{
					"security.openshift.io",
				},
				Resources: []string{
					"securitycontextconstraints",
				},
				ResourceNames: []string{
					"privileged",
				},
				Verbs: []string{
					"get",
					"patch",
					"update",
				},
			},
			{
				APIGroups: []string{
					"admissionregistration.k8s.io",
				},
				Resources: []string{
					"validatingwebhookconfigurations",
				},
				Verbs: []string{
					"get", "list", "watch", "create", "delete",
				},
			},
		},
	}

	// now append all rules needed by KubeVirt's components
	operatorRole.Rules = append(operatorRole.Rules, getKubeVirtComponentsRules()...)
	return operatorRole
}

func getKubeVirtComponentsRules() []rbacv1.PolicyRule {

	var rules []rbacv1.PolicyRule

	// namespace doesn't matter, we are only interested in the rules of both Roles and ClusterRoles
	all := GetAllApiServer("")
	all = append(all, GetAllController("")...)
	all = append(all, GetAllHandler("")...)
	all = append(all, GetAllCluster("")...)

	for _, resource := range all {
		switch resource.(type) {
		case *rbacv1.ClusterRole:
			role, _ := resource.(*rbacv1.ClusterRole)
			rules = append(rules, role.Rules...)
		case *rbacv1.Role:
			role, _ := resource.(*rbacv1.Role)
			rules = append(rules, role.Rules...)
		}
	}

	// OLM doesn't support role refs
	// so we need special handling for auth delegation for the apiserver,
	// by adding the rules of the system:auth-delegator role manually
	authDelegationRules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{
				"authentication.k8s.io",
			},
			Resources: []string{
				"tokenreviews",
			},
			Verbs: []string{
				"create",
			},
		},
		{
			APIGroups: []string{
				"authorization.k8s.io",
			},
			Resources: []string{
				"subjectaccessreviews",
			},
			Verbs: []string{
				"create",
			},
		},
	}
	rules = append(rules, authDelegationRules...)

	return rules
}

func newOperatorClusterRoleBinding(namespace string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt-operator",
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "kubevirt-operator",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      "kubevirt-operator",
			},
		},
	}
}
