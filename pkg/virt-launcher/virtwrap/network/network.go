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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package network

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"fmt"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const podInterface = "eth0"

var interfaceCacheFile = "/var/run/kubevirt-private/interface-cache-%s.json"
var qemuArgCacheFile = "/var/run/kubevirt-private/qemu-arg-%s.json"
var NetworkInterfaceFactory = getNetworkClass

var podInterfaceName = podInterface

type NetworkInterface interface {
	PlugInitial(vmi *v1.VirtualMachineInstance, iface *v1.Interface, network *v1.Network, pid int, podInterfaceName string) error
	Plug(vmi *v1.VirtualMachineInstance, iface *v1.Interface, network *v1.Network, domain *api.Domain, podInterfaceName string) error
	Unplug()
}

func getNetworksAndCniNetworks(vmi *v1.VirtualMachineInstance) (map[string]*v1.Network, map[string]int) {
	// prepare networks map
	networks := map[string]*v1.Network{}
	cniNetworks := map[string]int{}
	for _, network := range vmi.Spec.Networks {
		networks[network.Name] = network.DeepCopy()
		if networks[network.Name].Multus != nil && !networks[network.Name].Multus.Default {
			// multus pod interfaces start from 1
			cniNetworks[network.Name] = len(cniNetworks) + 1
		} else if networks[network.Name].Genie != nil {
			// genie pod interfaces start from 0
			cniNetworks[network.Name] = len(cniNetworks)
		}
	}
	return networks, cniNetworks
}

func getNetworkInterfaceFactory(networks map[string]*v1.Network, ifaceName string) (NetworkInterface, *v1.Network, error) {
	network, ok := networks[ifaceName]
	if !ok {
		return nil, nil, fmt.Errorf("failed to find a network %s", ifaceName)
	}
	vif, err := NetworkInterfaceFactory(network)
	if err != nil {
		return nil, network, err
	}
	return vif, network, nil
}

func getPodInterfaceName(networks map[string]*v1.Network, cniNetworks map[string]int, ifaceName string) string {
	if networks[ifaceName].Multus != nil && !networks[ifaceName].Multus.Default {
		// multus pod interfaces named netX
		return fmt.Sprintf("net%d", cniNetworks[ifaceName])
	} else if networks[ifaceName].Genie != nil {
		// genie pod interfaces named ethX
		return fmt.Sprintf("eth%d", cniNetworks[ifaceName])
	} else {
		return podInterface
	}
}

func SetupNetworkInterfaces(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	networks, cniNetworks := getNetworksAndCniNetworks(vmi)
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		networkInterfaceFactory, network, err := getNetworkInterfaceFactory(networks, iface.Name)
		if err != nil {
			return err
		}

		podInterfaceName = getPodInterfaceName(networks, cniNetworks, iface.Name)
		err = networkInterfaceFactory.Plug(vmi, &iface, network, domain, podInterfaceName)
		if err != nil {
			return err
		}
	}
	return nil
}

// This method will be called from virt-handler, and will perform actions that require permissions that virt-launcher is lacking.
func SetInitialNetworkConfig(vmi *v1.VirtualMachineInstance, pid int) error {
	networks, cniNetworks := getNetworksAndCniNetworks(vmi)
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		networkInterfaceFactory, network, err := getNetworkInterfaceFactory(networks, iface.Name)
		if err != nil {
			return err
		}

		podInterfaceName = getPodInterfaceName(networks, cniNetworks, iface.Name)
		err = networkInterfaceFactory.PlugInitial(vmi, &iface, network, pid, podInterfaceName)
		if err != nil {
			return err
		}
	}

	return nil
}

// a factory to get suitable network interface
func getNetworkClass(network *v1.Network) (NetworkInterface, error) {
	if network.Pod != nil || network.Multus != nil || network.Genie != nil {
		return new(PodInterface), nil
	}
	return nil, fmt.Errorf("Network not implemented")
}
