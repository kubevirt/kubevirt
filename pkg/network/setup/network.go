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
	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const primaryPodInterfaceName = "eth0"

type VMNetworkConfigurator struct {
	vmi          *v1.VirtualMachineInstance
	handler      netdriver.NetworkHandler
	cacheFactory cache.InterfaceCacheFactory
}

func newVMNetworkConfiguratorWithHandlerAndCache(vmi *v1.VirtualMachineInstance, handler netdriver.NetworkHandler, cacheFactory cache.InterfaceCacheFactory) *VMNetworkConfigurator {
	return &VMNetworkConfigurator{
		vmi:          vmi,
		handler:      handler,
		cacheFactory: cacheFactory,
	}
}

func NewVMNetworkConfigurator(vmi *v1.VirtualMachineInstance, cacheFactory cache.InterfaceCacheFactory) *VMNetworkConfigurator {
	return newVMNetworkConfiguratorWithHandlerAndCache(vmi, &netdriver.NetworkUtilsHandler{}, cacheFactory)
}

func (v VMNetworkConfigurator) getPhase1NICs(launcherPID *int) ([]podNIC, error) {
	nics := []podNIC{}

	if len(v.vmi.Spec.Domain.Devices.Interfaces) == 0 {
		return nics, nil
	}

	for i, _ := range v.vmi.Spec.Networks {
		nic, err := newPhase1PodNIC(v.vmi, &v.vmi.Spec.Networks[i], v.handler, v.cacheFactory, launcherPID)
		if err != nil {
			return nil, err
		}
		nics = append(nics, *nic)
	}
	return nics, nil

}

func (v VMNetworkConfigurator) getPhase2NICs(domain *api.Domain) ([]podNIC, error) {
	nics := []podNIC{}

	if len(v.vmi.Spec.Domain.Devices.Interfaces) == 0 {
		return nics, nil
	}

	for i, _ := range v.vmi.Spec.Networks {
		nic, err := newPhase2PodNIC(v.vmi, &v.vmi.Spec.Networks[i], v.handler, v.cacheFactory, domain)
		if err != nil {
			return nil, err
		}
		nics = append(nics, *nic)
	}
	return nics, nil

}

func (n *VMNetworkConfigurator) SetupPodNetworkPhase1(pid int) error {
	launcherPID := &pid
	nics, err := n.getPhase1NICs(launcherPID)
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

func (n *VMNetworkConfigurator) SetupPodNetworkPhase2(domain *api.Domain) error {
	nics, err := n.getPhase2NICs(domain)
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
