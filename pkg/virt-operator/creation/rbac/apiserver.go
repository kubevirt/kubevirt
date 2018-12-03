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

func CreateApiServerRBAC(clientset kubecli.KubevirtClient, kv *virtv1.KubeVirt) error {

	core := clientset.CoreV1()

	sa := newApiServerServiceAccount(kv.Namespace)
	_, err := core.ServiceAccounts(kv.Namespace).Create(sa)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("unable to create serviceaccount %+v: %v", sa, err)
	}

	rbac := clientset.RbacV1()

	cr := newApiServerClusterRole()
	_, err = rbac.ClusterRoles().Create(cr)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("unable to create clusterrole %+v: %v", cr, err)
	}

	clusterRoleBindings := []*rbacv1.ClusterRoleBinding{
		newApiServerClusterRoleBinding(kv.Namespace),
		newApiServerAuthDelegatorClusterRoleBinding(kv.Namespace),
	}
	for _, crb := range clusterRoleBindings {
		_, err := rbac.ClusterRoleBindings().Create(crb)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create clusterrolebinding %+v: %v", crb, err)
		}
	}

	r := newApiServerRole(kv.Namespace)
	_, err = rbac.Roles(kv.Namespace).Create(r)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("unable to create role %+v: %v", r, err)
	}

	rb := newApiServerRoleBinding(kv.Namespace)
	_, err = rbac.RoleBindings(kv.Namespace).Create(rb)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("unable to create rolebinding %+v: %v", rb, err)
	}

	return nil
}

func GetAllApiServer(namespace string) []interface{} {
	return []interface{}{
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
			Name:      "kubevirt-apiserver",
			Labels: map[string]string{
				"kubevirt.io": "",
			},
		},
	}
}

func newApiServerClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt-apiserver",
			Labels: map[string]string{
				"kubevirt.io": "",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"admissionregistration.k8s.io",
				},
				Resources: []string{
					"validatingwebhookconfigurations",
					"mutatingwebhookconfigurations",
				},
				Verbs: []string{
					"get", "create", "update",
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
					"get", "create", "update",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"pods",
				},
				Verbs: []string{
					"get", "list",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"pods/exec",
				},
				Verbs: []string{
					"create",
				},
			},
			{
				APIGroups: []string{
					"kubevirt.io",
				},
				Resources: []string{
					"virtualmachines",
					"virtualmachineinstances",
					"virtualmachineinstancemigrations",
				},
				Verbs: []string{
					"get", "list", "watch", "delete",
				},
			},
			{
				APIGroups: []string{
					"kubevirt.io",
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
				ResourceNames: []string{
					"extension-apiserver-authentication",
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
		},
	}
}

func newApiServerClusterRoleBinding(namespace string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "kubevirt-apiserver",
			Labels: map[string]string{
				"kubevirt.io": "",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "kubevirt-apiserver",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      "kubevirt-apiserver",
			},
		},
	}
}

func newApiServerAuthDelegatorClusterRoleBinding(namespace string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "kubevirt-apiserver-auth-delegator",
			Labels: map[string]string{
				"kubevirt.io": "",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "system:auth-delegator",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      "kubevirt-apiserver",
			},
		},
	}
}

func newApiServerRole(namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "kubevirt-apiserver",
			Labels: map[string]string{
				"kubevirt.io": "",
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
					"get", "list", "delete", "update", "create",
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
		},
	}
}

func newApiServerRoleBinding(namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "kubevirt-apiserver",
			Labels: map[string]string{
				"kubevirt.io": "",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "kubevirt-apiserver",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      "kubevirt-apiserver",
			},
		},
	}
}
