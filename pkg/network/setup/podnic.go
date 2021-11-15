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

	"kubevirt.io/kubevirt/pkg/network/domainspec"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"
	"kubevirt.io/kubevirt/pkg/network/cache"
	dhcpconfigurator "kubevirt.io/kubevirt/pkg/network/dhcp"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/errors"
	"kubevirt.io/kubevirt/pkg/network/infraconfigurators"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const defaultState = cache.PodIfaceNetworkPreparationPending

type podNIC struct {
	vmi               *v1.VirtualMachineInstance
	podInterfaceName  string
	launcherPID       *int
	vmiSpecIface      *v1.Interface
	vmiSpecNetwork    *v1.Network
	handler           netdriver.NetworkHandler
	cacheFactory      cache.InterfaceCacheFactory
	dhcpConfigurator  dhcpconfigurator.Configurator
	infraConfigurator infraconfigurators.PodNetworkInfraConfigurator
	domainGenerator   domainspec.LibvirtSpecGenerator
}

func newPhase1PodNIC(vmi *v1.VirtualMachineInstance, network *v1.Network, handler netdriver.NetworkHandler, cacheFactory cache.InterfaceCacheFactory, launcherPID *int) (*podNIC, error) {
	podnic, err := newPodNIC(vmi, network, handler, cacheFactory, launcherPID)
	if err != nil {
		return nil, err
	}

	if launcherPID == nil {
		return nil, fmt.Errorf("missing launcher PID to construct infra configurators")
	}

	if podnic.vmiSpecIface.Bridge != nil {
		podnic.infraConfigurator = infraconfigurators.NewBridgePodNetworkConfigurator(
			podnic.vmi,
			podnic.vmiSpecIface,
			generateInPodBridgeInterfaceName(podnic.podInterfaceName),
			*podnic.launcherPID,
			podnic.handler)
	} else if podnic.vmiSpecIface.Masquerade != nil {
		podnic.infraConfigurator = infraconfigurators.NewMasqueradePodNetworkConfigurator(
			podnic.vmi,
			podnic.vmiSpecIface,
			generateInPodBridgeInterfaceName(podnic.podInterfaceName),
			podnic.vmiSpecNetwork,
			*podnic.launcherPID,
			podnic.handler)
	}
	return podnic, nil
}

func newPhase2PodNIC(vmi *v1.VirtualMachineInstance, network *v1.Network, handler netdriver.NetworkHandler, cacheFactory cache.InterfaceCacheFactory, domain *api.Domain) (*podNIC, error) {
	podnic, err := newPodNIC(vmi, network, handler, cacheFactory, nil)
	if err != nil {
		return nil, err
	}

	podnic.dhcpConfigurator = podnic.newDHCPConfigurator()
	podnic.domainGenerator = podnic.newLibvirtSpecGenerator(domain)

	return podnic, nil
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

	return &podNIC{
		cacheFactory:     cacheFactory,
		handler:          handler,
		vmi:              vmi,
		vmiSpecNetwork:   network,
		podInterfaceName: podInterfaceName,
		vmiSpecIface:     correspondingNetworkIface,
		launcherPID:      launcherPID,
	}, nil
}

func (l *podNIC) setPodInterfaceCache() error {
	ifCache := &cache.PodCacheInterface{Iface: l.vmiSpecIface}

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
	err = l.cacheFactory.CacheForVMI(l.vmi).Write(l.vmiSpecIface.Name, ifCache)
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
	if l.vmiSpecIface.SRIOV != nil {
		return nil
	}

	state, err := l.state()
	if err != nil {
		return err
	}

	switch state {
	case cache.PodIfaceNetworkPreparationStarted:
		return errors.CreateCriticalNetworkError(fmt.Errorf("pod interface %s network preparation cannot be resumed", l.podInterfaceName))
	case cache.PodIfaceNetworkPreparationFinished:
		return nil
	}

	if err := l.setPodInterfaceCache(); err != nil {
		return err
	}

	if l.infraConfigurator == nil {
		return nil
	}

	if err := l.infraConfigurator.DiscoverPodNetworkInterface(l.podInterfaceName); err != nil {
		return err
	}

	dhcpConfig := l.infraConfigurator.GenerateNonRecoverableDHCPConfig()
	if dhcpConfig != nil {
		log.Log.V(4).Infof("The generated dhcpConfig: %s", dhcpConfig.String())
		if err := l.cacheFactory.CacheDHCPConfigForPid(getPIDString(l.launcherPID)).Write(l.podInterfaceName, dhcpConfig); err != nil {
			return fmt.Errorf("failed to save DHCP configuration: %w", err)
		}
	}

	domainIface := l.infraConfigurator.GenerateNonRecoverableDomainIfaceSpec()
	if domainIface != nil {
		log.Log.V(4).Infof("The generated libvirt domain interface: %+v", *domainIface)
		if err := l.storeCachedDomainIface(*domainIface); err != nil {
			return fmt.Errorf("failed to save libvirt domain interface: %w", err)
		}
	}

	if err := l.setState(cache.PodIfaceNetworkPreparationStarted); err != nil {
		return fmt.Errorf("failed setting state to PodIfaceNetworkPreparationStarted: %w", err)
	}

	// preparePodNetworkInterface must be called *after* the Generate
	// methods since it mutates the pod interface from which those
	// generator methods get their info from.
	if err := l.infraConfigurator.PreparePodNetworkInterface(); err != nil {
		log.Log.Reason(err).Error("failed to prepare pod networking")
		return errors.CreateCriticalNetworkError(err)
	}

	if err := l.setState(cache.PodIfaceNetworkPreparationFinished); err != nil {
		log.Log.Reason(err).Error("failed setting state to PodIfaceNetworkPreparationFinished")
		return errors.CreateCriticalNetworkError(err)
	}

	return nil
}

func (l *podNIC) PlugPhase2(domain *api.Domain) error {
	precond.MustNotBeNil(domain)

	// There is nothing to plug for SR-IOV devices
	if l.vmiSpecIface.SRIOV != nil {
		return nil
	}

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
			l.cacheFactory,
			getPIDString(l.launcherPID),
			generateInPodBridgeInterfaceName(l.podInterfaceName),
			l.handler,
			l.podInterfaceName,
			l.vmi.Spec.Domain.Devices.Interfaces,
			l.vmiSpecIface,
			l.vmi.Spec.Subdomain)
	} else if l.vmiSpecIface.Masquerade != nil {
		dhcpConfigurator = dhcpconfigurator.NewMasqueradeConfigurator(
			generateInPodBridgeInterfaceName(l.podInterfaceName),
			l.handler,
			l.vmiSpecIface,
			l.vmiSpecNetwork,
			l.podInterfaceName,
			l.vmi.Spec.Subdomain)
	}
	return dhcpConfigurator
}

func (l *podNIC) newLibvirtSpecGenerator(domain *api.Domain) domainspec.LibvirtSpecGenerator {
	if l.vmiSpecIface.Bridge != nil {
		cachedDomainIface, err := l.cachedDomainInterface()
		if err != nil {
			return nil
		}
		if cachedDomainIface == nil {
			cachedDomainIface = &api.Interface{}
		}
		return domainspec.NewBridgeLibvirtSpecGenerator(l.vmiSpecIface, domain, *cachedDomainIface, l.podInterfaceName, l.handler)
	}
	if l.vmiSpecIface.Masquerade != nil {
		return domainspec.NewMasqueradeLibvirtSpecGenerator(l.vmiSpecIface, l.vmiSpecNetwork, domain, l.podInterfaceName, l.handler)
	}
	if l.vmiSpecIface.Slirp != nil {
		return domainspec.NewSlirpLibvirtSpecGenerator(l.vmiSpecIface, domain)
	}
	if l.vmiSpecIface.Macvtap != nil {
		return domainspec.NewMacvtapLibvirtSpecGenerator(l.vmiSpecIface, domain, l.podInterfaceName, l.handler)
	}
	return nil
}

func (l *podNIC) cachedDomainInterface() (*api.Interface, error) {
	ifaceConfig, err := l.cacheFactory.CacheDomainInterfaceForPID(getPIDString(l.launcherPID)).Read(l.vmiSpecIface.Name)

	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return ifaceConfig, nil
}

func (l *podNIC) storeCachedDomainIface(domainIface api.Interface) error {
	return l.cacheFactory.CacheDomainInterfaceForPID(getPIDString(l.launcherPID)).Write(l.vmiSpecIface.Name, &domainIface)
}

func (l *podNIC) setState(state cache.PodIfaceState) error {
	podIfaceCaches := l.cacheFactory.CacheForVMI(l.vmi)
	podIfaceCache, err := podIfaceCaches.Read(l.vmiSpecIface.Name)
	if err != nil && !os.IsNotExist(err) {
		log.Log.Reason(err).Errorf("failed to read pod interface network state from cache, %s", err.Error())
		return err
	}
	if os.IsNotExist(err) {
		podIfaceCache = &cache.PodCacheInterface{}
	}
	podIfaceCache.State = state
	err = podIfaceCaches.Write(l.vmiSpecIface.Name, podIfaceCache)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to write pod interface network state to cache, %s", err.Error())
		return err
	}
	return nil
}

func (l *podNIC) state() (cache.PodIfaceState, error) {
	podIfaceCaches := l.cacheFactory.CacheForVMI(l.vmi)
	podIfaceCache, err := podIfaceCaches.Read(l.vmiSpecIface.Name)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultState, nil
		}
		log.Log.Reason(err).Errorf("failed to read pod interface network state from cache %s", err.Error())
		return defaultState, err
	}
	return podIfaceCache.State, nil
}

func generateInPodBridgeInterfaceName(podInterfaceName string) string {
	return fmt.Sprintf("k6t-%s", podInterfaceName)
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

func getPIDString(pid *int) string {
	if pid != nil {
		return fmt.Sprintf("%d", *pid)
	}
	return "self"
}
