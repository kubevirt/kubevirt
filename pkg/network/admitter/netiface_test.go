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

const (
	fieldValueRequired       = "FieldValueRequired"
	badInterfaceName         = "bad.name"
	fakeDomainPortField      = "fake.domain.devices.interfaces[0].ports[0].port"
	unknownProtocolMsg       = "Unknown protocol, only TCP or UDP allowed"
	testportName             = "testport"
	fakeDomainPortRangeField = "fake.domain.devices.interfaces[0].portRanges[0].start"
	fakeDomainField          = "fake"
	ntpServersMsg            = "NTP servers must be a list of valid IPv4 addresses."
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
			Type:    fieldValueRequired,
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
			Type:    fieldValueInvalidType,
			Message: "fake.domain.devices.interfaces[0].name 'default' not found.",
			Field:   fakePrimaryIfaceNameField,
		}))
	})

	It("should reject networks with duplicate names", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
		spec.Networks = []v1.Network{
			{
				Name: net1Name,
				NetworkSource: v1.NetworkSource{
					Pod: &v1.PodNetwork{},
				},
			},
			{
				Name: net1Name,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: testPortName},
				},
			},
		}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		causes := validator.Validate()

		Expect(causes).To(ConsistOf(metav1.StatusCause{
			Type:    fieldValueDuplicateType,
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
			{Name: net1Name, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
			{Name: net1Name, NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: testPortName}}},
		}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		causes := validator.Validate()

		Expect(causes).To(ContainElements(metav1.StatusCause{
			Type:    fieldValueDuplicateType,
			Message: "Only one interface can be connected to one specific network",
			Field:   "fake.domain.devices.interfaces[1].name",
		}))
	})

	It("should reject interface named with unsupported characters", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:                   badInterfaceName,
			InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
		}}
		spec.Networks = []v1.Network{{Name: badInterfaceName, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		Expect(validator.Validate()).To(ConsistOf(metav1.StatusCause{
			Type:    fieldValueInvalidType,
			Message: "Network interface name can only contain alphabetical characters, numbers, dashes (-) or underscores (_)",
			Field:   fakePrimaryIfaceNameField,
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
			Type:    fieldValueInvalidType,
			Message: expectedMessage,
			Field:   fakeDomainMACField,
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
			Type:    fieldValueInvalidType,
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
				Name:                   net1Name,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				Ports:                  ports,
			}}
			spec.Networks = []v1.Network{{Name: net1Name, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

			validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
			Expect(validator.Validate()).To(ConsistOf(expectedCauses))
		},
			Entry(
				"only the port name",
				[]v1.Port{{Name: testPortName}},
				[]metav1.StatusCause{{
					Type:    fieldValueRequired,
					Message: "Port field is mandatory.",
					Field:   fakeDomainPortField,
				}},
			),
			Entry(
				"bad protocol type",
				[]v1.Port{{Protocol: "bad", Port: 80}},
				[]metav1.StatusCause{{
					Type:    fieldValueInvalidType,
					Message: unknownProtocolMsg,
					Field:   "fake.domain.devices.interfaces[0].ports[0].protocol",
				}},
			),
			Entry(
				"port out of range",
				[]v1.Port{{Port: 80000}},
				[]metav1.StatusCause{{
					Type:    fieldValueInvalidType,
					Message: "Port field must be in range 0 < x < 65536.",
					Field:   fakeDomainPortField,
				}},
			),
			Entry(
				"two ports that have the same name",
				[]v1.Port{{Name: testportName, Port: 80}, {Name: testportName, Protocol: udpProtocol, Port: 80}},
				[]metav1.StatusCause{{
					Type:    fieldValueDuplicateType,
					Message: "Duplicate name of the port: testport",
					Field:   "fake.domain.devices.interfaces[0].ports[1].name",
				}},
			),
			Entry(
				"bad port name",
				[]v1.Port{{Name: "Test", Port: 80}},
				[]metav1.StatusCause{{
					Type:    fieldValueInvalidType,
					Message: "Invalid name of the port: Test",
					Field:   "fake.domain.devices.interfaces[0].ports[0].name",
				}},
			),
		)

		DescribeTable("should accept interface with", func(ports []v1.Port) {
			spec := &v1.VirtualMachineInstanceSpec{}
			spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name:                   net1Name,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				Ports:                  ports,
			}}
			spec.Networks = []v1.Network{{Name: net1Name, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

			validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
			Expect(validator.Validate()).To(BeEmpty())
		},
			Entry("single minimal port", []v1.Port{{Port: 80}}),
			Entry("multiple ports, same number, with protocol and without", []v1.Port{{Port: 80}, {Protocol: udpProtocol, Port: 80}}),
			Entry(
				"multiple ports, same number, different protocols",
				[]v1.Port{{Port: 80}, {Protocol: udpProtocol, Port: 80}, {Protocol: tcpProtocol, Port: 80}},
			),
		)
	})
	When("the interface portRanges is specified", func() {
		It("should reject portRanges when feature gate is disabled", func() {
			spec := &v1.VirtualMachineInstanceSpec{}
			spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name:                   net1Name,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				PortRanges:             []v1.PortRange{{Start: 80, End: 90}},
			}}
			spec.Networks = []v1.Network{{Name: net1Name, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

			validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{portRangesSpecGateEnabled: false})
			Expect(validator.Validate()).To(ConsistOf(metav1.StatusCause{
				Type:    fieldValueInvalidType,
				Message: "portRanges is specified on interface but the PortRangesSpec feature gate is not enabled",
				Field:   fakePrimaryIfaceNameField,
			}))
		})
		It("should reject when binding method is not masquerade", func() {
			spec := &v1.VirtualMachineInstanceSpec{}
			spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name:                   net1Name,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
				PortRanges:             []v1.PortRange{{Protocol: tcpProtocol, Start: 80, End: 90}},
			}}
			spec.Networks = []v1.Network{{Name: net1Name, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

			validator := admitter.NewValidator(
				k8sfield.NewPath("fake"), spec,
				stubClusterConfigChecker{portRangesSpecGateEnabled: true, bridgeBindingOnPodNetEnabled: true},
			)
			Expect(validator.Validate()).To(ConsistOf(metav1.StatusCause{
				Type:    fieldValueInvalidType,
				Message: "portRanges are only supported on masquerade interfaces",
				Field:   fakePrimaryIfaceNameField,
			}))
		})

		It("should reject when portRanges and ports are both set", func() {
			spec := &v1.VirtualMachineInstanceSpec{}
			spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name:                   net1Name,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				Ports:                  []v1.Port{{Port: 22}},
				PortRanges:             []v1.PortRange{{Protocol: tcpProtocol, Start: 80, End: 90}},
			}}
			spec.Networks = []v1.Network{{Name: net1Name, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

			validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{portRangesSpecGateEnabled: true})
			Expect(validator.Validate()).To(ConsistOf(metav1.StatusCause{
				Type:    fieldValueInvalidType,
				Message: "Cannot define both ports and portRanges on interface",
				Field:   fakePrimaryIfaceNameField,
			}))
		})

		DescribeTable("should reject portRanges with", func(portRanges []v1.PortRange, expectedCauses []metav1.StatusCause) {
			spec := &v1.VirtualMachineInstanceSpec{}
			spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name:                   net1Name,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				PortRanges:             portRanges,
			}}
			spec.Networks = []v1.Network{{Name: net1Name, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

			validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{portRangesSpecGateEnabled: true})
			Expect(validator.Validate()).To(ConsistOf(expectedCauses))
		},
			Entry(
				"bad protocol",
				[]v1.PortRange{{Protocol: "SCTP", Start: 80, End: 90}},
				[]metav1.StatusCause{{
					Type:    fieldValueInvalidType,
					Message: unknownProtocolMsg,
					Field:   "fake.domain.devices.interfaces[0].portRanges[0].protocol",
				}},
			),
			Entry(
				"start port out of range",
				[]v1.PortRange{{Protocol: tcpProtocol, Start: 0, End: 100}},
				[]metav1.StatusCause{{
					Type:    fieldValueInvalidType,
					Message: "Start must be a valid port number, 0 < x < 65536",
					Field:   fakeDomainPortRangeField,
				}},
			),
			Entry(
				"end port out of range",
				[]v1.PortRange{{Protocol: tcpProtocol, Start: 80, End: 70000}},
				[]metav1.StatusCause{{
					Type:    fieldValueInvalidType,
					Message: "End must be a valid port number, 0 < x < 65536",
					Field:   "fake.domain.devices.interfaces[0].portRanges[0].end",
				}},
			),
			Entry(
				"start greater than end",
				[]v1.PortRange{{Protocol: tcpProtocol, Start: 100, End: 80}},
				[]metav1.StatusCause{{
					Type:    fieldValueInvalidType,
					Message: "Start must be less than or equal to end",
					Field:   fakeDomainPortRangeField,
				}},
			),
			Entry(
				"two TCP ranges overlapping",
				[]v1.PortRange{{Protocol: tcpProtocol, Start: 80, End: 200}, {Protocol: tcpProtocol, Start: 150, End: 300}},
				[]metav1.StatusCause{{
					Type:    fieldValueInvalidType,
					Message: "TCP portRanges [80-200] and [150-300] overlap",
					Field:   "fake.domain.devices.interfaces[0].portRanges",
				}},
			),
		)

		DescribeTable("should accept portRanges with", func(portRanges []v1.PortRange) {
			spec := &v1.VirtualMachineInstanceSpec{}
			spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name:                   net1Name,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				PortRanges:             portRanges,
			}}
			spec.Networks = []v1.Network{{Name: net1Name, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

			validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{portRangesSpecGateEnabled: true})
			Expect(validator.Validate()).To(BeEmpty())
		},
			Entry("single range", []v1.PortRange{{Protocol: tcpProtocol, Start: 80, End: 90}}),
			Entry("single port (start == end)", []v1.PortRange{{Protocol: tcpProtocol, Start: 22, End: 22}}),
			Entry(
				"two non-overlapping TCP ranges",
				[]v1.PortRange{{Protocol: tcpProtocol, Start: 80, End: 100}, {Protocol: tcpProtocol, Start: 200, End: 300}},
			),
			Entry(
				"TCP and UDP ranges that overlap (allowed)",
				[]v1.PortRange{{Protocol: tcpProtocol, Start: 80, End: 200}, {Protocol: udpProtocol, Start: 150, End: 300}},
			),
		)
	})

	When("the interface DHCP options is specified", func() {
		DescribeTable("should reject interface DHCP options with", func(dhcpOpts v1.DHCPOptions, expectedCauses []metav1.StatusCause) {
			spec := &v1.VirtualMachineInstanceSpec{}
			spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name:                   net1Name,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				DHCPOptions:            &dhcpOpts,
			}}
			spec.Networks = []v1.Network{{Name: net1Name, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

			validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
			Expect(validator.Validate()).To(ConsistOf(expectedCauses))
		},
			Entry(
				"invalid DHCPPrivateOptions",
				v1.DHCPOptions{PrivateOptions: []v1.DHCPPrivateOptions{{Option: 223, Value: extraOptionsDomain}}},
				[]metav1.StatusCause{{
					Type:    fieldValueInvalidType,
					Message: "provided DHCPPrivateOptions are out of range, must be in range 224 to 254",
					Field:   fakeDomainField,
				}},
			),
			Entry(
				"duplicate DHCPPrivateOptions",
				v1.DHCPOptions{
					PrivateOptions: []v1.DHCPPrivateOptions{
						{Option: 240, Value: extraOptionsDomain},
						{Option: 240, Value: "sameextra.options.kubevirt.io"},
					},
				},
				[]metav1.StatusCause{{
					Type:    fieldValueInvalidType,
					Message: "Found Duplicates: you have provided duplicate DHCPPrivateOptions",
					Field:   fakeDomainField,
				}},
			),
			Entry(
				"non-IPv4 NTP servers",
				v1.DHCPOptions{NTPServers: []string{"::1", "hostname"}},
				[]metav1.StatusCause{{
					Type:    fieldValueInvalidType,
					Message: ntpServersMsg,
					Field:   "fake.domain.devices.interfaces[0].dhcpOptions.ntpServers[0]",
				}, {
					Type:    fieldValueInvalidType,
					Message: ntpServersMsg,
					Field:   "fake.domain.devices.interfaces[0].dhcpOptions.ntpServers[1]",
				}},
			),
		)

		DescribeTable("should accept interface DHCP options with", func(dhcpOpts v1.DHCPOptions) {
			spec := &v1.VirtualMachineInstanceSpec{}
			spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name:                   net1Name,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				DHCPOptions:            &dhcpOpts,
			}}
			spec.Networks = []v1.Network{{Name: net1Name, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

			validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
			Expect(validator.Validate()).To(BeEmpty())
		},
			Entry("  valid DHCPPrivateOptions", v1.DHCPOptions{
				PrivateOptions: []v1.DHCPPrivateOptions{{Option: 240, Value: extraOptionsDomain}},
			}),
			Entry(" valid NTP servers", v1.DHCPOptions{NTPServers: []string{"127.0.0.1", "127.0.0.2"}}),
			Entry(
				"unique DHCPPrivateOptions",
				v1.DHCPOptions{
					PrivateOptions: []v1.DHCPPrivateOptions{
						{Option: 240, Value: extraOptionsDomain},
						{Option: 241, Value: extraOptionsDomain},
					},
				},
			),
		)
	})
})
