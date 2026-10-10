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

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/admitter"
)

var _ = Describe("Validate network source", func() {
	It("support only a single pod network", func() {
		const net1Name = "default"
		const net2Name = "default2"
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(net1Name)),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(net2Name)),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithNetwork(&v1.Network{Name: net2Name, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}),
		)

		clusterConfig := stubClusterConfigChecker{bridgeBindingOnPodNetEnabled: true}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, clusterConfig)
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal("more than one interface is connected to a pod network in fake.interfaces"))
	})

	It("should reject when multiple types defined for a CNI network", func() {
		vmi := libvmi.New(libvmi.WithInterface(
			libvmi.InterfaceDeviceWithBridgeBinding("default")),
			libvmi.WithNetwork(&v1.Network{
				Name: "default",
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "default1"},
					Pod:    &v1.PodNetwork{},
				},
			}),
		)

		clusterConfig := stubClusterConfigChecker{bridgeBindingOnPodNetEnabled: true}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, clusterConfig)
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal("should have only one network type"))
	})

	It("when network source is not configured", func() {
		vmi := libvmi.New(libvmi.WithInterface(
			libvmi.InterfaceDeviceWithBridgeBinding("testnet1")),
			libvmi.WithNetwork(&v1.Network{
				NetworkSource: v1.NetworkSource{},
				Name:          "testnet1",
			}),
		)

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{})
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal("should have a network type"))
	})

	It("should accept resourceClaim network type", func() {
		const draNetName = "dra-net"
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.NewInterface(
				draNetName,
				libvmi.WithBindingPlugin(v1.PluginBinding{Name: "netbinding"}),
			)),
			libvmi.WithResourceClaim(v1.VirtualMachineInstanceResourceClaim{
				Name:              "claim1",
				ResourceClaimName: new("claim1"),
			}),
			libvmi.WithNetwork(libvmi.DRANetwork(draNetName, "claim1", "request1")),
		)

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		Expect(causes).To(BeEmpty())
	})

	DescribeTable("should reject when resourceClaim is combined with another network type",
		func(networkSource v1.NetworkSource) {
			vmi := libvmi.New(
				libvmi.WithInterface(libvmi.NewInterface(
					"default",
					libvmi.WithBindingPlugin(v1.PluginBinding{Name: "netbinding"})),
				),
				libvmi.WithResourceClaim(v1.VirtualMachineInstanceResourceClaim{Name: "claim1", ResourceClaimName: new("claim1")}),
				libvmi.WithNetwork(&v1.Network{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						ResourceClaim: &v1.ClaimRequest{ClaimName: "claim1", RequestName: "request1"},
						Pod:           networkSource.Pod,
						Multus:        networkSource.Multus,
					},
				}),
			)

			validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{networkDRAEnabled: true})
			causes := validator.Validate()
			Expect(causes).To(ContainElement(HaveField("Message", "should have only one network type")))
		},
		Entry("pod network", v1.NetworkSource{Pod: &v1.PodNetwork{}}),
		Entry("multus network", v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "default1"}}),
	)

	It("should reject multus network source without networkName", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("default")),
			libvmi.WithNetwork(libvmi.MultusNetwork("default", "")),
		)

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{})
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal("CNI delegating plugin must have a networkName"))
	})

	It("should reject multiple multus networks with a multus default", func() {
		const net1Name = "multus1"
		const net2Name = "multus2"
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(net1Name)),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(net2Name)),
			libvmi.WithNetwork(&v1.Network{
				Name: net1Name,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "multus-net1", Default: true},
				},
			}),
			libvmi.WithNetwork(&v1.Network{
				Name: net2Name,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "multus-net2", Default: true},
				},
			}),
		)

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{})
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
		Expect(causes[0].Field).To(Equal("fake.networks"))
		Expect(causes[0].Message).To(Equal("Multus CNI should only have one default network"))
	})

	It("should reject pod network with a multus default", func() {
		const defaultMultusNetName = "defaultmultus"
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("default")),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(defaultMultusNetName)),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithNetwork(&v1.Network{
				Name: defaultMultusNetName,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "multus-net1", Default: true},
				},
			}),
		)

		clusterConfig := stubClusterConfigChecker{bridgeBindingOnPodNetEnabled: true}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, clusterConfig)
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
		Expect(causes[0].Field).To(Equal("fake.networks"))
		Expect(causes[0].Message).To(Equal("Pod network cannot be defined when Multus default network is defined"))
	})

	It("should allow single multus network with a multus default", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("multus1")),
			libvmi.WithNetwork(&v1.Network{
				Name: "multus1",
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "multus-net1", Default: true},
				},
			}),
		)

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{})
		causes := validator.Validate()
		Expect(causes).To(BeEmpty())
	})

	It("should accept networks with a multus network source and bridge interface", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("default")),
			libvmi.WithNetwork(libvmi.MultusNetwork("default", "default")),
		)

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{})
		causes := validator.Validate()
		Expect(causes).To(BeEmpty())
	})

	It("should allow primary network and multiple secondary networks", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("default")),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("multus1")),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("multus2")),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithNetwork(libvmi.MultusNetwork("multus1", "multus-net1")),
			libvmi.WithNetwork(libvmi.MultusNetwork("multus2", "multus-net2")),
		)

		validator := admitter.NewValidator(
			k8sfield.NewPath("fake"),
			&vmi.Spec,
			stubClusterConfigChecker{bridgeBindingOnPodNetEnabled: true},
		)
		causes := validator.Validate()
		Expect(causes).To(BeEmpty())
	})
})
