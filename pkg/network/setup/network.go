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
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type VMNetworkConfigurator struct {
	vmi          *v1.VirtualMachineInstance
	handler      netdriver.NetworkHandler
	cacheCreator cacheCreator
}

type vmNetConfiguratorOption func(v *VMNetworkConfigurator)

func NewVMNetworkConfigurator(vmi *v1.VirtualMachineInstance, cacheCreator cacheCreator, opts ...vmNetConfiguratorOption) *VMNetworkConfigurator {
	v := &VMNetworkConfigurator{
		vmi:          vmi,
		handler:      &netdriver.NetworkUtilsHandler{},
		cacheCreator: cacheCreator,
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

func (v VMNetworkConfigurator) getPhase2NICs(domain *api.Domain, networks []v1.Network) ([]podNIC, error) {
	var nics []podNIC

	for i := range networks {
		iface := vmispec.LookupInterfaceByName(v.vmi.Spec.Domain.Devices.Interfaces, networks[i].Name)
		if iface == nil {
			return nil, fmt.Errorf("no iface matching with network %s", networks[i].Name)
		}

		// Binding plugin, SR-IOV and Slirp devices are not part of the phases
		if iface.Binding != nil || iface.SRIOV != nil || iface.Slirp != nil {
			continue
		}

		nic, err := newPhase2PodNIC(v.vmi, &networks[i], iface, v.handler, v.cacheCreator, domain)
		if err != nil {
			return nil, err
		}
		nics = append(nics, *nic)
	}
	return nics, nil
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
