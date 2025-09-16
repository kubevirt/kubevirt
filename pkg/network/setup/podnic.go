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
 * Copyright The KubeVirt Authors.
 *
 */

package network

import (
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"

	"kubevirt.io/kubevirt/pkg/network/cache"
	dhcpconfigurator "kubevirt.io/kubevirt/pkg/network/dhcp"
	"kubevirt.io/kubevirt/pkg/network/domainspec"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/link"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const defaultState = cache.PodIfaceNetworkPreparationPending

type podNIC struct {
	vmi              *v1.VirtualMachineInstance
	podInterfaceName string
	vmiSpecIface     *v1.Interface
	vmiSpecNetwork   *v1.Network
	handler          netdriver.NetworkHandler
	cacheCreator     cacheCreator
	dhcpConfigurator dhcpconfigurator.Configurator
	domainGenerator  domainspec.LibvirtSpecGenerator
}

func newPhase2PodNIC(vmi *v1.VirtualMachineInstance, network *v1.Network, iface *v1.Interface, handler netdriver.NetworkHandler, cacheCreator cacheCreator, domain *api.Domain, domainAttachment string) (*podNIC, error) {
	podnic := newPodNIC(vmi, network, iface, handler, cacheCreator)

	ifaceLink, err := link.DiscoverByNetwork(podnic.handler, podnic.vmi.Spec.Networks, *podnic.vmiSpecNetwork, vmi.Status.Interfaces)
	if err != nil {
		return nil, err
	}
	if ifaceLink == nil {
		podnic.podInterfaceName = ""
	} else {
		podnic.podInterfaceName = ifaceLink.Attrs().Name
	}

	podnic.dhcpConfigurator = podnic.newDHCPConfigurator()
	podnic.domainGenerator = podnic.newLibvirtSpecGenerator(domain, domainAttachment)

	return podnic, nil
}

func newPodNIC(vmi *v1.VirtualMachineInstance, network *v1.Network, iface *v1.Interface, handler netdriver.NetworkHandler, cacheCreator cacheCreator) *podNIC {
	return &podNIC{
		cacheCreator:   cacheCreator,
		handler:        handler,
		vmi:            vmi,
		vmiSpecNetwork: network,
		vmiSpecIface:   iface,
	}
}

func (l *podNIC) PlugPhase2(domain *api.Domain) error {
	precond.MustNotBeNil(domain)

	if err := l.domainGenerator.Generate(); err != nil {
		log.Log.Reason(err).Critical("failed to create libvirt configuration")
	}

	if l.dhcpConfigurator != nil {
		dhcpConfig, err := l.dhcpConfigurator.Generate()
		if err != nil {
			log.Log.Reason(err).Errorf("failed to get a dhcp configuration for: %s", l.podInterfaceName)
			return err
		}
		log.Log.V(4).Infof("The imported dhcpConfig: %s", dhcpConfig.String())
		if err := l.dhcpConfigurator.EnsureDHCPServerStarted(l.podInterfaceName, *dhcpConfig, l.vmiSpecIface.DHCPOptions); err != nil {
			log.Log.Reason(err).Criticalf("failed to ensure dhcp service running for: %s", l.podInterfaceName)
			panic(err)
		}
	}

	return nil
}

func (l *podNIC) newDHCPConfigurator() dhcpconfigurator.Configurator {
	var dhcpConfigurator dhcpconfigurator.Configurator
	if l.vmiSpecIface.Bridge != nil {
		dhcpConfigurator = dhcpconfigurator.NewBridgeConfigurator(
			l.cacheCreator,
			link.GenerateBridgeName(l.podInterfaceName),
			l.handler,
			l.podInterfaceName,
			l.vmi.Spec.Domain.Devices.Interfaces,
			l.vmiSpecIface,
			l.vmi.Spec.Subdomain)
	} else if l.vmiSpecIface.Masquerade != nil {
		dhcpConfigurator = dhcpconfigurator.NewMasqueradeConfigurator(
			link.GenerateBridgeName(l.podInterfaceName),
			l.handler,
			l.vmiSpecIface,
			l.vmiSpecNetwork,
			l.podInterfaceName,
			l.vmi.Spec.Subdomain)
	}
	return dhcpConfigurator
}

func (l *podNIC) newLibvirtSpecGenerator(domain *api.Domain, domainAttachment string) domainspec.LibvirtSpecGenerator {
	if domainAttachment == string(v1.Tap) {
		return domainspec.NewTapLibvirtSpecGenerator(l.vmiSpecIface, *l.vmiSpecNetwork, domain, l.podInterfaceName, l.handler)
	}
	return nil
}
