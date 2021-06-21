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
	"net"
	"os"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"
	"kubevirt.io/kubevirt/pkg/network/cache"
	dhcpconfigurator "kubevirt.io/kubevirt/pkg/network/dhcp"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/errors"
	"kubevirt.io/kubevirt/pkg/network/infraconfigurators"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type LibvirtSpecGenerator interface {
	generate(domainIface api.Interface) error
}

type podNIC struct {
	vmi              *v1.VirtualMachineInstance
	podInterfaceName string
	launcherPID      *int
	vmiSpecIface     *v1.Interface
	vmiSpecNetwork   *v1.Network
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
		vmiSpecNetwork:   network,
		podInterfaceName: podInterfaceName,
		vmiSpecIface:     correspondingNetworkIface,
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

	cachedDomainIface, err := l.cachedDomainInterface()
	if err != nil {
		return err
	}

	doesExist := cachedDomainIface != nil
	// ignore the bindMechanism.cachedDomainInterface for slirp and set the Pod interface cache
	if !doesExist {
		if err := l.setPodInterfaceCache(); err != nil {
			return err
		}
	}

	isSlirpIface := l.vmiSpecIface.Slirp != nil
	if isSlirpIface {
		return nil
	}

	if !doesExist {
		podNetworkingConfigurator, err := l.newPodNetworkConfigurator()
		if err != nil {
			return err
		}

		if err := podNetworkingConfigurator.DiscoverPodNetworkInterface(l.podInterfaceName); err != nil {
			return err
		}

		if l.dhcpConfigurator != nil {
			dhcpConfig := podNetworkingConfigurator.GenerateDHCPConfig()
			log.Log.V(4).Infof("The generated dhcpConfig: %s", dhcpConfig.String())
			if err := l.dhcpConfigurator.ExportConfiguration(*dhcpConfig); err != nil {
				log.Log.Reason(err).Error("failed to save dhcpConfig configuration")
				return errors.CreateCriticalNetworkError(err)
			}
		}

		domainIface := podNetworkingConfigurator.GenerateDomainIfaceSpec()
		// preparePodNetworkInterface must be called *after* the generate
		// methods since it mutates the pod interface from which those
		// generator methods get their info from.
		if err := podNetworkingConfigurator.PreparePodNetworkInterface(); err != nil {
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
	if l.vmiSpecIface.SRIOV != nil {
		return nil
	}

	libvirtSpecGenerator, err := l.newLibvirtSpecGenerator(domain)
	if err != nil {
		return err
	}

	if err := libvirtSpecGenerator.generate(l.getInfoForLibvirtDomainInterface()); err != nil {
		log.Log.Reason(err).Critical("failed to create libvirt configuration")
	}

	if l.dhcpConfigurator != nil {
		dhcpConfig, err := l.dhcpConfigurator.ImportConfiguration(l.podInterfaceName)
		if err != nil || dhcpConfig == nil {
			log.Log.Reason(err).Critical("failed to load cached dhcpConfig configuration")
		}
		log.Log.V(4).Infof("The imported dhcpConfig: %s", dhcpConfig.String())
		if err := l.dhcpConfigurator.EnsureDHCPServerStarted(l.podInterfaceName, *dhcpConfig, l.vmiSpecIface.DHCPOptions); err != nil {
			log.Log.Reason(err).Criticalf("failed to ensure dhcp service running for: %s", l.podInterfaceName)
			panic(err)
		}
	}

	return nil
}

func (l *podNIC) getInfoForLibvirtDomainInterface() api.Interface {
	if l.vmiSpecIface.Slirp == nil {
		domainIface, err := l.cachedDomainInterface()
		if err != nil {
			log.Log.Reason(err).Critical("failed to load cached interface configuration")
		}
		if domainIface == nil {
			log.Log.Reason(err).Critical("cached interface configuration doesn't exist")
		}
		return *domainIface
	}
	return api.Interface{}
}

func (l *podNIC) newLibvirtSpecGenerator(domain *api.Domain) (LibvirtSpecGenerator, error) {
	if l.vmiSpecIface.Bridge != nil {
		return newBridgeLibvirtSpecGenerator(l.vmiSpecIface, domain), nil
	}
	if l.vmiSpecIface.Masquerade != nil {
		return newMasqueradeLibvirtSpecGenerator(l.vmiSpecIface, domain), nil
	}
	if l.vmiSpecIface.Slirp != nil {
		return newSlirpLibvirtSpecGenerator(l.vmiSpecIface, domain), nil
	}
	if l.vmiSpecIface.Macvtap != nil {
		return newMacvtapLibvirtSpecGenerator(l.vmiSpecIface, domain), nil
	}
	return nil, fmt.Errorf("Not implemented")
}

func newMacvtapLibvirtSpecGenerator(iface *v1.Interface, domain *api.Domain) *MacvtapLibvirtSpecGenerator {
	return &MacvtapLibvirtSpecGenerator{
		vmiSpecIface: iface,
		domain:       domain,
	}
}

func newMasqueradeLibvirtSpecGenerator(iface *v1.Interface, domain *api.Domain) *MasqueradeLibvirtSpecGenerator {
	return &MasqueradeLibvirtSpecGenerator{
		vmiSpecIface: iface,
		domain:       domain,
	}
}

func newSlirpLibvirtSpecGenerator(iface *v1.Interface, domain *api.Domain) *SlirpLibvirtSpecGenerator {
	return &SlirpLibvirtSpecGenerator{
		vmiSpecIface: iface,
		domain:       domain,
	}
}

func newBridgeLibvirtSpecGenerator(iface *v1.Interface, domain *api.Domain) *BridgeLibvirtSpecGenerator {
	return &BridgeLibvirtSpecGenerator{
		vmiSpecIface: iface,
		domain:       domain,
	}
}

func (l *podNIC) newPodNetworkConfigurator() (infraconfigurators.PodNetworkInfraConfigurator, error) {
	if err != nil {
		return nil, err
	}
	if l.vmiSpecIface.Bridge != nil {
		return infraconfigurators.NewBridgePodNetworkConfigurator(
			l.vmi,
			l.vmiSpecIface,
			generateInPodBridgeInterfaceName(l.podInterfaceName),
			l.cacheFactory,
			*l.launcherPID,
			l.handler), nil
	}
	if l.vmiSpecIface.Masquerade != nil {
		return infraconfigurators.NewMasqueradePodNetworkConfigurator(
			l.vmi,
			l.vmiSpecIface,
			generateInPodBridgeInterfaceName(l.podInterfaceName),
			l.vmiSpecNetwork.Pod.VMNetworkCIDR,
			l.vmiSpecNetwork.Pod.VMIPv6NetworkCIDR,
			l.cacheFactory,
			*l.launcherPID,
			l.handler), nil
	}
	if l.vmiSpecIface.Slirp != nil {
		return nil, nil
	}
	if l.vmiSpecIface.Macvtap != nil {
		return infraconfigurators.NewMacvtapPodNetworkConfigurator(
			l.podInterfaceName,
			l.vmiSpecIface,
			l.handler), nil
	}
	return nil, fmt.Errorf("Not implemented")
}

type BridgeLibvirtSpecGenerator struct {
	vmiSpecIface *v1.Interface
	domain       *api.Domain
}

func (b *BridgeLibvirtSpecGenerator) generate(domainIface api.Interface) error {
	ifaces := b.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == b.vmiSpecIface.Name {
			ifaces[i].MTU = domainIface.MTU
			ifaces[i].MAC = domainIface.MAC
			ifaces[i].Target = domainIface.Target
			break
		}
	}
	return nil
}

func getPIDString(pid *int) string {
	if pid != nil {
		return fmt.Sprintf("%d", *pid)
	}
	return "self"
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

type MasqueradeLibvirtSpecGenerator struct {
	vmiSpecIface *v1.Interface
	domain       *api.Domain
}

func (b *MasqueradeLibvirtSpecGenerator) generate(domainIface api.Interface) error {
	ifaces := b.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == b.vmiSpecIface.Name {
			ifaces[i].MTU = domainIface.MTU
			ifaces[i].MAC = domainIface.MAC
			ifaces[i].Target = domainIface.Target
			break
		}
	}
	return nil
}

type SlirpLibvirtSpecGenerator struct {
	vmiSpecIface *v1.Interface
	domain       *api.Domain
}

func (b *SlirpLibvirtSpecGenerator) generate(api.Interface) error {
	// remove slirp interface from domain spec devices interfaces
	var foundIfaceModelType string
	ifaces := b.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == b.vmiSpecIface.Name {
			b.domain.Spec.Devices.Interfaces = append(ifaces[:i], ifaces[i+1:]...)
			foundIfaceModelType = iface.Model.Type
			break
		}
	}

	if foundIfaceModelType == "" {
		return fmt.Errorf("failed to find interface %s in vmi spec", b.vmiSpecIface.Name)
	}

	qemuArg := fmt.Sprintf("%s,netdev=%s,id=%s", foundIfaceModelType, b.vmiSpecIface.Name, b.vmiSpecIface.Name)
	if b.vmiSpecIface.MacAddress != "" {
		// We assume address was already validated in API layer so just pass it to libvirt as-is.
		qemuArg += fmt.Sprintf(",mac=%s", b.vmiSpecIface.MacAddress)
	}
	// Add interface configuration to qemuArgs
	b.domain.Spec.QEMUCmd.QEMUArg = append(b.domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: "-device"})
	b.domain.Spec.QEMUCmd.QEMUArg = append(b.domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: qemuArg})

	return nil
}

type MacvtapLibvirtSpecGenerator struct {
	vmiSpecIface *v1.Interface
	domain       *api.Domain
}

func (b *MacvtapLibvirtSpecGenerator) generate(domainIface api.Interface) error {
	ifaces := b.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == b.vmiSpecIface.Name {
			ifaces[i].MTU = domainIface.MTU
			ifaces[i].MAC = domainIface.MAC
			ifaces[i].Target = domainIface.Target
			break
		}
	}
	return nil
}

func generateInPodBridgeInterfaceName(podInterfaceName string) string {
	return fmt.Sprintf("k6t-%s", podInterfaceName)
}
