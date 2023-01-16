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

package driver

import (
	"fmt"
	"os"

	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"

	netutils "k8s.io/utils/net"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/cache"
	dhcpserver "kubevirt.io/kubevirt/pkg/network/dhcp/server"
	dhcpserverv6 "kubevirt.io/kubevirt/pkg/network/dhcp/serverv6"
	"kubevirt.io/kubevirt/pkg/network/dns"
)

const (
	LibvirtUserAndGroupId = "0"
)

type IPVersion int

const (
	IPv4 IPVersion = 4
	IPv6 IPVersion = 6
)

type NetworkHandler interface {
	LinkByName(name string) (netlink.Link, error)
	AddrList(link netlink.Link, family int) ([]netlink.Addr, error)
	ReadIPAddressesFromLink(interfaceName string) (string, string, error)
	RouteList(link netlink.Link, family int) ([]netlink.Route, error)
	LinkDel(link netlink.Link) error
	ParseAddr(s string) (*netlink.Addr, error)
	StartDHCP(nic *cache.DHCPConfig, bridgeInterfaceName string, dhcpOptions *v1.DHCPOptions) error
	HasIPv4GlobalUnicastAddress(interfaceName string) (bool, error)
	HasIPv6GlobalUnicastAddress(interfaceName string) (bool, error)
	IsIpv4Primary() (bool, error)
}

type NetworkUtilsHandler struct{}

func (h *NetworkUtilsHandler) LinkByName(name string) (netlink.Link, error) {
	return netlink.LinkByName(name)
}
func (h *NetworkUtilsHandler) AddrList(link netlink.Link, family int) ([]netlink.Addr, error) {
	return netlink.AddrList(link, family)
}
func (h *NetworkUtilsHandler) RouteList(link netlink.Link, family int) ([]netlink.Route, error) {
	return netlink.RouteList(link, family)
}
func (h *NetworkUtilsHandler) LinkDel(link netlink.Link) error {
	return netlink.LinkDel(link)
}
func (h *NetworkUtilsHandler) ParseAddr(s string) (*netlink.Addr, error) {
	return netlink.ParseAddr(s)
}

func (h *NetworkUtilsHandler) HasIPv4GlobalUnicastAddress(interfaceName string) (bool, error) {
	link, err := h.LinkByName(interfaceName)
	if err != nil {
		return false, err
	}
	addrList, err := h.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		return false, err
	}

	for _, addr := range addrList {
		if addr.IP.IsGlobalUnicast() {
			return true, nil
		}
	}
	return false, nil
}

func (h *NetworkUtilsHandler) HasIPv6GlobalUnicastAddress(interfaceName string) (bool, error) {
	link, err := h.LinkByName(interfaceName)
	if err != nil {
		return false, err
	}
	addrList, err := h.AddrList(link, netlink.FAMILY_V6)
	if err != nil {
		return false, err
	}

	for _, addr := range addrList {
		if addr.IP.IsGlobalUnicast() {
			return true, nil
		}
	}
	return false, nil
}

func (h *NetworkUtilsHandler) IsIpv4Primary() (bool, error) {
	podIP, exist := os.LookupEnv("MY_POD_IP")
	if !exist {
		return false, fmt.Errorf("MY_POD_IP doesnt exists")
	}

	return !netutils.IsIPv6String(podIP), nil
}

func (h *NetworkUtilsHandler) ReadIPAddressesFromLink(interfaceName string) (string, string, error) {
	link, err := h.LinkByName(interfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", interfaceName)
		return "", "", err
	}

	// get IP address
	addrList, err := h.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a address for interface: %s", interfaceName)
		return "", "", err
	}

	// no ip assigned. ipam disabled
	if len(addrList) == 0 {
		return "", "", nil
	}

	var ipv4, ipv6 string
	for _, addr := range addrList {
		if addr.IP.IsGlobalUnicast() {
			if netutils.IsIPv6(addr.IP) && ipv6 == "" {
				ipv6 = addr.IP.String()
			} else if !netutils.IsIPv6(addr.IP) && ipv4 == "" {
				ipv4 = addr.IP.String()
			}
		}
	}

	return ipv4, ipv6, nil
}

func (h *NetworkUtilsHandler) StartDHCP(nic *cache.DHCPConfig, bridgeInterfaceName string, dhcpOptions *v1.DHCPOptions) error {
	log.Log.V(4).Infof("StartDHCP network Nic: %+v", nic)
	nameservers, searchDomains, err := converter.GetResolvConfDetailsFromPod()
	if err != nil {
		return fmt.Errorf("Failed to get DNS servers from resolv.conf: %v", err)
	}

	domain := dns.DomainNameWithSubdomain(searchDomains, nic.Subdomain)
	if domain != "" {
		searchDomains = append([]string{domain}, searchDomains...)
	}

	if nic.IP.IPNet != nil {
		// panic in case the DHCP server failed during the vm creation
		// but ignore dhcp errors when the vm is destroyed or shutting down
		go func() {
			if err = DHCPServer(
				nic.MAC,
				nic.IP.IP,
				nic.IP.Mask,
				bridgeInterfaceName,
				nic.AdvertisingIPAddr,
				nic.Gateway,
				nameservers,
				nic.Routes,
				searchDomains,
				nic.Mtu,
				dhcpOptions,
			); err != nil {
				log.Log.Errorf("failed to run DHCP: %v", err)
				panic(err)
			}
		}()
	}

	if nic.IPv6.IPNet != nil {
		go func() {
			if err = DHCPv6Server(
				nic.IPv6.IP,
				bridgeInterfaceName,
			); err != nil {
				log.Log.Reason(err).Error("failed to run DHCPv6")
				panic(err)
			}
		}()
	}

	return nil
}

// Allow mocking for tests
var DHCPServer = dhcpserver.SingleClientDHCPServer
var DHCPv6Server = dhcpserverv6.SingleClientDHCPv6Server
