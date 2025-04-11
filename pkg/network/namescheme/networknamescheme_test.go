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

package namescheme_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/namescheme"
)

var _ = Describe("Network Name Scheme", func() {
	DescribeTable("create pod interfaces name scheme",
		func(nameSchemeFn func([]virtv1.Network) map[string]string, networkList []virtv1.Network, expectedNetworkNameScheme map[string]string) {
			podIfacesNameScheme := nameSchemeFn(networkList)

			Expect(podIfacesNameScheme).To(Equal(expectedNetworkNameScheme))
		},
		Entry("hashed, when network list is nil", namescheme.CreateHashedNetworkNameScheme, nil, map[string]string{}),
		Entry("hashed, when no multus networks exist",
			namescheme.CreateHashedNetworkNameScheme,
			[]virtv1.Network{
				newPodNetwork(),
			},
			map[string]string{
				"default": namescheme.PrimaryPodInterfaceName,
			}),
		Entry("hashed, when default multus networks exist",
			namescheme.CreateHashedNetworkNameScheme,
			[]virtv1.Network{
				createMultusDefaultNetwork("network0", "default/nad0"),
				createMultusSecondaryNetwork("network1", "default/nad1"),
				createMultusSecondaryNetwork("network2", "default/nad2"),
			},
			map[string]string{
				"network0": namescheme.PrimaryPodInterfaceName,
				"network1": "poda7662f44d65",
				"network2": "pod27f4a77f94e",
			}),
		Entry("hashed, when default pod networks exist",
			namescheme.CreateHashedNetworkNameScheme,
			[]virtv1.Network{
				newPodNetwork(),
				createMultusSecondaryNetwork("network1", "default/nad1"),
				createMultusSecondaryNetwork("network2", "default/nad2"),
			},
			map[string]string{
				"default":  namescheme.PrimaryPodInterfaceName,
				"network1": "poda7662f44d65",
				"network2": "pod27f4a77f94e",
			}),
		Entry("ordinal, when network list is nil", namescheme.CreateOrdinalNetworkNameScheme, nil, map[string]string{}),
		Entry("ordinal, when no multus networks exist",
			namescheme.CreateOrdinalNetworkNameScheme,
			[]virtv1.Network{
				newPodNetwork(),
			},
			map[string]string{
				"default": namescheme.PrimaryPodInterfaceName,
			}),
		Entry("ordinal, when default multus networks exist",
			namescheme.CreateOrdinalNetworkNameScheme,
			[]virtv1.Network{
				createMultusDefaultNetwork("network0", "default/nad0"),
				createMultusSecondaryNetwork("network1", "default/nad1"),
				createMultusSecondaryNetwork("network2", "default/nad2"),
			},
			map[string]string{
				"network0": namescheme.PrimaryPodInterfaceName,
				"network1": "net1",
				"network2": "net2",
			}),
		Entry("ordinal, when default pod networks exist",
			namescheme.CreateOrdinalNetworkNameScheme,
			[]virtv1.Network{
				newPodNetwork(),
				createMultusSecondaryNetwork("network1", "default/nad1"),
				createMultusSecondaryNetwork("network2", "default/nad2"),
			},
			map[string]string{
				"default":  namescheme.PrimaryPodInterfaceName,
				"network1": "net1",
				"network2": "net2",
			}),
	)

	Context("CreateFromNetworkStatuses", func() {
		const (
			network1Name         = "red"
			podIface1HashedName  = "podb1f51a511f1"
			podIface1OrdinalName = "net1"

			network2Name         = "green"
			podIface2HashedName  = "podba4788b226a"
			podIface2OrdinalName = "net2"
		)

		DescribeTable("should map VMI network names to pod interfaces names",
			func(networks []virtv1.Network, networkStatuses []networkv1.NetworkStatus, expectedNameScheme map[string]string) {
				Expect(namescheme.CreateFromNetworkStatuses(networks, networkStatuses)).To(Equal(expectedNameScheme))
			},
			Entry("when network status slice is empty",
				multusNetworks(network1Name, network2Name),
				[]networkv1.NetworkStatus{},
				map[string]string{network1Name: podIface1HashedName, network2Name: podIface2HashedName},
			),
			Entry("given only pod network",
				[]virtv1.Network{newPodNetwork()},
				[]networkv1.NetworkStatus{{Interface: namescheme.PrimaryPodInterfaceName}},
				map[string]string{"default": namescheme.PrimaryPodInterfaceName},
			),
			Entry("when the pod interfaces use a hashed naming scheme",
				multusNetworks(network1Name, network2Name),
				[]networkv1.NetworkStatus{{Interface: podIface1HashedName}, {Interface: podIface2HashedName}},
				map[string]string{network1Name: podIface1HashedName, network2Name: podIface2HashedName},
			),
			Entry("when the pod interfaces use an ordinal naming scheme",
				multusNetworks(network1Name, network2Name),
				[]networkv1.NetworkStatus{{Interface: podIface1OrdinalName}, {Interface: podIface2OrdinalName}},
				map[string]string{network1Name: podIface1OrdinalName, network2Name: podIface2OrdinalName},
			),
		)
	})

	Context("PodHasOrdinalInterfaceName", func() {
		DescribeTable("should return TRUE, given network status with ordinal interface names",
			func(podNetworkStatuses []networkv1.NetworkStatus) {
				Expect(namescheme.PodHasOrdinalInterfaceName(podNetworkStatuses)).To(BeTrue())
			},
			Entry("with primary pod network interface",
				[]networkv1.NetworkStatus{
					{Interface: "eth0"},
					{Interface: "net1"},
					{Interface: "net2"},
				}),
			Entry("without primary pod network interface",
				[]networkv1.NetworkStatus{
					{Interface: "net1"},
					{Interface: "net2"},
				}),
		)

		DescribeTable("should return FALSE",
			func(podNetworkStatuses []networkv1.NetworkStatus) {
				Expect(namescheme.PodHasOrdinalInterfaceName(podNetworkStatuses)).To(BeFalse())
			},
			Entry("When networks statutes is empty", []networkv1.NetworkStatus{}),
			Entry("when networks statutes has primary pod and hashed secondary network interface",
				[]networkv1.NetworkStatus{
					{Interface: "eth0"},
					{Interface: "podb1f51a511f1"},
					{Interface: "pod16477688c0e"},
				}),
			Entry("when networks statutes has hashed secondary network interface",
				[]networkv1.NetworkStatus{
					{Interface: "podb1f51a511f1"},
					{Interface: "pod16477688c0e"},
				}),
		)
	})

	Context("HashedPodInterfaceName", func() {
		DescribeTable("should return the given network name's hashed pod interface name",
			func(network virtv1.Network, expectedPodIfaceName string) {
				Expect(namescheme.HashedPodInterfaceName(network, []virtv1.VirtualMachineInstanceNetworkInterface{})).To(Equal(expectedPodIfaceName))
			},
			Entry("given default network name when default is pod network",
				newPodNetwork(),
				"eth0",
			),
			Entry("given default network name when default is Multus default network",
				createMultusDefaultNetwork("overlay-network", "pod-net-br"),
				"eth0",
			),
			Entry("given secondary network name",
				createMultusSecondaryNetwork("red", "test-br"),
				"podb1f51a511f1",
			),
		)
	})
	Context("OrdinalPodInterfaceName", func() {
		DescribeTable("should return empty string",
			func(networkName string, networks []virtv1.Network) {
				Expect(namescheme.OrdinalPodInterfaceName(networkName, networks)).To(BeEmpty())
			},
			Entry("given no networks",
				"red",
				nil,
			),
			Entry("given invalid network name",
				"blah",
				[]virtv1.Network{
					newPodNetwork(),
					createMultusSecondaryNetwork("blue", "test-br"),
					createMultusSecondaryNetwork("red", "test-br"),
				}),
		)

		DescribeTable("should return ordinal pod interface name",
			func(networkName string, networks []virtv1.Network, expectedPodIfaceName string) {
				Expect(namescheme.OrdinalPodInterfaceName(networkName, networks)).To(Equal(expectedPodIfaceName))
			},
			Entry("given default network name and default is pod network",
				"default",
				[]virtv1.Network{
					newPodNetwork(),
					createMultusSecondaryNetwork("blue", "test-br"),
					createMultusSecondaryNetwork("red", "test-br"),
				},
				"eth0",
			),
			Entry("given default network name and default is Multus default network",
				"overlay",
				[]virtv1.Network{
					createMultusDefaultNetwork("overlay", "pod-net-br"),
					createMultusSecondaryNetwork("blue", "test-br"),
					createMultusSecondaryNetwork("red", "test-br"),
				},
				"eth0",
			),
			Entry("given secondary network name",
				"red",
				[]virtv1.Network{
					newPodNetwork(),
					createMultusSecondaryNetwork("blue", "test-br"),
					createMultusSecondaryNetwork("red", "test-br"),
				},
				"net2",
			),
			Entry("given secondary network name with different order",
				"multus01",
				[]virtv1.Network{
					createMultusSecondaryNetwork("blue", "test-br"),
					createMultusSecondaryNetwork("multus01", "test-br"),
					newPodNetwork(),
				},
				"net2",
			),
			Entry("given secondary network name, only one secondary network",
				"multus01",
				[]virtv1.Network{
					createMultusSecondaryNetwork("multus01", "test-br"),
				},
				"net1",
			),
		)
	})
})

func multusNetworks(names ...string) []virtv1.Network {
	var networks []virtv1.Network
	for _, name := range names {
		networks = append(networks, createMultusNetwork(name, name+"net"))
	}
	return networks
}

func createMultusSecondaryNetwork(name, networkName string) virtv1.Network {
	return createMultusNetwork(name, networkName)
}

func createMultusDefaultNetwork(name, networkName string) virtv1.Network {
	multusNetwork := createMultusNetwork(name, networkName)
	multusNetwork.Multus.Default = true
	return multusNetwork
}

func createMultusNetwork(name, networkName string) virtv1.Network {
	return virtv1.Network{
		Name: name,
		NetworkSource: virtv1.NetworkSource{
			Multus: &virtv1.MultusNetwork{
				NetworkName: networkName,
			},
		},
	}
}

func newPodNetwork() virtv1.Network {
	return virtv1.Network{
		Name: "default",
		NetworkSource: virtv1.NetworkSource{
			Pod: &virtv1.PodNetwork{},
		},
	}
}
