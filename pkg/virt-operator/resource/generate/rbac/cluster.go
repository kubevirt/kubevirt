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

	"kubevirt.io/api/flavor"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/api/migrations"
)

const (
	GroupNameSubresources  = "subresources.kubevirt.io"
	GroupNameSnapshot      = "snapshot.kubevirt.io"
	GroupNameFlavor        = "flavor.kubevirt.io"
	GroupNamePool          = "pool.kubevirt.io"
	NameDefault            = "kubevirt.io:default"
	VMInstancesGuestOSInfo = "virtualmachineinstances/guestosinfo"
	VMInstancesFileSysList = "virtualmachineinstances/filesystemlist"
	VMInstancesUserList    = "virtualmachineinstances/userlist"
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
			APIVersion: VersionNamev1,
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: NameDefault,
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
					GroupNameSubresources,
				},
				Resources: []string{
					"version",
					"guestfs",
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
			APIVersion: VersionNamev1,
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: NameDefault,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
			Annotations: map[string]string{
				"rbac.authorization.kubernetes.io/autoupdate": "true",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: VersionName,
			Kind:     "ClusterRole",
			Name:     NameDefault,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "Group",
				APIGroup: VersionName,
				Name:     "system:authenticated",
			},
			{
				Kind:     "Group",
				APIGroup: VersionName,
				Name:     "system:unauthenticated",
			},
		},
	}
}

func newAdminClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: VersionNamev1,
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
					GroupNameSubresources,
				},
				Resources: []string{
					"virtualmachineinstances/console",
					"virtualmachineinstances/vnc",
					VMInstancesGuestOSInfo,
					VMInstancesFileSysList,
					VMInstancesUserList,
				},
				Verbs: []string{
					"get",
				},
			},
			{
				APIGroups: []string{
					GroupNameSubresources,
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
				},
			},
			{
				APIGroups: []string{
					GroupNameSubresources,
				},
				Resources: []string{
					"virtualmachines/start",
					"virtualmachines/stop",
					"virtualmachines/restart",
					"virtualmachines/addvolume",
					"virtualmachines/removevolume",
					"virtualmachines/migrate",
					"virtualmachines/memorydump",
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
					GroupNameSnapshot,
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
					GroupNameFlavor,
				},
				Resources: []string{
					flavor.PluralResourceName,
					flavor.ClusterPluralResourceName,
					flavor.PluralPreferenceResourceName,
					flavor.ClusterPluralPreferenceResourceName,
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch", "deletecollection",
				},
			},
			{
				APIGroups: []string{
					GroupNamePool,
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
			APIVersion: VersionNamev1,
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
					GroupNameSubresources,
				},
				Resources: []string{
					"virtualmachineinstances/console",
					"virtualmachineinstances/vnc",
					VMInstancesGuestOSInfo,
					VMInstancesFileSysList,
					VMInstancesUserList,
				},
				Verbs: []string{
					"get",
				},
			},
			{
				APIGroups: []string{
					GroupNameSubresources,
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
				},
			},
			{
				APIGroups: []string{
					GroupNameSubresources,
				},
				Resources: []string{
					"virtualmachines/start",
					"virtualmachines/stop",
					"virtualmachines/restart",
					"virtualmachines/addvolume",
					"virtualmachines/removevolume",
					"virtualmachines/migrate",
					"virtualmachines/memorydump",
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
					GroupNameSnapshot,
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
					GroupNameFlavor,
				},
				Resources: []string{
					flavor.PluralResourceName,
					flavor.ClusterPluralResourceName,
					flavor.PluralPreferenceResourceName,
					flavor.ClusterPluralPreferenceResourceName,
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					GroupNamePool,
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
					GroupName,
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
			APIVersion: VersionNamev1,
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
					GroupNameSubresources,
				},
				Resources: []string{
					VMInstancesGuestOSInfo,
					VMInstancesFileSysList,
					VMInstancesUserList,
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
					GroupNameSnapshot,
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
					GroupNameFlavor,
				},
				Resources: []string{
					flavor.PluralResourceName,
					flavor.ClusterPluralResourceName,
					flavor.PluralPreferenceResourceName,
					flavor.ClusterPluralPreferenceResourceName,
				},
				Verbs: []string{
					"get", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					GroupNamePool,
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
