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

func IsPodNetworkWithMasqueradeBindingInterface(networks []v1.Network, ifaces []v1.Interface) bool {
	if podNetwork := lookupPodNetwork(networks); podNetwork != nil {
		if podInterface := lookupInterfaceByNetwork(ifaces, podNetwork); podInterface != nil {
			return podInterface.Masquerade != nil
		}
	}
	return true
}

func LookupInterfaceStatusByMac(interfaces []v1.VirtualMachineInstanceNetworkInterface, macAddress string) *v1.VirtualMachineInstanceNetworkInterface {
	for _, iface := range interfaces {
		if iface.MAC == macAddress {
			iface := iface
			return &iface
		}
	}

	return nil
}

func lookupInterfaceByNetwork(ifaces []v1.Interface, network *v1.Network) *v1.Interface {
	for _, iface := range ifaces {
		if iface.Name == network.Name {
			iface := iface
			return &iface
		}
	}
	return nil
}
