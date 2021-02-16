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

package network

import (
	"fmt"
	"net"

	"github.com/coreos/go-iptables/iptables"

	v1 "kubevirt.io/client-go/api/v1"
)

var BridgeFakeIP = "169.254.75.1%d/32"

func GetNetworksAndCniNetworks(vmi *v1.VirtualMachineInstance) (map[string]*v1.Network, map[string]int) {
	networks := map[string]*v1.Network{}
	cniNetworks := map[string]int{}
	for _, network := range vmi.Spec.Networks {
		networks[network.Name] = network.DeepCopy()
		if networks[network.Name].Multus != nil && !networks[network.Name].Multus.Default {
			// multus pod interfaces start from 1
			cniNetworks[network.Name] = len(cniNetworks) + 1
		}
	}
	return networks, cniNetworks
}

func GetPodInterfaceName(network *v1.Network, cniNetworkIndex int) string {
	if network.Multus != nil && !network.Multus.Default {
		// multus pod interfaces named netX
		return fmt.Sprintf("net%d", cniNetworkIndex)
	} else {
		return PrimaryPodInterfaceName
	}
}

func RetrieveMacAddress(iface *v1.Interface) (*net.HardwareAddr, error) {
	if iface.MacAddress != "" {
		macAddress, err := net.ParseMAC(iface.MacAddress)
		if err != nil {
			return nil, err
		}
		return &macAddress, nil
	}
	return nil, nil
}

func GetFakeBridgeIP(ifaces []v1.Interface, ifaceName string) (string, error) {
	for i, iface := range ifaces {
		if iface.Name == ifaceName {
			return fmt.Sprintf(BridgeFakeIP, i), nil
		}
	}
	return "", fmt.Errorf("Failed to generate bridge fake address for interface %s", ifaceName)
}

func GetLoopbackAdrress(proto iptables.Protocol) string {
	if proto == iptables.ProtocolIPv4 {
		return "127.0.0.1"
	} else {
		return "::1"
	}
}
