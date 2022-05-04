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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
)

const (
	GroupNameSecurity = "security.openshift.io"
	serviceAccountFmt = "%s:%s:%s"
)
const OperatorServiceAccountName = "kubevirt-operator"

// Used for manifest generation only, not by the operator itself
func GetAllOperator(namespace string) []interface{} {
	return []interface{}{
		newOperatorServiceAccount(namespace),
		NewOperatorRole(namespace),
		newOperatorRoleBinding(namespace),
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
			Name:      OperatorServiceAccountName,
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
			APIVersion: VersionNamev1,
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: OperatorServiceAccountName,
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
					// pods/exec is required for testing upgrades - that can be removed when we stop
					// supporting upgrades from versions in which virt-api required pods/exec privileges
					"pods/exec",
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
					"",
				},
				Resources: []string{
					"configmaps",
				},
				Verbs: []string{
					"patch",
					"delete",
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
					"controllerrevisions",
				},
				Verbs: []string{
					"watch",
					"list",
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
					VersionName,
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
					GroupNameSecurity,
				},
				Resources: []string{
					"securitycontextconstraints",
				},
				Verbs: []string{
					"create",
					"get",
					"list",
					"watch",
				},
			},
			{
				APIGroups: []string{
					GroupNameSecurity,
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
					GroupNameSecurity,
				},
				Resources: []string{
					"securitycontextconstraints",
				},
				ResourceNames: []string{
					"kubevirt-handler",
					"kubevirt-controller",
				},
				Verbs: []string{
					"get",
					"list",
					"watch",
					"update",
					"delete",
				},
			},
			{
				APIGroups: []string{
					"admissionregistration.k8s.io",
				},
				Resources: []string{
					"validatingwebhookconfigurations",
					"mutatingwebhookconfigurations",
				},
				Verbs: []string{
					"get", "list", "watch", "create", "delete", "update", "patch",
				},
			},
			{
				APIGroups: []string{
					"apiregistration.k8s.io",
				},
				Resources: []string{
					"apiservices",
				},
				Verbs: []string{
					"get", "list", "watch", "create", "delete", "update", "patch",
				},
			},
			{
				APIGroups: []string{
					"monitoring.coreos.com",
				},
				Resources: []string{
					"servicemonitors",
					"prometheusrules",
				},
				Verbs: []string{
					"get", "list", "watch", "create", "delete", "update", "patch",
				},
			},
			// Until v0.43 a `get` verb was granted to these resources, but there is no get endpoint.
			// The get permission needs to be kept on the operator level so that updates work.
			{
				APIGroups: []string{
					"subresources.kubevirt.io",
				},
				Resources: []string{
					"virtualmachineinstances/pause",
					"virtualmachineinstances/unpause",
					"virtualmachineinstances/addvolume",
					"virtualmachineinstances/removevolume",
					"virtualmachineinstances/freeze",
					"virtualmachineinstances/unfreeze",
					"virtualmachineinstances/softreboot",
					"virtualmachineinstances/portforward",
				},
				Verbs: []string{
					"update",
					"get",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"namespaces",
				},
				Verbs: []string{
					"get",
					"list",
					"watch",
					"patch",
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
	all = append(all, GetAllCluster()...)

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
			APIVersion: VersionNamev1,
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: OperatorServiceAccountName,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: VersionName,
			Kind:     "ClusterRole",
			Name:     OperatorServiceAccountName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      OperatorServiceAccountName,
			},
		},
	}
}

func newOperatorRoleBinding(namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: VersionNamev1,
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt-operator-rolebinding",
			Namespace: namespace,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: VersionName,
			Kind:     "Role",
			Name:     OperatorServiceAccountName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      OperatorServiceAccountName,
			},
		},
	}
}

// NewOperatorRole creates a Role object for kubevirt-operator.
func NewOperatorRole(namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: VersionNamev1,
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      OperatorServiceAccountName,
			Namespace: namespace,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"secrets",
				},
				Verbs: []string{
					"create",
					"get",
					"list",
					"watch",
					"patch",
					"delete",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"configmaps",
				},
				Verbs: []string{
					"create",
					"get",
					"list",
					"watch",
					"patch",
					"delete",
				},
			},
		},
	}
}

func GetKubevirtComponentsServiceAccounts(namespace string) map[string]bool {
	usermap := make(map[string]bool)

	prefix := "system:serviceaccount"
	usermap[fmt.Sprintf(serviceAccountFmt, prefix, namespace, HandlerServiceAccountName)] = true
	usermap[fmt.Sprintf(serviceAccountFmt, prefix, namespace, ApiServiceAccountName)] = true
	usermap[fmt.Sprintf(serviceAccountFmt, prefix, namespace, ControllerServiceAccountName)] = true
	usermap[fmt.Sprintf(serviceAccountFmt, prefix, namespace, OperatorServiceAccountName)] = true

	return usermap
}
