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
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/admitter"
)

var _ = Describe("Validating VMI network spec", func() {
	DescribeTable("network interface state valid value", func(value v1.InterfaceState) {
		vm := libvmi.New(
			libvmi.WithInterface(v1.Interface{
				Name:                   "foo",
				State:                  value,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
			}),
			libvmi.WithNetwork(&v1.Network{
				Name:          "foo",
				NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "net"}},
			}),
		)
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vm.Spec, stubClusterConfigChecker{})
		Expect(validator.Validate()).To(BeEmpty())
	},
		Entry("is empty", v1.InterfaceState("")),
		Entry("is absent when bridge binding is used", v1.InterfaceStateAbsent),
		Entry("is up when bridge binding is used", v1.InterfaceStateLinkUp),
		Entry("is down when bridge binding is used", v1.InterfaceStateLinkDown),
	)

	It("network interface state value is invalid", func() {
		vm := libvmi.New(
			libvmi.WithNetwork(&v1.Network{
				Name:          "foo",
				NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
			}),
			libvmi.WithInterface(v1.Interface{
				Name:  "foo",
				State: v1.InterfaceState("foo"),
			}),
		)
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vm.Spec, stubClusterConfigChecker{})
		Expect(validator.Validate()).To(
			ConsistOf(metav1.StatusCause{
				Type:    "FieldValueInvalid",
				Message: "logical foo interface state value is unsupported: foo",
				Field:   "fake.domain.devices.interfaces[0].state",
			}))
	})

	DescribeTable("network interface state ", func(state v1.InterfaceState, messageRegex types.GomegaMatcher) {
		vm := libvmi.New(
			libvmi.WithInterface(v1.Interface{
				Name:                   "foo",
				State:                  state,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}},
			}),
			libvmi.WithNetwork(&v1.Network{
				Name:          "foo",
				NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "net"}},
			}),
		)
		statusCause := admitter.NewValidator(k8sfield.NewPath("fake"), &vm.Spec, stubClusterConfigChecker{}).Validate()
		Expect(statusCause).To(HaveLen(1))
		Expect(statusCause[0]).To(MatchAllFields(Fields{
			"Type":    Equal(metav1.CauseType("FieldValueInvalid")),
			"Field":   Equal("fake.domain.devices.interfaces[0].state"),
			"Message": messageRegex,
		}))
	},
		Entry("down is not supported for sriov", v1.InterfaceStateLinkDown, MatchRegexp("down.+SR-IOV")),
		Entry("up is not supported for sriov", v1.InterfaceStateLinkUp, MatchRegexp("up.+SR-IOV")),
		Entry("absent is not supported when bridge-binding is not used", v1.InterfaceStateAbsent, MatchRegexp("absent.+bridge")),
	)

	It("network interface state value of absent is not supported on the default network", func() {
		vm := libvmi.New(
			libvmi.WithNetwork(&v1.Network{
				Name:          "foo",
				NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
			}),
			libvmi.WithInterface(v1.Interface{
				Name:                   "foo",
				State:                  v1.InterfaceStateAbsent,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
			}),
		)
		clusterConfig := stubClusterConfigChecker{bridgeBindingOnPodNetEnabled: true}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vm.Spec, clusterConfig)
		Expect(validator.Validate()).To(
			ConsistOf(metav1.StatusCause{
				Type:    "FieldValueInvalid",
				Message: "\"foo\" interface's state \"absent\" is not supported on default networks",
				Field:   "fake.domain.devices.interfaces[0].state",
			}))
	})
})
