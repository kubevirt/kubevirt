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

	"kubevirt.io/kubevirt/pkg/network/admitter"
)

var _ = Describe("Validate network source", func() {
	It("support only a single pod network", func() {
		const net1Name = "default"
		const net2Name = "default2"
		vmi := v1.VirtualMachineInstance{}
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
			{Name: net1Name, InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
			{Name: net2Name, InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
		}
		vmi.Spec.Networks = []v1.Network{
			{Name: net1Name, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
			{Name: net2Name, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
		}
		clusterConfig := stubClusterConfigChecker{bridgeBindingOnPodNetEnabled: true}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, clusterConfig)
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal("more than one interface is connected to a pod network in fake.interfaces"))
	})

	It("should reject when multiple types defined for a CNI network", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
		spec.Networks = []v1.Network{
			{
				Name: "default",
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "default1"},
					Pod:    &v1.PodNetwork{},
				},
			},
		}

		clusterConfig := stubClusterConfigChecker{bridgeBindingOnPodNetEnabled: true}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, clusterConfig)
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal("should have only one network type"))
	})

	It("when network source is not configured", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		net1 := v1.Network{
			NetworkSource: v1.NetworkSource{},
			Name:          "testnet1",
		}
		iface1 := v1.Interface{Name: net1.Name}
		spec.Networks = []v1.Network{net1}
		spec.Domain.Devices.Interfaces = []v1.Interface{iface1}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal("should have a network type"))
	})

	It("should reject multus network source without networkName", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
		spec.Networks = []v1.Network{{
			Name:          "default",
			NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}},
		}}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal("CNI delegating plugin must have a networkName"))
	})

	It("should reject multiple multus networks with a multus default", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{
			*v1.DefaultBridgeNetworkInterface(),
			*v1.DefaultBridgeNetworkInterface(),
		}
		const net1Name = "multus1"
		const net2Name = "multus2"
		spec.Domain.Devices.Interfaces[0].Name = net1Name
		spec.Domain.Devices.Interfaces[1].Name = net2Name
		spec.Networks = []v1.Network{
			{
				Name: net1Name,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "multus-net1", Default: true},
				},
			},
			{
				Name: net2Name,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "multus-net2", Default: true},
				},
			},
		}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
		Expect(causes[0].Field).To(Equal("fake.networks"))
		Expect(causes[0].Message).To(Equal("Multus CNI should only have one default network"))
	})

	It("should reject pod network with a multus default", func() {
		const defaultMultusNetName = "defaultmultus"
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{
			*v1.DefaultBridgeNetworkInterface(),
			{
				Name: defaultMultusNetName,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Bridge: &v1.InterfaceBridge{},
				},
			},
		}

		spec.Networks = []v1.Network{
			{
				Name: "default",
				NetworkSource: v1.NetworkSource{
					Pod: &v1.PodNetwork{},
				},
			},
			{
				Name: defaultMultusNetName,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "multus-net1", Default: true},
				},
			},
		}

		clusterConfig := stubClusterConfigChecker{bridgeBindingOnPodNetEnabled: true}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, clusterConfig)
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
		Expect(causes[0].Field).To(Equal("fake.networks"))
		Expect(causes[0].Message).To(Equal("Pod network cannot be defined when Multus default network is defined"))
	})

	It("should allow single multus network with a multus default", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{
			*v1.DefaultBridgeNetworkInterface(),
		}
		spec.Domain.Devices.Interfaces[0].Name = "multus1"
		spec.Networks = []v1.Network{
			{
				Name: "multus1",
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "multus-net1", Default: true},
				},
			},
		}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		causes := validator.Validate()
		Expect(causes).To(BeEmpty())
	})

	It("should accept networks with a multus network source and bridge interface", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
		spec.Networks = []v1.Network{
			{
				Name: "default",
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "default"},
				},
			},
		}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		causes := validator.Validate()
		Expect(causes).To(BeEmpty())
	})

	It("should allow primary network and multiple secondary networks", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{
			{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Bridge: &v1.InterfaceBridge{},
				},
			},
			{
				Name: "multus1",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Bridge: &v1.InterfaceBridge{},
				},
			},
			{
				Name: "multus2",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Bridge: &v1.InterfaceBridge{},
				},
			},
		}

		spec.Networks = []v1.Network{
			{
				Name: "default",
				NetworkSource: v1.NetworkSource{
					Pod: &v1.PodNetwork{},
				},
			},
			{
				Name: "multus1",
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "multus-net1"},
				},
			},
			{
				Name: "multus2",
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "multus-net2"},
				},
			},
		}

		validator := admitter.NewValidator(
			k8sfield.NewPath("fake"),
			spec,
			stubClusterConfigChecker{bridgeBindingOnPodNetEnabled: true},
		)
		causes := validator.Validate()
		Expect(causes).To(BeEmpty())
	})
})
