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
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"

	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network/dhcp"

	lmf "github.com/subgraph/libmacouflage"
)

const (
	podInterface  = "eth0"
	guestDNS      = "8.8.8.8"
	DNSConfigFile = "/etc/resolv.conf"
)

var interfaceCacheFile = "/var/run/kubevirt-private/interface-cache.json"
var bridgeFakeIP = "10.11.12.13/24"

// only used by unit test suite
func setInterfaceCacheFile(path string) {
	interfaceCacheFile = path
}

type VIF struct {
	Name    string
	IP      netlink.Addr
	MAC     net.HardwareAddr
	Gateway net.IP
	Routes  *[]netlink.Route
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
	ReadDNSConfig(config string) []byte
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
	// panic in case the DHCP server failed during the vm creation
	// but ignore dhcp errors when the vm is destroyed or shutting down
	if err := DHCPServer(
		nic.MAC,
		nic.IP.IP,
		nic.IP.Mask,
		api.DefaultBridgeName,
		serverAddr.IP,
		nic.Gateway,
		net.ParseIP(guestDNS),
		nic.Routes,
	); err != nil {
		log.Log.Errorf("failed to run DHCP: %v", err)
		panic(err)
	}
}

// ReadDNSConfig will return all NS servers from resolv.conf or a default NS
func (h *NetworkUtilsHandler) ReadDNSConfig(config string) []byte {
	var servers []byte
	var nameserver string

	file, err := os.Open(config)
	if err != nil {
		log.Log.Warning("failed to open DNS config file")
		return net.ParseIP(guestDNS).To4()
	}
	reader := bufio.NewReader(file)
	defer file.Close()
	for line, _, err := reader.ReadLine(); err == nil; line, _, err = reader.ReadLine() {
		line := string(line)
		if strings.HasPrefix(line, "nameserver") {
			field := strings.SplitAfter(line, " ")
			nameserver = field[1]
		}
		if len(servers) < 12 {
			if ip := net.ParseIP(nameserver); ip != nil {
				servers = append(servers, ip.To4()...)
			}
		}
	}
	if len(servers) == 0 {
		servers = net.ParseIP(guestDNS).To4()
	}
	return servers
}

// Allow mocking for tests
var SetupPodNetwork = SetupDefaultPodNetwork
var DHCPServer = dhcp.SingleClientDHCPServer

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

	// There should alway be a pre-configured interface for the default pod interface.
	defaultIconf := domain.Spec.Devices.Interfaces[0]

	ifconf, err := getCachedInterface()
	if err != nil {
		return err
	}

	if ifconf == nil {
		vif := &VIF{Name: podInterface}
		podNicLink, err := discoverPodNetworkInterface(vif)
		if err != nil {
			return err
		}

		if err := preparePodNetworkInterfaces(vif, podNicLink); err != nil {
			log.Log.Reason(err).Critical("failed to prepared pod networking")
			panic(err)
		}

		// Start DHCP Server
		fakeServerAddr, _ := netlink.ParseAddr(bridgeFakeIP)
		go Handler.StartDHCP(vif, fakeServerAddr)

		// After the network is configured, cache the result
		// in case this function is called again.
		decorateInterfaceConfig(vif, &defaultIconf)
		err = setCachedInterface(&defaultIconf)
		if err != nil {
			panic(err)
		}
	}

	// TODO:(vladikr) Currently we support only one interface per vm.
	// Improve this once we'll start supporting more.
	if len(domain.Spec.Devices.Interfaces) == 0 {
		domain.Spec.Devices.Interfaces = append(domain.Spec.Devices.Interfaces, defaultIconf)
	} else {
		domain.Spec.Devices.Interfaces[0] = defaultIconf
	}

	return nil
}

func setCachedInterface(ifconf *api.Interface) error {
	buf, err := json.MarshalIndent(&ifconf, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling interface cache: %v", err)
	}
	err = ioutil.WriteFile(interfaceCacheFile, buf, 0644)
	if err != nil {
		return fmt.Errorf("error writing interface cache %v", err)
	}
	return nil
}

func getCachedInterface() (*api.Interface, error) {
	buf, err := ioutil.ReadFile(interfaceCacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	ifconf := api.Interface{}
	err = json.Unmarshal(buf, &ifconf)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling interface: %v", err)
	}
	return &ifconf, nil
}

func discoverPodNetworkInterface(nic *VIF) (netlink.Link, error) {
	nicLink, err := Handler.LinkByName(podInterface)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", podInterface)
		return nil, err
	}

	// get IP address
	addrList, err := Handler.AddrList(nicLink, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get an ip address for %s", podInterface)
		return nil, err
	}
	if len(addrList) == 0 {
		return nil, fmt.Errorf("No IP address found on %s", podInterface)
	}
	nic.IP = addrList[0]

	// Get interface gateway
	routes, err := Handler.RouteList(nicLink, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get routes for %s", podInterface)
		return nil, err
	}
	if len(routes) == 0 {
		return nil, fmt.Errorf("No gateway address found in routes for %s", podInterface)
	}
	nic.Gateway = routes[0].Gw
	var dhcpRoutes []netlink.Route
	if len(routes) > 1 {
		// Filter out irrelevant routes
		for _, route := range routes[1:] {
			if !route.Src.Equal(nic.IP.IP) {
				dhcpRoutes = append(dhcpRoutes, route)
			}
		}
		nic.Routes = &dhcpRoutes
	}

	// Get interface MAC address
	mac, err := Handler.GetMacDetails(podInterface)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get MAC for %s", podInterface)
		return nil, err
	}
	nic.MAC = mac
	return nicLink, nil
}

func preparePodNetworkInterfaces(nic *VIF, nicLink netlink.Link) error {
	// Remove IP from POD interface
	err := Handler.AddrDel(nicLink, &nic.IP)

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

	// Create a bridge
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: api.DefaultBridgeName,
		},
	}
	err = Handler.LinkAdd(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create a bridge")
		return err
	}
	netlink.LinkSetMaster(nicLink, bridge)

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
		log.Log.Reason(err).Errorf("failed to set macvlan IP")
		return err
	}

	return nil
}

func decorateInterfaceConfig(vif *VIF, ifconf *api.Interface) {

	ifconf.MAC = &api.MAC{MAC: vif.MAC.String()}
}
