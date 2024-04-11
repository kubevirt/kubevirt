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

package admitter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/client-go/api"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/admitter"
)

var _ = Describe("Validating VMI network spec", func() {

	DescribeTable("network interface state valid value", func(value v1.InterfaceState) {
		vm := api.NewMinimalVMI("testvm")
		vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:                   "foo",
			State:                  value,
			InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
		}
		vm.Spec.Networks = []v1.Network{
			{Name: "foo", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "net"}}},
		}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vm.Spec, stubSlirpClusterConfigChecker{})
		Expect(validator.Validate()).To(BeEmpty())
	},
		Entry("is empty", v1.InterfaceState("")),
		Entry("is absent when bridge binding is used", v1.InterfaceStateAbsent),
	)

	It("network interface state value is invalid", func() {
		vm := api.NewMinimalVMI("testvm")
		vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "foo", State: v1.InterfaceState("foo")}}
		vm.Spec.Networks = []v1.Network{{Name: "foo", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vm.Spec, stubSlirpClusterConfigChecker{})
		Expect(validator.Validate()).To(
			ConsistOf(metav1.StatusCause{
				Type:    "FieldValueInvalid",
				Message: "logical foo interface state value is unsupported: foo",
				Field:   "fake.domain.devices.interfaces[0].state",
			}))
	})

	It("network interface state value of absent is not supported when bridge-binding is not used", func() {
		vm := api.NewMinimalVMI("testvm")
		vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:                   "foo",
			State:                  v1.InterfaceStateAbsent,
			InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}},
		}}
		vm.Spec.Networks = []v1.Network{
			{Name: "foo", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "net"}}},
		}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vm.Spec, stubSlirpClusterConfigChecker{})
		Expect(validator.Validate()).To(
			ConsistOf(metav1.StatusCause{
				Type:    "FieldValueInvalid",
				Message: "\"foo\" interface's state \"absent\" is supported only for bridge binding",
				Field:   "fake.domain.devices.interfaces[0].state",
			}))
	})

	It("network interface state value of absent is not supported on the default network", func() {
		vm := api.NewMinimalVMI("testvm")
		vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:                   "foo",
			State:                  v1.InterfaceStateAbsent,
			InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
		}}
		vm.Spec.Networks = []v1.Network{{Name: "foo", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vm.Spec, stubSlirpClusterConfigChecker{})
		Expect(validator.Validate()).To(
			ConsistOf(metav1.StatusCause{
				Type:    "FieldValueInvalid",
				Message: "\"foo\" interface's state \"absent\" is not supported on default networks",
				Field:   "fake.domain.devices.interfaces[0].state",
			}))
	})
})
