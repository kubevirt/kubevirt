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
	"strconv"
	"strings"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

const (
	// maxIfaceNameLen equals max kernel interface name len (15) - length("-nic")
	// which is the suffix used for the bridge binding interface with IPAM.
	// (the interface created to hold the pod's IP address - and thus appease CNI).
	maxIfaceNameLen         = 11
	PrimaryPodInterfaceName = "eth0"
)

// CreateNetworkNameScheme iterates over the VMI's Networks, and creates for each a pod interface name.
// The returned map associates between the network name and the generated pod interface name.
// Primary network will use "eth0" and the secondary ones will be named with the hashed network name.
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
	return fmt.Sprintf("%x", hash.Sum(nil))[:maxIfaceNameLen]
}

// CreateOrdinalNetworkNameScheme iterates over the VMI's Networks, and creates for each a pod interface name.
// The returned map associates between the network name and the generated pod interface name.
// Primary network will use "eth0" and the secondary ones will use "net<id>" format, where id is an enumeration
// from 1 to n.
func CreateOrdinalNetworkNameScheme(vmiNetworks []v1.Network) map[string]string {
	networkNameSchemeMap := mapMultusNonDefaultNetworksToPodInterfaceOrdinalName(vmiNetworks)

	if multusDefaultNetwork := vmispec.LookUpDefaultNetwork(vmiNetworks); multusDefaultNetwork != nil {
		networkNameSchemeMap[multusDefaultNetwork.Name] = PrimaryPodInterfaceName
	}

	return networkNameSchemeMap
}

func mapMultusNonDefaultNetworksToPodInterfaceOrdinalName(networks []v1.Network) map[string]string {
	networkNameSchemeMap := map[string]string{}
	for i, network := range vmispec.FilterMultusNonDefaultNetworks(networks) {
		networkNameSchemeMap[network.Name] = secondaryInterfaceIndexedName(i + 1)
	}

	return networkNameSchemeMap
}

func secondaryInterfaceIndexedName(idx int) string {
	const interfaceNamePrefix = "net"
	return fmt.Sprintf("%s%d", interfaceNamePrefix, idx)
}

// CreateNetworkNameSchemeByPodNetworkStatus create pod network name scheme according to the given VMI spec networks
// and pod network status.
// In case the pod network status has at least one interface with ordinal interface name it returns the ordinal network
// name-scheme, Otherwise, returns the hashed network name scheme.
func CreateNetworkNameSchemeByPodNetworkStatus(networks []v1.Network, podNetworkStatus map[string]networkv1.NetworkStatus) map[string]string {
	if podHasOrdinalInterfaceName(podNetworkStatus) {
		return CreateOrdinalNetworkNameScheme(networks)
	}

	return CreateNetworkNameScheme(networks)
}

// podHasOrdinalInterfaceName check if the given pod network status has at least one pod interface with ordinal name
func podHasOrdinalInterfaceName(podNetworkStatus map[string]networkv1.NetworkStatus) bool {
	for _, networkStatus := range podNetworkStatus {
		if networkStatus.Interface == PrimaryPodInterfaceName {
			continue
		}
		if ordinalInterfaceName(networkStatus.Interface) {
			return true
		}
	}
	return false
}

// ordinalInterfaceName check if the given name is in form of the ordinal
// name scheme (e.g.: net1, net2..).
func ordinalInterfaceName(name string) bool {
	trimmedName := strings.TrimPrefix(name, "net")
	if _, err := strconv.Atoi(trimmedName); err != nil {
		return false
	}

	return true
}
