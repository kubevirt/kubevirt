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
	"k8s.io/apimachinery/pkg/runtime"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/api/migrations"
)

const (
	clusterAPIVersionName           = "rbac.authorization.k8s.io"
	clusterAPIVersionNamev1         = "rbac.authorization.k8s.io/v1"
	clusterAPIGroupName             = "kubevirt.io"
	clusterAPIGroupNameSubresources = "subresources.kubevirt.io"
	clusterAPIGroupNameSnapshot     = "snapshot.kubevirt.io"
	clusterAPIGroupNameFlavor       = "flavor.kubevirt.io"
  clusterAPIGroupNamePool        = "pool.kubevirt.io"
	clusterNameDefault              = "kubevirt.io:default"
	clusterVMInstancesGuestOSInfo   = "virtualmachineinstances/guestosinfo"
	clusterVMInstancesFileSysList   = "virtualmachineinstances/filesystemlist"
	clusterVMInstancesUserList      = "virtualmachineinstances/userlist"
)

func GetAllCluster() []runtime.Object {
	return []runtime.Object{
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
			APIVersion: clusterAPIVersionNamev1,
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterNameDefault,
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
					clusterAPIGroupNameSubresources,
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
			APIVersion: clusterAPIVersionNamev1,
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterNameDefault,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
			Annotations: map[string]string{
				"rbac.authorization.kubernetes.io/autoupdate": "true",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: clusterAPIVersionName,
			Kind:     "ClusterRole",
			Name:     clusterNameDefault,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "Group",
				APIGroup: clusterAPIVersionName,
				Name:     "system:authenticated",
			},
			{
				Kind:     "Group",
				APIGroup: clusterAPIVersionName,
				Name:     "system:unauthenticated",
			},
		},
	}
}

func newAdminClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: clusterAPIVersionNamev1,
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
					clusterAPIGroupNameSubresources,
				},
				Resources: []string{
					"virtualmachineinstances/console",
					"virtualmachineinstances/vnc",
					clusterVMInstancesGuestOSInfo,
					clusterVMInstancesFileSysList,
					clusterVMInstancesUserList,
				},
				Verbs: []string{
					"get",
				},
			},
			{
				APIGroups: []string{
					clusterAPIGroupNameSubresources,
				},
				Resources: []string{
					"virtualmachineinstances/pause",
					"virtualmachineinstances/unpause",
					"virtualmachineinstances/addvolume",
					"virtualmachineinstances/removevolume",
					"virtualmachineinstances/freeze",
					"virtualmachineinstances/unfreeze",
					"virtualmachineinstances/softreboot",
				},
				Verbs: []string{
					"update",
				},
			},
			{
				APIGroups: []string{
					clusterAPIGroupNameSubresources,
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
					clusterAPIGroupName,
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
					clusterAPIGroupNameSnapshot,
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
			{
				APIGroups: []string{
					clusterAPIGroupNameFlavor,
				},
				Resources: []string{
					"virtualmachineflavors",
					"virtualmachineclusterflavors",
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch", "deletecollection",
				},
			},
			{
				APIGroups: []string{
					clusterAPIGroupNamePool,
				},
				Resources: []string{
					"virtualmachinepools",
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch", "deletecollection",
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

func newEditClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: clusterAPIVersionNamev1,
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
					clusterAPIGroupNameSubresources,
				},
				Resources: []string{
					"virtualmachineinstances/console",
					"virtualmachineinstances/vnc",
					clusterVMInstancesGuestOSInfo,
					clusterVMInstancesFileSysList,
					clusterVMInstancesUserList,
				},
				Verbs: []string{
					"get",
				},
			},
			{
				APIGroups: []string{
					clusterAPIGroupNameSubresources,
				},
				Resources: []string{
					"virtualmachineinstances/pause",
					"virtualmachineinstances/unpause",
					"virtualmachineinstances/addvolume",
					"virtualmachineinstances/removevolume",
					"virtualmachineinstances/freeze",
					"virtualmachineinstances/unfreeze",
					"virtualmachineinstances/softreboot",
				},
				Verbs: []string{
					"update",
				},
			},
			{
				APIGroups: []string{
					clusterAPIGroupNameSubresources,
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
					clusterAPIGroupName,
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
					clusterAPIGroupNameSnapshot,
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
			{
				APIGroups: []string{
					clusterAPIGroupNameFlavor,
				},
				Resources: []string{
					"virtualmachineflavors",
					"virtualmachineclusterflavors",
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					clusterAPIGroupNamePool,
				},
				Resources: []string{
					"virtualmachinepools",
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					clusterAPIGroupName,
				},
				Resources: []string{
					"kubevirts",
				},
				Verbs: []string{
					"get", "list",
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

func newViewClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: clusterAPIVersionNamev1,
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
					clusterAPIGroupNameSubresources,
				},
				Resources: []string{
					clusterVMInstancesGuestOSInfo,
					clusterVMInstancesFileSysList,
					clusterVMInstancesUserList,
				},
				Verbs: []string{
					"get",
				},
			},
			{
				APIGroups: []string{
					clusterAPIGroupName,
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
					clusterAPIGroupNameSnapshot,
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
			{
				APIGroups: []string{
					clusterAPIGroupNameFlavor,
				},
				Resources: []string{
					"virtualmachineflavors",
					"virtualmachineclusterflavors",
				},
				Verbs: []string{
					"get", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					clusterAPIGroupNamePool,
				},
				Resources: []string{
					"virtualmachinepools",
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
		},
	}
}
