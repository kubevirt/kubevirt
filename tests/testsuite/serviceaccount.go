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
	SubresourceServiceAccountName = "kubevirt-subresource-test-sa"
	AdminServiceAccountName       = "kubevirt-admin-test-sa"
	EditServiceAccountName        = "kubevirt-edit-test-sa"
	ViewServiceAccountName        = "kubevirt-view-test-sa"
)

func createServiceAccounts() {
	createSubresourceServiceAccount()

	createServiceAccount(AdminServiceAccountName, "kubevirt.io:admin")
	createServiceAccount(ViewServiceAccountName, "kubevirt.io:view")
	createServiceAccount(EditServiceAccountName, "kubevirt.io:edit")
}

func cleanupServiceAccounts() {
	cleanupSubresourceServiceAccount()

	cleanupServiceAccount(AdminServiceAccountName)
	cleanupServiceAccount(ViewServiceAccountName)
	cleanupServiceAccount(EditServiceAccountName)
}

func cleanupSubresourceServiceAccount() {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	err = virtCli.CoreV1().ServiceAccounts(util.NamespaceTestDefault).Delete(context.Background(), SubresourceServiceAccountName, metav1.DeleteOptions{})
	if !k8serrors.IsNotFound(err) {
		util.PanicOnError(err)
	}

	err = virtCli.RbacV1().Roles(util.NamespaceTestDefault).Delete(context.Background(), SubresourceServiceAccountName, metav1.DeleteOptions{})
	if !k8serrors.IsNotFound(err) {
		util.PanicOnError(err)
	}

	err = virtCli.RbacV1().RoleBindings(util.NamespaceTestDefault).Delete(context.Background(), SubresourceServiceAccountName, metav1.DeleteOptions{})
	if !k8serrors.IsNotFound(err) {
		util.PanicOnError(err)
	}
}

func createServiceAccount(saName string, clusterRole string) {
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

func cleanupServiceAccount(saName string) {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	err = virtCli.RbacV1().RoleBindings(util.NamespaceTestDefault).Delete(context.Background(), saName, metav1.DeleteOptions{})
	if !k8serrors.IsNotFound(err) {
		util.PanicOnError(err)
	}

	err = virtCli.CoreV1().ServiceAccounts(util.NamespaceTestDefault).Delete(context.Background(), saName, metav1.DeleteOptions{})
	if !k8serrors.IsNotFound(err) {
		util.PanicOnError(err)
	}
}

func createSubresourceServiceAccount() {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	sa := k8sv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SubresourceServiceAccountName,
			Namespace: util.NamespaceTestDefault,
			Labels: map[string]string{
				util.KubevirtIoTest: "sa",
			},
		},
	}

	_, err = virtCli.CoreV1().ServiceAccounts(util.NamespaceTestDefault).Create(context.Background(), &sa, metav1.CreateOptions{})
	if !k8serrors.IsAlreadyExists(err) {
		util.PanicOnError(err)
	}

	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SubresourceServiceAccountName,
			Namespace: util.NamespaceTestDefault,
			Labels: map[string]string{
				util.KubevirtIoTest: "sa",
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
			Name:      SubresourceServiceAccountName,
			Namespace: util.NamespaceTestDefault,
			Labels: map[string]string{
				util.KubevirtIoTest: "sa",
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     SubresourceServiceAccountName,
			APIGroup: "rbac.authorization.k8s.io",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      SubresourceServiceAccountName,
				Namespace: util.NamespaceTestDefault,
			},
		},
	}

	_, err = virtCli.RbacV1().RoleBindings(util.NamespaceTestDefault).Create(context.Background(), &roleBinding, metav1.CreateOptions{})
	if !k8serrors.IsAlreadyExists(err) {
		util.PanicOnError(err)
	}
}
