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

package libnet

import (
	"fmt"
	"time"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/console"
)

func InterfaceExists(vmi *v1.VirtualMachineInstance, interfaceName string) error {
	const timeout = 15 * time.Second
	cmdCheck := fmt.Sprintf("ip link show %s\n", interfaceName)
	if err := console.RunCommand(vmi, cmdCheck, timeout); err != nil {
		return fmt.Errorf("could not check interface: interface %s was not found in the VMI %s: %w", interfaceName, vmi.Name, err)
	}
	return nil
}

func AddIPAddress(vmi *v1.VirtualMachineInstance, interfaceName, interfaceAddress string) error {
	const addrAddTimeout = time.Second * 5
	setStaticIPCmd := fmt.Sprintf("ip addr add %s dev %s\n", interfaceAddress, interfaceName)

	if err := console.RunCommand(vmi, setStaticIPCmd, addrAddTimeout); err != nil {
		return fmt.Errorf("could not configure address %s for interface %s on VMI %s: %w", interfaceAddress, interfaceName, vmi.Name, err)
	}

	return nil
}

func SetInterfaceUp(vmi *v1.VirtualMachineInstance, interfaceName string) error {
	const ifaceUpTimeout = time.Second * 5
	setUpCmd := fmt.Sprintf("ip link set %s up\n", interfaceName)

	if err := console.RunCommand(vmi, setUpCmd, ifaceUpTimeout); err != nil {
		return fmt.Errorf("could not set interface %s up on VMI %s: %w", interfaceName, vmi.Name, err)
	}

	return nil
}

func LookupNetworkByName(networks []v1.Network, name string) *v1.Network {
	for i, net := range networks {
		if net.Name == name {
			return &networks[i]
		}
	}

	return nil
}

func NewLinkStateAssersionCmd(mac string, desiredLinkState v1.InterfaceState) string {
	const (
		linkStateUPRegex   = "'state[[:space:]]+UP'"
		linkStateDOWNRegex = "'NO-CARRIER.+state[[:space:]]+DOWN'"
		ipLinkTemplate     = "ip -one link | grep %s | grep -E %s\n"
	)

	var linkStateRegex string

	switch desiredLinkState {
	case v1.InterfaceStateLinkUp:
		linkStateRegex = linkStateUPRegex
	case v1.InterfaceStateLinkDown:
		linkStateRegex = linkStateDOWNRegex
	case v1.InterfaceStateAbsent:
		// noop
	}

	return fmt.Sprintf(ipLinkTemplate, mac, linkStateRegex)
}
