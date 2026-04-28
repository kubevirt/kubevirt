/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package vmispec

import v1 "kubevirt.io/api/core/v1"

func LookupNetworkByName(networks []v1.Network, name string) *v1.Network {
	for i := range networks {
		if networks[i].Name == name {
			return &networks[i]
		}
	}
	return nil
}

func LookupPodNetwork(networks []v1.Network) *v1.Network {
	for _, network := range networks {
		if network.Pod != nil {
			net := network
			return &net
		}
	}
	return nil
}

func FilterMultusNonDefaultNetworks(networks []v1.Network) []v1.Network {
	return FilterNetworksSpec(networks, IsSecondaryMultusNetwork)
}

func FilterNetworksSpec(nets []v1.Network, predicate func(i v1.Network) bool) []v1.Network {
	var filteredNets []v1.Network
	for _, net := range nets {
		if predicate(net) {
			filteredNets = append(filteredNets, net)
		}
	}
	return filteredNets
}

func LookUpDefaultNetwork(networks []v1.Network) *v1.Network {
	for i, network := range networks {
		if !IsSecondaryMultusNetwork(network) {
			return &networks[i]
		}
	}
	return nil
}

func IsSecondaryMultusNetwork(net v1.Network) bool {
	return net.Multus != nil && !net.Multus.Default
}

func IndexNetworkSpecByName(networks []v1.Network) map[string]v1.Network {
	indexedNetworks := map[string]v1.Network{}
	for _, network := range networks {
		indexedNetworks[network.Name] = network
	}
	return indexedNetworks
}

func FilterNetworksByInterfaces(networks []v1.Network, interfaces []v1.Interface) []v1.Network {
	var nets []v1.Network
	networksByName := IndexNetworkSpecByName(networks)
	for _, iface := range interfaces {
		if net, exists := networksByName[iface.Name]; exists {
			nets = append(nets, net)
		}
	}
	return nets
}
