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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package admitters

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/apimachinery/resource"
)

var _ = Describe("Validating VMI network spec", func() {

	DescribeTable("network interface resources requests valid value", func(value string) {
		vm := api.NewMinimalVMI("testvm")
		vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
			resource.ResourceInterface: k8sresource.MustParse(value),
		}
		Expect(validateInterfaceRequestIsInRange(k8sfield.NewPath("fake"), &vm.Spec)).To(BeEmpty())
	},
		Entry("is an integer between 0 to 32", "5"),
		Entry("is the minimum", "0"),
		Entry("is the maximum", "32"),
	)

	DescribeTable("network interface resources requests invalid value", func(value string) {
		vm := api.NewMinimalVMI("testvm")
		vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
			resource.ResourceInterface: k8sresource.MustParse(value),
		}
		Expect(validateInterfaceRequestIsInRange(k8sfield.NewPath("fake"), &vm.Spec)).To(
			ConsistOf(metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "provided resources interface requests must be an integer between 0 to 32",
				Field:   "fake.domain.resources.requests.kubevirt.io/interface",
			}))
	},
		Entry("is not an integer", "1.2"),
		Entry("is negative", "-2"),
		Entry("is beyond the maximum", "33"),
	)
})
