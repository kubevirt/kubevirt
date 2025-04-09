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
 * Copyright 2024 Red Hat, Inc.
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
		vm := libvmi.New(
			libvmi.WithInterface(v1.Interface{
				Name:                   "foo",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
				Binding:                &v1.PluginBinding{Name: "boo"},
			}),
			libvmi.WithNetwork(&v1.Network{
				Name:          "foo",
				NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
			}),
		)
		clusterConfig := stubClusterConfigChecker{bridgeBindingOnPodNetEnabled: true, bindingPluginFGEnabled: true}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vm.Spec, clusterConfig)
		Expect(validator.Validate()).To(
			ConsistOf(metav1.StatusCause{
				Type:    "FieldValueInvalid",
				Message: "logical foo interface cannot have both binding plugin and interface binding method",
				Field:   "fake.domain.devices.interfaces[0].binding",
			}))
	})

	It("network interface has only plugin binding", func() {
		vm := libvmi.New(
			libvmi.WithInterface(v1.Interface{
				Name:    "foo",
				Binding: &v1.PluginBinding{Name: "boo"},
			}),
			libvmi.WithNetwork(&v1.Network{
				Name:          "foo",
				NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
			}),
		)
		clusterConfig := stubClusterConfigChecker{bindingPluginFGEnabled: true}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vm.Spec, clusterConfig)
		Expect(validator.Validate()).To(BeEmpty())
	})

	It("network interface has only binding method", func() {
		vm := libvmi.New(
			libvmi.WithNetwork(&v1.Network{
				Name:          "foo",
				NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
			}),
			libvmi.WithInterface(v1.Interface{
				Name:                   "foo",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
			}),
		)
		clusterConfig := stubClusterConfigChecker{bridgeBindingOnPodNetEnabled: true}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vm.Spec, clusterConfig)
		Expect(validator.Validate()).To(BeEmpty())
	})
})

var _ = Describe("Validating core binding", func() {
	It("should reject a masquerade interface on a network different than pod", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(v1.Interface{
				Name:                   "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				Ports:                  []v1.Port{{Name: "test"}},
			}),
			libvmi.WithNetwork(&v1.Network{
				Name:          "default",
				NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "test"}},
			}),
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
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithInterface(v1.Interface{
				Name:                   "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				MacAddress:             "02:00:00:00:00:00",
			}),
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
