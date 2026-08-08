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

package launcher

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"

	"kubevirt.io/kubevirt/pkg/network/cache"
	dhcpconfigurator "kubevirt.io/kubevirt/pkg/network/dhcp"
	"kubevirt.io/kubevirt/pkg/network/domainspec"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/link"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type cacheCreator interface {
	New(filePath string) *cache.Cache
}

type DHCPConfiguratorFactory func(iface *v1.Interface, network *v1.Network, podInterfaceName string) dhcpconfigurator.Configurator

type VMNetworkConfigurator struct {
	vmi               *v1.VirtualMachineInstance
	handler           netdriver.NetworkHandler
	cacheCreator      cacheCreator
	domainAttachments map[string]string

	dhcpConfiguratorFactory DHCPConfiguratorFactory
}

type vmNetConfiguratorOption func(v *VMNetworkConfigurator)

func NewVMNetworkConfigurator(
	vmi *v1.VirtualMachineInstance,
	cacheCreator cacheCreator,
	opts ...vmNetConfiguratorOption,
) *VMNetworkConfigurator {
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

func WithNetworkHandler(handler netdriver.NetworkHandler) vmNetConfiguratorOption {
	return func(v *VMNetworkConfigurator) {
		v.handler = handler
	}
}

func WithDHCPConfiguratorFactory(f DHCPConfiguratorFactory) vmNetConfiguratorOption {
	return func(v *VMNetworkConfigurator) {
		v.dhcpConfiguratorFactory = f
	}
}

func (n *VMNetworkConfigurator) SetupPodNetworkPhase2(domain *api.Domain, networks []v1.Network) error {
	precond.MustNotBeNil(domain)

	for i := range networks {
		iface := vmispec.LookupInterfaceByName(n.vmi.Spec.Domain.Devices.Interfaces, networks[i].Name)
		if iface == nil {
			return fmt.Errorf("no iface matching with network %s", networks[i].Name)
		}

		if n.domainAttachments[iface.Name] != string(v1.Tap) {
			continue
		}

		podIfaceName, err := n.discoverPodInterfaceName(&networks[i])
		if err != nil {
			return err
		}

		n.enrichTapDomainInterface(domain, iface, &networks[i], podIfaceName)

		if err := n.ensureDHCP(iface, &networks[i], podIfaceName); err != nil {
			return err
		}
	}
	return nil
}

func (n *VMNetworkConfigurator) discoverPodInterfaceName(network *v1.Network) (string, error) {
	ifaceLink, err := link.DiscoverByNetwork(n.handler, n.vmi.Spec.Networks, *network, n.vmi.Status.Interfaces)
	if err != nil {
		return "", err
	}
	if ifaceLink == nil {
		return "", nil
	}
	return ifaceLink.Attrs().Name, nil
}

func (n *VMNetworkConfigurator) enrichTapDomainInterface(
	domain *api.Domain,
	iface *v1.Interface,
	network *v1.Network,
	podIfaceName string,
) {
	generator := domainspec.NewTapLibvirtSpecGenerator(iface, *network, domain, podIfaceName, n.handler)
	if err := generator.Generate(); err != nil {
		log.Log.Reason(err).Critical("failed to create libvirt configuration")
	}
}

func (n *VMNetworkConfigurator) ensureDHCP(iface *v1.Interface, network *v1.Network, podIfaceName string) error {
	var configurator dhcpconfigurator.Configurator
	if n.dhcpConfiguratorFactory != nil {
		configurator = n.dhcpConfiguratorFactory(iface, network, podIfaceName)
	} else {
		configurator = n.newDHCPConfigurator(iface, network, podIfaceName)
	}

	if configurator == nil {
		return nil
	}

	dhcpConfig, err := configurator.Generate()
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a dhcp configuration for: %s", podIfaceName)
		return err
	}
	log.Log.V(4).Infof("The imported dhcpConfig: %s", dhcpConfig.String())
	if err := configurator.EnsureDHCPServerStarted(podIfaceName, *dhcpConfig, iface.DHCPOptions); err != nil {
		log.Log.Reason(err).Criticalf("failed to ensure dhcp service running for: %s", podIfaceName)
		panic(err)
	}

	return nil
}

func (n *VMNetworkConfigurator) newDHCPConfigurator(
	iface *v1.Interface,
	network *v1.Network,
	podIfaceName string,
) dhcpconfigurator.Configurator {
	if iface.Bridge != nil {
		return dhcpconfigurator.NewBridgeConfigurator(
			n.cacheCreator,
			link.GenerateBridgeName(podIfaceName),
			n.handler,
			podIfaceName,
			n.vmi.Spec.Domain.Devices.Interfaces,
			iface,
			n.vmi.Spec.Subdomain)
	}
	if iface.Masquerade != nil {
		return dhcpconfigurator.NewMasqueradeConfigurator(
			link.GenerateBridgeName(podIfaceName),
			n.handler,
			iface,
			network,
			podIfaceName,
			n.vmi.Spec.Subdomain)
	}
	return nil
}
