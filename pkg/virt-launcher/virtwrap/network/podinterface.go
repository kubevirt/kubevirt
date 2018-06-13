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
	"strings"

	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var bridgeFakeIP = "169.254.75.86/32"

// DefaultProtocol is the default port protocol
const DefaultProtocol string = "TCP"

// DefaultVMCIDR is the default CIRD for vm network
const DefaultVMCIDR = "10.0.2.0/24"

type BindMechanism interface {
	setup() error
}

type PodInterface struct{}

func (l *PodInterface) Unplug() {}

// Plug connect a Pod network device to the virtual machine
func (l *PodInterface) Plug(iface *v1.Interface, network *v1.Network, domain *api.Domain) error {
	precond.MustNotBeNil(domain)
	initHandler()

	driver, err := getBinding(iface, network, domain)
	if err != nil {
		return err
	}

	err = driver.setup()
	if err != nil {
		return err
	}

	return nil
}

func getBinding(iface *v1.Interface, network *v1.Network, domain *api.Domain) (BindMechanism, error) {
	if iface.Bridge != nil {
		vif := &VIF{Name: podInterface}
		return &BridgePodInterface{iface: iface, vif: vif, domain: domain}, nil
	}
	if iface.Slirp != nil {
		SlirpConfig := api.Arg{Value: fmt.Sprintf("user,id=%s", iface.Name)}
		return &SlirpPodInterface{iface: iface, network: network, domain: domain, SlirpConfig: SlirpConfig}, nil
	}
	return nil, fmt.Errorf("Not implemented")
}

type BridgePodInterface struct {
	vif        *VIF
	iface      *v1.Interface
	podNicLink netlink.Link
	domain     *api.Domain
}

func (b *BridgePodInterface) setup() error {

	interfaces := b.domain.Spec.Devices.Interfaces

	// There should always be a pre-configured interface for the default pod interface.
	if len(interfaces) == 0 {
		return fmt.Errorf("failed to find a default interface configuration")
	}
	defaultIconf := interfaces[0]

	ifconf, err := getCachedInterface()
	if err != nil {
		return err
	}

	if ifconf == nil {
		err := b.discoverPodNetworkInterface()
		if err != nil {
			return err
		}
		if err := b.preparePodNetworkInterfaces(); err != nil {
			log.Log.Reason(err).Critical("failed to prepared pod networking")
			panic(err)
		}

		b.startDHCPServer()

		// After the network is configured, cache the result
		// in case this function is called again.
		b.decorateInterfaceConfig(&defaultIconf)
		err = setCachedInterface(&defaultIconf)
		if err != nil {
			panic(err)
		}
		ifconf = &defaultIconf
	}

	// TODO:(vladikr) Currently we support only one interface per vm.
	interfaces[0] = *ifconf

	return nil
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

	// Get interface MAC address
	mac, err := Handler.GetMacDetails(podInterface)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get MAC for %s", podInterface)
		return err
	}
	b.vif.MAC = mac

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

	return nil
}

func (b *BridgePodInterface) startDHCPServer() {
	// Start DHCP Server
	fakeServerAddr, _ := netlink.ParseAddr(bridgeFakeIP)
	Handler.StartDHCP(b.vif, fakeServerAddr)
}

func (b *BridgePodInterface) decorateInterfaceConfig(ifconf *api.Interface) {
	ifconf.MAC = &api.MAC{MAC: b.vif.MAC.String()}
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
	iface       *v1.Interface
	network     *v1.Network
	domain      *api.Domain
	SlirpConfig api.Arg
}

func (s *SlirpPodInterface) setup() error {
	err := s.configVMCIDR()
	if err != nil {
		return err
	}

	err = s.configDNSSearchName()
	if err != nil {
		return err
	}

	err = s.configPortForward()
	if err != nil {
		return err
	}

	err = s.commitConfiguration()
	if err != nil {
		return err
	}

	return nil
}

func (p *SlirpPodInterface) configPortForward() error {
	if p.iface.Slirp.Ports == nil {
		return nil
	}

	portForwardMap := make(map[int32]string)

	for _, forwardPort := range p.iface.Slirp.Ports {
		protocol := DefaultProtocol

		if forwardPort.Port == 0 {
			return fmt.Errorf("Port must be configure")
		}

		// Check protocol, its case sensitive like kubernetes
		if forwardPort.Protocol != "" {
			if forwardPort.Protocol != "TCP" && forwardPort.Protocol != "UDP" {
				return fmt.Errorf("Unknow protocol only TCP or UDP allowed")
			} else {
				protocol = forwardPort.Protocol
			}
		}
		//Check for duplicate pod port allocation
		if portProtocol, ok := portForwardMap[forwardPort.Port]; ok && portProtocol == protocol {
			return fmt.Errorf("Duplicated pod port allocation")
		}

		// Check if PodPort is configure If not Get the same Port as the vm port
		if forwardPort.PodPort == 0 {
			forwardPort.PodPort = forwardPort.Port
		}

		portForwardMap[forwardPort.Port] = protocol
		p.SlirpConfig.Value += fmt.Sprintf(",hostfwd=%s::%d-:%d", strings.ToLower(string(protocol)), forwardPort.PodPort, forwardPort.Port)

	}

	return nil
}

func (p *SlirpPodInterface) configVMCIDR() error {
	if p.network.Pod == nil {
		return fmt.Errorf("Slirp works only with pod network")
	}

	vmNetworkCIDR := ""
	if p.network.Pod.VMNetworkCIDR != "" {
		_, _, err := net.ParseCIDR(p.network.Pod.VMNetworkCIDR)
		if err != nil {
			return fmt.Errorf("Failed parsing CIDR")
		}
		vmNetworkCIDR = p.network.Pod.VMNetworkCIDR
	} else {
		vmNetworkCIDR = DefaultVMCIDR
	}

	// Insert configuration to qemu commandline
	p.SlirpConfig.Value += fmt.Sprintf(",net=%s", vmNetworkCIDR)

	return nil
}

func (p *SlirpPodInterface) configDNSSearchName() error {
	// remove the search string from the output and convert to string
	_, dnsSearchNames, err := getResolvConfDetailsFromPod()
	if err != nil {
		return err
	}

	// Insert configuration to qemu commandline
	for _, dnsSearchName := range dnsSearchNames {
		p.SlirpConfig.Value += fmt.Sprintf(",dnssearch=%s", dnsSearchName)
	}

	return nil
}

func (p *SlirpPodInterface) commitConfiguration() error {
	p.domain.Spec.QEMUCmd.QEMUArg = append(p.domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: "-netdev"})
	p.domain.Spec.QEMUCmd.QEMUArg = append(p.domain.Spec.QEMUCmd.QEMUArg, p.SlirpConfig)

	return nil
}
