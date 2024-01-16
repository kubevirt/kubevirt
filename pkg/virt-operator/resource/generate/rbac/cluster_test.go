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
 * Copyright the KubeVirt Authors.
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
				clusterRole := getObject(clusterObjects, reflect.TypeOf(&rbacv1.ClusterRole{}), NameDefault).(*rbacv1.ClusterRole)
				Expect(clusterRole).ToNot(BeNil())
				expectExactRuleExists(clusterRole.Rules, apiGroup, resource, verbs...)

			},
				Entry(fmt.Sprintf("get and list %s/%s", virtv1.SubresourceGroupName, ApiVersion), virtv1.SubresourceGroupName, ApiVersion, "get", "list"),
				Entry(fmt.Sprintf("get and list %s/%s", virtv1.SubresourceGroupName, ApiGuestFs), virtv1.SubresourceGroupName, ApiGuestFs, "get", "list"),
			)
		})

		Context("default cluster role binding", func() {

			It("should contain RoleRef to default cluster role", func() {
				clusterRoleBinding := getObject(clusterObjects, reflect.TypeOf(&rbacv1.ClusterRoleBinding{}), NameDefault).(*rbacv1.ClusterRoleBinding)
				Expect(clusterRoleBinding).ToNot(BeNil())
				expectRoleRefToBe(clusterRoleBinding.RoleRef, "ClusterRole", NameDefault)
			})

			DescribeTable("should contain subject to refer", func(kind, name string, verbs ...string) {
				clusterRoleBinding := getObject(clusterObjects, reflect.TypeOf(&rbacv1.ClusterRoleBinding{}), NameDefault).(*rbacv1.ClusterRoleBinding)
				Expect(clusterRoleBinding).ToNot(BeNil())
				expectSubjectExists(clusterRoleBinding.Subjects, kind, name)
			},
				Entry("system:authenticated", "Group", "system:authenticated"),
				Entry("system:unauthenticated", "Group", "system:unauthenticated"),
			)
		})

		Context("admin cluster role", func() {

			DescribeTable("should contain rule to", func(apiGroup, resource string, verbs ...string) {
				clusterRole := getObject(clusterObjects, reflect.TypeOf(&rbacv1.ClusterRole{}), "kubevirt.io:admin").(*rbacv1.ClusterRole)
				Expect(clusterRole).ToNot(BeNil())
				expectExactRuleExists(clusterRole.Rules, apiGroup, resource, verbs...)
			},
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesConsole), virtv1.SubresourceGroupName, ApiVMInstancesConsole, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesVNC), virtv1.SubresourceGroupName, ApiVMInstancesVNC, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesVNCScreenshot), virtv1.SubresourceGroupName, ApiVMInstancesVNCScreenshot, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesPortForward), virtv1.SubresourceGroupName, ApiVMInstancesPortForward, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesGuestOSInfo), virtv1.SubresourceGroupName, ApiVMInstancesGuestOSInfo, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesFileSysList), virtv1.SubresourceGroupName, ApiVMInstancesFileSysList, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesUserList), virtv1.SubresourceGroupName, ApiVMInstancesUserList, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesSEVFetchCertChain), virtv1.SubresourceGroupName, ApiVMInstancesSEVFetchCertChain, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesSEVQueryLaunchMeasurement), virtv1.SubresourceGroupName, ApiVMInstancesSEVQueryLaunchMeasurement, "get"),

				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesPause), virtv1.SubresourceGroupName, ApiVMInstancesPause, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesUnpause), virtv1.SubresourceGroupName, ApiVMInstancesUnpause, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesAddVolume), virtv1.SubresourceGroupName, ApiVMInstancesAddVolume, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesRemoveVolume), virtv1.SubresourceGroupName, ApiVMInstancesRemoveVolume, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesFreeze), virtv1.SubresourceGroupName, ApiVMInstancesFreeze, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesUnfreeze), virtv1.SubresourceGroupName, ApiVMInstancesUnfreeze, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesSoftReboot), virtv1.SubresourceGroupName, ApiVMInstancesSoftReboot, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesSEVSetupSession), virtv1.SubresourceGroupName, ApiVMInstancesSEVSetupSession, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesSEVInjectLaunchSecret), virtv1.SubresourceGroupName, ApiVMInstancesSEVInjectLaunchSecret, "update"),

				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMExpandSpec), virtv1.SubresourceGroupName, ApiVMExpandSpec, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMPortForward), virtv1.SubresourceGroupName, ApiVMPortForward, "get"),

				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMStart), virtv1.SubresourceGroupName, ApiVMStart, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMStop), virtv1.SubresourceGroupName, ApiVMInstancesSEVInjectLaunchSecret, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMRestart), virtv1.SubresourceGroupName, ApiVMStop, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMAddVolume), virtv1.SubresourceGroupName, ApiVMRestart, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMRemoveVolume), virtv1.SubresourceGroupName, ApiVMAddVolume, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMMigrate), virtv1.SubresourceGroupName, ApiVMMigrate, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMMemoryDump), virtv1.SubresourceGroupName, ApiVMMemoryDump, "update"),

				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiExpandVmSpec), virtv1.SubresourceGroupName, ApiExpandVmSpec, "update"),

				Entry(fmt.Sprintf("do all operations to %s/%s", GroupName, ApiVM), GroupName, ApiVM, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),
				Entry(fmt.Sprintf("do all operations to %s/%s", GroupName, ApiVMInstances), GroupName, ApiVMInstances, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),
				Entry(fmt.Sprintf("do all operations to %s/%s", GroupName, ApiVMIPresets), GroupName, ApiVMIPresets, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),
				Entry(fmt.Sprintf("do all operations to %s/%s", GroupName, ApiVMIReplicasets), GroupName, ApiVMIReplicasets, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),
				Entry(fmt.Sprintf("do all operations to %s/%s", GroupName, ApiVMIMigrations), GroupName, ApiVMIMigrations, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),

				Entry(fmt.Sprintf("do all operations to %s/%s", snapshot.GroupName, ApiVMSnapshots), snapshot.GroupName, ApiVMSnapshots, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),
				Entry(fmt.Sprintf("do all operations to %s/%s", snapshot.GroupName, ApiVMSnapshotContents), snapshot.GroupName, ApiVMSnapshotContents, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),
				Entry(fmt.Sprintf("do all operations to %s/%s", snapshot.GroupName, ApiVMRestores), snapshot.GroupName, ApiVMRestores, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),

				Entry(fmt.Sprintf("do all operations to %s/%s", export.GroupName, ApiVMExports), export.GroupName, ApiVMExports, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),

				Entry(fmt.Sprintf("do all operations to %s/%s", clone.GroupName, ApiVMClones), clone.GroupName, ApiVMClones, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),

				Entry(fmt.Sprintf("do all operations to %s/%s", instancetype.GroupName, instancetype.PluralResourceName), instancetype.GroupName, instancetype.PluralResourceName, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),
				Entry(fmt.Sprintf("do all operations to %s/%s", instancetype.GroupName, instancetype.ClusterPluralResourceName), instancetype.GroupName, instancetype.ClusterPluralResourceName, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),
				Entry(fmt.Sprintf("do all operations to %s/%s", instancetype.GroupName, instancetype.PluralPreferenceResourceName), instancetype.GroupName, instancetype.PluralPreferenceResourceName, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),
				Entry(fmt.Sprintf("do all operations to %s/%s", instancetype.GroupName, instancetype.ClusterPluralPreferenceResourceName), instancetype.GroupName, instancetype.ClusterPluralPreferenceResourceName, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),

				Entry(fmt.Sprintf("do all operations to %s/%s", pool.GroupName, ApiVMPools), pool.GroupName, ApiVMPools, "get", "delete", "create", "update", "patch", "list", "watch", "deletecollection"),

				Entry(fmt.Sprintf("get, list, watch %s/%s", migrations.GroupName, migrations.ResourceMigrationPolicies), migrations.GroupName, migrations.ResourceMigrationPolicies, "get", "list", "watch"),
			)
		})

		Context("edit cluster role", func() {

			DescribeTable("should contain rule to", func(apiGroup, resource string, verbs ...string) {
				clusterRole := getObject(clusterObjects, reflect.TypeOf(&rbacv1.ClusterRole{}), "kubevirt.io:edit").(*rbacv1.ClusterRole)
				Expect(clusterRole).ToNot(BeNil())
				expectExactRuleExists(clusterRole.Rules, apiGroup, resource, verbs...)
			},
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesConsole), virtv1.SubresourceGroupName, ApiVMInstancesConsole, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesVNC), virtv1.SubresourceGroupName, ApiVMInstancesVNC, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesVNCScreenshot), virtv1.SubresourceGroupName, ApiVMInstancesVNCScreenshot, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesPortForward), virtv1.SubresourceGroupName, ApiVMInstancesPortForward, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesGuestOSInfo), virtv1.SubresourceGroupName, ApiVMInstancesGuestOSInfo, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesFileSysList), virtv1.SubresourceGroupName, ApiVMInstancesFileSysList, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesUserList), virtv1.SubresourceGroupName, ApiVMInstancesUserList, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesSEVFetchCertChain), virtv1.SubresourceGroupName, ApiVMInstancesSEVFetchCertChain, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesSEVQueryLaunchMeasurement), virtv1.SubresourceGroupName, ApiVMInstancesSEVQueryLaunchMeasurement, "get"),

				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesPause), virtv1.SubresourceGroupName, ApiVMInstancesPause, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesUnpause), virtv1.SubresourceGroupName, ApiVMInstancesUnpause, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesAddVolume), virtv1.SubresourceGroupName, ApiVMInstancesAddVolume, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesRemoveVolume), virtv1.SubresourceGroupName, ApiVMInstancesRemoveVolume, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesFreeze), virtv1.SubresourceGroupName, ApiVMInstancesFreeze, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesUnfreeze), virtv1.SubresourceGroupName, ApiVMInstancesUnfreeze, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesSoftReboot), virtv1.SubresourceGroupName, ApiVMInstancesSoftReboot, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesSEVSetupSession), virtv1.SubresourceGroupName, ApiVMInstancesSEVSetupSession, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesSEVInjectLaunchSecret), virtv1.SubresourceGroupName, ApiVMInstancesSEVInjectLaunchSecret, "update"),

				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMExpandSpec), virtv1.SubresourceGroupName, ApiVMExpandSpec, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMPortForward), virtv1.SubresourceGroupName, ApiVMPortForward, "get"),

				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMStart), virtv1.SubresourceGroupName, ApiVMStart, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMStop), virtv1.SubresourceGroupName, ApiVMInstancesSEVInjectLaunchSecret, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMRestart), virtv1.SubresourceGroupName, ApiVMStop, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMAddVolume), virtv1.SubresourceGroupName, ApiVMRestart, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMRemoveVolume), virtv1.SubresourceGroupName, ApiVMAddVolume, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMMigrate), virtv1.SubresourceGroupName, ApiVMMigrate, "update"),
				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiVMMemoryDump), virtv1.SubresourceGroupName, ApiVMMemoryDump, "update"),

				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiExpandVmSpec), virtv1.SubresourceGroupName, ApiExpandVmSpec, "update"),

				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", GroupName, ApiVM), GroupName, ApiVM, "get", "delete", "create", "update", "patch", "list", "watch"),
				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", GroupName, ApiVMInstances), GroupName, ApiVMInstances, "get", "delete", "create", "update", "patch", "list", "watch"),
				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", GroupName, ApiVMIPresets), GroupName, ApiVMIPresets, "get", "delete", "create", "update", "patch", "list", "watch"),
				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", GroupName, ApiVMIReplicasets), GroupName, ApiVMIReplicasets, "get", "delete", "create", "update", "patch", "list", "watch"),
				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", GroupName, ApiVMIMigrations), GroupName, ApiVMIMigrations, "get", "delete", "create", "update", "patch", "list", "watch"),

				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", snapshot.GroupName, ApiVMSnapshots), snapshot.GroupName, ApiVMSnapshots, "get", "delete", "create", "update", "patch", "list", "watch"),
				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", snapshot.GroupName, ApiVMSnapshotContents), snapshot.GroupName, ApiVMSnapshotContents, "get", "delete", "create", "update", "patch", "list", "watch"),
				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", snapshot.GroupName, ApiVMRestores), snapshot.GroupName, ApiVMRestores, "get", "delete", "create", "update", "patch", "list", "watch"),

				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", export.GroupName, ApiVMExports), export.GroupName, ApiVMExports, "get", "delete", "create", "update", "patch", "list", "watch"),

				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", clone.GroupName, ApiVMClones), clone.GroupName, ApiVMClones, "get", "delete", "create", "update", "patch", "list", "watch"),

				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", instancetype.GroupName, instancetype.PluralResourceName), instancetype.GroupName, instancetype.PluralResourceName, "get", "delete", "create", "update", "patch", "list", "watch"),
				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", instancetype.GroupName, instancetype.ClusterPluralResourceName), instancetype.GroupName, instancetype.ClusterPluralResourceName, "get", "delete", "create", "update", "patch", "list", "watch"),
				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", instancetype.GroupName, instancetype.PluralPreferenceResourceName), instancetype.GroupName, instancetype.PluralPreferenceResourceName, "get", "delete", "create", "update", "patch", "list", "watch"),
				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", instancetype.GroupName, instancetype.ClusterPluralPreferenceResourceName), instancetype.GroupName, instancetype.ClusterPluralPreferenceResourceName, "get", "delete", "create", "update", "patch", "list", "watch"),

				Entry(fmt.Sprintf("get, delete, create, update, patch, list, watch %s/%s", pool.GroupName, ApiVMPools), pool.GroupName, ApiVMPools, "get", "delete", "create", "update", "patch", "list", "watch"),

				Entry(fmt.Sprintf("get, list %s/%s", GroupName, ApiKubevirts), GroupName, ApiKubevirts, "get", "list"),

				Entry(fmt.Sprintf("get, list, watch %s/%s", migrations.GroupName, migrations.ResourceMigrationPolicies), migrations.GroupName, migrations.ResourceMigrationPolicies, "get", "list", "watch"),
			)
		})

		Context("view cluster role", func() {

			DescribeTable("should contain rule to", func(apiGroup, resource string, verbs ...string) {
				clusterRole := getObject(clusterObjects, reflect.TypeOf(&rbacv1.ClusterRole{}), "kubevirt.io:view").(*rbacv1.ClusterRole)
				Expect(clusterRole).ToNot(BeNil())
				expectExactRuleExists(clusterRole.Rules, apiGroup, resource, verbs...)
			},
				Entry(fmt.Sprintf("get, list %s/%s", GroupName, ApiKubevirts), GroupName, ApiKubevirts, "get", "list"),

				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMExpandSpec), virtv1.SubresourceGroupName, ApiVMExpandSpec, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesGuestOSInfo), virtv1.SubresourceGroupName, ApiVMInstancesGuestOSInfo, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesFileSysList), virtv1.SubresourceGroupName, ApiVMInstancesFileSysList, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesUserList), virtv1.SubresourceGroupName, ApiVMInstancesUserList, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesSEVFetchCertChain), virtv1.SubresourceGroupName, ApiVMInstancesSEVFetchCertChain, "get"),
				Entry(fmt.Sprintf("get %s/%s", virtv1.SubresourceGroupName, ApiVMInstancesSEVQueryLaunchMeasurement), virtv1.SubresourceGroupName, ApiVMInstancesSEVQueryLaunchMeasurement, "get"),

				Entry(fmt.Sprintf("update %s/%s", virtv1.SubresourceGroupName, ApiExpandVmSpec), virtv1.SubresourceGroupName, ApiExpandVmSpec, "update"),

				Entry(fmt.Sprintf("get, list, watch %s/%s", GroupName, ApiVM), GroupName, ApiVM, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", GroupName, ApiVMInstances), GroupName, ApiVMInstances, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", GroupName, ApiVMIPresets), GroupName, ApiVMIPresets, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", GroupName, ApiVMIReplicasets), GroupName, ApiVMIReplicasets, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", GroupName, ApiVMIMigrations), GroupName, ApiVMIMigrations, "get", "list", "watch"),

				Entry(fmt.Sprintf("get, list, watch %s/%s", snapshot.GroupName, ApiVMSnapshots), snapshot.GroupName, ApiVMSnapshots, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", snapshot.GroupName, ApiVMSnapshotContents), snapshot.GroupName, ApiVMSnapshotContents, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", snapshot.GroupName, ApiVMRestores), snapshot.GroupName, ApiVMRestores, "get", "list", "watch"),

				Entry(fmt.Sprintf("get, list, watch %s/%s", export.GroupName, ApiVMExports), export.GroupName, ApiVMExports, "get", "list", "watch"),

				Entry(fmt.Sprintf("get, list, watch %s/%s", clone.GroupName, ApiVMClones), clone.GroupName, ApiVMClones, "get", "list", "watch"),

				Entry(fmt.Sprintf("get, list, watch %s/%s", instancetype.GroupName, instancetype.PluralResourceName), instancetype.GroupName, instancetype.PluralResourceName, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", instancetype.GroupName, instancetype.ClusterPluralResourceName), instancetype.GroupName, instancetype.ClusterPluralResourceName, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", instancetype.GroupName, instancetype.PluralPreferenceResourceName), instancetype.GroupName, instancetype.PluralPreferenceResourceName, "get", "list", "watch"),
				Entry(fmt.Sprintf("get, list, watch %s/%s", instancetype.GroupName, instancetype.ClusterPluralPreferenceResourceName), instancetype.GroupName, instancetype.ClusterPluralPreferenceResourceName, "get", "list", "watch"),

				Entry(fmt.Sprintf("get, list, watch %s/%s", pool.GroupName, ApiVMPools), pool.GroupName, ApiVMPools, "get", "list", "watch"),

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
