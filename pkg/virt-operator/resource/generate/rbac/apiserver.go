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
 * Copyright 2018 Red Hat, Inc.
 *
 */
package rbac

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/api/flavor"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/migrations"
)

const (
	VersionName   = "rbac.authorization.k8s.io"
	VersionNamev1 = "rbac.authorization.k8s.io/v1"
	GroupName     = "kubevirt.io"
)

const ApiServiceAccountName = "kubevirt-apiserver"

func GetAllApiServer(namespace string) []runtime.Object {
	return []runtime.Object{
		newApiServerServiceAccount(namespace),
		newApiServerClusterRole(),
		newApiServerClusterRoleBinding(namespace),
		newApiServerAuthDelegatorClusterRoleBinding(namespace),
		newApiServerRole(namespace),
		newApiServerRoleBinding(namespace),
	}
}

func newApiServerServiceAccount(namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      ApiServiceAccountName,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
	}
}

func newApiServerClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: VersionNamev1,
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ApiServiceAccountName,
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
					"pods",
				},
				Verbs: []string{
					"get", "list", "delete", "patch",
				},
			},
			{
				APIGroups: []string{
					GroupName,
				},
				Resources: []string{
					"virtualmachines",
					"virtualmachineinstances",
				},
				Verbs: []string{
					"get", "list", "watch", "patch", "update",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"persistentvolumeclaims",
				},
				Verbs: []string{
					"get",
				},
			},
			{
				APIGroups: []string{
					GroupName,
				},
				Resources: []string{
					"virtualmachines/status",
				},
				Verbs: []string{
					"patch",
				},
			},
			{
				APIGroups: []string{
					GroupName,
				},
				Resources: []string{
					"virtualmachineinstancemigrations",
				},
				Verbs: []string{
					"create", "get", "list", "watch", "patch",
				},
			},
			{
				APIGroups: []string{
					GroupName,
				},
				Resources: []string{
					"virtualmachineinstancepresets",
				},
				Verbs: []string{
					"watch", "list",
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
					"get", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"limitranges",
				},
				Verbs: []string{
					"watch", "list",
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
				},
			},
			{
				APIGroups: []string{
					GroupName,
				},
				Resources: []string{
					"kubevirts",
				},
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
			},
			{
				APIGroups: []string{
					"snapshot.kubevirt.io",
				},
				Resources: []string{
					"virtualmachinesnapshots",
					"virtualmachinerestores",
				},
				Verbs: []string{
					"get", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					"cdi.kubevirt.io",
				},
				Resources: []string{
					"datasources",
				},
				Verbs: []string{
					"get", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					"flavor.kubevirt.io",
				},
				Resources: []string{
					flavor.PluralResourceName,
					flavor.ClusterPluralResourceName,
					flavor.PluralPreferenceResourceName,
					flavor.ClusterPluralPreferenceResourceName,
				},
				Verbs: []string{
					"list", "watch",
				},
			},
			{
				APIGroups: []string{
					migrations.GroupName,
				},
				Resources: []string{
					migrations.ResourceMigrationPolicies,
				},
				Verbs: []string{
					"get", "list", "watch",
				},
			},
		},
	}
}

func newApiServerClusterRoleBinding(namespace string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: VersionNamev1,
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ApiServiceAccountName,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: VersionName,
			Kind:     "ClusterRole",
			Name:     ApiServiceAccountName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      ApiServiceAccountName,
			},
		},
	}
}

func newApiServerAuthDelegatorClusterRoleBinding(namespace string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: VersionNamev1,
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt-apiserver-auth-delegator",
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: VersionName,
			Kind:     "ClusterRole",
			Name:     "system:auth-delegator",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      ApiServiceAccountName,
			},
		},
	}
}

func newApiServerRole(namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: VersionNamev1,
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      ApiServiceAccountName,
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
					"configmaps",
				},
				Verbs: []string{
					"get", "list", "watch",
				},
			},
		},
	}
}

func newApiServerRoleBinding(namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: VersionNamev1,
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      ApiServiceAccountName,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: VersionName,
			Kind:     "Role",
			Name:     ApiServiceAccountName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      ApiServiceAccountName,
			},
		},
	}
}
