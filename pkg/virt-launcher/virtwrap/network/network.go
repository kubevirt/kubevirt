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

type podNIC interface {
	PlugPhase2(vmi *v1.VirtualMachineInstance, iface *v1.Interface, network *v1.Network, domain *api.Domain, podInterfaceName string) error
}

func invokePodNICFactory(networks map[string]*v1.Network, ifaceName string) (podNIC, error) {
	network, ok := networks[ifaceName]
	if !ok {
		return nil, fmt.Errorf("failed to find a network %s", ifaceName)
	}
	return podNICFactory(network)
}

func SetupPodNetworkPhase2(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	networks, cniNetworks := networkdriver.GetNetworksAndCniNetworks(vmi)
	for i, iface := range vmi.Spec.Domain.Devices.Interfaces {
		podnic, err := invokePodNICFactory(networks, iface.Name)
		if err != nil {
			return err
		}
		network := networks[iface.Name]
		cniNetworkIndex := cniNetworks[iface.Name]
		podInterfaceName := networkdriver.GetPodInterfaceName(network, cniNetworkIndex)
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
