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
 * Copyright The KubeVirt Authors.
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
	defaultClusterRoleName          = "kubevirt.io:default"
	instancetypeViewClusterRoleName = "instancetype.kubevirt.io:view"

	apiVersion            = "version"
	apiGuestFs            = "guestfs"
	apiExpandVmSpec       = "expand-vm-spec"
	apiKubevirts          = "kubevirts"
	apiVM                 = "virtualmachines"
	apiVMInstances        = "virtualmachineinstances"
	apiVMIPresets         = "virtualmachineinstancepresets"
	apiVMIReplicasets     = "virtualmachineinstancereplicasets"
	apiVMIMigrations      = "virtualmachineinstancemigrations"
	apiVMSnapshots        = "virtualmachinesnapshots"
	apiVMSnapshotContents = "virtualmachinesnapshotcontents"
	apiVMRestores         = "virtualmachinerestores"
	apiVMExports          = "virtualmachineexports"
	apiVMClones           = "virtualmachineclones"
	apiVMPools            = "virtualmachinepools"

	apiVMExpandSpec   = "virtualmachines/expand-spec"
	apiVMPortForward  = "virtualmachines/portforward"
	apiVMStart        = "virtualmachines/start"
	apiVMStop         = "virtualmachines/stop"
	apiVMRestart      = "virtualmachines/restart"
	apiVMAddVolume    = "virtualmachines/addvolume"
	apiVMRemoveVolume = "virtualmachines/removevolume"
	apiVMMigrate      = "virtualmachines/migrate"
	apiVMMemoryDump   = "virtualmachines/memorydump"

	apiVMInstancesConsole                   = "virtualmachineinstances/console"
	apiVMInstancesVNC                       = "virtualmachineinstances/vnc"
	apiVMInstancesVNCScreenshot             = "virtualmachineinstances/vnc/screenshot"
	apiVMInstancesPortForward               = "virtualmachineinstances/portforward"
	apiVMInstancesPause                     = "virtualmachineinstances/pause"
	apiVMInstancesUnpause                   = "virtualmachineinstances/unpause"
	apiVMInstancesAddVolume                 = "virtualmachineinstances/addvolume"
	apiVMInstancesRemoveVolume              = "virtualmachineinstances/removevolume"
	apiVMInstancesFreeze                    = "virtualmachineinstances/freeze"
	apiVMInstancesUnfreeze                  = "virtualmachineinstances/unfreeze"
	apiVMInstancesSoftReboot                = "virtualmachineinstances/softreboot"
	apiVMInstancesReset                     = "virtualmachineinstances/reset"
	apiVMInstancesGuestOSInfo               = "virtualmachineinstances/guestosinfo"
	apiVMInstancesFileSysList               = "virtualmachineinstances/filesystemlist"
	apiVMInstancesUserList                  = "virtualmachineinstances/userlist"
	apiVMInstancesSEVFetchCertChain         = "virtualmachineinstances/sev/fetchcertchain"
	apiVMInstancesSEVQueryLaunchMeasurement = "virtualmachineinstances/sev/querylaunchmeasurement"
	apiVMInstancesSEVSetupSession           = "virtualmachineinstances/sev/setupsession"
	apiVMInstancesSEVInjectLaunchSecret     = "virtualmachineinstances/sev/injectlaunchsecret"
	apiVMInstancesUSBRedir                  = "virtualmachineinstances/usbredir"
)

func GetAllCluster() []runtime.Object {
	return []runtime.Object{
		newDefaultClusterRole(),
		newDefaultClusterRoleBinding(),
		newAdminClusterRole(),
		newEditClusterRole(),
		newViewClusterRole(),
		newInstancetypeViewClusterRole(),
		newInstancetypeViewClusterRoleBinding(),
		newMigrateClusterRole(),
	}
}

func newDefaultClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: VersionNamev1,
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultClusterRoleName,
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
					GroupName,
				},
				Resources: []string{
					apiKubevirts,
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
					apiVersion,
					apiGuestFs,
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
			Name: defaultClusterRoleName,
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
			Name:     defaultClusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "Group",
				APIGroup: VersionName,
				Name:     "system:authenticated",
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
					apiVMInstancesConsole,
					apiVMInstancesVNC,
					apiVMInstancesVNCScreenshot,
					apiVMInstancesPortForward,
					apiVMInstancesGuestOSInfo,
					apiVMInstancesFileSysList,
					apiVMInstancesUserList,
					apiVMInstancesSEVFetchCertChain,
					apiVMInstancesSEVQueryLaunchMeasurement,
					apiVMInstancesUSBRedir,
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
					apiVMInstancesPause,
					apiVMInstancesUnpause,
					apiVMInstancesAddVolume,
					apiVMInstancesRemoveVolume,
					apiVMInstancesFreeze,
					apiVMInstancesUnfreeze,
					apiVMInstancesSoftReboot,
					apiVMInstancesReset,
					apiVMInstancesSEVSetupSession,
					apiVMInstancesSEVInjectLaunchSecret,
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
					apiVMExpandSpec,
					apiVMPortForward,
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
					apiVMStart,
					apiVMStop,
					apiVMRestart,
					apiVMAddVolume,
					apiVMRemoveVolume,
					apiVMMemoryDump,
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
					apiExpandVmSpec,
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
					apiVM,
					apiVMInstances,
					apiVMIPresets,
					apiVMIReplicasets,
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch", "deletecollection",
				},
			},
			{
				APIGroups: []string{
					GroupName,
				},
				Resources: []string{
					apiVMIMigrations,
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
					apiVMSnapshots,
					apiVMSnapshotContents,
					apiVMRestores,
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
					apiVMExports,
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
					apiVMClones,
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
					apiVMPools,
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
					apiVMInstancesConsole,
					apiVMInstancesVNC,
					apiVMInstancesVNCScreenshot,
					apiVMInstancesPortForward,
					apiVMInstancesGuestOSInfo,
					apiVMInstancesFileSysList,
					apiVMInstancesUserList,
					apiVMInstancesSEVFetchCertChain,
					apiVMInstancesSEVQueryLaunchMeasurement,
					apiVMInstancesUSBRedir,
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
					apiVMInstancesPause,
					apiVMInstancesUnpause,
					apiVMInstancesAddVolume,
					apiVMInstancesRemoveVolume,
					apiVMInstancesFreeze,
					apiVMInstancesUnfreeze,
					apiVMInstancesSoftReboot,
					apiVMInstancesReset,
					apiVMInstancesSEVSetupSession,
					apiVMInstancesSEVInjectLaunchSecret,
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
					apiVMExpandSpec,
					apiVMPortForward,
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
					apiVMStart,
					apiVMStop,
					apiVMRestart,
					apiVMAddVolume,
					apiVMRemoveVolume,
					apiVMMemoryDump,
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
					apiExpandVmSpec,
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
					apiVM,
					apiVMInstances,
					apiVMIPresets,
					apiVMIReplicasets,
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
					apiVMIMigrations,
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
					apiVMSnapshots,
					apiVMSnapshotContents,
					apiVMRestores,
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
					apiVMExports,
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
					apiVMClones,
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
					apiVMPools,
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
					apiKubevirts,
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

func newMigrateClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: VersionNamev1,
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt.io:migrate",
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
					apiVMMigrate,
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
					apiVMIMigrations,
				},
				Verbs: []string{
					"get", "delete", "create", "update", "patch", "list", "watch", "deletecollection",
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
					apiKubevirts,
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
					apiVMExpandSpec,
					apiVMInstancesGuestOSInfo,
					apiVMInstancesFileSysList,
					apiVMInstancesUserList,
					apiVMInstancesSEVFetchCertChain,
					apiVMInstancesSEVQueryLaunchMeasurement,
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
					apiExpandVmSpec,
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
					apiVM,
					apiVMInstances,
					apiVMIPresets,
					apiVMIReplicasets,
					apiVMIMigrations,
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
					apiVMSnapshots,
					apiVMSnapshotContents,
					apiVMRestores,
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
					apiVMExports,
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
					apiVMClones,
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
					apiVMPools,
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
			Name: instancetypeViewClusterRoleName,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
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

func newInstancetypeViewClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: VersionNamev1,
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: instancetypeViewClusterRoleName,
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
			Name:     instancetypeViewClusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "Group",
				APIGroup: VersionName,
				Name:     "system:authenticated",
			},
		},
	}
}
