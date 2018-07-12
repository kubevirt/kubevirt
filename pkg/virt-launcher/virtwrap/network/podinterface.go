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

	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var bridgeFakeIP = "169.254.75.86/32"

// DefaultProtocol is the default port protocol
const DefaultProtocol string = "TCP"

// DefaultVMCIDR is the default CIDR for vm network
const DefaultVMCIDR = "10.0.2.0/24"

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
func (l *PodInterface) Plug(iface *v1.Interface, network *v1.Network, domain *api.Domain) error {
	precond.MustNotBeNil(domain)
	initHandler()

	driver, err := getBinding(iface, domain)
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

func getBinding(iface *v1.Interface, domain *api.Domain) (BindMechanism, error) {
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
		vif := &VIF{Name: podInterface}
		populateMacAddress(vif, iface)
		return &BridgePodInterface{iface: iface, vif: vif, domain: domain, podInterfaceNum: podInterfaceNum}, nil
	}
	if iface.Slirp != nil {
		return &SlirpPodInterface{iface: iface, domain: domain, podInterfaceNum: podInterfaceNum}, nil
	}
	return nil, fmt.Errorf("Not implemented")
}

type BridgePodInterface struct {
	vif             *VIF
	iface           *v1.Interface
	podNicLink      netlink.Link
	domain          *api.Domain
	podInterfaceNum int
}

func (b *BridgePodInterface) discoverPodNetworkInterface() error {
	link, err := Handler.LinkByName(podInterface)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", podInterface)
		return err
	}
	b.podNicLink = link

	// get IP address
	addrList, err := Handler.AddrList(b.podNicLink, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get an ip address for %s", podInterface)
		return err
	}
	if len(addrList) == 0 {
		return fmt.Errorf("No IP address found on %s", podInterface)
	}
	b.vif.IP = addrList[0]

	// Handle interface routes
	if err := b.setInterfaceRoutes(); err != nil {
		return err
	}

	if len(b.vif.MAC) == 0 {
		// Get interface MAC address
		mac, err := Handler.GetMacDetails(podInterface)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to get MAC for %s", podInterface)
			return err
		}
		b.vif.MAC = mac
	}

	// Get interface MTU
	b.vif.Mtu = uint16(b.podNicLink.Attrs().MTU)
	return nil
}

func (b *BridgePodInterface) preparePodNetworkInterfaces() error {
	// Remove IP from POD interface
	err := Handler.AddrDel(b.podNicLink, &b.vif.IP)

	if err != nil {
		log.Log.Reason(err).Errorf("failed to delete link for interface: %s", podInterface)
		return err
	}

	// Set interface link to down to change its MAC address
	err = Handler.LinkSetDown(b.podNicLink)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link down for interface: %s", podInterface)
		return err
	}

	_, err = Handler.SetRandomMac(podInterface)
	if err != nil {
		return err
	}

	err = Handler.LinkSetUp(b.podNicLink)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", podInterface)
		return err
	}

	if err := b.createDefaultBridge(); err != nil {
		return err
	}

	b.startDHCPServer()

	return nil
}

func (b *BridgePodInterface) startDHCPServer() {
	// Start DHCP Server
	fakeServerAddr, _ := netlink.ParseAddr(bridgeFakeIP)
	Handler.StartDHCP(b.vif, fakeServerAddr)
}

func (b *BridgePodInterface) decorateConfig() error {
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
		log.Log.Reason(err).Errorf("failed to get routes for %s", podInterface)
		return err
	}
	if len(routes) == 0 {
		return fmt.Errorf("No gateway address found in routes for %s", podInterface)
	}
	b.vif.Gateway = routes[0].Gw
	if len(routes) > 1 {
		dhcpRoutes := filterPodNetworkRoutes(routes, b.vif)
		b.vif.Routes = &dhcpRoutes
	}
	return nil
}

func (b *BridgePodInterface) createDefaultBridge() error {
	// Create a bridge
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: api.DefaultBridgeName,
		},
	}
	err := Handler.LinkAdd(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create a bridge")
		return err
	}
	netlink.LinkSetMaster(b.podNicLink, bridge)

	err = Handler.LinkSetUp(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", api.DefaultBridgeName)
		return err
	}

	// set fake ip on a bridge
	fakeaddr, err := Handler.ParseAddr(bridgeFakeIP)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", api.DefaultBridgeName)
		return err
	}

	if err := Handler.AddrAdd(bridge, fakeaddr); err != nil {
		log.Log.Reason(err).Errorf("failed to set bridge IP")
		return err
	}

	return nil
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
