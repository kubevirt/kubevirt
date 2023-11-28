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
	"kubevirt.io/api/clone"
	"kubevirt.io/api/export"
	"kubevirt.io/api/pool"
	"kubevirt.io/api/snapshot"

	"kubevirt.io/api/instancetype"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/api/migrations"
)

const (
	NameDefault = "kubevirt.io:default"

	ApiVersion            = "version"
	ApiGuestFs            = "guestfs"
	ApiExpandVmSpec       = "expand-vm-spec"
	ApiKubevirts          = "kubevirts"
	ApiVM                 = "virtualmachines"
	ApiVMInstances        = "virtualmachineinstances"
	ApiVMIPresets         = "virtualmachineinstancepresets"
	ApiVMIReplicasets     = "virtualmachineinstancereplicasets"
	ApiVMIMigrations      = "virtualmachineinstancemigrations"
	ApiVMSnapshots        = "virtualmachinesnapshots"
	ApiVMSnapshotContents = "virtualmachinesnapshotcontents"
	ApiVMRestores         = "virtualmachinerestores"
	ApiVMExports          = "virtualmachineexports"
	ApiVMClones           = "virtualmachineclones"
	ApiVMPools            = "virtualmachinepools"

	ApiVMExpandSpec   = "virtualmachines/expand-spec"
	ApiVMPortForward  = "virtualmachines/portforward"
	ApiVMStart        = "virtualmachines/start"
	ApiVMStop         = "virtualmachines/stop"
	ApiVMRestart      = "virtualmachines/restart"
	ApiVMAddVolume    = "virtualmachines/addvolume"
	ApiVMRemoveVolume = "virtualmachines/removevolume"
	ApiVMMigrate      = "virtualmachines/migrate"
	ApiVMMemoryDump   = "virtualmachines/memorydump"

	ApiVMInstancesConsole                   = "virtualmachineinstances/console"
	ApiVMInstancesVNC                       = "virtualmachineinstances/vnc"
	ApiVMInstancesVNCScreenshot             = "virtualmachineinstances/vnc/screenshot"
	ApiVMInstancesPortForward               = "virtualmachineinstances/portforward"
	ApiVMInstancesPause                     = "virtualmachineinstances/pause"
	ApiVMInstancesUnpause                   = "virtualmachineinstances/unpause"
	ApiVMInstancesAddVolume                 = "virtualmachineinstances/addvolume"
	ApiVMInstancesRemoveVolume              = "virtualmachineinstances/removevolume"
	ApiVMInstancesFreeze                    = "virtualmachineinstances/freeze"
	ApiVMInstancesUnfreeze                  = "virtualmachineinstances/unfreeze"
	ApiVMInstancesSoftReboot                = "virtualmachineinstances/softreboot"
	ApiVMInstancesGuestOSInfo               = "virtualmachineinstances/guestosinfo"
	ApiVMInstancesFileSysList               = "virtualmachineinstances/filesystemlist"
	ApiVMInstancesUserList                  = "virtualmachineinstances/userlist"
	ApiVMInstancesSEVFetchCertChain         = "virtualmachineinstances/sev/fetchcertchain"
	ApiVMInstancesSEVQueryLaunchMeasurement = "virtualmachineinstances/sev/querylaunchmeasurement"
	ApiVMInstancesSEVSetupSession           = "virtualmachineinstances/sev/setupsession"
	ApiVMInstancesSEVInjectLaunchSecret     = "virtualmachineinstances/sev/injectlaunchsecret"
)

func GetAllCluster() []runtime.Object {
	return []runtime.Object{
		newDefaultClusterRole(),
		newDefaultClusterRoleBinding(),
		newAdminClusterRole(),
		newEditClusterRole(),
		newViewClusterRole(),
		newInstancetypeViewClusterRole(),
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
					virtv1.SubresourceGroupName,
				},
				Resources: []string{
					ApiVersion,
					ApiGuestFs,
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
					virtv1.SubresourceGroupName,
				},
				Resources: []string{
					ApiVMInstancesConsole,
					ApiVMInstancesVNC,
					ApiVMInstancesVNCScreenshot,
					ApiVMInstancesPortForward,
					ApiVMInstancesGuestOSInfo,
					ApiVMInstancesFileSysList,
					ApiVMInstancesUserList,
					ApiVMInstancesSEVFetchCertChain,
					ApiVMInstancesSEVQueryLaunchMeasurement,
				},
				Verbs: []string{
					"get",
				},
			},
			{
				APIGroups: []string{
					virtv1.SubresourceGroupName,
				},
				Resources: []string{
					ApiVMInstancesPause,
					ApiVMInstancesUnpause,
					ApiVMInstancesAddVolume,
					ApiVMInstancesRemoveVolume,
					ApiVMInstancesFreeze,
					ApiVMInstancesUnfreeze,
					ApiVMInstancesSoftReboot,
					ApiVMInstancesSEVSetupSession,
					ApiVMInstancesSEVInjectLaunchSecret,
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
					ApiVMExpandSpec,
					ApiVMPortForward,
				},
				Verbs: []string{
					"get",
				},
			},
			{
				APIGroups: []string{
					virtv1.SubresourceGroupName,
				},
				Resources: []string{
					ApiVMStart,
					ApiVMStop,
					ApiVMRestart,
					ApiVMAddVolume,
					ApiVMRemoveVolume,
					ApiVMMigrate,
					ApiVMMemoryDump,
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
					ApiExpandVmSpec,
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
					ApiVM,
					ApiVMInstances,
					ApiVMIPresets,
					ApiVMIReplicasets,
					ApiVMIMigrations,
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch", "deletecollection",
				},
			},
			{
				APIGroups: []string{
					snapshot.GroupName,
				},
				Resources: []string{
					ApiVMSnapshots,
					ApiVMSnapshotContents,
					ApiVMRestores,
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch", "deletecollection",
				},
			},
			{
				APIGroups: []string{
					export.GroupName,
				},
				Resources: []string{
					ApiVMExports,
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch", "deletecollection",
				},
			},
			{
				APIGroups: []string{
					clone.GroupName,
				},
				Resources: []string{
					ApiVMClones,
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch", "deletecollection",
				},
			},
			{
				APIGroups: []string{
					instancetype.GroupName,
				},
				Resources: []string{
					instancetype.PluralResourceName,
					instancetype.ClusterPluralResourceName,
					instancetype.PluralPreferenceResourceName,
					instancetype.ClusterPluralPreferenceResourceName,
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch", "deletecollection",
				},
			},
			{
				APIGroups: []string{
					pool.GroupName,
				},
				Resources: []string{
					ApiVMPools,
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
					virtv1.SubresourceGroupName,
				},
				Resources: []string{
					ApiVMInstancesConsole,
					ApiVMInstancesVNC,
					ApiVMInstancesVNCScreenshot,
					ApiVMInstancesPortForward,
					ApiVMInstancesGuestOSInfo,
					ApiVMInstancesFileSysList,
					ApiVMInstancesUserList,
					ApiVMInstancesSEVFetchCertChain,
					ApiVMInstancesSEVQueryLaunchMeasurement,
				},
				Verbs: []string{
					"get",
				},
			},
			{
				APIGroups: []string{
					virtv1.SubresourceGroupName,
				},
				Resources: []string{
					ApiVMInstancesPause,
					ApiVMInstancesUnpause,
					ApiVMInstancesAddVolume,
					ApiVMInstancesRemoveVolume,
					ApiVMInstancesFreeze,
					ApiVMInstancesUnfreeze,
					ApiVMInstancesSoftReboot,
					ApiVMInstancesSEVSetupSession,
					ApiVMInstancesSEVInjectLaunchSecret,
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
					ApiVMExpandSpec,
					ApiVMPortForward,
				},
				Verbs: []string{
					"get",
				},
			},
			{
				APIGroups: []string{
					virtv1.SubresourceGroupName,
				},
				Resources: []string{
					ApiVMStart,
					ApiVMStop,
					ApiVMRestart,
					ApiVMAddVolume,
					ApiVMRemoveVolume,
					ApiVMMigrate,
					ApiVMMemoryDump,
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
					ApiExpandVmSpec,
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
					ApiVM,
					ApiVMInstances,
					ApiVMIPresets,
					ApiVMIReplicasets,
					ApiVMIMigrations,
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					snapshot.GroupName,
				},
				Resources: []string{
					ApiVMSnapshots,
					ApiVMSnapshotContents,
					ApiVMRestores,
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					export.GroupName,
				},
				Resources: []string{
					ApiVMExports,
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					clone.GroupName,
				},
				Resources: []string{
					ApiVMClones,
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					instancetype.GroupName,
				},
				Resources: []string{
					instancetype.PluralResourceName,
					instancetype.ClusterPluralResourceName,
					instancetype.PluralPreferenceResourceName,
					instancetype.ClusterPluralPreferenceResourceName,
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					pool.GroupName,
				},
				Resources: []string{
					ApiVMPools,
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
					ApiKubevirts,
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
					GroupName,
				},
				Resources: []string{
					ApiKubevirts,
				},
				Verbs: []string{
					"get", "list",
				},
			},
			{
				APIGroups: []string{
					virtv1.SubresourceGroupName,
				},
				Resources: []string{
					ApiVMExpandSpec,
					ApiVMInstancesGuestOSInfo,
					ApiVMInstancesFileSysList,
					ApiVMInstancesUserList,
					ApiVMInstancesSEVFetchCertChain,
					ApiVMInstancesSEVQueryLaunchMeasurement,
				},
				Verbs: []string{
					"get",
				},
			},
			{
				APIGroups: []string{
					virtv1.SubresourceGroupName,
				},
				Resources: []string{
					ApiExpandVmSpec,
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
					ApiVM,
					ApiVMInstances,
					ApiVMIPresets,
					ApiVMIReplicasets,
					ApiVMIMigrations,
				},
				Verbs: []string{
					"get", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					snapshot.GroupName,
				},
				Resources: []string{
					ApiVMSnapshots,
					ApiVMSnapshotContents,
					ApiVMRestores,
				},
				Verbs: []string{
					"get", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					export.GroupName,
				},
				Resources: []string{
					ApiVMExports,
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
					ApiVMClones,
				},
				Verbs: []string{
					"get", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					instancetype.GroupName,
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
					pool.GroupName,
				},
				Resources: []string{
					ApiVMPools,
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
func newInstancetypeViewClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: VersionNamev1,
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "instancetype.kubevirt.io:view",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					instancetype.GroupName,
				},
				Resources: []string{
					instancetype.ClusterPluralResourceName,
					instancetype.ClusterPluralPreferenceResourceName,
				},
				Verbs: []string{
					"get", "list", "watch",
				},
			},
		},
	}
}
