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
	"strconv"
	"strings"

	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/network"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var bridgeFakeIP = "169.254.75.1%d/32"

const (
	LibvirtLocalConnectionPort = 22222
	LibvirtDirectMigrationPort = 49152
	LibvirtBlockMigrationPort  = 49153
)

type BindMechanism interface {
	discoverPodNetworkInterface(podIfaceName string) error
	preparePodNetworkInterface() error
	generateDomainIfaceSpec() api.Interface
	generateDhcpConfig() *cache.DhcpConfig

	// The following entry points require domain initialized for the
	// binding and can be used in phase2 only.
	decorateConfig(domainIface api.Interface) error
}

type BridgeBindMechanism struct {
	vmi                 *v1.VirtualMachineInstance
	iface               *v1.Interface
	podNicLink          netlink.Link
	domain              *api.Domain
	bridgeInterfaceName string
	arpIgnore           bool
	cacheFactory        cache.InterfaceCacheFactory
	launcherPID         *int
	queueCount          uint32
	handler             netdriver.NetworkHandler
	tapDeviceName       string
	podIfaceIP          netlink.Addr
	ipamEnabled         bool
	mac                 *net.HardwareAddr
	routes              []netlink.Route
}

func (b *BridgeBindMechanism) discoverPodNetworkInterface(podIfaceName string) error {
	link, err := b.handler.LinkByName(podIfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", podIfaceName)
		return err
	}
	b.podNicLink = link

	addrList, err := b.handler.AddrList(b.podNicLink, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get an ip address for %s", podIfaceName)
		return err
	}
	if len(addrList) == 0 {
		b.ipamEnabled = false
	} else {
		b.podIfaceIP = addrList[0]
		b.ipamEnabled = true
		if err := b.learnInterfaceRoutes(); err != nil {
			return err
		}
	}

	b.tapDeviceName = generateTapDeviceName(podIfaceName)
	if b.mac == nil {
		b.mac = &b.podNicLink.Attrs().HardwareAddr
	}

	if err := validateMTU(b.podNicLink.Attrs().MTU); err != nil {
		return err
	}

	return nil
}

func (b *BridgeBindMechanism) generateDhcpConfig() *cache.DhcpConfig {
	if !b.ipamEnabled {
		return &cache.DhcpConfig{Name: b.podNicLink.Attrs().Name, IPAMDisabled: true}
	}
	fakeBridgeIP, err := b.getFakeBridgeIP()
	if err != nil {
		return nil
	}
	fakeServerAddr, err := netlink.ParseAddr(fakeBridgeIP)
	if err != nil || fakeServerAddr == nil {
		return nil
	}
	dhcpConfig := &cache.DhcpConfig{
		MAC:               *b.mac,
		Name:              b.podNicLink.Attrs().Name,
		IPAMDisabled:      !b.ipamEnabled,
		IP:                b.podIfaceIP,
		AdvertisingIPAddr: fakeServerAddr.IP,
	}
	if b.podNicLink != nil {
		dhcpConfig.Mtu = uint16(b.podNicLink.Attrs().MTU)
	}

	if b.ipamEnabled && len(b.routes) > 0 {
		b.decorateDhcpConfigRoutes(dhcpConfig)
	}
	return dhcpConfig
}

func (b *BridgeBindMechanism) getFakeBridgeIP() (string, error) {
	ifaces := b.vmi.Spec.Domain.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Name == b.iface.Name {
			return fmt.Sprintf(bridgeFakeIP, i), nil
		}
	}
	return "", fmt.Errorf("Failed to generate bridge fake address for interface %s", b.iface.Name)
}

func (b *BridgeBindMechanism) preparePodNetworkInterface() error {
	// Set interface link to down to change its MAC address
	if err := b.handler.LinkSetDown(b.podNicLink); err != nil {
		log.Log.Reason(err).Errorf("failed to bring link down for interface: %s", b.podNicLink.Attrs().Name)
		return err
	}

	if b.ipamEnabled {
		// Remove IP from POD interface
		err := b.handler.AddrDel(b.podNicLink, &b.podIfaceIP)

		if err != nil {
			log.Log.Reason(err).Errorf("failed to delete address for interface: %s", b.podNicLink.Attrs().Name)
			return err
		}

		if err := b.switchPodInterfaceWithDummy(); err != nil {
			log.Log.Reason(err).Error("failed to switch pod interface with a dummy")
			return err
		}
	}

	if _, err := b.handler.SetRandomMac(b.podNicLink.Attrs().Name); err != nil {
		return err
	}

	if err := b.createBridge(); err != nil {
		return err
	}

	err := createAndBindTapToBridge(b.handler, b.tapDeviceName, b.bridgeInterfaceName, b.queueCount, *b.launcherPID, b.podNicLink.Attrs().MTU, netdriver.LibvirtUserAndGroupId)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create tap device named %s", b.tapDeviceName)
		return err
	}

	if b.arpIgnore {
		if err := b.handler.ConfigureIpv4ArpIgnore(); err != nil {
			log.Log.Reason(err).Errorf("failed to set arp_ignore=1 on interface %s", b.bridgeInterfaceName)
			return err
		}
	}

	if err := b.handler.LinkSetUp(b.podNicLink); err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.podNicLink.Attrs().Name)
		return err
	}

	if err := b.handler.LinkSetLearningOff(b.podNicLink); err != nil {
		log.Log.Reason(err).Errorf("failed to disable mac learning for interface: %s", b.podNicLink.Attrs().Name)
		return err
	}

	return nil
}

func (b *BridgeBindMechanism) generateDomainIfaceSpec() api.Interface {
	return api.Interface{
		MAC: &api.MAC{MAC: b.mac.String()},
		MTU: &api.MTU{Size: strconv.Itoa(b.podNicLink.Attrs().MTU)},
		Target: &api.InterfaceTarget{
			Device:  b.tapDeviceName,
			Managed: "no",
		},
	}
}

func (b *BridgeBindMechanism) decorateConfig(domainIface api.Interface) error {
	ifaces := b.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == b.iface.Name {
			ifaces[i].MTU = domainIface.MTU
			ifaces[i].MAC = domainIface.MAC
			ifaces[i].Target = domainIface.Target
			break
		}
	}
	return nil
}

func (b *BridgeBindMechanism) learnInterfaceRoutes() error {
	routes, err := b.handler.RouteList(b.podNicLink, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get routes for %s", b.podNicLink.Attrs().Name)
		return err
	}
	if len(routes) == 0 {
		return fmt.Errorf("No gateway address found in routes for %s", b.podNicLink.Attrs().Name)
	}
	b.routes = routes
	return nil
}

func (b *BridgeBindMechanism) decorateDhcpConfigRoutes(dhcpConfig *cache.DhcpConfig) {
	dhcpConfig.AdvertisingIPAddr = b.routes[0].Gw
	if len(b.routes) > 1 {
		dhcpRoutes := netdriver.FilterPodNetworkRoutes(b.routes, dhcpConfig.IP.IP)
		dhcpConfig.Routes = &dhcpRoutes
	}
}

func (b *BridgeBindMechanism) createBridge() error {
	// Create a bridge
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: b.bridgeInterfaceName,
		},
	}
	err := b.handler.LinkAdd(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create a bridge")
		return err
	}

	err = b.handler.LinkSetMaster(b.podNicLink, bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to connect interface %s to bridge %s", b.podNicLink.Attrs().Name, bridge.Name)
		return err
	}

	err = b.handler.LinkSetUp(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.bridgeInterfaceName)
		return err
	}

	// set fake ip on a bridge
	addr, err := b.getFakeBridgeIP()
	if err != nil {
		return err
	}
	fakeaddr, err := b.handler.ParseAddr(addr)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.bridgeInterfaceName)
		return err
	}

	if err := b.handler.AddrAdd(bridge, fakeaddr); err != nil {
		log.Log.Reason(err).Errorf("failed to set bridge IP")
		return err
	}

	if err = b.handler.DisableTXOffloadChecksum(b.bridgeInterfaceName); err != nil {
		log.Log.Reason(err).Error("failed to disable TX offload checksum on bridge interface")
		return err
	}

	return nil
}

func (b *BridgeBindMechanism) switchPodInterfaceWithDummy() error {
	originalPodInterfaceName := b.podNicLink.Attrs().Name
	newPodInterfaceName := fmt.Sprintf("%s-nic", originalPodInterfaceName)
	dummy := &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: originalPodInterfaceName}}

	// Set arp_ignore=1 on the bridge interface to avoid
	// the interface being seen by Duplicate Address Detection (DAD).
	// Without this, some VMs will lose their ip address after a few
	// minutes.
	b.arpIgnore = true

	// Rename pod interface to free the original name for a new dummy interface
	err := b.handler.LinkSetName(b.podNicLink, newPodInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to rename interface : %s", b.podNicLink.Attrs().Name)
		return err
	}

	b.podNicLink, err = b.handler.LinkByName(newPodInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", newPodInterfaceName)
		return err
	}

	// Create a dummy interface named after the original interface
	err = b.handler.LinkAdd(dummy)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create dummy interface : %s", originalPodInterfaceName)
		return err
	}

	// Replace original pod interface IP address to the dummy
	// Since the dummy is not connected to anything, it should not affect networking
	// Replace will add if ip doesn't exist or modify the ip
	err = b.handler.AddrReplace(dummy, &b.podIfaceIP)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to replace original IP address to dummy interface: %s", originalPodInterfaceName)
		return err
	}

	return nil
}

type MasqueradeBindMechanism struct {
	vmi                 *v1.VirtualMachineInstance
	iface               *v1.Interface
	podNicLink          netlink.Link
	domain              *api.Domain
	bridgeInterfaceName string
	vmNetworkCIDR       string
	vmIPv6NetworkCIDR   string
	gatewayAddr         *netlink.Addr
	gatewayIpv6Addr     *netlink.Addr
	cacheFactory        cache.InterfaceCacheFactory
	launcherPID         *int
	queueCount          uint32
	handler             netdriver.NetworkHandler
	podIfaceIPv4Addr    netlink.Addr
	podIfaceIPv6Addr    netlink.Addr
	mac                 *net.HardwareAddr
}

func (b *MasqueradeBindMechanism) discoverPodNetworkInterface(podIfaceName string) error {
	link, err := b.handler.LinkByName(podIfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", podIfaceName)
		return err
	}
	b.podNicLink = link

	if err := validateMTU(b.podNicLink.Attrs().MTU); err != nil {
		return err
	}

	if err := b.configureIPv4Addresses(); err != nil {
		return err
	}

	ipv6Enabled, err := b.handler.IsIpv6Enabled(podIfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to verify whether ipv6 is configured on %s", podIfaceName)
		return err
	}
	if ipv6Enabled {
		if err := b.configureIPv6Addresses(); err != nil {
			return err
		}
	}

	return nil
}

func (b *MasqueradeBindMechanism) configureIPv4Addresses() error {
	b.setDefaultCidr(iptables.ProtocolIPv4)
	vmIPv4Addr, gatewayIPv4, err := b.generateGatewayAndVmIPAddrs(iptables.ProtocolIPv4)
	if err != nil {
		return err
	}
	b.podIfaceIPv4Addr = *vmIPv4Addr
	b.gatewayAddr = gatewayIPv4
	return nil
}

func (b *MasqueradeBindMechanism) configureIPv6Addresses() error {
	b.setDefaultCidr(iptables.ProtocolIPv6)
	vmIPv6Addr, gatewayIPv6, err := b.generateGatewayAndVmIPAddrs(iptables.ProtocolIPv6)
	if err != nil {
		return err
	}
	b.podIfaceIPv6Addr = *vmIPv6Addr
	b.gatewayIpv6Addr = gatewayIPv6
	return nil
}

func (b *MasqueradeBindMechanism) generateDhcpConfig() *cache.DhcpConfig {
	dhcpConfig := &cache.DhcpConfig{
		Name: b.podNicLink.Attrs().Name,
		IP:   b.podIfaceIPv4Addr,
		IPv6: b.podIfaceIPv6Addr,
	}
	if b.mac != nil {
		dhcpConfig.MAC = *b.mac
	}
	if b.podNicLink != nil {
		dhcpConfig.Mtu = uint16(b.podNicLink.Attrs().MTU)
	}
	if b.gatewayAddr != nil {
		dhcpConfig.AdvertisingIPAddr = b.gatewayAddr.IP.To4()
	}
	if b.gatewayIpv6Addr != nil {
		dhcpConfig.AdvertisingIPv6Addr = b.gatewayIpv6Addr.IP.To16()
	}

	return dhcpConfig
}

func (b *MasqueradeBindMechanism) setDefaultCidr(protocol iptables.Protocol) {
	if protocol == iptables.ProtocolIPv4 {
		if b.vmNetworkCIDR == "" {
			b.vmNetworkCIDR = api.DefaultVMCIDR
		}
	} else {
		if b.vmIPv6NetworkCIDR == "" {
			b.vmIPv6NetworkCIDR = api.DefaultVMIpv6CIDR
		}
	}
}

func (b *MasqueradeBindMechanism) generateGatewayAndVmIPAddrs(protocol iptables.Protocol) (*netlink.Addr, *netlink.Addr, error) {
	cidrToConfigure := b.vmNetworkCIDR
	if protocol == iptables.ProtocolIPv6 {
		cidrToConfigure = b.vmIPv6NetworkCIDR
	}

	vmIP, gatewayIP, err := b.handler.GetHostAndGwAddressesFromCIDR(cidrToConfigure)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get gw and vm available addresses from CIDR %s", cidrToConfigure)
		return nil, nil, err
	}

	gatewayAddr, err := b.handler.ParseAddr(gatewayIP)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse gateway address %s err %v", gatewayAddr, err)
	}
	vmAddr, err := b.handler.ParseAddr(vmIP)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse vm address %s err %v", vmAddr, err)
	}
	return gatewayAddr, vmAddr, nil
}

func (b *MasqueradeBindMechanism) preparePodNetworkInterface() error {
	if err := b.createBridge(); err != nil {
		return err
	}

	tapDeviceName := generateTapDeviceName(b.podNicLink.Attrs().Name)
	err := createAndBindTapToBridge(b.handler, tapDeviceName, b.bridgeInterfaceName, b.queueCount, *b.launcherPID, b.podNicLink.Attrs().MTU, netdriver.LibvirtUserAndGroupId)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create tap device named %s", tapDeviceName)
		return err
	}

	err = b.createNatRules(iptables.ProtocolIPv4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create ipv4 nat rules for vm error: %v", err)
		return err
	}

	ipv6Enabled, err := b.handler.IsIpv6Enabled(b.podNicLink.Attrs().Name)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to verify whether ipv6 is configured on %s", b.podNicLink.Attrs().Name)
		return err
	}
	if ipv6Enabled {
		err = b.createNatRules(iptables.ProtocolIPv6)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to create ipv6 nat rules for vm error: %v", err)
			return err
		}
	}

	return nil
}

func (b *MasqueradeBindMechanism) generateDomainIfaceSpec() api.Interface {
	domainIface := api.Interface{
		MTU: &api.MTU{Size: strconv.Itoa(b.podNicLink.Attrs().MTU)},
		Target: &api.InterfaceTarget{
			Device:  generateTapDeviceName(b.podNicLink.Attrs().Name),
			Managed: "no",
		},
	}
	if b.mac != nil {
		domainIface.MAC = &api.MAC{MAC: b.mac.String()}
	}
	return domainIface
}

func (b *MasqueradeBindMechanism) decorateConfig(domainIface api.Interface) error {
	ifaces := b.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == b.iface.Name {
			ifaces[i].MTU = domainIface.MTU
			ifaces[i].MAC = domainIface.MAC
			ifaces[i].Target = domainIface.Target
			break
		}
	}
	return nil
}

func (b *MasqueradeBindMechanism) createBridge() error {
	mac, err := net.ParseMAC(network.StaticMasqueradeBridgeMAC)
	if err != nil {
		return err
	}
	// Create a bridge
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name:         b.bridgeInterfaceName,
			MTU:          b.podNicLink.Attrs().MTU,
			HardwareAddr: mac,
		},
	}
	err = b.handler.LinkAdd(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create a bridge")
		return err
	}

	err = b.handler.LinkSetUp(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.bridgeInterfaceName)
		return err
	}

	if err := b.handler.AddrAdd(bridge, b.gatewayAddr); err != nil {
		log.Log.Reason(err).Errorf("failed to set bridge IP")
		return err
	}

	ipv6Enabled, err := b.handler.IsIpv6Enabled(b.podNicLink.Attrs().Name)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to verify whether ipv6 is configured on %s", b.podNicLink.Attrs().Name)
		return err
	}
	if ipv6Enabled {
		if err := b.handler.AddrAdd(bridge, b.gatewayIpv6Addr); err != nil {
			log.Log.Reason(err).Errorf("failed to set bridge IPv6")
			return err
		}
	}

	if err = b.handler.DisableTXOffloadChecksum(b.bridgeInterfaceName); err != nil {
		log.Log.Reason(err).Error("failed to disable TX offload checksum on bridge interface")
		return err
	}

	return nil
}

func (b *MasqueradeBindMechanism) createNatRules(protocol iptables.Protocol) error {
	err := b.handler.ConfigureIpForwarding(protocol)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to configure ip forwarding")
		return err
	}

	if b.handler.NftablesLoad(protocol) == nil {
		return b.createNatRulesUsingNftables(protocol)
	} else if b.handler.HasNatIptables(protocol) {
		return b.createNatRulesUsingIptables(protocol)
	}
	return fmt.Errorf("Couldn't configure ip nat rules")
}

func (b *MasqueradeBindMechanism) createNatRulesUsingIptables(protocol iptables.Protocol) error {
	err := b.handler.IptablesNewChain(protocol, "nat", "KUBEVIRT_PREINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.IptablesNewChain(protocol, "nat", "KUBEVIRT_POSTINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.IptablesAppendRule(protocol, "nat", "POSTROUTING", "-s", b.geVmIfaceIpByProtocol(protocol), "-j", "MASQUERADE")
	if err != nil {
		return err
	}

	err = b.handler.IptablesAppendRule(protocol, "nat", "PREROUTING", "-i", b.podNicLink.Attrs().Name, "-j", "KUBEVIRT_PREINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.IptablesAppendRule(protocol, "nat", "POSTROUTING", "-o", b.bridgeInterfaceName, "-j", "KUBEVIRT_POSTINBOUND")
	if err != nil {
		return err
	}

	err = b.skipForwardingForReservedPortsUsingIptables(protocol)
	if err != nil {
		return err
	}

	if len(b.iface.Ports) == 0 {
		err = b.handler.IptablesAppendRule(protocol, "nat", "KUBEVIRT_PREINBOUND",
			"-j",
			"DNAT",
			"--to-destination", b.geVmIfaceIpByProtocol(protocol))
		if err != nil {
			return err
		}

		err = b.handler.IptablesAppendRule(protocol, "nat", "KUBEVIRT_POSTINBOUND",
			"--source", getLoopbackAdrress(protocol),
			"-j",
			"SNAT",
			"--to-source", b.getGatewayByProtocol(protocol))
		if err != nil {
			return err
		}

		err = b.handler.IptablesAppendRule(protocol, "nat", "OUTPUT",
			"--destination", getLoopbackAdrress(protocol),
			"-j",
			"DNAT",
			"--to-destination", b.geVmIfaceIpByProtocol(protocol))
		if err != nil {
			return err
		}

		return nil
	}

	for _, port := range b.iface.Ports {
		if port.Protocol == "" {
			port.Protocol = "tcp"
		}

		err = b.handler.IptablesAppendRule(protocol, "nat", "KUBEVIRT_POSTINBOUND",
			"-p",
			strings.ToLower(port.Protocol),
			"--dport",
			strconv.Itoa(int(port.Port)),
			"--source", getLoopbackAdrress(protocol),
			"-j",
			"SNAT",
			"--to-source", b.getGatewayByProtocol(protocol))
		if err != nil {
			return err
		}

		err = b.handler.IptablesAppendRule(protocol, "nat", "KUBEVIRT_PREINBOUND",
			"-p",
			strings.ToLower(port.Protocol),
			"--dport",
			strconv.Itoa(int(port.Port)),
			"-j",
			"DNAT",
			"--to-destination", b.geVmIfaceIpByProtocol(protocol))
		if err != nil {
			return err
		}

		err = b.handler.IptablesAppendRule(protocol, "nat", "OUTPUT",
			"-p",
			strings.ToLower(port.Protocol),
			"--dport",
			strconv.Itoa(int(port.Port)),
			"--destination", getLoopbackAdrress(protocol),
			"-j",
			"DNAT",
			"--to-destination", b.geVmIfaceIpByProtocol(protocol))
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *MasqueradeBindMechanism) skipForwardingForReservedPortsUsingIptables(protocol iptables.Protocol) error {
	chainWhereDnatIsPerformed := "OUTPUT"
	chainWhereSnatIsPerformed := "KUBEVIRT_POSTINBOUND"
	for _, chain := range []string{chainWhereDnatIsPerformed, chainWhereSnatIsPerformed} {
		err := b.handler.IptablesAppendRule(protocol, "nat", chain,
			"-p", "tcp", "--match", "multiport",
			"--dports", fmt.Sprintf("%s", strings.Join(portsUsedByLiveMigration(), ",")),
			"--source", getLoopbackAdrress(protocol),
			"-j", "RETURN")
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *MasqueradeBindMechanism) createNatRulesUsingNftables(proto iptables.Protocol) error {
	err := b.handler.NftablesNewChain(proto, "nat", "KUBEVIRT_PREINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.NftablesNewChain(proto, "nat", "KUBEVIRT_POSTINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.NftablesAppendRule(proto, "nat", "postrouting", b.handler.GetNFTIPString(proto), "saddr", b.geVmIfaceIpByProtocol(proto), "counter", "masquerade")
	if err != nil {
		return err
	}

	err = b.handler.NftablesAppendRule(proto, "nat", "prerouting", "iifname", b.podNicLink.Attrs().Name, "counter", "jump", "KUBEVIRT_PREINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.NftablesAppendRule(proto, "nat", "postrouting", "oifname", b.bridgeInterfaceName, "counter", "jump", "KUBEVIRT_POSTINBOUND")
	if err != nil {
		return err
	}

	err = b.skipForwardingForReservedPortsUsingNftables(proto)
	if err != nil {
		return err
	}

	if len(b.iface.Ports) == 0 {
		err = b.handler.NftablesAppendRule(proto, "nat", "KUBEVIRT_PREINBOUND",
			"counter", "dnat", "to", b.geVmIfaceIpByProtocol(proto))
		if err != nil {
			return err
		}

		err = b.handler.NftablesAppendRule(proto, "nat", "KUBEVIRT_POSTINBOUND",
			b.handler.GetNFTIPString(proto), "saddr", getLoopbackAdrress(proto),
			"counter", "snat", "to", b.getGatewayByProtocol(proto))
		if err != nil {
			return err
		}

		err = b.handler.NftablesAppendRule(proto, "nat", "output",
			b.handler.GetNFTIPString(proto), "daddr", getLoopbackAdrress(proto),
			"counter", "dnat", "to", b.geVmIfaceIpByProtocol(proto))
		if err != nil {
			return err
		}

		return nil
	}

	for _, port := range b.iface.Ports {
		if port.Protocol == "" {
			port.Protocol = "tcp"
		}

		err = b.handler.NftablesAppendRule(proto, "nat", "KUBEVIRT_POSTINBOUND",
			strings.ToLower(port.Protocol),
			"dport",
			strconv.Itoa(int(port.Port)),
			b.handler.GetNFTIPString(proto), "saddr", b.getSrcAddressesToSnat(proto),
			"counter", "snat", "to", b.getGatewayByProtocol(proto))
		if err != nil {
			return err
		}

		if !hasIstioSidecarInjectionEnabled(b.vmi) {
			err = b.handler.NftablesAppendRule(proto, "nat", "KUBEVIRT_PREINBOUND",
				strings.ToLower(port.Protocol),
				"dport",
				strconv.Itoa(int(port.Port)),
				"counter", "dnat", "to", b.geVmIfaceIpByProtocol(proto))
			if err != nil {
				return err
			}
		}

		addressesToDnat, err := b.getDstAddressesToDnat(proto)
		if err != nil {
			return err
		}
		err = b.handler.NftablesAppendRule(proto, "nat", "output",
			b.handler.GetNFTIPString(proto), "daddr", addressesToDnat,
			strings.ToLower(port.Protocol),
			"dport",
			strconv.Itoa(int(port.Port)),
			"counter", "dnat", "to", b.geVmIfaceIpByProtocol(proto))
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *MasqueradeBindMechanism) skipForwardingForReservedPortsUsingNftables(proto iptables.Protocol) error {
	chainWhereDnatIsPerformed := "output"
	chainWhereSnatIsPerformed := "KUBEVIRT_POSTINBOUND"
	for _, chain := range []string{chainWhereDnatIsPerformed, chainWhereSnatIsPerformed} {
		err := b.handler.NftablesAppendRule(proto, "nat", chain,
			"tcp", "dport", fmt.Sprintf("{ %s }", strings.Join(portsUsedByLiveMigration(), ", ")),
			b.handler.GetNFTIPString(proto), "saddr", getLoopbackAdrress(proto),
			"counter", "return")
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *MasqueradeBindMechanism) getGatewayByProtocol(proto iptables.Protocol) string {
	if proto == iptables.ProtocolIPv4 {
		return b.gatewayAddr.IP.String()
	} else {
		return b.gatewayIpv6Addr.IP.String()
	}
}

func (b *MasqueradeBindMechanism) geVmIfaceIpByProtocol(proto iptables.Protocol) string {
	if proto == iptables.ProtocolIPv4 {
		return b.podIfaceIPv4Addr.IP.String()
	} else {
		return b.podIfaceIPv6Addr.IP.String()
	}
}

func (b *MasqueradeBindMechanism) getSrcAddressesToSnat(proto iptables.Protocol) string {
	addresses := []string{getLoopbackAdrress(proto)}
	if hasIstioSidecarInjectionEnabled(b.vmi) && proto == iptables.ProtocolIPv4 {
		addresses = append(addresses, getEnvoyLoopbackAddress())
	}
	return fmt.Sprintf("{ %s }", strings.Join(addresses, ", "))
}

func (b *MasqueradeBindMechanism) getDstAddressesToDnat(proto iptables.Protocol) (string, error) {
	addresses := []string{getLoopbackAdrress(proto)}
	if hasIstioSidecarInjectionEnabled(b.vmi) && proto == iptables.ProtocolIPv4 {
		ipv4, _, err := b.handler.ReadIPAddressesFromLink(b.podNicLink.Attrs().Name)
		if err != nil {
			return "", err
		}
		addresses = append(addresses, ipv4)
	}
	return fmt.Sprintf("{ %s }", strings.Join(addresses, ", ")), nil
}

func hasIstioSidecarInjectionEnabled(vmi *v1.VirtualMachineInstance) bool {
	if val, ok := vmi.GetAnnotations()["sidecar.istio.io/inject"]; ok {
		return strings.ToLower(val) == "true"
	}
	return false
}

func getEnvoyLoopbackAddress() string {
	return "127.0.0.6"
}

func getLoopbackAdrress(proto iptables.Protocol) string {
	if proto == iptables.ProtocolIPv4 {
		return "127.0.0.1"
	} else {
		return "::1"
	}
}

func portsUsedByLiveMigration() []string {
	return []string{
		fmt.Sprint(LibvirtLocalConnectionPort),
		fmt.Sprint(LibvirtDirectMigrationPort),
		fmt.Sprint(LibvirtBlockMigrationPort),
	}
}

type SlirpBindMechanism struct {
	iface  *v1.Interface
	domain *api.Domain
}

func (b *SlirpBindMechanism) discoverPodNetworkInterface(podIfaceName string) error {
	return nil
}

func (b *SlirpBindMechanism) preparePodNetworkInterface() error {
	return nil
}

func (b *SlirpBindMechanism) generateDomainIfaceSpec() api.Interface {
	return api.Interface{}
}

func (b *SlirpBindMechanism) decorateConfig(api.Interface) error {
	// remove slirp interface from domain spec devices interfaces
	var foundIfaceModelType string
	ifaces := b.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == b.iface.Name {
			b.domain.Spec.Devices.Interfaces = append(ifaces[:i], ifaces[i+1:]...)
			foundIfaceModelType = iface.Model.Type
			break
		}
	}

	if foundIfaceModelType == "" {
		return fmt.Errorf("failed to find interface %s in vmi spec", b.iface.Name)
	}

	qemuArg := fmt.Sprintf("%s,netdev=%s,id=%s", foundIfaceModelType, b.iface.Name, b.iface.Name)
	if b.iface.MacAddress != "" {
		// We assume address was already validated in API layer so just pass it to libvirt as-is.
		qemuArg += fmt.Sprintf(",mac=%s", b.iface.MacAddress)
	}
	// Add interface configuration to qemuArgs
	b.domain.Spec.QEMUCmd.QEMUArg = append(b.domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: "-device"})
	b.domain.Spec.QEMUCmd.QEMUArg = append(b.domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: qemuArg})

	return nil
}

func (b *SlirpBindMechanism) generateDhcpConfig() *cache.DhcpConfig {
	return nil
}

type MacvtapBindMechanism struct {
	vmi          *v1.VirtualMachineInstance
	iface        *v1.Interface
	domain       *api.Domain
	podNicLink   netlink.Link
	mac          *net.HardwareAddr
	cacheFactory cache.InterfaceCacheFactory
	launcherPID  *int
	handler      netdriver.NetworkHandler
}

func (b *MacvtapBindMechanism) discoverPodNetworkInterface(podIfaceName string) error {
	link, err := b.handler.LinkByName(podIfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", podIfaceName)
		return err
	}
	b.podNicLink = link

	return nil
}

func (b *MacvtapBindMechanism) podIfaceMAC() string {
	if b.mac != nil {
		return b.mac.String()
	} else {
		return b.podNicLink.Attrs().HardwareAddr.String()
	}
}

func (b *MacvtapBindMechanism) preparePodNetworkInterface() error {
	return nil
}

func (b *MacvtapBindMechanism) generateDomainIfaceSpec() api.Interface {
	return api.Interface{
		MAC: &api.MAC{MAC: b.podIfaceMAC()},
		MTU: &api.MTU{Size: strconv.Itoa(b.podNicLink.Attrs().MTU)},
		Target: &api.InterfaceTarget{
			Device:  b.podNicLink.Attrs().Name,
			Managed: "no",
		},
	}
}

func (b *MacvtapBindMechanism) decorateConfig(domainIface api.Interface) error {
	ifaces := b.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == b.iface.Name {
			ifaces[i].MTU = domainIface.MTU
			ifaces[i].MAC = domainIface.MAC
			ifaces[i].Target = domainIface.Target
			break
		}
	}
	return nil
}

func (b *MacvtapBindMechanism) generateDhcpConfig() *cache.DhcpConfig {
	return nil
}

func createAndBindTapToBridge(handler netdriver.NetworkHandler, deviceName string, bridgeIfaceName string, queueNumber uint32, launcherPID int, mtu int, tapOwner string) error {
	err := handler.CreateTapDevice(deviceName, queueNumber, launcherPID, mtu, tapOwner)
	if err != nil {
		return err
	}
	return handler.BindTapDeviceToBridge(deviceName, bridgeIfaceName)
}

func generateTapDeviceName(podInterfaceName string) string {
	return "tap" + podInterfaceName[3:]
}

func validateMTU(mtu int) error {
	if mtu < 0 || mtu > 65535 {
		return fmt.Errorf("MTU value out of range ")
	}
	return nil
}
