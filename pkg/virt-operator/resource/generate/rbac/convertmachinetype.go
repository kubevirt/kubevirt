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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package rbac

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	virtv1 "kubevirt.io/api/core/v1"
)

const ConvertMachineTypeServiceAccountName = "convert-machine-type"

func GetAllConvertMachineType(namespace string) []runtime.Object {
	return []runtime.Object{
		newConvertMachineTypeServiceAccount(namespace),
		newConvertMachineTypeClusterRole(),
		newConvertMachineTypeClusterRoleBinding(namespace),
	}
}

func newConvertMachineTypeServiceAccount(namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      ConvertMachineTypeServiceAccountName,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
	}
}

func newConvertMachineTypeClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: VersionNamev1,
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ConvertMachineTypeServiceAccountName,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					virtv1.SubresourceGroupName,
				},
				Resources: []string{
					"virtualmachines/restart",
				},
				Verbs: []string{
					"update",
				},
			},
			{
				APIGroups: []string{
					virtv1.SubresourceGroupName,
				},
				Resources: []string{
					"expand-vm-spec",
				},
				Verbs: []string{
					"update",
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
					"get", "update", "patch", "list", "watch",
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
					"kubevirts",
				},
				Verbs: []string{
					"get", "list",
				},
			},
		},
	}
}

func newConvertMachineTypeClusterRoleBinding(namespace string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ConvertMachineTypeServiceAccountName,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     ConvertMachineTypeServiceAccountName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      ConvertMachineTypeServiceAccountName,
			},
		},
	}
}
