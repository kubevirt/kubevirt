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
 * Copyright The KubeVirt Authors.
 *
 */

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

package driver

import (
	"fmt"

	"github.com/vishvananda/netlink"

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
	StartDHCP(nic *cache.DHCPConfig, bridgeInterfaceName string, dhcpOptions *v1.DHCPOptions) error
	HasIPv4GlobalUnicastAddress(interfaceName string) (bool, error)
	HasIPv6GlobalUnicastAddress(interfaceName string) (bool, error)
}

type NetworkUtilsHandler struct{}

func (h *NetworkUtilsHandler) LinkByName(name string) (netlink.Link, error) {
	return netlink.LinkByName(name)
}
func (h *NetworkUtilsHandler) HasIPv4GlobalUnicastAddress(interfaceName string) (bool, error) {
	link, err := h.LinkByName(interfaceName)
	if err != nil {
		return false, err
	}
	addrList, err := netlink.AddrList(link, netlink.FAMILY_V4)
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
	addrList, err := netlink.AddrList(link, netlink.FAMILY_V6)
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

func (h *NetworkUtilsHandler) StartDHCP(nic *cache.DHCPConfig, bridgeInterfaceName string, dhcpOptions *v1.DHCPOptions) error {
	log.Log.V(4).Infof("StartDHCP network Nic: %+v", nic)
	nameservers, searchDomains, err := dns.GetResolvConfDetailsFromPod()
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
				nameservers.IPv4,
				nic.Routes,
				searchDomains,
				nic.Mtu,
				dhcpOptions,
			); err != nil {
				log.Log.Errorf("failed to run DHCP Server: %v", err)
				panic(err)
			}
		}()
	}

	if nic.IPv6.IPNet != nil {
		go func() {
			if err = DHCPv6Server(
				nic.IPv6.IP,
				bridgeInterfaceName,
				nameservers.IPv6,
			); err != nil {
				log.Log.Reason(err).Error("failed to run DHCPv6 Server")
				panic(err)
			}
		}()
	}

	return nil
}

// Allow mocking for tests
var DHCPServer = dhcpserver.SingleClientDHCPServer
var DHCPv6Server = dhcpserverv6.SingleClientDHCPv6Server
