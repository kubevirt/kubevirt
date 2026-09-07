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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/admitter"
)

var _ = Describe("Validating network binding combinations", func() {
	It("network interface has both binding plugin and interface binding method", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.NewInterface(
				"foo",
				libvmi.WithBridgeBinding(),
				libvmi.WithBindingPlugin(v1.PluginBinding{Name: "boo"}),
			)),
			libvmi.WithNetwork(&v1.Network{
				Name:          "foo",
				NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
			}),
		)
		clusterConfig := stubClusterConfigChecker{bridgeBindingOnPodNetEnabled: true}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, clusterConfig)
		Expect(validator.Validate()).To(
			ConsistOf(metav1.StatusCause{
				Type:    "FieldValueInvalid",
				Message: "logical foo interface must have exactly one binding method or binding plugin",
				Field:   "fake.domain.devices.interfaces[0]",
			}))
	})

	It("network interface has only plugin binding", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.NewInterface(
				"foo",
				libvmi.WithBindingPlugin(v1.PluginBinding{Name: "boo"}),
			)),
			libvmi.WithNetwork(&v1.Network{
				Name:          "foo",
				NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
			}),
		)
		clusterConfig := stubClusterConfigChecker{}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, clusterConfig)
		Expect(validator.Validate()).To(BeEmpty())
	})

	It("network interface has neither binding plugin nor interface binding method", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.NewInterface(
				"foo",
			)),
			libvmi.WithNetwork(&v1.Network{
				Name:          "foo",
				NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
			}),
		)
		clusterConfig := stubClusterConfigChecker{}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, clusterConfig)
		Expect(validator.Validate()).To(
			ConsistOf(metav1.StatusCause{
				Type:    "FieldValueInvalid",
				Message: "logical foo interface must have exactly one binding method or binding plugin",
				Field:   "fake.domain.devices.interfaces[0]",
			}))
	})

	It("network interface has more than one binding method", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(v1.Interface{
				Name: "foo",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Bridge:     &v1.InterfaceBridge{},
					Masquerade: &v1.InterfaceMasquerade{},
				},
			}),
			libvmi.WithNetwork(&v1.Network{
				Name:          "foo",
				NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
			}),
		)
		clusterConfig := stubClusterConfigChecker{bridgeBindingOnPodNetEnabled: true}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, clusterConfig)
		Expect(validator.Validate()).To(
			ConsistOf(metav1.StatusCause{
				Type:    "FieldValueInvalid",
				Message: "logical foo interface must have exactly one binding method or binding plugin",
				Field:   "fake.domain.devices.interfaces[0]",
			}))
	})

	It("network interface has only binding method", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.NewInterface(
				"foo",
				libvmi.WithBridgeBinding(),
			)),
			libvmi.WithNetwork(&v1.Network{
				Name:          "foo",
				NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
			}),
		)
		clusterConfig := stubClusterConfigChecker{bridgeBindingOnPodNetEnabled: true}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, clusterConfig)
		Expect(validator.Validate()).To(BeEmpty())
	})
})

var _ = Describe("Validating core binding", func() {
	It("should reject a masquerade interface on a network different than pod", func() {
		vmi := libvmi.New(

			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding(v1.Port{Name: "test"})),

			libvmi.WithNetwork(libvmi.MultusNetwork("default", "test")),
		)

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{})
		causes := validator.Validate()

		Expect(causes).To(ConsistOf(metav1.StatusCause{
			Type:    "FieldValueInvalid",
			Message: "Masquerade interface only implemented with pod network",
			Field:   "fake.domain.devices.interfaces[0].name",
		}))
	})

	It("should reject a masquerade interface with a specified reserved MAC address", func() {
		vmi := libvmi.New(

			libvmi.WithInterface(libvmi.NewInterface(
				"default",
				libvmi.WithMasqueradeBinding(),
				libvmi.WithMac("02:00:00:00:00:00"),
			)),

			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{})
		causes := validator.Validate()

		Expect(causes).To(ConsistOf(metav1.StatusCause{
			Type:    "FieldValueInvalid",
			Message: "The requested MAC address is reserved for the in-pod bridge. Please choose another one.",
			Field:   "fake.domain.devices.interfaces[0].macAddress",
		}))
	})

	It("should reject a bridge interface on a pod network when it is not permitted", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{})
		causes := validator.Validate()

		Expect(causes).To(ConsistOf(metav1.StatusCause{
			Type:    "FieldValueInvalid",
			Message: "Bridge on pod network configuration is not enabled under kubevirt-config",
			Field:   "fake.domain.devices.interfaces[0].name",
		}))
	})
})
