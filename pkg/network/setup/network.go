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

func (v VMNetworkConfigurator) getPhase1NICs(launcherPID *int, networks []v1.Network) ([]podNIC, error) {
	var nics []podNIC

	for i := range networks {
		nic, err := newPhase1PodNIC(v.vmi, &networks[i], v.handler, v.cacheCreator, launcherPID)
		if err != nil {
			return nil, err
		}
		// SR-IOV devices are not part of the phases.
		if nic.vmiSpecIface.SRIOV != nil {
			continue
		}
		nics = append(nics, *nic)
	}
	return nics, nil
}

func (v VMNetworkConfigurator) getPhase2NICs(domain *api.Domain, networks []v1.Network) ([]podNIC, error) {
	var nics []podNIC

	for i := range networks {
		nic, err := newPhase2PodNIC(v.vmi, &networks[i], v.handler, v.cacheCreator, domain)
		if err != nil {
			return nil, err
		}
		// SR-IOV devices are not part of the phases.
		if nic.vmiSpecIface.SRIOV != nil {
			continue
		}
		nics = append(nics, *nic)
	}
	return nics, nil
}

func (n *VMNetworkConfigurator) SetupPodNetworkPhase1(launcherPID int, networks []v1.Network, configState ConfigState) error {
	nics, err := n.getPhase1NICs(&launcherPID, networks)
	if err != nil {
		return err
	}

	err = configState.Run(
		nics,
		func(nic *podNIC) error {
			return nic.discoverAndStoreCache()
		},
		func(nic *podNIC) error {
			if nic.infraConfigurator == nil {
				return nil
			}
			return nic.infraConfigurator.PreparePodNetworkInterface()
		},
	)
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
