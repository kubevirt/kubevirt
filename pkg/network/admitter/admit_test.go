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
	DescribeTable("link state is supported for bridge, masquerade, and binding plugin interfaces",
		func(iface v1.Interface, network *v1.Network) {
			vm := libvmi.New(
				libvmi.WithInterface(iface),
				libvmi.WithNetwork(network),
			)
			causes := admitter.NewValidator(k8sfield.NewPath("fake"), &vm.Spec, stubClusterConfigChecker{}).Validate()
			Expect(causes).To(BeEmpty())
		},
		Entry("masquerade up",
			libvmi.NewInterface("foo", libvmi.WithMasqueradeBinding(), libvmi.WithState(v1.InterfaceStateLinkUp)),
			&v1.Network{Name: "foo", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
		),
		Entry("masquerade down",
			libvmi.NewInterface("foo", libvmi.WithMasqueradeBinding(), libvmi.WithState(v1.InterfaceStateLinkDown)),
			&v1.Network{Name: "foo", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
		),
		Entry("bridge up",
			libvmi.NewInterface("foo", libvmi.WithBridgeBinding(), libvmi.WithState(v1.InterfaceStateLinkUp)),
			libvmi.MultusNetwork("foo", "net"),
		),
		Entry("bridge down",
			libvmi.NewInterface("foo", libvmi.WithBridgeBinding(), libvmi.WithState(v1.InterfaceStateLinkDown)),
			libvmi.MultusNetwork("foo", "net"),
		),
		Entry("bridge absent",
			libvmi.NewInterface("foo", libvmi.WithBridgeBinding(), libvmi.WithState(v1.InterfaceStateAbsent)),
			libvmi.MultusNetwork("foo", "net"),
		),
		Entry("binding plugin up",
			libvmi.NewInterface("foo", libvmi.WithBindingPlugin(v1.PluginBinding{Name: "test"}), libvmi.WithState(v1.InterfaceStateLinkUp)),
			libvmi.MultusNetwork("foo", "net"),
		),
		Entry("binding plugin down",
			libvmi.NewInterface("foo", libvmi.WithBindingPlugin(v1.PluginBinding{Name: "test"}), libvmi.WithState(v1.InterfaceStateLinkDown)),
			libvmi.MultusNetwork("foo", "net"),
		),
	)

	It("network interface state value is invalid", func() {
		vm := libvmi.New(
			libvmi.WithNetwork(&v1.Network{
				Name:          "foo",
				NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
			}),
			libvmi.WithInterface(v1.Interface{
				Name:                   "foo",
				State:                  v1.InterfaceState("foo"),
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
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

	DescribeTable("network interface state is not supported",
		func(iface v1.Interface, network *v1.Network, clusterConfig stubClusterConfigChecker, messageRegex types.GomegaMatcher) {
			vm := libvmi.New(
				libvmi.WithInterface(iface),
				libvmi.WithNetwork(network),
			)
			statusCause := admitter.NewValidator(k8sfield.NewPath("fake"), &vm.Spec, clusterConfig).Validate()
			Expect(statusCause).To(HaveLen(1))
			Expect(statusCause[0]).To(MatchAllFields(Fields{
				"Type":    Equal(metav1.CauseType("FieldValueInvalid")),
				"Field":   Equal("fake.domain.devices.interfaces[0].state"),
				"Message": messageRegex,
			}))
		},
		Entry("down is not supported for sriov",
			libvmi.NewInterface("foo", libvmi.WithSRIOVBinding(), libvmi.WithState(v1.InterfaceStateLinkDown)),
			libvmi.MultusNetwork("foo", "net"),
			stubClusterConfigChecker{},
			MatchRegexp("down.+binding type"),
		),
		Entry("up is not supported for sriov",
			libvmi.NewInterface("foo", libvmi.WithSRIOVBinding(), libvmi.WithState(v1.InterfaceStateLinkUp)),
			libvmi.MultusNetwork("foo", "net"),
			stubClusterConfigChecker{},
			MatchRegexp("up.+binding type"),
		),
		Entry("absent is not supported when bridge-binding is not used",
			libvmi.NewInterface("foo", libvmi.WithSRIOVBinding(), libvmi.WithState(v1.InterfaceStateAbsent)),
			libvmi.MultusNetwork("foo", "net"),
			stubClusterConfigChecker{},
			MatchRegexp("absent.+bridge"),
		),
		Entry("down is not supported for passt",
			libvmi.NewInterface("foo", libvmi.WithPasstBinding(), libvmi.WithState(v1.InterfaceStateLinkDown)),
			&v1.Network{Name: "foo", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
			stubClusterConfigChecker{passtBindingFeatureGateEnabled: true},
			MatchRegexp("down.+binding type"),
		),
		Entry("up is not supported for passt",
			libvmi.NewInterface("foo", libvmi.WithPasstBinding(), libvmi.WithState(v1.InterfaceStateLinkUp)),
			&v1.Network{Name: "foo", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
			stubClusterConfigChecker{passtBindingFeatureGateEnabled: true},
			MatchRegexp("up.+binding type"),
		),
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
