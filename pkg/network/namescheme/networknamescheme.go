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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package namescheme

import (
	"crypto/md5"
	"fmt"
	"io"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

const (
	MaxIfaceNameLen         = 11
	PrimaryPodInterfaceName = "eth0"
)

// CreateNetworkNameScheme iterates over the VMI's Networks, and creates for each a pod interface name.
// The returned map associates between the network name and the generated pod interface name.
// Primary network will use "eth0" and the secondary ones will use "net<id>" format, where id is an enumeration
// from 1 to n.
func CreateNetworkNameScheme(vmiNetworks []v1.Network) map[string]string {
	networkNameSchemeMap := mapMultusNonDefaultNetworksToPodInterfaceName(vmiNetworks)

	if multusDefaultNetwork := vmispec.LookUpDefaultNetwork(vmiNetworks); multusDefaultNetwork != nil {
		networkNameSchemeMap[multusDefaultNetwork.Name] = PrimaryPodInterfaceName
	}

	return networkNameSchemeMap
}

// CreateHashedNetworkNamingScheme iterates over the VMI's Networks, and creates for each a pod interface name.
// The returned map associates between the network name and the generated pod interface name.
// The generated interface names will be named "net<id>" where id is the network name md5.
func CreateHashedNetworkNamingScheme(vmiNetworks []v1.Network) map[string]string {
	return mapMultusNonDefaultNetworksToHashPodInterfaceName(vmiNetworks)
}

func mapMultusNonDefaultNetworksToPodInterfaceName(networks []v1.Network) map[string]string {
	networkNameSchemeMap := map[string]string{}
	for i, network := range vmispec.FilterMultusNonDefaultNetworks(networks) {
		networkNameSchemeMap[network.Name] = SecondaryInterfaceName(i + 1)
	}
	return networkNameSchemeMap
}

func mapMultusNonDefaultNetworksToHashPodInterfaceName(networks []v1.Network) map[string]string {
	networkNameSchemeMap := map[string]string{}
	for _, network := range vmispec.FilterMultusNonDefaultNetworks(networks) {
		networkNameSchemeMap[network.Name] = HashNetworkName(network.Name)
	}
	return networkNameSchemeMap
}

func SecondaryInterfaceName(idx int) string {
	return fmt.Sprintf("net%d", idx)
}

func HashNetworkName(networkName string) string {
	hash := md5.New()
	_, _ = io.WriteString(hash, networkName)
	return fmt.Sprintf("net%x", hash.Sum(nil))[:MaxIfaceNameLen]
}
