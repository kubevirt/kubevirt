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
			"interface fake.domain.devices.interfaces[0].name has too long MAC address (de:ad:00:00:be:af:be:af).",
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
		Entry("valid address in lowercase and colon separated", "de:ad:00:00:be:af"),
		Entry("valid address in uppercase and colon separated", "DE:AD:00:00:BE:AF"),
		Entry("valid address in lowercase and dash separated", "de-ad-00-00-be-af"),
		Entry("valid address in uppercase and dash separated", "DE-AD-00-00-BE-AF"),
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

	When("the interface port is specified", func() {
		DescribeTable("should reject interface port with", func(ports []v1.Port, expectedCauses []metav1.StatusCause) {
			spec := &v1.VirtualMachineInstanceSpec{}
			spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name:                   "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				Ports:                  ports,
			}}
			spec.Networks = []v1.Network{{Name: "default", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

			validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
			Expect(validator.Validate()).To(ConsistOf(expectedCauses))
		},
			Entry(
				"only the port name",
				[]v1.Port{{Name: "test"}},
				[]metav1.StatusCause{{
					Type:    "FieldValueRequired",
					Message: "Port field is mandatory.",
					Field:   "fake.domain.devices.interfaces[0].ports[0]",
				}},
			),
			Entry(
				"bad protocol type",
				[]v1.Port{{Protocol: "bad", Port: 80}},
				[]metav1.StatusCause{{
					Type:    "FieldValueInvalid",
					Message: "Unknown protocol, only TCP or UDP allowed",
					Field:   "fake.domain.devices.interfaces[0].ports[0].protocol",
				}},
			),
			Entry(
				"port out of range",
				[]v1.Port{{Port: 80000}},
				[]metav1.StatusCause{{
					Type:    "FieldValueInvalid",
					Message: "Port field must be in range 0 < x < 65536.",
					Field:   "fake.domain.devices.interfaces[0].ports[0]",
				}},
			),
			Entry(
				"two ports that have the same name",
				[]v1.Port{{Name: "testport", Port: 80}, {Name: "testport", Protocol: "UDP", Port: 80}},
				[]metav1.StatusCause{{
					Type:    "FieldValueDuplicate",
					Message: "Duplicate name of the port: testport",
					Field:   "fake.domain.devices.interfaces[0].ports[1].name",
				}},
			),
			Entry(
				"bad port name",
				[]v1.Port{{Name: "Test", Port: 80}},
				[]metav1.StatusCause{{
					Type:    "FieldValueInvalid",
					Message: "Invalid name of the port: Test",
					Field:   "fake.domain.devices.interfaces[0].ports[0].name",
				}},
			),
		)

		DescribeTable("should accept interface with", func(ports []v1.Port) {
			spec := &v1.VirtualMachineInstanceSpec{}
			spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name:                   "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				Ports:                  ports,
			}}
			spec.Networks = []v1.Network{{Name: "default", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

			validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
			Expect(validator.Validate()).To(BeEmpty())
		},
			Entry("single minimal port", []v1.Port{{Port: 80}}),
			Entry("multiple ports, same number, with protocol and without", []v1.Port{{Port: 80}, {Protocol: "UDP", Port: 80}}),
			Entry(
				"multiple ports, same number, different protocols",
				[]v1.Port{{Port: 80}, {Protocol: "UDP", Port: 80}, {Protocol: "TCP", Port: 80}},
			),
		)
	})

	When("the interface DHCP options is specified", func() {
		DescribeTable("should reject interface DHCP options with", func(dhcpOpts v1.DHCPOptions, expectedCauses []metav1.StatusCause) {
			spec := &v1.VirtualMachineInstanceSpec{}
			spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name:                   "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				DHCPOptions:            &dhcpOpts,
			}}
			spec.Networks = []v1.Network{{Name: "default", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

			validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
			Expect(validator.Validate()).To(ConsistOf(expectedCauses))
		},
			Entry(
				"invalid DHCPPrivateOptions",
				v1.DHCPOptions{PrivateOptions: []v1.DHCPPrivateOptions{{Option: 223, Value: "extra.options.kubevirt.io"}}},
				[]metav1.StatusCause{{
					Type:    "FieldValueInvalid",
					Message: "provided DHCPPrivateOptions are out of range, must be in range 224 to 254",
					Field:   "fake",
				}},
			),
			Entry(
				"duplicate DHCPPrivateOptions",
				v1.DHCPOptions{
					PrivateOptions: []v1.DHCPPrivateOptions{
						{Option: 240, Value: "extra.options.kubevirt.io"},
						{Option: 240, Value: "sameextra.options.kubevirt.io"},
					},
				},
				[]metav1.StatusCause{{
					Type:    "FieldValueInvalid",
					Message: "Found Duplicates: you have provided duplicate DHCPPrivateOptions",
					Field:   "fake",
				}},
			),
			Entry(
				"non-IPv4 NTP servers",
				v1.DHCPOptions{NTPServers: []string{"::1", "hostname"}},
				[]metav1.StatusCause{{
					Type:    "FieldValueInvalid",
					Message: "NTP servers must be a list of valid IPv4 addresses.",
					Field:   "fake.domain.devices.interfaces[0].dhcpOptions.ntpServers[0]",
				}, {
					Type:    "FieldValueInvalid",
					Message: "NTP servers must be a list of valid IPv4 addresses.",
					Field:   "fake.domain.devices.interfaces[0].dhcpOptions.ntpServers[1]",
				}},
			),
		)

		DescribeTable("should accept interface DHCP options with", func(dhcpOpts v1.DHCPOptions) {
			spec := &v1.VirtualMachineInstanceSpec{}
			spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name:                   "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				DHCPOptions:            &dhcpOpts,
			}}
			spec.Networks = []v1.Network{{Name: "default", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

			validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
			Expect(validator.Validate()).To(BeEmpty())
		},
			Entry("  valid DHCPPrivateOptions", v1.DHCPOptions{
				PrivateOptions: []v1.DHCPPrivateOptions{{Option: 240, Value: "extra.options.kubevirt.io"}},
			}),
			Entry(" valid NTP servers", v1.DHCPOptions{NTPServers: []string{"127.0.0.1", "127.0.0.2"}}),
			Entry(
				"unique DHCPPrivateOptions",
				v1.DHCPOptions{
					PrivateOptions: []v1.DHCPPrivateOptions{
						{Option: 240, Value: "extra.options.kubevirt.io"},
						{Option: 241, Value: "extra.options.kubevirt.io"},
					},
				},
			),
		)
	})
})
