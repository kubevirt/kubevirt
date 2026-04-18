/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
	vmi               *v1.VirtualMachineInstance
	handler           netdriver.NetworkHandler
	cacheCreator      cacheCreator
	domainAttachments map[string]string
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

func WithDomainAttachments(domainAttachments map[string]string) vmNetConfiguratorOption {
	return func(v *VMNetworkConfigurator) {
		v.domainAttachments = domainAttachments
	}
}

func (v VMNetworkConfigurator) getPhase2NICs(domain *api.Domain, networks []v1.Network) ([]podNIC, error) {
	var nics []podNIC

	for i := range networks {
		iface := vmispec.LookupInterfaceByName(v.vmi.Spec.Domain.Devices.Interfaces, networks[i].Name)
		if iface == nil {
			return nil, fmt.Errorf("no iface matching with network %s", networks[i].Name)
		}

		// Passt, Binding plugin (with non tap domain attachment) and SR-IOV devices are not part of the phases
		if (iface.PasstBinding != nil || iface.Binding != nil && v.domainAttachments[iface.Name] != string(v1.Tap)) || iface.SRIOV != nil {
			continue
		}

		nic, err := newPhase2PodNIC(v.vmi, &networks[i], iface, v.handler, v.cacheCreator, domain, v.domainAttachments[iface.Name])
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
