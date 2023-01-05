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
	"context"
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

func (v VMNetworkConfigurator) getPhase1NICs(launcherPID *int) ([]podNIC, error) {
	return v.getPhase1NicsWithGenerator(launcherPID, standardPodNicGenerator{})
}

func (v VMNetworkConfigurator) getPhase1NicsWithGenerator(launcherPID *int, podNicGenerator podNicGenerator) ([]podNIC, error) {
	var nics []podNIC

	if len(v.vmi.Spec.Domain.Devices.Interfaces) == 0 {
		return nics, nil
	}

	relevantNetworks := podNicGenerator.relevantNetworks(v.vmi)
	for i := range relevantNetworks {
		nic, err := podNicGenerator.generate(v.vmi, &relevantNetworks[i], v.handler, v.cacheCreator, launcherPID)
		if err != nil {
			return nil, err
		}
		nics = append(nics, *nic)
	}
	return nics, nil
}

func (v VMNetworkConfigurator) getPhase2NICs(domain *api.Domain) ([]podNIC, error) {
	return v.getPhase2NICsWithGenerator(domain, standardPodNicGenerator{})
}

func (v VMNetworkConfigurator) getPhase2NICsWithGenerator(domain *api.Domain, podNicGenerator podNicGenerator) ([]podNIC, error) {
	var nics []podNIC

	relevantNetworks := podNicGenerator.relevantNetworks(v.vmi)
	for i := range relevantNetworks {
		nic, err := newPhase2PodNIC(v.vmi, &relevantNetworks[i], v.handler, v.cacheCreator, domain)
		if err != nil {
			return nil, err
		}
		nics = append(nics, *nic)
	}
	return nics, nil
}

func (n *VMNetworkConfigurator) SetupPodNetworkPhase1(launcherPID int) error {
	nics, err := n.getPhase1NICs(&launcherPID)
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

func (n *VMNetworkConfigurator) CreatePodAuxiliaryInfra(pid int, ifaceName string) error {
	launcherPID := &pid
	nics, err := n.getPhase1NicsWithGenerator(launcherPID, newHotplugPodNicGenerator(ifaceName))
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

func (n *VMNetworkConfigurator) SetupPodNetworkPhase2(domain *api.Domain, ifacesCtxMap map[string]context.Context) error {
	nics, err := n.getPhase2NICs(domain)
	if err != nil {
		return err
	}
	for _, nic := range nics {
		ctx := ifacesCtxMap[nic.vmiSpecIface.Name]
		if err := nic.PlugPhase2(ctx, domain); err != nil {
			return fmt.Errorf("failed plugging phase2 at nic '%s': %w", nic.podInterfaceName, err)
		}
	}
	return nil
}

func (n *VMNetworkConfigurator) StartDHCP(ctx context.Context, domain *api.Domain, ifaceName string) error {
	nics, err := n.getPhase2NICsWithGenerator(domain, newHotplugPodNicGenerator(ifaceName))
	if err != nil {
		return err
	}
	for _, nic := range nics {
		if err := nic.StartDHCP(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (n *VMNetworkConfigurator) StopDHCP(ctx context.Context, cancel context.CancelFunc, domain *api.Domain, ifaceName string) error {
	nics, err := n.getPhase2NICsWithGenerator(domain, newHotUnplugPodNicGenerator(ifaceName))
	if err != nil {
		return err
	}
	for _, nic := range nics {
		if err := nic.StopDHCP(ctx, cancel); err != nil {
			return err
		}
	}
	return nil
}

func isIfaceAlreadyAvailableInVM(ifacesToHotplug map[string]struct{}, networkName string) bool {
	_, found := ifacesToHotplug[networkName]
	return len(ifacesToHotplug) > 0 && !found
}

func indexedInterfacesToHotplug(ifaceNames []string) map[string]struct{} {
	indexedIfacesToHotplug := map[string]struct{}{}
	for _, ifaceName := range ifaceNames {
		indexedIfacesToHotplug[ifaceName] = struct{}{}
	}
	return indexedIfacesToHotplug
}
