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
	"crypto/sha256"
	"fmt"
	"io"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

const (
	// MaxIfaceNameLen equals max kernel interface name len (15) - length("-nic")
	// which is the suffix used for the bridge binding interface with IPAM.
	// (the interface created to hold the pod's IP address - and thus appease CNI).
	MaxIfaceNameLen         = 11
	PrimaryPodInterfaceName = "eth0"
)

// CreateNetworkNameScheme iterates over the VMI's Networks, and creates for each a pod interface name.
// The returned map associates between the network name and the generated pod interface name.
// Primary network will use "eth0" and the secondary ones will be named "net<id>" where id is the network name sha256.
func CreateNetworkNameScheme(vmiNetworks []v1.Network) map[string]string {
	networkNameSchemeMap := mapMultusNonDefaultNetworksToPodInterfaceName(vmiNetworks)
	if multusDefaultNetwork := vmispec.LookUpDefaultNetwork(vmiNetworks); multusDefaultNetwork != nil {
		networkNameSchemeMap[multusDefaultNetwork.Name] = PrimaryPodInterfaceName
	}
	return networkNameSchemeMap
}

func mapMultusNonDefaultNetworksToPodInterfaceName(networks []v1.Network) map[string]string {
	networkNameSchemeMap := map[string]string{}
	for _, network := range vmispec.FilterMultusNonDefaultNetworks(networks) {
		networkNameSchemeMap[network.Name] = hashNetworkName(network.Name)
	}
	return networkNameSchemeMap
}

func hashNetworkName(networkName string) string {
	hash := sha256.New()
	_, _ = io.WriteString(hash, networkName)
	return fmt.Sprintf("%x", hash.Sum(nil))[:MaxIfaceNameLen]
}
