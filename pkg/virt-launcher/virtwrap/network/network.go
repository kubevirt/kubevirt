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

	networkdriver "kubevirt.io/kubevirt/pkg/network"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var podNICFactory = newpodNIC

type PodCacheInterface struct {
	Iface  *v1.Interface `json:"iface,omitempty"`
	PodIP  string        `json:"podIP,omitempty"`
	PodIPs []string      `json:"podIPs,omitempty"`
}

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

func getNetworksAndCniNetworks(vmi *v1.VirtualMachineInstance) (map[string]*v1.Network, map[string]int) {
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

func invokePodNICFactory(networks map[string]*v1.Network, ifaceName string) (podNIC, error) {
	network, ok := networks[ifaceName]
	if !ok {
		return nil, fmt.Errorf("failed to find a network %s", ifaceName)
	}
	return podNICFactory(network)
}

func getPodInterfaceName(networks map[string]*v1.Network, cniNetworks map[string]int, ifaceName string) string {
	if networks[ifaceName].Multus != nil && !networks[ifaceName].Multus.Default {
		// multus pod interfaces named netX
		return fmt.Sprintf("net%d", cniNetworks[ifaceName])
	} else {
		return networkdriver.PrimaryPodInterfaceName
	}
}

func SetupPodNetworkPhase1(vmi *v1.VirtualMachineInstance, pid int) error {
	err := networkdriver.CreateVirtHandlerCacheDir(vmi.ObjectMeta.UID)
	if err != nil {
		return err
	}
	networks, cniNetworks := getNetworksAndCniNetworks(vmi)
	for i, iface := range vmi.Spec.Domain.Devices.Interfaces {
		podnic, err := invokePodNICFactory(networks, iface.Name)
		if err != nil {
			return err
		}
		podInterfaceName := getPodInterfaceName(networks, cniNetworks, iface.Name)
		err = podNIC.PlugPhase1(podnic, vmi, &vmi.Spec.Domain.Devices.Interfaces[i], networks[iface.Name], podInterfaceName, pid)
		if err != nil {
			return err
		}
	}
	return nil
}

func SetupPodNetworkPhase2(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	networks, cniNetworks := getNetworksAndCniNetworks(vmi)
	for i, iface := range vmi.Spec.Domain.Devices.Interfaces {
		podnic, err := invokePodNICFactory(networks, iface.Name)
		if err != nil {
			return err
		}
		podInterfaceName := getPodInterfaceName(networks, cniNetworks, iface.Name)
		err = podNIC.PlugPhase2(podnic, vmi, &vmi.Spec.Domain.Devices.Interfaces[i], networks[iface.Name], domain, podInterfaceName)
		if err != nil {
			return err
		}
	}
	return nil
}

func newpodNIC(network *v1.Network) (podNIC, error) {
	if network.Pod != nil || network.Multus != nil {
		return new(podNICImpl), nil
	}
	return nil, fmt.Errorf("Network not implemented")
}
