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

package vmispec

import v1 "kubevirt.io/api/core/v1"

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
	var multusNetworks []v1.Network
	for _, network := range networks {
		if IsSecondaryMultusNetwork(network) {
			multusNetworks = append(multusNetworks, network)
		}
	}
	return multusNetworks
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
