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
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/client-go/api/v1"
)

func GetAllCluster(namespace string) []interface{} {
	return []interface{}{
		newDefaultClusterRole(),
		newDefaultClusterRoleBinding(),
		newAdminClusterRole(),
		newEditClusterRole(),
		newViewClusterRole(),
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
				virtv1.AppLabel:               "",
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
				virtv1.AppLabel: "",
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
				virtv1.AppLabel: "",
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
				},
				Verbs: []string{
					"get",
				},
			},
			{
				APIGroups: []string{
					"subresources.kubevirt.io",
				},
				Resources: []string{
					"virtualmachineinstances/pause",
					"virtualmachineinstances/unpause",
				},
				Verbs: []string{
					"get",
					"update",
				},
			},
			{
				APIGroups: []string{
					"subresources.kubevirt.io",
				},
				Resources: []string{
					"virtualmachines/start",
					"virtualmachines/stop",
					"virtualmachines/restart",
				},
				Verbs: []string{
					"update",
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
					"virtualmachineinstancemigrations",
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch", "deletecollection",
				},
			},
			{
				APIGroups: []string{
					"snapshot.kubevirt.io",
				},
				Resources: []string{
					"virtualmachinesnapshots",
					"virtualmachinesnapshotcontents",
					"virtualmachinerestores",
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
				virtv1.AppLabel: "",
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
				},
				Verbs: []string{
					"get",
				},
			},
			{
				APIGroups: []string{
					"subresources.kubevirt.io",
				},
				Resources: []string{
					"virtualmachineinstances/pause",
					"virtualmachineinstances/unpause",
				},
				Verbs: []string{
					"get",
					"update",
				},
			},
			{
				APIGroups: []string{
					"subresources.kubevirt.io",
				},
				Resources: []string{
					"virtualmachines/start",
					"virtualmachines/stop",
					"virtualmachines/restart",
				},
				Verbs: []string{
					"update",
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
					"virtualmachineinstancemigrations",
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					"snapshot.kubevirt.io",
				},
				Resources: []string{
					"virtualmachinesnapshots",
					"virtualmachinesnapshotcontents",
					"virtualmachinerestores",
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
				virtv1.AppLabel: "",
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
					"virtualmachineinstancemigrations",
				},
				Verbs: []string{
					"get", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					"snapshot.kubevirt.io",
				},
				Resources: []string{
					"virtualmachinesnapshots",
					"virtualmachinesnapshotcontents",
					"virtualmachinerestores",
				},
				Verbs: []string{
					"get", "list", "watch",
				},
			},
		},
	}
}
