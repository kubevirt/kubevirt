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

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network/dhcp"

	lmf "github.com/subgraph/libmacouflage"
)

const (
	podInterface     = "eth0"
	macVlanIfaceName = "macvlan0"
	macVlanFakeIP    = "10.11.12.13/24"
	guestDNS         = "8.8.8.8"
)

type VIF struct {
	Name    string
	IP      netlink.Addr
	MAC     net.HardwareAddr
	Gateway net.IP
}

type NetworkHandler interface {
	LinkByName(name string) (netlink.Link, error)
	AddrList(link netlink.Link, family int) ([]netlink.Addr, error)
	RouteList(link netlink.Link, family int) ([]netlink.Route, error)
	AddrDel(link netlink.Link, addr *netlink.Addr) error
	AddrAdd(link netlink.Link, addr *netlink.Addr) error
	LinkSetDown(link netlink.Link) error
	LinkSetUp(link netlink.Link) error
	LinkAdd(link netlink.Link) error
	ParseAddr(s string) (*netlink.Addr, error)
	ChangeMacAddr(iface string) (net.HardwareAddr, error)
	GetMacDetails(iface string) (net.HardwareAddr, error)
	StartDHCP(nic *VIF, serverAddr *netlink.Addr)
}

type NetworkUtilsHandler struct{}

var Handler NetworkHandler

func (h *NetworkUtilsHandler) LinkByName(name string) (netlink.Link, error) {
	return netlink.LinkByName(name)
}
func (h *NetworkUtilsHandler) AddrList(link netlink.Link, family int) ([]netlink.Addr, error) {
	return netlink.AddrList(link, family)
}
func (h *NetworkUtilsHandler) RouteList(link netlink.Link, family int) ([]netlink.Route, error) {
	return netlink.RouteList(link, family)
}
func (h *NetworkUtilsHandler) AddrDel(link netlink.Link, addr *netlink.Addr) error {
	return netlink.AddrDel(link, addr)
}
func (h *NetworkUtilsHandler) LinkSetDown(link netlink.Link) error {
	return netlink.LinkSetDown(link)
}
func (h *NetworkUtilsHandler) LinkSetUp(link netlink.Link) error {
	return netlink.LinkSetUp(link)
}
func (h *NetworkUtilsHandler) LinkAdd(link netlink.Link) error {
	return netlink.LinkAdd(link)
}
func (h *NetworkUtilsHandler) ParseAddr(s string) (*netlink.Addr, error) {
	return netlink.ParseAddr(s)
}
func (h *NetworkUtilsHandler) AddrAdd(link netlink.Link, addr *netlink.Addr) error {
	return netlink.AddrAdd(link, addr)
}

// GetMacDetails from an interface
func (h *NetworkUtilsHandler) GetMacDetails(iface string) (net.HardwareAddr, error) {
	currentMac, err := lmf.GetCurrentMac(iface)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get mac information for interface: %s", iface)
		return nil, err
	}
	return currentMac, nil
}

// ChangeMacAddr changes the MAC address for a agiven interface
func (h *NetworkUtilsHandler) ChangeMacAddr(iface string) (net.HardwareAddr, error) {
	var mac net.HardwareAddr

	currentMac, err := Handler.GetMacDetails(iface)
	if err != nil {
		return nil, err
	}

	changed, err := lmf.SpoofMacRandom(iface, false)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to spoof MAC for iface: %s", iface)
		return nil, err
	}

	if changed {
		mac, err = Handler.GetMacDetails(iface)
		if err != nil {
			return nil, err
		}
		log.Log.Reason(err).Errorf("Updated Mac for iface: %s - %s", iface, mac)
	}
	return currentMac, nil
}

func (h *NetworkUtilsHandler) StartDHCP(nic *VIF, serverAddr *netlink.Addr) {
	// Start DHCP
	go func() {
		dhcp.SingleClientDHCPServer(
			nic.MAC,
			nic.IP.IP,
			nic.IP.Mask,
			macVlanIfaceName,
			serverAddr.IP,
			nic.Gateway,
			net.ParseIP(guestDNS),
		)
	}()
}

// Allow mocking for tests
var SetupPodNetwork = SetupDefaultPodNetwork

func initHandler() {
	if Handler == nil {
		Handler = &NetworkUtilsHandler{}
	}
}

// SetupDefaultPodNetwork will prepare the pod management network to be used by a virtual machine
// which will own the pod network IP and MAC. Pods MAC address will be changed to a
// random address and IP will be deleted. This will also create a macvlan device with a fake IP.
// DHCP server will be started and bounded to the macvlan interface to server the original pod ip
// to the guest OS
func SetupDefaultPodNetwork(domain *api.Domain) error {
	precond.MustNotBeNil(domain)
	initHandler()

	nic := &VIF{Name: podInterface}

	nicLink, err := Handler.LinkByName(podInterface)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", podInterface)
		return err
	}

	// get IP address
	addrList, err := Handler.AddrList(nicLink, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get an ip address for %s", podInterface)
		return err
	}
	if len(addrList) == 0 {
		return fmt.Errorf("No IP address found on %s", podInterface)
	}
	nic.IP = addrList[0]

	// Get interface gateway
	routes, err := Handler.RouteList(nicLink, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get routes for %s", podInterface)
		return err
	}
	if len(routes) == 0 {
		return fmt.Errorf("No gateway address found in routes for %s", podInterface)
	}
	nic.Gateway = routes[0].Gw

	// Get interface MAC address
	mac, err := Handler.GetMacDetails(podInterface)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get MAC for %s", podInterface)
		return err
	}
	nic.MAC = mac

	// Remove IP from POD interface
	err = Handler.AddrDel(nicLink, &nic.IP)

	if err != nil {
		log.Log.Reason(err).Errorf("failed to delete link for interface: %s", podInterface)
		return err
	}

	// Set interface link to down to change its MAC address
	err = Handler.LinkSetDown(nicLink)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link down for interface: %s", podInterface)
		return err
	}

	_, err = Handler.ChangeMacAddr(podInterface)
	if err != nil {
		return err
	}

	err = Handler.LinkSetUp(nicLink)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", podInterface)
		return err
	}

	// Create a macvlan link
	macvlan := &netlink.Macvlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:        macVlanIfaceName,
			ParentIndex: nicLink.Attrs().Index,
		},
		Mode: netlink.MACVLAN_MODE_BRIDGE,
	}

	//Create macvlan interface
	if err := Handler.LinkAdd(macvlan); err != nil {
		log.Log.Reason(err).Errorf("failed to create macvlan interface")
		return err
	}

	//get macvlan link
	macvlink, err := Handler.LinkByName(macVlanIfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", macVlanIfaceName)
		return err
	}
	err = Handler.LinkSetUp(macvlink)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", macVlanIfaceName)
		return err
	}

	// set fake ip on macvlan interface
	fakeaddr, err := Handler.ParseAddr(macVlanFakeIP)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", macVlanIfaceName)
		return err
	}

	if err := Handler.AddrAdd(macvlink, fakeaddr); err != nil {
		log.Log.Reason(err).Errorf("failed to set macvlan IP")
		return err
	}

	// Start DHCP Server
	Handler.StartDHCP(nic, fakeaddr)

	if err := plugNetworkDevice(domain, nic); err != nil {
		return err
	}

	return nil
}

func plugNetworkDevice(domain *api.Domain, vif *VIF) error {

	// get VIF config
	ifconf, err := decorateInterfaceConfig(vif)
	if err != nil {
		log.Log.Reason(err).Error("failed to get VIF config.")
		return err
	}

	//TODO:(vladikr) Currently we support only one interface per vm. Improve this once we'll start supporting more.
	if len(domain.Spec.Devices.Interfaces) == 0 {
		domain.Spec.Devices.Interfaces = append(domain.Spec.Devices.Interfaces, *ifconf)
	}
	for idx, _ := range domain.Spec.Devices.Interfaces {
		domain.Spec.Devices.Interfaces[idx] = *ifconf
	}

	return nil
}

func decorateInterfaceConfig(vif *VIF) (*api.Interface, error) {

	inter := api.Interface{}
	inter.Type = "direct"
	inter.TrustGuestRxFilters = "yes"
	inter.Source = api.InterfaceSource{Device: vif.Name, Mode: "bridge"}
	inter.MAC = &api.MAC{MAC: vif.MAC.String()}
	inter.Model = &api.Model{Type: "virtio"}

	return &inter, nil
}
