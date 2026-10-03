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
	"fmt"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
)

var _ = Describe("Export proxy SA Cluster role", func() {
	const expectedNamespace = "kubevirt"
	Context("GetAllExportProxy", func() {
		allObjects := GetAllExportProxy(expectedNamespace)

		It("should not be nil", func() {
			Expect(allObjects).ToNot(BeNil())
		})

		DescribeTable("cluster role should contain rule to", func(apiGroup, resource string, verbs ...string) {
			clusterRole, ok := getObject(allObjects, reflect.TypeOf(&rbacv1.ClusterRole{}), ExportProxyServiceAccountName).(*rbacv1.ClusterRole)
			Expect(ok).To(BeTrue())
			Expect(clusterRole).ToNot(BeNil())
			expectExactRuleExists(clusterRole.Rules, apiGroup, resource, verbs...)
		},
			Entry(fmt.Sprintf("get/list/watch %s/%s", "export.kubevirt.io", "virtualmachineexports"), "export.kubevirt.io", "virtualmachineexports", "get", "list", "watch"),
			Entry(fmt.Sprintf("list/watch %s/%s", "kubevirt.io", "kubevirts"), "kubevirt.io", "kubevirts", "list", "watch"),
			Entry(fmt.Sprintf("get/list/watch %s/%s", "", "services"), "", "services", "get", "list", "watch"),
		)
	})
})
