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

	"kubevirt.io/kubevirt/pkg/network/admitter"
)

var _ = Describe("Validating VMI network spec", func() {
	It("should reject network with missing interface", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{}
		spec.Networks = []v1.Network{{
			Name:          "not-the-default",
			NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
		}}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		causes := validator.Validate()

		Expect(causes).To(ConsistOf(metav1.StatusCause{
			Type:    "FieldValueRequired",
			Message: "fake.networks[0].name 'not-the-default' not found.",
			Field:   "fake.networks[0].name",
		}))
	})

	It("should reject interface with missing network", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
		spec.Networks = []v1.Network{}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		causes := validator.Validate()

		Expect(causes).To(ConsistOf(metav1.StatusCause{
			Type:    "FieldValueInvalid",
			Message: "fake.domain.devices.interfaces[0].name 'default' not found.",
			Field:   "fake.domain.devices.interfaces[0].name",
		}))
	})

	It("should reject networks with duplicate names", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
		spec.Networks = []v1.Network{
			{
				Name: "default",
				NetworkSource: v1.NetworkSource{
					Pod: &v1.PodNetwork{},
				},
			},
			{
				Name: "default",
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "test"},
				},
			},
		}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		causes := validator.Validate()

		Expect(causes).To(ConsistOf(metav1.StatusCause{
			Type:    "FieldValueDuplicate",
			Message: "Network with name \"default\" already exists, every network must have a unique name",
			Field:   "fake.networks[1].name",
		}))
	})

	It("should reject interfaces with duplicate names", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{
			*v1.DefaultBridgeNetworkInterface(),
			*v1.DefaultBridgeNetworkInterface(),
		}
		spec.Networks = []v1.Network{
			{Name: "default", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
			{Name: "default", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "test"}}},
		}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		causes := validator.Validate()

		Expect(causes).To(ContainElements(metav1.StatusCause{
			Type:    "FieldValueDuplicate",
			Message: "Only one interface can be connected to one specific network",
			Field:   "fake.domain.devices.interfaces[1].name",
		}))
	})
})
