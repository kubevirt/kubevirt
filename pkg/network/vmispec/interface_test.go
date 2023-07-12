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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package vmispec_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
)

var _ = Describe("VMI network spec", func() {

	Context("pod network", func() {
		const podNet0 = "podnet0"

		networks := []v1.Network{podNetwork(podNet0)}

		It("does not exist", func() {
			ifaces := []v1.Interface{interfaceWithBridgeBinding(podNet0)}
			Expect(netvmispec.IsPodNetworkWithMasqueradeBindingInterface([]v1.Network{}, ifaces)).To(BeTrue())
		})

		It("is used by a masquerade interface", func() {
			ifaces := []v1.Interface{interfaceWithMasqueradeBinding(podNet0)}
			Expect(netvmispec.IsPodNetworkWithMasqueradeBindingInterface(networks, ifaces)).To(BeTrue())
		})

		It("used by a non-masquerade interface", func() {
			ifaces := []v1.Interface{interfaceWithBridgeBinding(podNet0)}
			Expect(netvmispec.IsPodNetworkWithMasqueradeBindingInterface(networks, ifaces)).To(BeFalse())
		})
	})

	Context("SR-IOV", func() {
		It("finds no SR-IOV interfaces in list", func() {
			ifaces := []v1.Interface{
				{
					Name:                   "net0",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				},
				{
					Name:                   "net1",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
				},
			}

			Expect(netvmispec.FilterSRIOVInterfaces(ifaces)).To(BeEmpty())
			Expect(netvmispec.SRIOVInterfaceExist(ifaces)).To(BeFalse())
		})

		It("finds two SR-IOV interfaces in list", func() {
			sriov_net1 := v1.Interface{
				Name:                   "sriov-net1",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}},
			}
			sriov_net2 := v1.Interface{
				Name:                   "sriov-net2",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}},
			}

			ifaces := []v1.Interface{
				{
					Name:                   "masq-net0",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				},
				sriov_net1,
				sriov_net2,
			}

			Expect(netvmispec.FilterSRIOVInterfaces(ifaces)).To(Equal([]v1.Interface{sriov_net1, sriov_net2}))
			Expect(netvmispec.SRIOVInterfaceExist(ifaces)).To(BeTrue())
		})
	})

	const iface1, iface2, iface3, iface4, iface5 = "iface1", "iface2", "iface3", "iface4", "iface5"

	Context("pop interface by network", func() {
		const netName = "net1"
		network := podNetwork(netName)
		expectedStatusIfaces := vmiStatusInterfaces(iface1, iface2)

		It("has no network", func() {
			statusIface, statusIfaces := netvmispec.PopInterfaceByNetwork(vmiStatusInterfaces(iface1, iface2), nil)
			Expect(statusIface).To(BeNil())
			Expect(statusIfaces).To(Equal(expectedStatusIfaces))
		})

		It("has no interfaces", func() {
			statusIface, _ := netvmispec.PopInterfaceByNetwork(nil, &network)
			Expect(statusIface).To(BeNil())
		})

		It("interface not found", func() {
			statusIface, _ := netvmispec.PopInterfaceByNetwork(vmiStatusInterfaces(iface1, iface2), &network)
			Expect(statusIface).To(BeNil())
		})

		DescribeTable("pop interface from position", func(statusIfaces []v1.VirtualMachineInstanceNetworkInterface) {
			expectedStatusIface := v1.VirtualMachineInstanceNetworkInterface{Name: netName}
			statusIface, statusIfaces := netvmispec.PopInterfaceByNetwork(statusIfaces, &network)
			Expect(*statusIface).To(Equal(expectedStatusIface))
			Expect(statusIfaces).To(Equal(expectedStatusIfaces))
		},
			Entry("first", vmiStatusInterfaces(netName, iface1, iface2)),
			Entry("last", vmiStatusInterfaces(iface1, iface2, netName)),
			Entry("mid", vmiStatusInterfaces(iface1, netName, iface2)),
		)
	})
})

func podNetwork(name string) v1.Network {
	return v1.Network{
		Name:          name,
		NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
	}
}

func interfaceWithBridgeBinding(name string) v1.Interface {
	return v1.Interface{
		Name:                   name,
		InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
	}
}

func interfaceWithMasqueradeBinding(name string) v1.Interface {
	return v1.Interface{
		Name:                   name,
		InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
	}
}

func vmiStatusInterfaces(names ...string) []v1.VirtualMachineInstanceNetworkInterface {
	var statusInterfaces []v1.VirtualMachineInstanceNetworkInterface
	for _, name := range names {
		iface := v1.VirtualMachineInstanceNetworkInterface{Name: name}
		statusInterfaces = append(statusInterfaces, iface)
	}
	return statusInterfaces
}

func vmiSpecInterfaces(names ...string) []v1.Interface {
	var specInterfaces []v1.Interface
	for _, name := range names {
		specInterfaces = append(specInterfaces, v1.Interface{Name: name})
	}
	return specInterfaces
}
