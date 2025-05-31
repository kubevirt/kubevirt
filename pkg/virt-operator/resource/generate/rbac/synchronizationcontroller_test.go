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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
)

var _ = Describe("Synchronization controller SA Cluster role and cluster role bindings", func() {
	const expectedNamespace = "kubevirt"
	Context("GetAllSynchronizationController", func() {
		allObjects := GetAllSynchronizationController(expectedNamespace)

		It("should not be nil", func() {
			Expect(allObjects).ToNot(BeNil())
		})

		DescribeTable("cluster role should contain rule to", func(apiGroup, resource string, verbs ...string) {
			clusterRole, ok := getObject(allObjects, reflect.TypeOf(&rbacv1.ClusterRole{}), SynchronizationControllerServiceAccountName).(*rbacv1.ClusterRole)
			if ok {
				Expect(clusterRole).ToNot(BeNil())
				expectExactRuleExists(clusterRole.Rules, apiGroup, resource, verbs...)
			}
		},
			Entry(fmt.Sprintf("get/list/watch %s/%s", GroupName, apiKubevirts), GroupName, apiKubevirts, "get", "list", "watch"),
			Entry(fmt.Sprintf("get/list/watch/update/patch %s/%s", GroupName, apiVMInstances), GroupName, apiVMInstances, "get", "list", "watch", "update", "patch"),
			Entry(fmt.Sprintf("get/list/watch %s/%s", GroupName, apiVMIMigrations), GroupName, apiVMIMigrations, "get", "list", "watch"),
			Entry(fmt.Sprintf("get/list/watch %s/%s", "apiextensions.k8s.io", "customresourcedefinitions"), "apiextensions.k8s.io", "customresourcedefinitions", "get", "list", "watch"),
			Entry(fmt.Sprintf("update/create/patch %s/%s", "", "events"), "", "events", "update", "create", "patch"),
		)

		DescribeTable("cluster role should contain rule to", func(apiGroup, resource string, verbs ...string) {
			role, ok := getObject(allObjects, reflect.TypeOf(&rbacv1.Role{}), SynchronizationControllerServiceAccountName).(*rbacv1.Role)
			if ok {
				Expect(role).ToNot(BeNil())
				expectExactRuleExists(role.Rules, apiGroup, resource, verbs...)
			}
		},
			Entry(fmt.Sprintf("get/list/watch %s/%s", "", "configmaps"), "", "configmaps", "get", "list", "watch"),
			Entry(fmt.Sprintf("get/list/watch/delete/create/update/patch %s/%s", "coordination.k8s.io", "leases"), "coordination.k8s.io", "leases", "get", "list", "watch", "delete", "update", "create", "patch"),
		)
	})
})
