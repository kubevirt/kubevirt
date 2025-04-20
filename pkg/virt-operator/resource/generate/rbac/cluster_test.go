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
 */

package rbac

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"kubevirt.io/api/clone"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/export"
	"kubevirt.io/api/instancetype"
	"kubevirt.io/api/migrations"
	"kubevirt.io/api/pool"
	"kubevirt.io/api/snapshot"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
)

var _ = Describe("Cluster role and cluster role bindings", func() {

	Context("GetAllCluster", func() {

		clusterObjects := GetAllCluster()

		It("should not be nil", func() {
			Expect(clusterObjects).ToNot(BeNil())
		})

		Context("default cluster role", func() {

			DescribeTable("should contain rule to", func(apiGroup, resource string, verbs ...string) {
				clusterRole := getObject(clusterObjects, reflect.TypeOf(&rbacv1.ClusterRole{}), defaultClusterRoleName).(*rbacv1.ClusterRole)
				Expect(clusterRole).ToNot(BeNil())
				expectExactRuleExists(clusterRole.Rules, apiGroup, resource, verbs...)

			},
				Entry(fmt.Sprintf("get and list %s/%s", GroupName, apiKubevirts), GroupName, apiKubevirts, "get", "list"),
				Entry(fmt.Sprintf("get and list %s/%s", virtv1.SubresourceGroupName, apiVersion), virtv1.SubresourceGroupName, apiVersion, "get", "list"),
				Entry(fmt.Sprintf("get and list %s/%s", virtv1.SubresourceGroupName, apiGuestFs), virtv1.SubresourceGroupName, apiGuestFs, "get", "list"),
			)
		})

		Context("default cluster role binding", func() {

			It("should contain RoleRef to default cluster role", func() {
				clusterRoleBinding := getObject(clusterObjects, reflect.TypeOf(&rbacv1.ClusterRoleBinding{}), defaultClusterRoleName).(*rbacv1.ClusterRoleBinding)
				Expect(clusterRoleBinding).ToNot(BeNil())
				expectRoleRefToBe(clusterRoleBinding.RoleRef, "ClusterRole", defaultClusterRoleName)
			})

			DescribeTable("should contain subject to refer", func(kind, name string, verbs ...string) {
				clusterRoleBinding := getObject(clusterObjects, reflect.TypeOf(&rbacv1.ClusterRoleBinding{}), defaultClusterRoleName).(*rbacv1.ClusterRoleBinding)
				Expect(clusterRoleBinding).ToNot(BeNil())
				expectSubjectExists(clusterRoleBinding.Subjects, kind, name)
			},
				Entry("system:authenticated", "Group", "system:authenticated"),
			)
		})

		Context("admin cluster role", func() {

			DescribeTable("should contain rule to", func(apiGroup, resource string, verbs ...string) {
				clusterRole := getObject(clusterObjects, reflect.TypeOf(&rbacv1.ClusterRole{}), "kubevirt.io:admin").(*rbacv1.ClusterRole)
				Expect(clusterRole).ToNot(BeNil())
				expectExactRuleExists(clusterRole.Rules, apiGroup, resource, verbs...)
			},
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesConsole), virtv1.SubresourceGroupName, apiVMInstancesConsole, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesVNC), virtv1.SubresourceGroupName, apiVMInstancesVNC, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesVNCScreenshot), virtv1.SubresourceGroupName, apiVMInstancesVNCScreenshot, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesPortForward), virtv1.SubresourceGroupName, apiVMInstancesPortForward, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesGuestOSInfo), virtv1.SubresourceGroupName, apiVMInstancesGuestOSInfo, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesFileSysList), virtv1.SubresourceGroupName, apiVMInstancesFileSysList, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesUserList), virtv1.SubresourceGroupName, apiVMInstancesUserList, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesSEVFetchCertChain), virtv1.SubresourceGroupName, apiVMInstancesSEVFetchCertChain, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesSEVQueryLaunchMeasurement), virtv1.SubresourceGroupName, apiVMInstancesSEVQueryLaunchMeasurement, "get"),

				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesPause), virtv1.SubresourceGroupName, apiVMInstancesPause, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesUnpause), virtv1.SubresourceGroupName, apiVMInstancesUnpause, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesAddVolume), virtv1.SubresourceGroupName, apiVMInstancesAddVolume, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesRemoveVolume), virtv1.SubresourceGroupName, apiVMInstancesRemoveVolume, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesFreeze), virtv1.SubresourceGroupName, apiVMInstancesFreeze, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesUnfreeze), virtv1.SubresourceGroupName, apiVMInstancesUnfreeze, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesReset), virtv1.SubresourceGroupName, apiVMInstancesReset, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesSoftReboot), virtv1.SubresourceGroupName, apiVMInstancesSoftReboot, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesSEVSetupSession), virtv1.SubresourceGroupName, apiVMInstancesSEVSetupSession, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesSEVInjectLaunchSecret), virtv1.SubresourceGroupName, apiVMInstancesSEVInjectLaunchSecret, "update"),

				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMExpandSpec), virtv1.SubresourceGroupName, apiVMExpandSpec, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMPortForward), virtv1.SubresourceGroupName, apiVMPortForward, "get"),

				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMStart), virtv1.SubresourceGroupName, apiVMStart, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMStop), virtv1.SubresourceGroupName, apiVMInstancesSEVInjectLaunchSecret, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMRestart), virtv1.SubresourceGroupName, apiVMStop, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMAddVolume), virtv1.SubresourceGroupName, apiVMRestart, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMRemoveVolume), virtv1.SubresourceGroupName, apiVMAddVolume, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMMemoryDump), virtv1.SubresourceGroupName, apiVMMemoryDump, "update"),

				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiExpandVmSpec), virtv1.SubresourceGroupName, apiExpandVmSpec, "update"),

				Entry(fmt.Sprintf("do all operations to %s/%s", GroupName, apiVM), GroupName, apiVM, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),
				Entry(fmt.Sprintf("do all operations to %s/%s", GroupName, apiVMInstances), GroupName, apiVMInstances, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),
				Entry(fmt.Sprintf("do all operations to %s/%s", GroupName, apiVMIPresets), GroupName, apiVMIPresets, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),
				Entry(fmt.Sprintf("do all operations to %s/%s", GroupName, apiVMIReplicasets), GroupName, apiVMIReplicasets, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),

				Entry(fmt.Sprintf("do all operations to %s/%s", snapshot.GroupName, apiVMSnapshots), snapshot.GroupName, apiVMSnapshots, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),
				Entry(fmt.Sprintf("do all operations to %s/%s", snapshot.GroupName, apiVMSnapshotContents), snapshot.GroupName, apiVMSnapshotContents, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),
				Entry(fmt.Sprintf("do all operations to %s/%s", snapshot.GroupName, apiVMRestores), snapshot.GroupName, apiVMRestores, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),

				Entry(fmt.Sprintf("do all operations to %s/%s", export.GroupName, apiVMExports), export.GroupName, apiVMExports, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),

				Entry(fmt.Sprintf("do all operations to %s/%s", clone.GroupName, apiVMClones), clone.GroupName, apiVMClones, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),

				Entry(fmt.Sprintf("do all operations to %s/%s", instancetype.GroupName, instancetype.PluralResourceName), instancetype.GroupName, instancetype.PluralResourceName, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),
				Entry(fmt.Sprintf("do all operations to %s/%s", instancetype.GroupName, instancetype.ClusterPluralResourceName), instancetype.GroupName, instancetype.ClusterPluralResourceName, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),
				Entry(fmt.Sprintf("do all operations to %s/%s", instancetype.GroupName, instancetype.PluralPreferenceResourceName), instancetype.GroupName, instancetype.PluralPreferenceResourceName, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),
				Entry(fmt.Sprintf("do all operations to %s/%s", instancetype.GroupName, instancetype.ClusterPluralPreferenceResourceName), instancetype.GroupName, instancetype.ClusterPluralPreferenceResourceName, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),

				Entry(fmt.Sprintf("do all operations to %s/%s", pool.GroupName, apiVMPools), pool.GroupName, apiVMPools, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),

				Entry(fmt.Sprintf("get, list, watch %s/%s", migrations.GroupName, migrations.ResourceMigrationPolicies), migrations.GroupName, migrations.ResourceMigrationPolicies, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", GroupName, apiVMIMigrations), GroupName, apiVMIMigrations, "get", "list", "watch"),
			)
		})

		Context("edit cluster role", func() {

			DescribeTable("should contain rule to", func(apiGroup, resource string, verbs ...string) {
				clusterRole := getObject(clusterObjects, reflect.TypeOf(&rbacv1.ClusterRole{}), "kubevirt.io:edit").(*rbacv1.ClusterRole)
				Expect(clusterRole).ToNot(BeNil())
				expectExactRuleExists(clusterRole.Rules, apiGroup, resource, verbs...)
			},
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesConsole), virtv1.SubresourceGroupName, apiVMInstancesConsole, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesVNC), virtv1.SubresourceGroupName, apiVMInstancesVNC, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesVNCScreenshot), virtv1.SubresourceGroupName, apiVMInstancesVNCScreenshot, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesPortForward), virtv1.SubresourceGroupName, apiVMInstancesPortForward, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesGuestOSInfo), virtv1.SubresourceGroupName, apiVMInstancesGuestOSInfo, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesFileSysList), virtv1.SubresourceGroupName, apiVMInstancesFileSysList, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesUserList), virtv1.SubresourceGroupName, apiVMInstancesUserList, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesSEVFetchCertChain), virtv1.SubresourceGroupName, apiVMInstancesSEVFetchCertChain, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesSEVQueryLaunchMeasurement), virtv1.SubresourceGroupName, apiVMInstancesSEVQueryLaunchMeasurement, "get"),

				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesPause), virtv1.SubresourceGroupName, apiVMInstancesPause, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesUnpause), virtv1.SubresourceGroupName, apiVMInstancesUnpause, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesAddVolume), virtv1.SubresourceGroupName, apiVMInstancesAddVolume, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesRemoveVolume), virtv1.SubresourceGroupName, apiVMInstancesRemoveVolume, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesFreeze), virtv1.SubresourceGroupName, apiVMInstancesFreeze, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesUnfreeze), virtv1.SubresourceGroupName, apiVMInstancesUnfreeze, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesReset), virtv1.SubresourceGroupName, apiVMInstancesReset, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesSoftReboot), virtv1.SubresourceGroupName, apiVMInstancesSoftReboot, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesSEVSetupSession), virtv1.SubresourceGroupName, apiVMInstancesSEVSetupSession, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMInstancesSEVInjectLaunchSecret), virtv1.SubresourceGroupName, apiVMInstancesSEVInjectLaunchSecret, "update"),

				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMExpandSpec), virtv1.SubresourceGroupName, apiVMExpandSpec, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMPortForward), virtv1.SubresourceGroupName, apiVMPortForward, "get"),

				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMStart), virtv1.SubresourceGroupName, apiVMStart, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMStop), virtv1.SubresourceGroupName, apiVMInstancesSEVInjectLaunchSecret, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMRestart), virtv1.SubresourceGroupName, apiVMStop, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMAddVolume), virtv1.SubresourceGroupName, apiVMRestart, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMRemoveVolume), virtv1.SubresourceGroupName, apiVMAddVolume, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMMemoryDump), virtv1.SubresourceGroupName, apiVMMemoryDump, "update"),

				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiExpandVmSpec), virtv1.SubresourceGroupName, apiExpandVmSpec, "update"),

				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", GroupName, apiVM), GroupName, apiVM, "get", "delete", "create", "update", "patch", "list", "watch"),
				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", GroupName, apiVMInstances), GroupName, apiVMInstances, "get", "delete", "create", "update", "patch", "list", "watch"),
				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", GroupName, apiVMIPresets), GroupName, apiVMIPresets, "get", "delete", "create", "update", "patch", "list", "watch"),
				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", GroupName, apiVMIReplicasets), GroupName, apiVMIReplicasets, "get", "delete", "create", "update", "patch", "list", "watch"),

				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", snapshot.GroupName, apiVMSnapshots), snapshot.GroupName, apiVMSnapshots, "get", "delete", "create", "update", "patch", "list", "watch"),
				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", snapshot.GroupName, apiVMSnapshotContents), snapshot.GroupName, apiVMSnapshotContents, "get", "delete", "create", "update", "patch", "list", "watch"),
				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", snapshot.GroupName, apiVMRestores), snapshot.GroupName, apiVMRestores, "get", "delete", "create", "update", "patch", "list", "watch"),

				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", export.GroupName, apiVMExports), export.GroupName, apiVMExports, "get", "delete", "create", "update", "patch", "list", "watch"),

				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", clone.GroupName, apiVMClones), clone.GroupName, apiVMClones, "get", "delete", "create", "update", "patch", "list", "watch"),

				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", instancetype.GroupName, instancetype.PluralResourceName), instancetype.GroupName, instancetype.PluralResourceName, "get", "delete", "create", "update", "patch", "list", "watch"),
				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", instancetype.GroupName, instancetype.ClusterPluralResourceName), instancetype.GroupName, instancetype.ClusterPluralResourceName, "get", "delete", "create", "update", "patch", "list", "watch"),
				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", instancetype.GroupName, instancetype.PluralPreferenceResourceName), instancetype.GroupName, instancetype.PluralPreferenceResourceName, "get", "delete", "create", "update", "patch", "list", "watch"),
				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", instancetype.GroupName, instancetype.ClusterPluralPreferenceResourceName), instancetype.GroupName, instancetype.ClusterPluralPreferenceResourceName, "get", "delete", "create", "update", "patch", "list", "watch"),

				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", pool.GroupName, apiVMPools), pool.GroupName, apiVMPools, "get", "delete", "create", "update", "patch", "list", "watch"),

				Entry(fmt.Sprintf("get, list %s/%s", GroupName, apiKubevirts), GroupName, apiKubevirts, "get", "list"),

				Entry(fmt.Sprintf("get, list, watch %s/%s", migrations.GroupName, migrations.ResourceMigrationPolicies), migrations.GroupName, migrations.ResourceMigrationPolicies, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", GroupName, apiVMIMigrations), GroupName, apiVMIMigrations, "get", "list", "watch"),
			)
		})

		Context("migrate cluster role", func() {

			DescribeTable("should contain rule to", func(apiGroup, resource string, verbs ...string) {
				clusterRole := getObject(clusterObjects, reflect.TypeOf(&rbacv1.ClusterRole{}), "kubevirt.io:migrate").(*rbacv1.ClusterRole)
				Expect(clusterRole).ToNot(BeNil())
				expectExactRuleExists(clusterRole.Rules, apiGroup, resource, verbs...)
			},
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiVMMigrate), virtv1.SubresourceGroupName, apiVMMigrate, "update"),
				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", GroupName, apiVMIMigrations), GroupName, apiVMIMigrations, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),
			)
		})

		Context("view cluster role", func() {

			DescribeTable("should contain rule to", func(apiGroup, resource string, verbs ...string) {
				clusterRole := getObject(clusterObjects, reflect.TypeOf(&rbacv1.ClusterRole{}), "kubevirt.io:view").(*rbacv1.ClusterRole)
				Expect(clusterRole).ToNot(BeNil())
				expectExactRuleExists(clusterRole.Rules, apiGroup, resource, verbs...)
			},
				Entry(fmt.Sprintf("get, list %s/%s", GroupName, apiKubevirts), GroupName, apiKubevirts, "get", "list"),

				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMExpandSpec), virtv1.SubresourceGroupName, apiVMExpandSpec, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesGuestOSInfo), virtv1.SubresourceGroupName, apiVMInstancesGuestOSInfo, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesFileSysList), virtv1.SubresourceGroupName, apiVMInstancesFileSysList, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesUserList), virtv1.SubresourceGroupName, apiVMInstancesUserList, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesSEVFetchCertChain), virtv1.SubresourceGroupName, apiVMInstancesSEVFetchCertChain, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, apiVMInstancesSEVQueryLaunchMeasurement), virtv1.SubresourceGroupName, apiVMInstancesSEVQueryLaunchMeasurement, "get"),

				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, apiExpandVmSpec), virtv1.SubresourceGroupName, apiExpandVmSpec, "update"),

				Entry(fmt.Sprintf("get, list, watch %s/%s", GroupName, apiVM), GroupName, apiVM, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", GroupName, apiVMInstances), GroupName, apiVMInstances, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", GroupName, apiVMIPresets), GroupName, apiVMIPresets, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", GroupName, apiVMIReplicasets), GroupName, apiVMIReplicasets, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", GroupName, apiVMIMigrations), GroupName, apiVMIMigrations, "get", "list", "watch"),

				Entry(fmt.Sprintf("get, list, watch %s/%s", snapshot.GroupName, apiVMSnapshots), snapshot.GroupName, apiVMSnapshots, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", snapshot.GroupName, apiVMSnapshotContents), snapshot.GroupName, apiVMSnapshotContents, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", snapshot.GroupName, apiVMRestores), snapshot.GroupName, apiVMRestores, "get", "list", "watch"),

				Entry(fmt.Sprintf("get, list, watch %s/%s", export.GroupName, apiVMExports), export.GroupName, apiVMExports, "get", "list", "watch"),

				Entry(fmt.Sprintf("get, list, watch %s/%s", clone.GroupName, apiVMClones), clone.GroupName, apiVMClones, "get", "list", "watch"),

				Entry(fmt.Sprintf("get, list, watch %s/%s", instancetype.GroupName, instancetype.PluralResourceName), instancetype.GroupName, instancetype.PluralResourceName, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", instancetype.GroupName, instancetype.ClusterPluralResourceName), instancetype.GroupName, instancetype.ClusterPluralResourceName, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", instancetype.GroupName, instancetype.PluralPreferenceResourceName), instancetype.GroupName, instancetype.PluralPreferenceResourceName, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", instancetype.GroupName, instancetype.ClusterPluralPreferenceResourceName), instancetype.GroupName, instancetype.ClusterPluralPreferenceResourceName, "get", "list", "watch"),

				Entry(fmt.Sprintf("get, list, watch %s/%s", pool.GroupName, apiVMPools), pool.GroupName, apiVMPools, "get", "list", "watch"),

				Entry(fmt.Sprintf("get, list, watch %s/%s", migrations.GroupName, migrations.ResourceMigrationPolicies), migrations.GroupName, migrations.ResourceMigrationPolicies, "get", "list", "watch"),
			)
		})

		Context("instance type view cluster role", func() {

			DescribeTable("should contain rule to", func(apiGroup, resource string, verbs ...string) {
				clusterRole := getObject(clusterObjects, reflect.TypeOf(&rbacv1.ClusterRole{}), instancetypeViewClusterRoleName).(*rbacv1.ClusterRole)
				Expect(clusterRole).ToNot(BeNil())
				expectExactRuleExists(clusterRole.Rules, apiGroup, resource, verbs...)
			},
				Entry(fmt.Sprintf("get, list, watch %s/%s", instancetype.GroupName, instancetype.ClusterPluralResourceName), instancetype.GroupName, instancetype.ClusterPluralResourceName, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", instancetype.GroupName, instancetype.ClusterPluralPreferenceResourceName), instancetype.GroupName, instancetype.ClusterPluralPreferenceResourceName, "get", "list", "watch"),
			)
		})

		Context("instance type view cluster role binding", func() {

			It("should contain RoleRef to instancetype view cluster role", func() {
				clusterRoleBinding := getObject(clusterObjects, reflect.TypeOf(&rbacv1.ClusterRoleBinding{}), instancetypeViewClusterRoleName).(*rbacv1.ClusterRoleBinding)
				Expect(clusterRoleBinding).ToNot(BeNil())
				expectRoleRefToBe(clusterRoleBinding.RoleRef, "ClusterRole", instancetypeViewClusterRoleName)
			})

			DescribeTable("should contain subject to refer", func(kind, name string, verbs ...string) {
				clusterRoleBinding := getObject(clusterObjects, reflect.TypeOf(&rbacv1.ClusterRoleBinding{}), instancetypeViewClusterRoleName).(*rbacv1.ClusterRoleBinding)
				Expect(clusterRoleBinding).ToNot(BeNil())
				expectSubjectExists(clusterRoleBinding.Subjects, kind, name)
			},
				Entry("system:authenticated", "Group", "system:authenticated"),
			)
		})
	})

})

func getObject(items []runtime.Object, tp reflect.Type, name string) runtime.Object {
	for _, item := range items {
		typeOf := reflect.TypeOf(item)
		if typeOf == tp {
			unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(item)
			if err != nil {
				continue
			}
			str, _, _ := unstructured.NestedString(unstructuredObj, "metadata", "name")
			if str == name {
				return item
			}
		}
	}
	return nil
}

func expectExactRuleExists(rules []rbacv1.PolicyRule, apiGroup, resource string, verbs ...string) {
	for _, rule := range rules {
		if contains(rule.APIGroups, apiGroup) &&
			contains(rule.Resources, resource) &&
			len(rule.Verbs) == len(verbs) {
			for _, verb := range verbs {
				if contains(rule.Verbs, verb) {
					return
				}
			}
		}
	}

	Fail(fmt.Sprintf("Rule (apiGroup: %s, resource: %s, verbs: %v) not found", apiGroup, resource, verbs))
}

func expectSubjectExists(subjects []rbacv1.Subject, kind, name string) {
	for _, subject := range subjects {
		if subject.Kind == kind &&
			subject.Name == name {
			return
		}
	}

	Fail(fmt.Sprintf("Subject (kind: %s, name: %s) not found", kind, name))
}

func expectRoleRefToBe(roleRef rbacv1.RoleRef, kind, name string) {
	Expect(roleRef.Kind).To(BeEquivalentTo(kind))
	Expect(roleRef.Name).To(BeEquivalentTo(name))
}

func contains[T comparable](elems []T, v T) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}
