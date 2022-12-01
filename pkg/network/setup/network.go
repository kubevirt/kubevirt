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

func (v VMNetworkConfigurator) getPhase1NICs(launcherPID *int, networksToPlug ...v1.Network) ([]podNIC, error) {
	var nics []podNIC

	if len(networksToPlug) == 0 {
		networksToPlug = v.vmi.Spec.Networks
	}
	for i := range networksToPlug {
		nic, err := newPhase1PodNIC(v.vmi, &networksToPlug[i], v.handler, v.cacheCreator, launcherPID)
		if err != nil {
			return nil, err
		}
		nics = append(nics, *nic)
	}
	return nics, nil
}

func (v VMNetworkConfigurator) getPhase2NICs(domain *api.Domain, networksToPlug ...v1.Network) ([]podNIC, error) {
	var nics []podNIC

	if len(networksToPlug) == 0 {
		networksToPlug = v.vmi.Spec.Networks
	}
	for i := range networksToPlug {
		nic, err := newPhase2PodNIC(v.vmi, &networksToPlug[i], v.handler, v.cacheCreator, domain)
		if err != nil {
			return nil, err
		}
		nics = append(nics, *nic)
	}
	return nics, nil
}

func (n *VMNetworkConfigurator) SetupPodNetworkPhase1(launcherPID int, networksToPlug ...v1.Network) error {
	nics, err := n.getPhase1NICs(&launcherPID, networksToPlug...)
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

func (n *VMNetworkConfigurator) SetupPodNetworkPhase2(domain *api.Domain, networksToPlug ...v1.Network) error {
	nics, err := n.getPhase2NICs(domain, networksToPlug...)
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
