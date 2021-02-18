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
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network/cache"
)

const primaryPodInterfaceName = "eth0"

var podNICFactory = newpodNIC

// Network configuration is split into two parts, or phases, each executed in a
// different context.
// Phase1 is run by virt-handler and heavylifts most configuration steps. It
// also creates the tap device that will be passed by name to virt-launcher,
// thus allowing unprivileged libvirt to consume a pre-configured device.
// Phase2 is run by virt-launcher in the pod context and completes steps left
// out of virt-handler. The reason to have a separate phase for virt-launcher
// and not just have all the work done by virt-handler is because there is no
// ready solution for DHCP server startup in virt-handler context yet. This is
// a temporary limitation and the split is expected to go once the final gap is
// closed.
// Moving all configuration steps into virt-handler will also allow to
// downgrade privileges for virt-launcher, specifically, to remove NET_ADMIN
// capability. Future patches should address that. See:
// https://github.com/kubevirt/kubevirt/issues/3085
type podNIC interface {
	PlugPhase1(vmi *v1.VirtualMachineInstance, iface *v1.Interface, network *v1.Network, podInterfaceName string, pid int) error
	PlugPhase2(vmi *v1.VirtualMachineInstance, iface *v1.Interface, network *v1.Network, domain *api.Domain, podInterfaceName string) error
}

func SetupPodNetworkPhase1(vmi *v1.VirtualMachineInstance, pid int, cacheFactory cache.InterfaceCacheFactory) error {
	networks := mapNetworksByName(vmi.Spec.Networks)
	primaryNet := lookupPrimaryNetwork(vmi.Spec.Networks)
	secondaryNets := filterSecondaryMultusNetworks(vmi.Spec.Networks)
	podInterfaceNames := mapPodInterfaceNameByNetwork(primaryNet, secondaryNets)
	for i, iface := range vmi.Spec.Domain.Devices.Interfaces {
		network, ok := networks[iface.Name]
		if !ok {
			return fmt.Errorf("failed to find a network %s", iface.Name)
		}
		podnic := podNICFactory(cacheFactory)
		podInterfaceName := podInterfaceNames[network.Name]
		err := podNIC.PlugPhase1(podnic, vmi, &vmi.Spec.Domain.Devices.Interfaces[i], &network, podInterfaceName, pid)
		if err != nil {
			return err
		}
	}
	return nil
}

func SetupPodNetworkPhase2(vmi *v1.VirtualMachineInstance, domain *api.Domain, cacheFactory cache.InterfaceCacheFactory) error {
	networks := mapNetworksByName(vmi.Spec.Networks)
	primaryNet := lookupPrimaryNetwork(vmi.Spec.Networks)
	secondaryNets := filterSecondaryMultusNetworks(vmi.Spec.Networks)
	podInterfaceNames := mapPodInterfaceNameByNetwork(primaryNet, secondaryNets)
	for i, iface := range vmi.Spec.Domain.Devices.Interfaces {
		network, ok := networks[iface.Name]
		if !ok {
			return fmt.Errorf("failed to find a network %s", iface.Name)
		}
		podnic := podNICFactory(cacheFactory)
		podInterfaceName := podInterfaceNames[network.Name]
		err := podNIC.PlugPhase2(podnic, vmi, &vmi.Spec.Domain.Devices.Interfaces[i], &network, domain, podInterfaceName)
		if err != nil {
			return err
		}
	}
	return nil
}

func mapNetworksByName(nets []v1.Network) map[string]v1.Network {
	networks := map[string]v1.Network{}
	for _, net := range nets {
		networks[net.Name] = net
	}
	return networks
}

func mapPodInterfaceNameByNetwork(primaryMultusNetwork *v1.Network, secondaryMultusNetworks []v1.Network) map[string]string {
	m := map[string]string{}
	for i, net := range secondaryMultusNetworks {
		m[net.Name] = getSecondaryPodInterfaceName(i)
	}
	if primaryMultusNetwork != nil {
		m[primaryMultusNetwork.Name] = primaryPodInterfaceName
	}
	return m
}

func getSecondaryPodInterfaceName(index int) string {
	return fmt.Sprintf("net%d", index+1)
}

func lookupPrimaryNetwork(nets []v1.Network) *v1.Network {
	for _, net := range nets {
		if !isSecondaryMultusNetwork(net) {
			return &net
		}
	}
	return nil
}

func filterSecondaryMultusNetworks(nets []v1.Network) []v1.Network {
	var secondary []v1.Network
	for _, net := range nets {
		if isSecondaryMultusNetwork(net) {
			secondary = append(secondary, net)
		}
	}
	return secondary
}

func isSecondaryMultusNetwork(net v1.Network) bool {
	return net.Multus != nil && !net.Multus.Default
}

func newpodNIC(cacheFactory cache.InterfaceCacheFactory) podNIC {
	return &podNICImpl{cacheFactory: cacheFactory, handler: Handler}
}
