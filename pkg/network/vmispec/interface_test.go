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
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
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
		})
	})

	const iface1, iface2, iface3, iface4, iface5 = "iface1", "iface2", "iface3", "iface4", "iface5"

	DescribeTable("return VMI spec interface names, given",
		func(interfaces []v1.Interface, expectedNames []string) {
			Expect(netvmispec.InterfacesNames(interfaces)).To(Equal(expectedNames))
		},
		Entry("no interfaces", nil, nil),
		Entry("single interface", vmiSpecInterfaces(iface1), []string{iface1}),
		Entry("more then one interface", vmiSpecInterfaces(iface1, iface2, iface3), []string{iface1, iface2, iface3}),
	)

	It("filter status interfaces, given 0 interfaces and 0 names", func() {
		Expect(netvmispec.FilterStatusInterfacesByNames(nil, nil)).To(BeEmpty())
	})
	It("filter status interfaces, given 0 interfaces and 3 names", func() {
		names := []string{iface1, iface2, iface3}
		Expect(netvmispec.FilterStatusInterfacesByNames(nil, names)).To(BeEmpty())
	})
	It("filter status interfaces, given 3 interfaces and 0 names", func() {
		statusInterfaces := vmiStatusInterfaces(iface1, iface2, iface3)
		Expect(netvmispec.FilterStatusInterfacesByNames(statusInterfaces, nil)).To(BeEmpty())
	})
	It("filter status interfaces, given 5 interfaces and 2 names", func() {
		statusInterfaces := vmiStatusInterfaces(iface1, iface4, iface3, iface5, iface2)
		names := []string{iface4, iface5}
		expectedInterfaces := vmiStatusInterfaces(names...)
		Expect(netvmispec.FilterStatusInterfacesByNames(statusInterfaces, names)).To(Equal(expectedInterfaces))
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
