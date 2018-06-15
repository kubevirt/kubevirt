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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network/dhcp"

	lmf "github.com/subgraph/libmacouflage"
)

type VIF struct {
	Name    string
	IP      netlink.Addr
	MAC     net.HardwareAddr
	Gateway net.IP
	Routes  *[]netlink.Route
	Mtu     uint16
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
	SetRandomMac(iface string) (net.HardwareAddr, error)
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

// SetRandomMac changes the MAC address for a agiven interface to a randomly generated, preserving the vendor prefix
func (h *NetworkUtilsHandler) SetRandomMac(iface string) (net.HardwareAddr, error) {
	var mac net.HardwareAddr

	currentMac, err := Handler.GetMacDetails(iface)
	if err != nil {
		return nil, err
	}

	changed, err := lmf.SpoofMacSameVendor(iface, false)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to spoof MAC for an interface: %s", iface)
		return nil, err
	}

	if changed {
		mac, err = Handler.GetMacDetails(iface)
		if err != nil {
			return nil, err
		}
		log.Log.Reason(err).Errorf("updated MAC for interface: %s - %s", iface, mac)
	}
	return currentMac, nil
}

func (h *NetworkUtilsHandler) StartDHCP(nic *VIF, serverAddr *netlink.Addr) {
	nameservers, searchDomains, err := getResolvConfDetailsFromPod()
	if err != nil {
		log.Log.Errorf("Failed to get DNS servers from resolv.conf: %v", err)
		panic(err)
	}

	// panic in case the DHCP server failed during the vm creation
	// but ignore dhcp errors when the vm is destroyed or shutting down
	go func() {
		if err = DHCPServer(
			nic.MAC,
			nic.IP.IP,
			nic.IP.Mask,
			api.DefaultBridgeName,
			serverAddr.IP,
			nic.Gateway,
			nameservers,
			nic.Routes,
			searchDomains,
			nic.Mtu,
		); err != nil {
			log.Log.Errorf("failed to run DHCP: %v", err)
			panic(err)
		}
	}()
}

// Allow mocking for tests
var SetupPodNetwork = SetupNetworkInterfaces
var DHCPServer = dhcp.SingleClientDHCPServer

func initHandler() {
	if Handler == nil {
		Handler = &NetworkUtilsHandler{}
	}
}

func getInterfaceCacheFile(name string) string {
	return fmt.Sprintf(interfaceCacheFile, name)
}

func setCachedInterface(name string, ifconf *api.Interface) error {
	buf, err := json.MarshalIndent(&ifconf, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling interface cache: %v", err)
	}
	err = ioutil.WriteFile(getInterfaceCacheFile(name), buf, 0644)
	if err != nil {
		return fmt.Errorf("error writing interface cache %v", err)
	}
	return nil
}

func getCachedInterface(name string) (*api.Interface, error) {
	buf, err := ioutil.ReadFile(getInterfaceCacheFile(name))
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

// filter out irrelevant routes
func filterPodNetworkRoutes(routes []netlink.Route, nic *VIF) (filteredRoutes []netlink.Route) {
	for _, route := range routes {
		// don't create empty static routes
		if route.Dst == nil && route.Src.Equal(nil) && route.Gw.Equal(nil) {
			continue
		}

		// don't create static route for src == nic
		if route.Src != nil && route.Src.Equal(nic.IP.IP) {
			continue
		}

		filteredRoutes = append(filteredRoutes, route)
	}
	return
}

// only used by unit test suite
func setInterfaceCacheFile(path string) {
	interfaceCacheFile = path
}

// returns nameservers [][]byte, searchdomains []string, error
func getResolvConfDetailsFromPod() ([][]byte, []string, error) {
	b, err := ioutil.ReadFile(resolvConf)
	if err != nil {
		return nil, nil, err
	}

	nameservers, err := ParseNameservers(string(b))
	if err != nil {
		return nil, nil, err
	}

	searchDomains, err := ParseSearchDomains(string(b))
	if err != nil {
		return nil, nil, err
	}

	log.Log.Reason(err).Infof("Found nameservers in %s: %s", resolvConf, bytes.Join(nameservers, []byte{' '}))
	log.Log.Reason(err).Infof("Found search domains in %s: %s", resolvConf, strings.Join(searchDomains, " "))

	return nameservers, searchDomains, err
}

func ParseNameservers(content string) ([][]byte, error) {
	var nameservers [][]byte

	re, err := regexp.Compile("([0-9]{1,3}.?){4}")
	if err != nil {
		return nameservers, err
	}

	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, nameserverPrefix) {
			nameserver := re.FindString(line)
			if nameserver != "" {
				nameservers = append(nameservers, net.ParseIP(nameserver).To4())
			}
		}
	}

	if err = scanner.Err(); err != nil {
		return nameservers, err
	}

	// apply a default DNS if none found from pod
	if len(nameservers) == 0 {
		nameservers = append(nameservers, net.ParseIP(defaultDNS).To4())
	}

	return nameservers, nil
}

func ParseSearchDomains(content string) ([]string, error) {
	var searchDomains []string

	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, domainSearchPrefix) {
			doms := strings.Fields(strings.TrimPrefix(line, domainSearchPrefix))
			for _, dom := range doms {
				searchDomains = append(searchDomains, dom)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(searchDomains) == 0 {
		searchDomains = append(searchDomains, defaultSearchDomain)
	}

	return searchDomains, nil
}
