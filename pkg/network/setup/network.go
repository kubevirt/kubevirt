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

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/link"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type VMNetworkConfigurator struct {
	vmi          *v1.VirtualMachineInstance
	handler      netdriver.NetworkHandler
	cacheCreator cacheCreator
	launcherPid  *int
}

func newVMNetworkConfiguratorWithHandlerAndCache(vmi *v1.VirtualMachineInstance, handler netdriver.NetworkHandler, cacheCreator cacheCreator, launcherPid *int) *VMNetworkConfigurator {
	return &VMNetworkConfigurator{
		vmi:          vmi,
		handler:      handler,
		cacheCreator: cacheCreator,
		launcherPid:  launcherPid,
	}
}

func NewVMNetworkConfigurator(vmi *v1.VirtualMachineInstance, cacheCreator cacheCreator, launcherPid *int) *VMNetworkConfigurator {
	return newVMNetworkConfiguratorWithHandlerAndCache(vmi, &netdriver.NetworkUtilsHandler{}, cacheCreator, launcherPid)
}

func (v VMNetworkConfigurator) getPhase1NICs(launcherPID *int, networks []v1.Network) ([]podNIC, error) {
	var nics []podNIC

	for i := range networks {
		// SR-IOV devices are not part of the phases.
		if iface := vmispec.LookupInterfaceByName(v.vmi.Spec.Domain.Devices.Interfaces, networks[i].Name); iface.SRIOV != nil {
			continue
		}

		nic, err := newPhase1PodNIC(v.vmi, &networks[i], v.handler, v.cacheCreator, launcherPID)
		if err != nil {
			return nil, err
		}
		nics = append(nics, *nic)
	}
	return nics, nil
}

func (v VMNetworkConfigurator) getPhase2NICs(domain *api.Domain, networks []v1.Network) ([]podNIC, error) {
	var nics []podNIC

	for i := range networks {
		// SR-IOV devices are not part of the phases.
		if iface := vmispec.LookupInterfaceByName(v.vmi.Spec.Domain.Devices.Interfaces, networks[i].Name); iface.SRIOV != nil {
			continue
		}

		nic, err := newPhase2PodNIC(v.vmi, &networks[i], v.handler, v.cacheCreator, domain)
		if err != nil {
			return nil, err
		}
		nics = append(nics, *nic)
	}
	return nics, nil
}

func (n *VMNetworkConfigurator) SetupPodNetworkPhase1(launcherPID int, networks []v1.Network, configState ConfigStateExecutor) error {
	nics, err := n.getPhase1NICs(&launcherPID, networks)
	if err != nil {
		return err
	}

	err = configState.Run(
		nics,
		preConfigStateRun,
		func(nic *podNIC) error {
			return nic.discoverAndStoreCache()
		},
		func(nic *podNIC) error {
			if nic.infraConfigurator == nil {
				return nil
			}
			return nic.infraConfigurator.PreparePodNetworkInterface()
		})
	if err != nil {
		return fmt.Errorf("failed setup pod network phase1: %w", err)
	}
	return nil
}

func (n *VMNetworkConfigurator) SetupPodNetworkPhase2(domain *api.Domain, networks []v1.Network) error {
	nics, err := n.getPhase2NICs(domain, networks)
	if err != nil {
		return err
	}
	for _, nic := range nics {
		if err := nic.PlugPhase2(domain); err != nil {
			return fmt.Errorf("failed plugging phase2 at nic '%s': %w", nic.podInterfaceName, err)
		}
	}
	return nil
}

func preConfigStateRun(nics []podNIC) ([]podNIC, error) {
	nics, err := discoverPodInterfaces(nics)
	if err != nil {
		return nil, err
	}

	return filterOutAbsentIfaces(nics), nil
}

func discoverPodInterfaces(nics []podNIC) ([]podNIC, error) {
	for idx, nic := range nics {
		podIfaceName, err := discoverPodInterfaceName(nic.handler, nic.vmi.Spec.Networks, *nic.vmiSpecNetwork)
		if err != nil {
			return nil, err
		}
		nics[idx].podInterfaceName = podIfaceName
	}
	return nics, nil
}

func discoverPodInterfaceName(handler netdriver.NetworkHandler, networks []v1.Network, subjectNetwork v1.Network) (string, error) {
	ifaceLink, err := link.DiscoverByNetwork(handler, networks, subjectNetwork)
	if err != nil {
		return "", err
	} else {
		if ifaceLink == nil {
			// couldn't find any interface
			return "", nil
		}
		return ifaceLink.Attrs().Name, nil
	}
}

func filterOutAbsentIfaces(nics []podNIC) []podNIC {
	var filteredNics []podNIC
	for _, nic := range nics {
		toAdd := nic.vmiSpecIface.State != v1.InterfaceStateAbsent ||
			(nic.vmiSpecIface.State == v1.InterfaceStateAbsent &&
				namescheme.OrdinalSecondaryInterfaceName(nic.podInterfaceName))
		if toAdd {
			filteredNics = append(filteredNics, nic)
		}
	}
	return filteredNics
}

func (n *VMNetworkConfigurator) UnplugPodNetworksPhase1(vmi *v1.VirtualMachineInstance, networks []v1.Network, configState ConfigStateExecutor) error {
	networkByName := vmispec.IndexNetworkSpecByName(networks)
	err := configState.Unplug(
		networks,
		func(netsToFilter []v1.Network) ([]string, error) {
			return n.filterOutOrdinalInterfaces(netsToFilter, vmi)
		},
		func(network string) error {
			unpluggedPodNic := NewUnpluggedpodnic(string(vmi.UID), networkByName[network], n.handler, *n.launcherPid, n.cacheCreator)
			return unpluggedPodNic.UnplugPhase1()
		})
	if err != nil {
		return fmt.Errorf("failed unplug pod networks phase1: %w", err)
	}
	return nil
}

func (n *VMNetworkConfigurator) filterOutOrdinalInterfaces(networks []v1.Network, vmi *v1.VirtualMachineInstance) ([]string, error) {
	networksToUnplug := []string{}
	for _, net := range networks {
		podIfaceName, err := discoverPodInterfaceName(n.handler, vmi.Spec.Networks, net)
		if err != nil {
			return nil, err
		}
		// podIfaceName can be empty in case the interface was already unplugged from the pod or not yet plugged
		if podIfaceName == "" || !namescheme.OrdinalSecondaryInterfaceName(podIfaceName) {
			networksToUnplug = append(networksToUnplug, net.Name)
		}
	}
	return networksToUnplug, nil
}
