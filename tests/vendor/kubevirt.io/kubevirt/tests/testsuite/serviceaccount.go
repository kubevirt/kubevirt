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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package testsuite

import (
	"context"

	k8sv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/util"
)

const (
	AdminServiceAccountName                   = "kubevirt-admin-test-sa"
	EditServiceAccountName                    = "kubevirt-edit-test-sa"
	ViewServiceAccountName                    = "kubevirt-view-test-sa"
	SubresourceServiceAccountName             = "kubevirt-subresource-test-sa"
	SubresourceUnprivilegedServiceAccountName = "kubevirt-subresource-test-unprivileged-sa"
)

func createServiceAccounts() {
	createServiceAccount(AdminServiceAccountName)
	createRoleBinding(AdminServiceAccountName, "kubevirt.io:admin")

	createServiceAccount(EditServiceAccountName)
	createRoleBinding(EditServiceAccountName, "kubevirt.io:edit")

	createServiceAccount(ViewServiceAccountName)
	createRoleBinding(ViewServiceAccountName, "kubevirt.io:view")

	createServiceAccount(SubresourceServiceAccountName)
	createSubresourceRole(SubresourceServiceAccountName)

	createServiceAccount(SubresourceUnprivilegedServiceAccountName)
}

func cleanupServiceAccounts() {
	cleanupServiceAccount(AdminServiceAccountName)
	cleanupServiceAccount(EditServiceAccountName)
	cleanupServiceAccount(ViewServiceAccountName)
	cleanupServiceAccount(SubresourceServiceAccountName)
	cleanupServiceAccount(SubresourceUnprivilegedServiceAccountName)
}
func createServiceAccount(saName string) {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	sa := k8sv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: util.NamespaceTestDefault,
			Labels: map[string]string{
				util.KubevirtIoTest: saName,
			},
		},
	}

	_, err = virtCli.CoreV1().ServiceAccounts(util.NamespaceTestDefault).Create(context.Background(), &sa, metav1.CreateOptions{})
	if !k8serrors.IsAlreadyExists(err) {
		util.PanicOnError(err)
	}

	secret := k8sv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: util.NamespaceTestDefault,
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": saName,
			},
		},
		Type: k8sv1.SecretTypeServiceAccountToken,
	}

	_, err = virtCli.CoreV1().Secrets(util.NamespaceTestDefault).Create(context.Background(), &secret, metav1.CreateOptions{})
	if !k8serrors.IsAlreadyExists(err) {
		util.PanicOnError(err)
	}
}

func createRoleBinding(saName string, clusterRole string) {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: util.NamespaceTestDefault,
			Labels: map[string]string{
				util.KubevirtIoTest: saName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     clusterRole,
			APIGroup: "rbac.authorization.k8s.io",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: util.NamespaceTestDefault,
			},
		},
	}

	_, err = virtCli.RbacV1().RoleBindings(util.NamespaceTestDefault).Create(context.Background(), &roleBinding, metav1.CreateOptions{})
	if !k8serrors.IsAlreadyExists(err) {
		util.PanicOnError(err)
	}
}

func createSubresourceRole(saName string) {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: util.NamespaceTestDefault,
			Labels: map[string]string{
				util.KubevirtIoTest: saName,
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"subresources.kubevirt.io"},
				Resources: []string{"virtualmachines/start", "expand-vm-spec"},
				Verbs:     []string{"update"},
			},
		},
	}

	_, err = virtCli.RbacV1().Roles(util.NamespaceTestDefault).Create(context.Background(), &role, metav1.CreateOptions{})
	if !k8serrors.IsAlreadyExists(err) {
		util.PanicOnError(err)
	}

	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: util.NamespaceTestDefault,
			Labels: map[string]string{
				util.KubevirtIoTest: saName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     saName,
			APIGroup: "rbac.authorization.k8s.io",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: util.NamespaceTestDefault,
			},
		},
	}

	_, err = virtCli.RbacV1().RoleBindings(util.NamespaceTestDefault).Create(context.Background(), &roleBinding, metav1.CreateOptions{})
	if !k8serrors.IsAlreadyExists(err) {
		util.PanicOnError(err)
	}
}

func cleanupServiceAccount(saName string) {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	err = virtCli.CoreV1().ServiceAccounts(util.NamespaceTestDefault).Delete(context.Background(), saName, metav1.DeleteOptions{})
	if !k8serrors.IsNotFound(err) {
		util.PanicOnError(err)
	}

	err = virtCli.CoreV1().Secrets(util.NamespaceTestDefault).Delete(context.Background(), saName, metav1.DeleteOptions{})
	if !k8serrors.IsNotFound(err) {
		util.PanicOnError(err)
	}

	err = virtCli.RbacV1().Roles(util.NamespaceTestDefault).Delete(context.Background(), saName, metav1.DeleteOptions{})
	if !k8serrors.IsNotFound(err) {
		util.PanicOnError(err)
	}

	err = virtCli.RbacV1().RoleBindings(util.NamespaceTestDefault).Delete(context.Background(), saName, metav1.DeleteOptions{})
	if !k8serrors.IsNotFound(err) {
		util.PanicOnError(err)
	}
}
