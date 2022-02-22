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
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const primaryPodInterfaceName = "eth0"

type VMNetworkConfigurator struct {
	vmi          *v1.VirtualMachineInstance
	handler      netdriver.NetworkHandler
	cacheCreator cacheCreator
}

func newVMNetworkConfiguratorWithHandlerAndCache(vmi *v1.VirtualMachineInstance, handler netdriver.NetworkHandler, cacheCreator cacheCreator) *VMNetworkConfigurator {
	return &VMNetworkConfigurator{
		vmi:          vmi,
		handler:      handler,
		cacheCreator: cacheCreator,
	}
}

func NewVMNetworkConfigurator(vmi *v1.VirtualMachineInstance, cacheCreator cacheCreator) *VMNetworkConfigurator {
	return newVMNetworkConfiguratorWithHandlerAndCache(vmi, &netdriver.NetworkUtilsHandler{}, cacheCreator)
}

func (v VMNetworkConfigurator) getPhase1NICs(launcherPID *int, ifaceName string) ([]podNIC, error) {
	nics := []podNIC{}

	for i := range v.vmi.Spec.Networks {
		nic, err := newPhase1PodNIC(v.vmi, &v.vmi.Spec.Networks[i], v.handler, v.cacheCreator, launcherPID)
		if err != nil {
			return nil, err
		}
		if len(ifaceName) > 0 { //get NIC by iface name
			if ifaceName == v.vmi.Spec.Networks[i].Name {
				nics = append(nics, *nic)
				return nics, nil
			}
		} else { //get all NICs
			nics = append(nics, *nic)
		}
	}

	return nics, nil
}

func (v VMNetworkConfigurator) getPhase2NICs(domain *api.Domain, ifaceName string) ([]podNIC, error) {
	nics := []podNIC{}

	for i := range v.vmi.Spec.Networks {
		nic, err := newPhase2PodNIC(v.vmi, &v.vmi.Spec.Networks[i], v.handler, v.cacheCreator, domain)
		if err != nil {
			return nil, err
		}
		if len(ifaceName) > 0 { //get NIC by iface name
			if ifaceName == v.vmi.Spec.Networks[i].Name {
				nics = append(nics, *nic)
				return nics, nil
			}
		} else { //get all NICs
			nics = append(nics, *nic)
		}
	}
	return nics, nil
}

func (n *VMNetworkConfigurator) SetupPodNetworkPhase1(launcherPID int) error {
	nics, err := n.getPhase1NICs(&launcherPID, "")
	if err != nil {
		return err
	}
	for _, nic := range nics {
		if err := nic.PlugPhase1(); err != nil {
			return fmt.Errorf("failed plugging phase1 at nic '%s': %w", nic.podInterfaceName, err)
		}
	}
	return nil
}

func (n *VMNetworkConfigurator) SetupPodNetworkPhase2(domain *api.Domain, stopChan chan string) error {
	nics, err := n.getPhase2NICs(domain, "")
	if err != nil {
		return err
	}
	for _, nic := range nics {
		if err := nic.PlugPhase2(domain, stopChan); err != nil {
			return fmt.Errorf("failed plugging phase2 at nic '%s': %w", nic.podInterfaceName, err)
		}
	}
	return nil
}

//UnplugPodNetworkPhase2
func (n *VMNetworkConfigurator) RemovePodAuxiliaryInfraForInterface(pid int, ifaceName string) error {
	launcherPID := &pid
	//we use setupPhase1NICs to create NIC instance with infraConfigurator inside
	nics, err := n.getPhase1NICs(launcherPID, ifaceName)
	if err != nil {
		return err
	} else if len(nics) != 1 {
		return fmt.Errorf("could not find a single interface named %s. Found %d", ifaceName, len(nics))
	}

	if err := nics[0].UnplugPhase2(); err != nil {
		return fmt.Errorf("failed unplugging phase2 at nic '%s': %w", nics[0].podInterfaceName, err)
	}

	return nil
}

//UnplugPodNetworkPhase1
func (n *VMNetworkConfigurator) StopDHCPServerForInterface(domain *api.Domain, ifaceName string, stopChan chan string) error {
	//we use setupPhase2NICs to create NIC instance with dhcpConfigurator inside
	nics, err := n.getPhase2NICs(domain, ifaceName)
	if err != nil {
		return err
	} else if len(nics) != 1 {
		return fmt.Errorf("could not find a single interface named %s. Found %d", ifaceName, len(nics))
	}

	if err := nics[0].UnplugPhase1(stopChan); err != nil {
		return fmt.Errorf("failed unplugging phase1 at nic '%s': %w", nics[0].podInterfaceName, err)
	}

	return nil
}
