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

var interfaceCacheFile = "/var/run/kubevirt-network/%s/interface-cache-%s.json"
var qemuArgCacheFile = "/var/run/kubevirt-network/%s/qemu-arg-%s.json"
var vifCacheFile = "/var/run/kubevirt-network/%s/vif-cache-%s.json"
var NetworkInterfaceFactory = getNetworkClass

var podInterfaceName = podInterface

type plugFunction func(vif NetworkInterface, vmi *v1.VirtualMachineInstance, iface *v1.Interface, network *v1.Network, domain *api.Domain, podInterfaceName string) error

// Network configuration is split into two parts, or phases, each executed in a
// different context. Phase1 is run by virt-handler and heavylifts most
// configuration steps. Phase2 is run by virt-launcher in the pod context and
// completes steps left out of virt-handler. The reason to have a separate phase
// for virt-launcher and not just have all the work done by virt-handler is
// because there is no ready solution for DHCP server startup in virt-handler
// context yet. This is a temporary limitation and the split is expected to go
// once the final gap is closed. Moving all configuration steps into virt-handler
// will also allow to downgrade privileges for virt-launcher, specifically, to
// remove NET_ADMIN capability. Future patches should address that.
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
