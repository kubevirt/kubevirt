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
	"maps"
	"regexp"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

const (
	// maxIfaceNameLen equals max kernel interface name len (15) - length("-nic")
	// which is the suffix used for the bridge binding interface with IPAM.
	// (the interface created to hold the pod's IP address - and thus appease CNI).
	maxIfaceNameLen   = 11
	HashedIfacePrefix = "pod"

	PrimaryPodInterfaceName = "eth0"
)

// CreateHashedNetworkNameScheme iterates over the VMI's Networks, and creates for each a pod interface name.
// The returned map associates between the network name and the generated pod interface name.
// Primary network will use "eth0" and the secondary ones will be named with the hashed network name.
func CreateHashedNetworkNameScheme(vmiNetworks []v1.Network) map[string]string {
	networkNameSchemeMap := mapMultusNonDefaultNetworksToPodInterfaceName(vmiNetworks)

	if multusDefaultNetwork := vmispec.LookUpDefaultNetwork(vmiNetworks); multusDefaultNetwork != nil {
		networkNameSchemeMap[multusDefaultNetwork.Name] = PrimaryPodInterfaceName
	}

	return networkNameSchemeMap
}

func HashedPodInterfaceName(network v1.Network, ifaceStatuses []v1.VirtualMachineInstanceNetworkInterface) string {
	if vmispec.IsSecondaryMultusNetwork(network) {
		return GenerateHashedInterfaceName(network.Name)
	}

	if primaryIfaceStatus := vmispec.LookupInterfaceStatusByName(ifaceStatuses, network.Name); primaryIfaceStatus != nil &&
		primaryIfaceStatus.PodInterfaceName != "" {
		return primaryIfaceStatus.PodInterfaceName
	}

	return PrimaryPodInterfaceName
}

func mapMultusNonDefaultNetworksToPodInterfaceName(networks []v1.Network) map[string]string {
	networkNameSchemeMap := map[string]string{}
	for _, network := range vmispec.FilterMultusNonDefaultNetworks(networks) {
		networkNameSchemeMap[network.Name] = GenerateHashedInterfaceName(network.Name)
	}
	return networkNameSchemeMap
}

func GenerateHashedInterfaceName(networkName string) string {
	hash := sha256.New()
	_, _ = io.WriteString(hash, networkName)
	hashedName := fmt.Sprintf("%x", hash.Sum(nil))[:maxIfaceNameLen]
	return fmt.Sprintf("%s%s", HashedIfacePrefix, hashedName)
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

// OrdinalPodInterfaceName returns the ordinal interface name for the given network name.
// Rereuse the `CreateOrdinalNetworkNameScheme` for various networks helps find the target interface name.
func OrdinalPodInterfaceName(name string, networks []v1.Network) string {
	networkNameSchemeMap := CreateOrdinalNetworkNameScheme(networks)
	if ordinalName, exist := networkNameSchemeMap[name]; exist {
		return ordinalName
	}
	return ""
}

func mapMultusNonDefaultNetworksToPodInterfaceOrdinalName(networks []v1.Network) map[string]string {
	networkNameSchemeMap := map[string]string{}
	for i, network := range vmispec.FilterMultusNonDefaultNetworks(networks) {
		networkNameSchemeMap[network.Name] = generateOrdinalInterfaceName(i + 1)
	}

	return networkNameSchemeMap
}

func generateOrdinalInterfaceName(idx int) string {
	const ordinalIfacePrefix = "net"
	return fmt.Sprintf("%s%d", ordinalIfacePrefix, idx)
}

// CreateFromNetworkStatuses creates a mapping of network name to pod interface name
// based on the given VMI spec networks and pod network statuses.
// In case the pod network status list has at least one interface with ordinal interface name -
// the function will return an ordinal network name-scheme.
func CreateFromNetworkStatuses(networks []v1.Network, networkStatuses []networkv1.NetworkStatus) map[string]string {
	if PodHasOrdinalInterfaceName(networkStatuses) {
		return CreateOrdinalNetworkNameScheme(networks)
	}

	return CreateHashedNetworkNameScheme(networks)
}

// PodHasOrdinalInterfaceName checks if the given pod network status has at least one pod interface with ordinal name
func PodHasOrdinalInterfaceName(networkStatuses []networkv1.NetworkStatus) bool {
	for _, networkStatus := range networkStatuses {
		if OrdinalSecondaryInterfaceName(networkStatus.Interface) {
			return true
		}
	}
	return false
}

// OrdinalSecondaryInterfaceName check if the given name is in form of the ordinal
// name scheme (e.g.: net1, net2..).
// Primary iface name (eth0) is treated as non-ordinal interface name.
func OrdinalSecondaryInterfaceName(name string) bool {
	const ordinalIfaceNameRegex = `^net\d+$`
	match, err := regexp.MatchString(ordinalIfaceNameRegex, name)
	if err != nil {
		return false
	}
	return match
}

func UpdatePrimaryPodIfaceNameFromVMIStatus(
	podIfaceNamesByNetworkName map[string]string,
	networks []v1.Network,
	ifaceStatuses []v1.VirtualMachineInstanceNetworkInterface,
) map[string]string {
	primaryNetwork := vmispec.LookUpDefaultNetwork(networks)
	if primaryNetwork == nil {
		return podIfaceNamesByNetworkName
	}

	primaryIfaceStatus := vmispec.LookupInterfaceStatusByName(ifaceStatuses, primaryNetwork.Name)
	if primaryIfaceStatus == nil || primaryIfaceStatus.PodInterfaceName == "" {
		return podIfaceNamesByNetworkName
	}

	updatedPodIfaceNamesByNetworkName := maps.Clone(podIfaceNamesByNetworkName)
	updatedPodIfaceNamesByNetworkName[primaryNetwork.Name] = primaryIfaceStatus.PodInterfaceName

	return updatedPodIfaceNamesByNetworkName
}
