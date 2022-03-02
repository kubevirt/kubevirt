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

import (
	v1 "kubevirt.io/api/core/v1"
)

const (
	InfoSourceDomain      string = "domain"
	InfoSourceGuestAgent  string = "guest-agent"
	InfoSourceDomainAndGA string = InfoSourceDomain + ", " + InfoSourceGuestAgent
)

func FilterSRIOVInterfaces(ifaces []v1.Interface) []v1.Interface {
	var sriovIfaces []v1.Interface
	for _, iface := range ifaces {
		if iface.SRIOV != nil {
			sriovIfaces = append(sriovIfaces, iface)
		}
	}
	return sriovIfaces
}

func FilterInterfacesSpec(ifaces []v1.Interface, predicate func(i v1.Interface) bool) []v1.Interface {
	var filteredIfaces []v1.Interface
	for _, iface := range ifaces {
		if predicate(iface) {
			filteredIfaces = append(filteredIfaces, iface)
		}
	}
	return filteredIfaces
}

func IsPodNetworkWithMasqueradeBindingInterface(networks []v1.Network, ifaces []v1.Interface) bool {
	if podNetwork := lookupPodNetwork(networks); podNetwork != nil {
		if podInterface := LookupInterfaceByNetwork(ifaces, podNetwork); podInterface != nil {
			return podInterface.Masquerade != nil
		}
	}
	return true
}

func LookupInterfaceStatusByMac(interfaces []v1.VirtualMachineInstanceNetworkInterface, macAddress string) *v1.VirtualMachineInstanceNetworkInterface {
	for index := range interfaces {
		if interfaces[index].MAC == macAddress {
			return &interfaces[index]
		}
	}
	return nil
}

func LookupInterfaceStatusByName(interfaces []v1.VirtualMachineInstanceNetworkInterface, name string) *v1.VirtualMachineInstanceNetworkInterface {
	for index := range interfaces {
		if interfaces[index].Name == name {
			return &interfaces[index]
		}
	}
	return nil
}

func IndexInterfaceSpecByName(interfaces []v1.Interface) map[string]v1.Interface {
	ifacesByName := map[string]v1.Interface{}
	for _, ifaceSpec := range interfaces {
		ifacesByName[ifaceSpec.Name] = ifaceSpec
	}
	return ifacesByName
}

func IndexInterfaceSpecByMac(interfaces []v1.Interface) map[string]v1.Interface {
	ifacesByMac := map[string]v1.Interface{}
	for _, ifaceSpec := range interfaces {
		if mac := ifaceSpec.MacAddress; mac != "" {
			ifacesByMac[mac] = ifaceSpec
		}
	}
	return ifacesByMac
}

func LookupInterfaceByNetwork(ifaces []v1.Interface, network *v1.Network) *v1.Interface {
	for _, iface := range ifaces {
		if iface.Name == network.Name {
			iface := iface
			return &iface
		}
	}
	return nil
}

// InterfacesNames returns slice with the names of the given interfaces.
func InterfacesNames(interfaces []v1.Interface) []string {
	var ifaceNames []string
	for _, iface := range interfaces {
		ifaceNames = append(ifaceNames, iface.Name)
	}
	return ifaceNames
}

// FilterStatusInterfacesByNames returns filtered slice of interfaces by the given slice of names.
// Matching by the interface 'Name' attribute.
func FilterStatusInterfacesByNames(interfaces []v1.VirtualMachineInstanceNetworkInterface, names []string) []v1.VirtualMachineInstanceNetworkInterface {
	lookupNameSet := map[string]struct{}{}
	for _, name := range names {
		lookupNameSet[name] = struct{}{}
	}

	var filtered []v1.VirtualMachineInstanceNetworkInterface
	for _, iface := range interfaces {
		if _, exists := lookupNameSet[iface.Name]; exists {
			filtered = append(filtered, iface)
		}
	}

	return filtered
}
