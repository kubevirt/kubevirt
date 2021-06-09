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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package network

import (
	"fmt"
	"os"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"
	"kubevirt.io/kubevirt/pkg/network/cache"
	dhcpconfigurator "kubevirt.io/kubevirt/pkg/network/dhcp"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
)

type podNIC struct {
	vmi              *v1.VirtualMachineInstance
	podInterfaceName string
	launcherPID      *int
	iface            *v1.Interface
	network          *v1.Network
	handler          netdriver.NetworkHandler
	cacheFactory     cache.InterfaceCacheFactory
	dhcpConfigurator *dhcpconfigurator.Configurator
}

func newPodNIC(vmi *v1.VirtualMachineInstance, network *v1.Network, handler netdriver.NetworkHandler, cacheFactory cache.InterfaceCacheFactory, launcherPID *int) (*podNIC, error) {
	if network.Pod == nil && network.Multus == nil {
		return nil, fmt.Errorf("Network not implemented")
	}

	correspondingNetworkIface := findInterfaceByNetworkName(vmi, network)
	if correspondingNetworkIface == nil {
		return nil, fmt.Errorf("no iface matching with network %s", network.Name)
	}

	podInterfaceName, err := composePodInterfaceName(vmi, network)
	if err != nil {
		return nil, err
	}

	var dhcpConfigurator *dhcpconfigurator.Configurator
	if correspondingNetworkIface.Bridge != nil {
		dhcpConfigurator = dhcpconfigurator.NewConfiguratorWithClientFilter(
			cacheFactory,
			getPIDString(launcherPID),
			generateInPodBridgeInterfaceName(podInterfaceName),
			handler)
	} else if correspondingNetworkIface.Masquerade != nil {
		dhcpConfigurator = dhcpconfigurator.NewConfigurator(
			cacheFactory,
			getPIDString(launcherPID),
			generateInPodBridgeInterfaceName(podInterfaceName),
			handler)
	}
	return &podNIC{
		cacheFactory:     cacheFactory,
		handler:          handler,
		vmi:              vmi,
		network:          network,
		podInterfaceName: podInterfaceName,
		iface:            correspondingNetworkIface,
		launcherPID:      launcherPID,
		dhcpConfigurator: dhcpConfigurator,
	}, nil
}

func composePodInterfaceName(vmi *v1.VirtualMachineInstance, network *v1.Network) (string, error) {
	if isSecondaryMultusNetwork(*network) {
		multusIndex := findMultusIndex(vmi, network)
		if multusIndex == -1 {
			return "", fmt.Errorf("Network name %s not found", network.Name)
		}
		return fmt.Sprintf("net%d", multusIndex), nil
	}
	return primaryPodInterfaceName, nil
}

func findInterfaceByNetworkName(vmi *v1.VirtualMachineInstance, network *v1.Network) *v1.Interface {
	for i, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.Name == network.Name {
			return &vmi.Spec.Domain.Devices.Interfaces[i]
		}
	}
	return nil
}

func findMultusIndex(vmi *v1.VirtualMachineInstance, networkToFind *v1.Network) int {
	idxMultus := 0
	for _, network := range vmi.Spec.Networks {
		if isSecondaryMultusNetwork(network) {
			// multus pod interfaces start from 1
			idxMultus++
			if network.Name == networkToFind.Name {
				return idxMultus
			}
		}
	}
	return -1
}

func isSecondaryMultusNetwork(net v1.Network) bool {
	return net.Multus != nil && !net.Multus.Default
}

func (l *podNIC) setPodInterfaceCache() error {
	ifCache := &cache.PodCacheInterface{Iface: l.iface}

	ipv4, ipv6, err := l.handler.ReadIPAddressesFromLink(l.podInterfaceName)
	if err != nil {
		return err
	}

	switch {
	case ipv4 != "" && ipv6 != "":
		ifCache.PodIPs, err = l.sortIPsBasedOnPrimaryIP(ipv4, ipv6)
		if err != nil {
			return err
		}
	case ipv4 != "":
		ifCache.PodIPs = []string{ipv4}
	case ipv6 != "":
		ifCache.PodIPs = []string{ipv6}
	default:
		return nil
	}

	ifCache.PodIP = ifCache.PodIPs[0]
	err = l.cacheFactory.CacheForVMI(l.vmi).Write(l.iface.Name, ifCache)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to write pod Interface to ifCache, %s", err.Error())
		return err
	}

	return nil
}

// sortIPsBasedOnPrimaryIP returns a sorted slice of IP/s based on the detected cluster primary IP.
// The operation clones the Pod status IP list order logic.
func (l *podNIC) sortIPsBasedOnPrimaryIP(ipv4, ipv6 string) ([]string, error) {
	ipv4Primary, err := l.handler.IsIpv4Primary()
	if err != nil {
		return nil, err
	}

	if ipv4Primary {
		return []string{ipv4, ipv6}, nil
	}

	return []string{ipv6, ipv4}, nil
}

func (l *podNIC) PlugPhase1() error {

	// There is nothing to plug for SR-IOV devices
	if l.iface.SRIOV != nil {
		return nil
	}

	cachedDomainIface, err := l.cachedDomainInterface()
	if err != nil {
		return err
	}

	doesExist := cachedDomainIface != nil
	// ignore the bindMechanism.cachedDomainInterface for slirp and set the Pod interface cache
	if !doesExist || l.iface.Slirp != nil {
		err := l.setPodInterfaceCache()
		if err != nil {
			return err
		}
	}
	if !doesExist {
		bindMechanism, err := l.getPhase1Binding()
		if err != nil {
			return err
		}

		if err := bindMechanism.discoverPodNetworkInterface(l.podInterfaceName); err != nil {
			return err
		}

		if l.dhcpConfigurator != nil {
			dhcpConfig := bindMechanism.generateDhcpConfig()
			if err := l.dhcpConfigurator.ExportConfiguration(*dhcpConfig); err != nil {
				log.Log.Reason(err).Error("failed to save dhcpConfig configuration")
				return errors.CreateCriticalNetworkError(err)
			}
		}

		domainIface := bindMechanism.generateDomainIfaceSpec()
		// preparePodNetworkInterface must be called *after* the generate
		// methods since it mutates the pod interface from which those
		// generator methods get their info from.
		if err := bindMechanism.preparePodNetworkInterface(); err != nil {
			log.Log.Reason(err).Error("failed to prepare pod networking")
			return errors.CreateCriticalNetworkError(err)
		}

		// caching the domain interface *must* be the last thing done in phase
		// 1, since retrieving it is the criteria to configure the pod
		// networking infrastructure.
		if err := l.storeCachedDomainIface(domainIface); err != nil {
			log.Log.Reason(err).Error("failed to save interface configuration")
			return errors.CreateCriticalNetworkError(err)
		}

	}

	return nil
}

func (l *podNIC) PlugPhase2(domain *api.Domain) error {
	precond.MustNotBeNil(domain)

	// There is nothing to plug for SR-IOV devices
	if l.iface.SRIOV != nil {
		return nil
	}

	domainIface, err := l.cachedDomainInterface()
	if err != nil {
		log.Log.Reason(err).Critical("failed to load cached interface configuration")
	}
	if domainIface == nil {
		log.Log.Reason(err).Critical("cached interface configuration doesn't exist")
	}

	bindMechanism, err := l.getPhase2Binding(domain)
	if err != nil {
		return err
	}

	if err := bindMechanism.decorateConfig(*domainIface); err != nil {
		log.Log.Reason(err).Critical("failed to create libvirt configuration")
	}

	if l.dhcpConfigurator != nil {
		dhcpConfig, err := l.dhcpConfigurator.ImportConfiguration(l.podInterfaceName)
		if err != nil || dhcpConfig == nil {
			log.Log.Reason(err).Critical("failed to load cached dhcpConfig configuration")
		}
		if err := l.dhcpConfigurator.EnsureDhcpServerStarted(l.podInterfaceName, *dhcpConfig, l.iface.DHCPOptions); err != nil {
			log.Log.Reason(err).Criticalf("failed to ensure dhcp service running for: %s", l.podInterfaceName)
			panic(err)
		}
	}

	return nil
}

func (l *podNIC) getPhase1Binding() (BindMechanism, error) {
	return l.getPhase2Binding(nil)
}

func (l *podNIC) getPhase2Binding(domain *api.Domain) (BindMechanism, error) {
	if l.iface.Bridge != nil {
		return newBridgeBinding(l.vmi, l.iface, domain, l.podInterfaceName, l.cacheFactory, l.launcherPID, l.handler)
	}
	if l.iface.Masquerade != nil {
		return newMasqueradeBinding(l.vmi, l.iface, domain, l.network.Pod.VMNetworkCIDR, l.network.Pod.VMIPv6NetworkCIDR, l.podInterfaceName, l.cacheFactory, l.launcherPID, l.handler)
	}
	if l.iface.Slirp != nil {
		return &SlirpBindMechanism{iface: l.iface, domain: domain}, nil
	}
	if l.iface.Macvtap != nil {
		return newMacvtapBinding(l.vmi, l.iface, domain, l.cacheFactory, l.launcherPID, l.handler)
	}
	return nil, fmt.Errorf("Not implemented")
}

func getPIDString(pid *int) string {
	if pid != nil {
		return fmt.Sprintf("%d", *pid)
	}
	return "self"
}

func (l *podNIC) cachedDomainInterface() (*api.Interface, error) {
	ifaceConfig, err := l.cacheFactory.CacheDomainInterfaceForPID(getPIDString(l.launcherPID)).Read(l.iface.Name)

	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return ifaceConfig, nil
}

func (l *podNIC) storeCachedDomainIface(domainIface api.Interface) error {
	return l.cacheFactory.CacheDomainInterfaceForPID(getPIDString(l.launcherPID)).Write(l.iface.Name, &domainIface)
}

func calculateNetworkQueues(vmi *v1.VirtualMachineInstance) uint32 {
	if isMultiqueue(vmi) {
		return converter.CalculateNetworkQueues(vmi)
	}
	return 0
}

func isMultiqueue(vmi *v1.VirtualMachineInstance) bool {
	return (vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue != nil) &&
		(*vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue)
}
