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
	"fmt"

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

	It("should reject interface named with unsupported characters", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:                   "bad.name",
			InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
		}}
		spec.Networks = []v1.Network{{Name: "bad.name", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		Expect(validator.Validate()).To(ConsistOf(metav1.StatusCause{
			Type:    "FieldValueInvalid",
			Message: "Network interface name can only contain alphabetical characters, numbers, dashes (-) or underscores (_)",
			Field:   "fake.domain.devices.interfaces[0].name",
		}))
	})

	It("should reject invalid interface model", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}
		spec.Domain.Devices.Interfaces[0].Model = "invalid_model"
		spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		Expect(validator.Validate()).To(ConsistOf(metav1.StatusCause{
			Type:    "FieldValueNotSupported",
			Message: "interface fake.domain.devices.interfaces[0].name uses model invalid_model that is not supported.",
			Field:   "fake.domain.devices.interfaces[0].model",
		}))
	})

	It("should accept valid interface model", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}
		spec.Domain.Devices.Interfaces[0].Model = v1.VirtIO
		spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		Expect(validator.Validate()).To(BeEmpty())
	})

	DescribeTable("should reject invalid MAC addresses", func(macAddress, expectedMessage string) {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}
		spec.Domain.Devices.Interfaces[0].MacAddress = macAddress
		spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		Expect(validator.Validate()).To(ConsistOf(metav1.StatusCause{
			Type:    "FieldValueInvalid",
			Message: expectedMessage,
			Field:   "fake.domain.devices.interfaces[0].macAddress",
		}))
	},
		Entry(
			"too short address",
			"de:ad:00:00:be",
			"interface fake.domain.devices.interfaces[0].name has malformed MAC address (de:ad:00:00:be).",
		),
		Entry(
			"too short address with '-'",
			"de-ad-00-00-be",
			"interface fake.domain.devices.interfaces[0].name has malformed MAC address (de-ad-00-00-be).",
		),
		Entry(
			"too long address",
			"de:ad:00:00:be:af:be:af",
			"interface fake.domain.devices.interfaces[0].name has MAC address (de:ad:00:00:be:af:be:af) that is too long.",
		),
	)

	DescribeTable("should accept valid MAC addresses", func(macAddress string) {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}
		spec.Domain.Devices.Interfaces[0].MacAddress = macAddress
		spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		Expect(validator.Validate()).To(BeEmpty())
	},
		Entry("valid address", "de:ad:00:00:be:af"),
		Entry("valid address with '-'", "de-ad-00-00-be-af"),
	)

	DescribeTable("should reject invalid PCI addresses", func(pciAddress string) {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}
		spec.Domain.Devices.Interfaces[0].PciAddress = pciAddress
		spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		Expect(validator.Validate()).To(ConsistOf(metav1.StatusCause{
			Type:    "FieldValueInvalid",
			Message: fmt.Sprintf("interface fake.domain.devices.interfaces[0].name has malformed PCI address (%s).", pciAddress),
			Field:   "fake.domain.devices.interfaces[0].pciAddress",
		}))
	},
		Entry("too many dots", "0000:80.10.1"),
		Entry("too many parts'-'", "0000:80:80:1.0"),
		Entry("function out of range", "0000:80:11.15"),
	)

	DescribeTable("should accept valid PCI address", func(pciAddress string) {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}
		spec.Domain.Devices.Interfaces[0].PciAddress = pciAddress
		spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		Expect(validator.Validate()).To(BeEmpty())
	},
		Entry("valid address A", "0000:81:11.1"),
		Entry("valid address B", "0001:02:00.0"),
	)
})
