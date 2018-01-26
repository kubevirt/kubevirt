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

	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/log"
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

// SetupDefaultPodNetwork will prepare the pod management network to be used by a virtual machine
// which will own the pod network IP and MAC. Pods MAC address will be changed to a
// random address and IP will be deleted. This will also create a macvlan device with a fake IP.
// DHCP server will be started and bounded to the macvlan interface to server the original pod ip
// to the guest OS
func SetupDefaultPodNetwork() (net.HardwareAddr, error) {
	// Get IP and MAC
	// Change eth0 MAC
	// Create macvlan and set fake address
	// remove eth0 IP
	// Start DHCP

	nic := &VIF{Name: podInterface}
	link, err := netlink.LinkByName(podInterface)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", podInterface)
		return nil, err
	}

	// get IP address
	addrList, err := netlink.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get an ip address for %s", podInterface)
		return nil, err
	}
	if len(addrList) == 0 {
		return nil, fmt.Errorf("No IP address found on %s", podInterface)
	}
	nic.IP = addrList[0]

	// Get interface gateway
	routes, err := netlink.RouteList(link, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get routes for %s", podInterface)
		return nil, err
	}
	if len(routes) == 0 {
		return nil, fmt.Errorf("No gateway address found in routes for %s", podInterface)
	}
	nic.Gateway = routes[0].Gw

	// Get interface MAC address
	mac, err := GetMacDetails(podInterface)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get MAC for %s", podInterface)
		return nil, err
	}
	nic.MAC = mac

	// Remove IP from POD interface
	err = netlink.AddrDel(link, &nic.IP)

	if err != nil {
		log.Log.Reason(err).Errorf("failed to delete link for interface: %s", podInterface)
		return nil, err
	}

	// Set interface link to down to change its MAC address
	err = netlink.LinkSetDown(link)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link down for interface: %s", podInterface)
		return nil, err
	}

	_, err = ChangeMacAddr(podInterface)
	if err != nil {
		return nil, err
	}

	err = netlink.LinkSetUp(link)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", podInterface)
		return nil, err
	}

	// Create a macvlan link
	macvlan := &netlink.Macvlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:        macVlanIfaceName,
			ParentIndex: link.Attrs().Index,
		},
		Mode: netlink.MACVLAN_MODE_BRIDGE,
	}

	//Create macvlan interface
	if err := netlink.LinkAdd(macvlan); err != nil {
		log.Log.Reason(err).Errorf("failed to create macvlan interface")
		return nil, err
	}

	//get macvlan link
	macvlink, err := netlink.LinkByName(macVlanIfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", macVlanIfaceName)
		return nil, err
	}
	err = netlink.LinkSetUp(macvlink)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", macVlanIfaceName)
		return nil, err
	}

	// set fake ip on macvlan interface
	fakeaddr, err := netlink.ParseAddr(macVlanFakeIP)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", macVlanIfaceName)
		return nil, err
	}

	if err := netlink.AddrAdd(macvlink, fakeaddr); err != nil {
		log.Log.Reason(err).Errorf("failed to set macvlan IP")
		return nil, err
	}

	// Start DHCP
	go dhcp.SingleClientDHCPServer(
		nic.MAC,
		nic.IP.IP,
		nic.IP.Mask,
		macVlanIfaceName,
		fakeaddr.IP,
		nic.Gateway,
		net.ParseIP(guestDNS),
	)

	return nic.MAC, nil
}

// GetMacDetails from an interface
func GetMacDetails(iface string) (net.HardwareAddr, error) {
	currentMac, err := lmf.GetCurrentMac(iface)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get mac information for interface: %s", iface)
		return nil, err
	}
	return currentMac, nil
}

// ChangeMacAddr changes the MAC address for a agiven interface
func ChangeMacAddr(iface string) (net.HardwareAddr, error) {
	var mac net.HardwareAddr

	currentMac, err := GetMacDetails(iface)
	if err != nil {
		return nil, err
	}

	changed, err := lmf.SpoofMacRandom(iface, false)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to spoof MAC for iface: %s", iface)
		return nil, err
	}

	if changed {
		mac, err = GetMacDetails(iface)
		if err != nil {
			return nil, err
		}
		log.Log.Reason(err).Errorf("Updated Mac for iface: %s - %s", iface, mac)
	}
	return currentMac, nil
}
