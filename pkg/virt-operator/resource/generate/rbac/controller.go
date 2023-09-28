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

	"kubevirt.io/api/clone"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"

	"kubevirt.io/api/instancetype"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/migrations"
)

func GetAllController(namespace string) []runtime.Object {
	return []runtime.Object{
		newControllerServiceAccount(namespace),
		newControllerClusterRole(),
		newControllerClusterRoleBinding(namespace),
		newControllerRole(namespace),
		newControllerRoleBinding(namespace),
	}
}

func newControllerServiceAccount(namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      components.ControllerServiceAccountName,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
	}
}

func newControllerRole(namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: VersionNamev1,
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      components.ControllerServiceAccountName,
			Namespace: namespace,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					GroupNameRoute,
				},
				Resources: []string{
					"routes",
				},
				Verbs: []string{
					"list",
					"get",
					"watch",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"secrets",
				},
				Verbs: []string{
					"list",
					"get",
					"watch",
				},
			},
			{
				APIGroups: []string{
					"networking.k8s.io",
				},
				Resources: []string{
					"ingresses",
				},
				Verbs: []string{
					"list",
					"get",
					"watch",
				},
			},
			{
				APIGroups: []string{
					"coordination.k8s.io",
				},
				Resources: []string{
					"leases",
				},
				Verbs: []string{
					"get", "list", "watch", "delete", "update", "create", "patch",
				},
			},
		},
	}
}

func newControllerRoleBinding(namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: VersionNamev1,
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      components.ControllerServiceAccountName,
			Namespace: namespace,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: VersionName,
			Kind:     "Role",
			Name:     components.ControllerServiceAccountName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      components.ControllerServiceAccountName,
			},
		},
	}
}

func newControllerClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: components.ControllerServiceAccountName,
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
					"namespaces",
				},
				Verbs: []string{
					"get",
					"list",
					"watch",
					"patch",
				},
			},
			{
				APIGroups: []string{
					"policy",
				},
				Resources: []string{
					"poddisruptionbudgets",
				},
				Verbs: []string{
					"get", "list", "watch", "delete", "create", "patch",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"pods", "configmaps", "endpoints", "services",
				},
				Verbs: []string{
					"get", "list", "watch", "delete", "update", "create", "patch",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"events",
				},
				Verbs: []string{
					"update", "create", "patch",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"secrets",
				},
				Verbs: []string{
					"create",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"pods/finalizers",
				},
				Verbs: []string{
					"update",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"pods/eviction",
				},
				Verbs: []string{
					"create",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"pods/status",
				},
				Verbs: []string{
					"patch",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"nodes",
				},
				Verbs: []string{
					"get", "list", "watch", "update", "patch",
				},
			},
			{
				APIGroups: []string{
					"apps",
				},
				Resources: []string{
					"daemonsets",
				},
				Verbs: []string{
					"list",
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
					"get",
					"update",
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
					"get", "list", "watch", "create", "update", "delete", "patch",
				},
			},
			{
				APIGroups: []string{
					"snapshot.kubevirt.io",
				},
				Resources: []string{
					"*",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"export.kubevirt.io",
				},
				Resources: []string{
					"*",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"pool.kubevirt.io",
				},
				Resources: []string{
					"virtualmachinepools",
					"virtualmachinepools/finalizers",
					"virtualmachinepools/status",
					"virtualmachinepools/scale",
				},

				Verbs: []string{
					"watch",
					"list",
					"create",
					"delete",
					"update",
					"patch",
					"get",
				},
			},
			{
				APIGroups: []string{
					"kubevirt.io",
				},
				Resources: []string{
					"*",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"subresources.kubevirt.io",
				},
				Resources: []string{
					"virtualmachineinstances/addvolume",
					"virtualmachineinstances/removevolume",
					"virtualmachineinstances/freeze",
					"virtualmachineinstances/unfreeze",
					"virtualmachineinstances/softreboot",
					"virtualmachineinstances/sev/setupsession",
					"virtualmachineinstances/sev/injectlaunchsecret",
				},
				Verbs: []string{
					"update",
				},
			},
			{
				APIGroups: []string{
					"cdi.kubevirt.io",
				},
				Resources: []string{
					"*",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"k8s.cni.cncf.io",
				},
				Resources: []string{
					"network-attachment-definitions",
				},
				Verbs: []string{"get"},
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
					"authorization.k8s.io",
				},
				Resources: []string{
					"subjectaccessreviews",
				},
				Verbs: []string{
					"create",
				},
			},
			{
				APIGroups: []string{
					"snapshot.storage.k8s.io",
				},
				Resources: []string{
					"volumesnapshotclasses",
				},
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
			},
			{
				APIGroups: []string{
					"snapshot.storage.k8s.io",
				},
				Resources: []string{
					"volumesnapshots",
				},
				Verbs: []string{
					"get",
					"list",
					"watch",
					"create",
					"update",
					"delete",
				},
			},
			{
				APIGroups: []string{
					"storage.k8s.io",
				},
				Resources: []string{
					"storageclasses",
				},
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
			},
			{
				APIGroups: []string{
					"instancetype.kubevirt.io",
				},
				Resources: []string{
					instancetype.PluralResourceName,
					instancetype.ClusterPluralResourceName,
					instancetype.PluralPreferenceResourceName,
					instancetype.ClusterPluralPreferenceResourceName,
				},
				Verbs: []string{
					"get", "list", "watch",
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
			{
				APIGroups: []string{
					clone.GroupName,
				},
				Resources: []string{
					clone.ResourceVMClonePlural,
					clone.ResourceVMClonePlural + "/status",
					clone.ResourceVMClonePlural + "/finalizers",
				},
				Verbs: []string{
					"get", "list", "watch", "update", "patch", "delete",
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
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"resourcequotas",
				},
				Verbs: []string{
					"list",
					"watch",
				},
			},
		},
	}
}

func newControllerClusterRoleBinding(namespace string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: components.ControllerServiceAccountName,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     components.ControllerServiceAccountName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      components.ControllerServiceAccountName,
			},
		},
	}
}
