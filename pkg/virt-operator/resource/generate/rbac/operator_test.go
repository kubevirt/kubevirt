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
 * Copyright 2021 Red Hat, Inc.
 */

package rbac

import (
	"fmt"
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

var _ = Describe("RBAC", func() {

	const expectedNamespace = "default"

	Context("GetAllOperator", func() {

		forOperator := GetAllOperator(expectedNamespace)

		It("isn't nil", func() {
			Expect(forOperator).ToNot(BeNil())
		})

		It("has service account", func() {
			serviceAccount := getFirstItemOfType(forOperator, reflect.TypeOf(&v1.ServiceAccount{})).(*v1.ServiceAccount)
			Expect(serviceAccount.Namespace).To(BeEquivalentTo(expectedNamespace))
		})

		It("has rbac role", func() {
			role := getFirstItemOfType(forOperator, reflect.TypeOf(&rbacv1.Role{})).(*rbacv1.Role)
			Expect(role.Namespace).To(BeEquivalentTo(expectedNamespace))
		})

		It("has rbac role binding", func() {
			roleBinding := getFirstItemOfType(forOperator, reflect.TypeOf(&rbacv1.RoleBinding{})).(*rbacv1.RoleBinding)
			Expect(roleBinding.Namespace).To(BeEquivalentTo(expectedNamespace))
		})

		It("has cluster role", func() {
			clusterRole := getFirstItemOfType(forOperator, reflect.TypeOf(&rbacv1.ClusterRole{})).(*rbacv1.ClusterRole)
			Expect(clusterRole).ToNot(BeNil())
		})

		It("has cluster role binding", func() {
			clusterRoleBinding := getFirstItemOfType(forOperator, reflect.TypeOf(&rbacv1.ClusterRoleBinding{})).(*rbacv1.ClusterRoleBinding)
			Expect(clusterRoleBinding.Subjects[0].Namespace).To(BeEquivalentTo(expectedNamespace))
		})

	})

	Context("GetKubevirtComponentsServiceAccounts", func() {

		serviceAccounts := GetKubevirtComponentsServiceAccounts(expectedNamespace)

		DescribeTable("has service account",
			func(name string) {
				Expect(serviceAccounts).To(HaveKey(MatchRegexp(fmt.Sprintf(".*%s.*", name))))
			},
			Entry("for Handler", HandlerServiceAccountName),
			Entry("for Api", ApiServiceAccountName),
			Entry("for Controller", ControllerServiceAccountName),
			Entry("for Operator", OperatorServiceAccountName),
		)

	})

})

func getFirstItemOfType(items []interface{}, tp reflect.Type) interface{} {
	for _, item := range items {
		typeOf := reflect.TypeOf(item)
		if typeOf == tp {
			return item
		}
	}
	return nil
}
