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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

func CreateClusterRBAC(clientset kubecli.KubevirtClient, kv *virtv1.KubeVirt) error {

	core := clientset.CoreV1()
	sa := newPrivilegedServiceAccount(kv.Namespace)
	_, err := core.ServiceAccounts(kv.Namespace).Create(sa)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("unable to create serviceaccount %+v: %v", sa, err)
	}

	rbac := clientset.RbacV1()

	clusterRoles := []*rbacv1.ClusterRole{
		newDefaultClusterRole(),
		newAdminClusterRole(),
		newEditClusterRole(),
		newViewClusterRole(),
	}
	for _, cr := range clusterRoles {
		_, err := rbac.ClusterRoles().Create(cr)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create clusterrole %+v: %v", cr, err)
		}
	}

	clusterRoleBindings := []*rbacv1.ClusterRoleBinding{
		newDefaultClusterRoleBinding(),
		newPrivilegedClusterRoleBinding(kv.Namespace),
	}
	for _, crb := range clusterRoleBindings {
		_, err := rbac.ClusterRoleBindings().Create(crb)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create clusterrolebinding %+v: %v", crb, err)
		}
	}

	return nil
}

func GetAllCluster(namespace string) []interface{} {
	return []interface{}{
		newDefaultClusterRole(),
		newDefaultClusterRoleBinding(),
		newAdminClusterRole(),
		newEditClusterRole(),
		newViewClusterRole(),
		newPrivilegedServiceAccount(namespace),
		newPrivilegedClusterRoleBinding(namespace),
	}
}

func newDefaultClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt.io:default",
			Labels: map[string]string{
				"kubevirt.io":                 "",
				"kubernetes.io/bootstrapping": "rbac-defaults",
			},
			Annotations: map[string]string{
				"rbac.authorization.kubernetes.io/autoupdate": "true",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"subresources.kubevirt.io",
				},
				Resources: []string{
					"version",
				},
				Verbs: []string{
					"get", "list",
				},
			},
		},
	}
}

func newDefaultClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt.io:default",
			Labels: map[string]string{
				"kubevirt.io": "",
			},
			Annotations: map[string]string{
				"rbac.authorization.kubernetes.io/autoupdate": "true",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "kubevirt.io:default",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "Group",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     "system:authenticated",
			},
			{
				Kind:     "Group",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     "system:unauthenticated",
			},
		},
	}
}

func newAdminClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt.io:admin",
			Labels: map[string]string{
				"kubevirt.io":                                  "",
				"rbac.authorization.k8s.io/aggregate-to-admin": "true",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"subresources.kubevirt.io",
				},
				Resources: []string{
					"virtualmachineinstances/console",
					"virtualmachineinstances/vnc",
					"virtualmachines/restart",
				},
				Verbs: []string{
					"get",
				},
			},
			{
				APIGroups: []string{
					"kubevirt.io",
				},
				Resources: []string{
					"virtualmachines",
					"virtualmachineinstances",
					"virtualmachineinstancepresets",
					"virtualmachineinstancereplicasets",
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch", "deletecollection",
				},
			},
		},
	}
}

func newEditClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt.io:edit",
			Labels: map[string]string{
				"kubevirt.io":                                 "",
				"rbac.authorization.k8s.io/aggregate-to-edit": "true",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"subresources.kubevirt.io",
				},
				Resources: []string{
					"virtualmachineinstances/console",
					"virtualmachineinstances/vnc",
					"virtualmachines/restart",
				},
				Verbs: []string{
					"get",
				},
			},
			{
				APIGroups: []string{
					"kubevirt.io",
				},
				Resources: []string{
					"virtualmachines",
					"virtualmachineinstances",
					"virtualmachineinstancepresets",
					"virtualmachineinstancereplicasets",
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch",
				},
			},
		},
	}
}

func newViewClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt.io:view",
			Labels: map[string]string{
				"kubevirt.io":                                 "",
				"rbac.authorization.k8s.io/aggregate-to-view": "true",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"kubevirt.io",
				},
				Resources: []string{
					"virtualmachines",
					"virtualmachineinstances",
					"virtualmachineinstancepresets",
					"virtualmachineinstancereplicasets",
				},
				Verbs: []string{
					"get", "list", "watch",
				},
			},
		},
	}
}

func newPrivilegedServiceAccount(namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "kubevirt-privileged",
			Labels: map[string]string{
				"kubevirt.io": "",
			},
		},
	}
}

func newPrivilegedClusterRoleBinding(namespace string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "kubevirt-privileged-cluster-admin",
			Labels: map[string]string{
				"kubevirt.io": "",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      "kubevirt-privileged",
			},
		},
	}
}
