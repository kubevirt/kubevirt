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

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

package network

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var bridgeFakeIP = "169.254.75.1%d/32"

type BindMechanism interface {
	discoverPodNetworkInterface() error
	preparePodNetworkInterfaces() error
	decorateConfig() error
	loadCachedInterface(name string) (bool, error)
	setCachedInterface(name string) error
}

type PodInterface struct{}

func (l *PodInterface) Unplug() {}

func findInterfaceByName(ifaces []api.Interface, name string) (int, error) {
	for i, iface := range ifaces {
		if iface.Alias.Name == name {
			return i, nil
		}
	}
	return 0, fmt.Errorf("failed to find interface with alias set to %s", name)
}

// Plug connect a Pod network device to the virtual machine
func (l *PodInterface) Plug(iface *v1.Interface, network *v1.Network, domain *api.Domain, podInterfaceName string) error {
	precond.MustNotBeNil(domain)
	initHandler()

	// There is nothing to plug for SR-IOV devices
	if iface.SRIOV != nil {
		return nil
	}

	driver, err := getBinding(iface, network, domain, podInterfaceName)
	if err != nil {
		return err
	}

	isExist, err := driver.loadCachedInterface(iface.Name)
	if err != nil {
		return err
	}

	if !isExist {
		err := driver.discoverPodNetworkInterface()
		if err != nil {
			return err
		}

		if err := driver.preparePodNetworkInterfaces(); err != nil {
			log.Log.Reason(err).Critical("failed to prepared pod networking")
			panic(err)
		}

		// After the network is configured, cache the result
		// in case this function is called again.
		err = driver.decorateConfig()
		if err != nil {
			log.Log.Reason(err).Critical("failed to create libvirt configuration")
			panic(err)
		}

		err = driver.setCachedInterface(iface.Name)
		if err != nil {
			log.Log.Reason(err).Critical("failed to save interface configuration")
			panic(err)
		}
	}

	return nil
}

func getBinding(iface *v1.Interface, network *v1.Network, domain *api.Domain, podInterfaceName string) (BindMechanism, error) {
	podInterfaceNum, err := findInterfaceByName(domain.Spec.Devices.Interfaces, iface.Name)
	if err != nil {
		return nil, err
	}

	populateMacAddress := func(vif *VIF, iface *v1.Interface) error {
		if iface.MacAddress != "" {
			macAddress, err := net.ParseMAC(iface.MacAddress)
			if err != nil {
				return err
			}
			vif.MAC = macAddress
		}
		return nil
	}

	if iface.Bridge != nil {
		vif := &VIF{Name: podInterfaceName}
		populateMacAddress(vif, iface)
		return &BridgePodInterface{iface: iface,
			vif:                 vif,
			domain:              domain,
			podInterfaceNum:     podInterfaceNum,
			podInterfaceName:    podInterfaceName,
			bridgeInterfaceName: fmt.Sprintf("k6t-%s", podInterfaceName)}, nil
	}
	if iface.Masquerade != nil {
		vif := &VIF{Name: podInterfaceName}
		populateMacAddress(vif, iface)
		return &MasqueradePodInterface{iface: iface,
			vif:                 vif,
			domain:              domain,
			podInterfaceNum:     podInterfaceNum,
			podInterfaceName:    podInterfaceName,
			vmNetworkCIDR:       network.Pod.VMNetworkCIDR,
			bridgeInterfaceName: fmt.Sprintf("k6t-%s", podInterfaceName)}, nil
	}
	if iface.Slirp != nil {
		return &SlirpPodInterface{iface: iface, domain: domain, podInterfaceNum: podInterfaceNum}, nil
	}
	return nil, fmt.Errorf("Not implemented")
}

type BridgePodInterface struct {
	vif                 *VIF
	iface               *v1.Interface
	podNicLink          netlink.Link
	domain              *api.Domain
	isLayer2            bool
	podInterfaceNum     int
	podInterfaceName    string
	bridgeInterfaceName string
}

func (b *BridgePodInterface) discoverPodNetworkInterface() error {
	link, err := Handler.LinkByName(b.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", b.podInterfaceName)
		return err
	}
	b.podNicLink = link

	// get IP address
	addrList, err := Handler.AddrList(b.podNicLink, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get an ip address for %s", b.podInterfaceName)
		return err
	}
	if len(addrList) == 0 {
		b.isLayer2 = true
	} else {
		b.vif.IP = addrList[0]
		b.isLayer2 = false
	}

	if len(b.vif.MAC) == 0 {
		// Get interface MAC address
		mac, err := Handler.GetMacDetails(b.podInterfaceName)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to get MAC for %s", b.podInterfaceName)
			return err
		}
		b.vif.MAC = mac
	}

	if b.podNicLink.Attrs().MTU < 0 || b.podNicLink.Attrs().MTU > 65535 {
		return fmt.Errorf("MTU value out of range ")
	}

	// Get interface MTU
	b.vif.Mtu = uint16(b.podNicLink.Attrs().MTU)

	if !b.isLayer2 {
		// Handle interface routes
		if err := b.setInterfaceRoutes(); err != nil {
			return err
		}
	}
	return nil
}

func (b *BridgePodInterface) preparePodNetworkInterfaces() error {
	// Set interface link to down to change its MAC address
	if err := Handler.LinkSetDown(b.podNicLink); err != nil {
		log.Log.Reason(err).Errorf("failed to bring link down for interface: %s", b.podInterfaceName)
		return err
	}

	if _, err := Handler.SetRandomMac(b.podInterfaceName); err != nil {
		return err
	}

	if err := Handler.LinkSetUp(b.podNicLink); err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.podInterfaceName)
		return err
	}

	if err := b.createBridge(); err != nil {
		return err
	}

	if !b.isLayer2 {
		// Remove IP from POD interface
		err := Handler.AddrDel(b.podNicLink, &b.vif.IP)

		if err != nil {
			log.Log.Reason(err).Errorf("failed to delete address for interface: %s", b.podInterfaceName)
			return err
		}

		b.startDHCPServer()
	}

	if err := Handler.LinkSetLearningOff(b.podNicLink); err != nil {
		log.Log.Reason(err).Errorf("failed to disable mac learning for interface: %s", b.podInterfaceName)
		return err
	}

	return nil
}

func (b *BridgePodInterface) startDHCPServer() {
	// Start DHCP Server
	fakeServerAddr, _ := netlink.ParseAddr(fmt.Sprintf(bridgeFakeIP, b.podInterfaceNum))
	Handler.StartDHCP(b.vif, fakeServerAddr, b.bridgeInterfaceName, b.iface.DHCPOptions)
}

func (b *BridgePodInterface) decorateConfig() error {
	b.domain.Spec.Devices.Interfaces[b.podInterfaceNum].MTU = &api.MTU{Size: strconv.Itoa(b.podNicLink.Attrs().MTU)}
	b.domain.Spec.Devices.Interfaces[b.podInterfaceNum].MAC = &api.MAC{MAC: b.vif.MAC.String()}

	return nil
}

func (b *BridgePodInterface) loadCachedInterface(name string) (bool, error) {
	var ifaceConfig api.Interface

	isExist, err := readFromCachedFile(name, interfaceCacheFile, &ifaceConfig)
	if err != nil {
		return false, err
	}

	if isExist {
		b.domain.Spec.Devices.Interfaces[b.podInterfaceNum] = ifaceConfig
		return true, nil
	}

	return false, nil
}

func (b *BridgePodInterface) setCachedInterface(name string) error {
	err := writeToCachedFile(&b.domain.Spec.Devices.Interfaces[b.podInterfaceNum], interfaceCacheFile, name)
	return err
}

func (b *BridgePodInterface) setInterfaceRoutes() error {
	routes, err := Handler.RouteList(b.podNicLink, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get routes for %s", b.podInterfaceName)
		return err
	}
	if len(routes) == 0 {
		return fmt.Errorf("No gateway address found in routes for %s", b.podInterfaceName)
	}
	b.vif.Gateway = routes[0].Gw
	if len(routes) > 1 {
		dhcpRoutes := filterPodNetworkRoutes(routes, b.vif)
		b.vif.Routes = &dhcpRoutes
	}
	return nil
}

func (b *BridgePodInterface) createBridge() error {
	// Create a bridge
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: b.bridgeInterfaceName,
		},
	}
	err := Handler.LinkAdd(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create a bridge")
		return err
	}

	err = Handler.LinkSetMaster(b.podNicLink, bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to connect interface %s to bridge %s", b.podInterfaceName, bridge.Name)
		return err
	}

	err = Handler.LinkSetUp(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.bridgeInterfaceName)
		return err
	}

	// set fake ip on a bridge
	fakeaddr, err := Handler.ParseAddr(fmt.Sprintf(bridgeFakeIP, b.podInterfaceNum))
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.bridgeInterfaceName)
		return err
	}

	if err := Handler.AddrAdd(bridge, fakeaddr); err != nil {
		log.Log.Reason(err).Errorf("failed to set bridge IP")
		return err
	}

	return nil
}

type MasqueradePodInterface struct {
	vif                 *VIF
	iface               *v1.Interface
	podNicLink          netlink.Link
	domain              *api.Domain
	podInterfaceNum     int
	podInterfaceName    string
	bridgeInterfaceName string
	vmNetworkCIDR       string
	gatewayAddr         *netlink.Addr
}

func (p *MasqueradePodInterface) discoverPodNetworkInterface() error {
	link, err := Handler.LinkByName(p.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", p.podInterfaceName)
		return err
	}
	p.podNicLink = link

	if p.podNicLink.Attrs().MTU < 0 || p.podNicLink.Attrs().MTU > 65535 {
		return fmt.Errorf("MTU value out of range ")
	}

	// Get interface MTU
	p.vif.Mtu = uint16(p.podNicLink.Attrs().MTU)

	if p.vmNetworkCIDR == "" {
		p.vmNetworkCIDR = api.DefaultVMCIDR
	}

	defaultGateway, vm, err := Handler.GetHostAndGwAddressesFromCIDR(p.vmNetworkCIDR)
	if err != nil {
		log.Log.Errorf("failed to get gw and vm available addresses from CIDR %s", p.vmNetworkCIDR)
		return err
	}

	gatewayAddr, err := Handler.ParseAddr(defaultGateway)
	if err != nil {
		return fmt.Errorf("failed to parse gateway ip address %s", defaultGateway)
	}
	p.vif.Gateway = gatewayAddr.IP.To4()
	p.gatewayAddr = gatewayAddr

	vmAddr, err := Handler.ParseAddr(vm)
	if err != nil {
		return fmt.Errorf("failed to parse vm ip address %s", vm)
	}
	p.vif.IP = *vmAddr

	return nil
}

func (p *MasqueradePodInterface) preparePodNetworkInterfaces() error {
	// Create an master bridge interface
	bridgeNicName := fmt.Sprintf("%s-nic", p.bridgeInterfaceName)
	bridgeNic := &netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{
			Name: bridgeNicName,
		},
	}
	err := Handler.LinkAdd(bridgeNic)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create a interface: %s", bridgeNic.Name)
		return err
	}

	if p.iface.MacAddress == "" {
		p.vif.MAC, err = Handler.GenerateRandomMac()
		if err != nil {
			log.Log.Reason(err).Errorf("failed to generate random mac address")
			return err
		}
	}

	err = Handler.LinkSetUp(bridgeNic)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", bridgeNic.Name)
		return err
	}

	if err := p.createBridge(); err != nil {
		return err
	}

	err = p.createNatRules()
	if err != nil {
		log.Log.Errorf("failed to create nat rules for vm error: %v", err)
		return err
	}

	p.startDHCPServer()

	return nil
}

func (p *MasqueradePodInterface) startDHCPServer() {
	// Start DHCP Server
	Handler.StartDHCP(p.vif, p.gatewayAddr, p.bridgeInterfaceName, p.iface.DHCPOptions)
}

func (p *MasqueradePodInterface) decorateConfig() error {
	p.domain.Spec.Devices.Interfaces[p.podInterfaceNum].MTU = &api.MTU{Size: strconv.Itoa(p.podNicLink.Attrs().MTU)}
	p.domain.Spec.Devices.Interfaces[p.podInterfaceNum].MAC = &api.MAC{MAC: p.vif.MAC.String()}

	return nil
}

func (p *MasqueradePodInterface) loadCachedInterface(name string) (bool, error) {
	var ifaceConfig api.Interface

	isExist, err := readFromCachedFile(name, interfaceCacheFile, &ifaceConfig)
	if err != nil {
		return false, err
	}

	if isExist {
		p.domain.Spec.Devices.Interfaces[p.podInterfaceNum] = ifaceConfig
		return true, nil
	}

	return false, nil
}

func (p *MasqueradePodInterface) setCachedInterface(name string) error {
	err := writeToCachedFile(&p.domain.Spec.Devices.Interfaces[p.podInterfaceNum], interfaceCacheFile, name)
	return err
}

func (p *MasqueradePodInterface) createBridge() error {
	// Get dummy link
	bridgeNicName := fmt.Sprintf("%s-nic", p.bridgeInterfaceName)
	bridgeNicLink, err := Handler.LinkByName(bridgeNicName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to find dummy interface for bridge")
	}

	// Create a bridge
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: p.bridgeInterfaceName,
		},
	}
	err = Handler.LinkAdd(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create a bridge")
		return err
	}

	err = Handler.LinkSetMaster(bridgeNicLink, bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to connect %s interface to bridge %s", bridgeNicName, p.bridgeInterfaceName)
		return err
	}

	err = Handler.LinkSetUp(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", p.bridgeInterfaceName)
		return err
	}

	if err := Handler.AddrAdd(bridge, p.gatewayAddr); err != nil {
		log.Log.Reason(err).Errorf("failed to set bridge IP")
		return err
	}

	return nil
}

func (p *MasqueradePodInterface) createNatRules() error {

	err := Handler.IptablesNewChain("nat", "KUBEVIRT_PREINBOUND")
	if err != nil {
		return err
	}

	err = Handler.IptablesNewChain("nat", "KUBEVIRT_POSTINBOUND")
	if err != nil {
		return err
	}

	err = Handler.IptablesAppendRule("nat", "POSTROUTING", "-s", p.vif.IP.IP.String(), "-j", "MASQUERADE")
	if err != nil {
		return err
	}

	err = Handler.IptablesAppendRule("nat", "PREROUTING", "-i", p.podInterfaceName, "-j", "KUBEVIRT_PREINBOUND")
	if err != nil {
		return err
	}

	err = Handler.IptablesAppendRule("nat", "POSTROUTING", "-o", p.bridgeInterfaceName, "-j", "KUBEVIRT_POSTINBOUND")
	if err != nil {
		return err
	}

	for _, port := range p.iface.Ports {
		if port.Protocol == "" {
			port.Protocol = "tcp"
		}

		err = Handler.IptablesAppendRule("nat", "KUBEVIRT_POSTINBOUND",
			"-p",
			strings.ToLower(port.Protocol),
			"--dport",
			strconv.Itoa(int(port.Port)),
			"-j",
			"SNAT",
			"--to-source", p.gatewayAddr.IP.String())
		if err != nil {
			return err
		}

		err = Handler.IptablesAppendRule("nat", "KUBEVIRT_PREINBOUND",
			"-p",
			strings.ToLower(port.Protocol),
			"--dport",
			strconv.Itoa(int(port.Port)),
			"-j",
			"DNAT",
			"--to-destination", p.vif.IP.IP.String())
		if err != nil {
			return err
		}

		err = Handler.IptablesAppendRule("nat", "OUTPUT",
			"-p",
			strings.ToLower(port.Protocol),
			"--dport",
			strconv.Itoa(int(port.Port)),
			"--destination", "127.0.0.1",
			"-j",
			"DNAT",
			"--to-destination", p.vif.IP.IP.String())
		if err != nil {
			return err
		}
	}
	return err
}

type SlirpPodInterface struct {
	iface           *v1.Interface
	domain          *api.Domain
	podInterfaceNum int
}

func (s *SlirpPodInterface) discoverPodNetworkInterface() error {
	s.domain.Spec.QEMUCmd.QEMUArg = append(s.domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: "-device"})
	return nil
}

func (s *SlirpPodInterface) preparePodNetworkInterfaces() error {
	interfaces := s.domain.Spec.Devices.Interfaces
	domainInterface := interfaces[s.podInterfaceNum]
	s.domain.Spec.QEMUCmd.QEMUArg = append(s.domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: fmt.Sprintf("%s,netdev=%s", domainInterface.Model.Type, s.iface.Name)})

	s.domain.Spec.Devices.Interfaces = append(interfaces[:s.podInterfaceNum], interfaces[s.podInterfaceNum+1:]...)
	s.podInterfaceNum = len(s.domain.Spec.QEMUCmd.QEMUArg) - 1

	return nil
}

func (s *SlirpPodInterface) decorateConfig() error {
	s.domain.Spec.QEMUCmd.QEMUArg[s.podInterfaceNum].Value += fmt.Sprintf(",id=%s", s.iface.Name)
	if s.iface.MacAddress != "" {
		// We assume address was already validated in API layer so just pass it to libvirt as-is.
		s.domain.Spec.QEMUCmd.QEMUArg[s.podInterfaceNum].Value += fmt.Sprintf(",mac=%s", s.iface.MacAddress)
	}
	return nil
}

func (s *SlirpPodInterface) loadCachedInterface(name string) (bool, error) {
	var qemuArg api.Arg
	interfaces := s.domain.Spec.Devices.Interfaces

	isExist, err := readFromCachedFile(name, qemuArgCacheFile, &qemuArg)
	if err != nil {
		return false, err
	}

	if isExist {
		// remove slirp interface from domain spec devices interfaces
		interfaces = append(interfaces[:s.podInterfaceNum], interfaces[s.podInterfaceNum+1:]...)

		// Add interface configuration to qemuArgs
		s.domain.Spec.QEMUCmd.QEMUArg = append(s.domain.Spec.QEMUCmd.QEMUArg, qemuArg)
		return true, nil
	}

	return false, nil
}

func (s *SlirpPodInterface) setCachedInterface(name string) error {
	err := writeToCachedFile(&s.domain.Spec.QEMUCmd.QEMUArg[s.podInterfaceNum], qemuArgCacheFile, name)
	return err
}
