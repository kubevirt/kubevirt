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
	"github.com/onsi/gomega/gstruct"

	rbacv1 "k8s.io/api/rbac/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

var _ = Describe("RBAC", func() {

	const expectedNamespace = "default"

	Context("GetAllController", func() {
		forController := GetAllController(expectedNamespace)

		DescribeTable("has finalizer rbac for installs with OwnerReferencesPermissionEnforcement", func(apiGroup, resource string) {
			clusterRole := getObject(forController, reflect.TypeOf(&rbacv1.ClusterRole{}), components.ControllerServiceAccountName).(*rbacv1.ClusterRole)
			Expect(clusterRole).ToNot(BeNil())
			Expect(clusterRole.Rules).To(
				ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"APIGroups": ContainElement(apiGroup),
					"Resources": ContainElement(fmt.Sprintf("%s/finalizers", resource)),
					"Verbs":     ContainElement("update"),
				})), "appropriate rule for finalizers not found",
			)
		},
			Entry("for vmclones", "clone.kubevirt.io", "virtualmachineclones"),
			Entry("for vmexports", "export.kubevirt.io", "virtualmachineexports"),
			Entry("for vmpools", "pool.kubevirt.io", "virtualmachinepools"),
			Entry("for vmsnapshots", "snapshot.kubevirt.io", "virtualmachinesnapshots"),
			Entry("for vmsnapshotcontents", "snapshot.kubevirt.io", "virtualmachinesnapshotcontents"),
			Entry("for vms", "kubevirt.io", "virtualmachines"),
			Entry("for vmis", "kubevirt.io", "virtualmachineinstances"),
		)
	})
})
