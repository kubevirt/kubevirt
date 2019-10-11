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

var interfaceCacheFile = "/var/run/kubevirt/interface-cache-%s-%s.json"
var qemuArgCacheFile = "/var/run/kubevirt/qemu-arg-%s-%s.json"
var vifCacheFile = "/var/run/kubevirt/vif-cache-%s-%s.json"
var NetworkInterfaceFactory = getNetworkClass

var podInterfaceName = podInterface

type plugFunction func(vif NetworkInterface, vmi *v1.VirtualMachineInstance, iface *v1.Interface, network *v1.Network, domain *api.Domain, podInterfaceName string) error

type NetworkInterface interface {
	PlugPhase1(vmi *v1.VirtualMachineInstance, iface *v1.Interface, network *v1.Network, domain *api.Domain, podInterfaceName string) error
	PlugPhase2(vmi *v1.VirtualMachineInstance, iface *v1.Interface, network *v1.Network, domain *api.Domain, podInterfaceName string) error
	Unplug()
}

func SetupNetworkInterfacesPhase1(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	return _setupNetworkInterfaces(vmi, domain, NetworkInterface.PlugPhase1)
}

func SetupNetworkInterfacesPhase2(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	return _setupNetworkInterfaces(vmi, domain, NetworkInterface.PlugPhase2)
}

func _setupNetworkInterfaces(vmi *v1.VirtualMachineInstance, domain *api.Domain, plug plugFunction) error {
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

	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		network, ok := networks[iface.Name]
		if !ok {
			return fmt.Errorf("failed to find a network %s", iface.Name)
		}
		vif, err := NetworkInterfaceFactory(network)
		if err != nil {
			return err
		}

		if networks[iface.Name].Multus != nil && !networks[iface.Name].Multus.Default {
			// multus pod interfaces named netX
			podInterfaceName = fmt.Sprintf("net%d", cniNetworks[iface.Name])
		} else if networks[iface.Name].Genie != nil {
			// genie pod interfaces named ethX
			podInterfaceName = fmt.Sprintf("eth%d", cniNetworks[iface.Name])
		} else {
			podInterfaceName = podInterface
		}

		err = plug(vif, vmi, &iface, network, domain, podInterfaceName)
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
