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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package nmstate

import (
	"net"
	"strings"

	"kubevirt.io/kubevirt/pkg/pointer"

	vishnetlink "github.com/vishvananda/netlink"
)

func (n NMState) Read() (*Status, error) {
	links, err := n.adapter.LinkList()
	if err != nil {
		return nil, err
	}

	var status Status

	if status.Interfaces, err = n.readInterfaces(links); err != nil {
		return nil, err
	}

	if status.Routes.Running, err = n.readRoutes(); err != nil {
		return nil, err
	}

	if status.LinuxStack, err = n.readLinuxStack(); err != nil {
		return nil, err
	}

	return &status, nil
}

func (n NMState) readInterfaces(links []vishnetlink.Link) ([]Interface, error) {
	var ifaces []Interface
	for _, link := range links {
		iface := Interface{
			Name:       link.Attrs().Name,
			Index:      link.Attrs().Index,
			TypeName:   normalizeLinkTypeName(link),
			State:      link.Attrs().OperState.String(),
			MacAddress: link.Attrs().HardwareAddr.String(),
			MTU:        link.Attrs().MTU,
			IPv4: IP{
				Enabled: pointer.P(false),
			},
			IPv6: IP{
				Enabled: pointer.P(false),
			},
		}
		if index := link.Attrs().MasterIndex; index > 0 {
			bridgeLink, err := n.adapter.LinkByIndex(index)
			if err != nil {
				return nil, err
			}
			iface.Controller = bridgeLink.Attrs().Name
		}

		et, err := n.readEthtool(iface.Name)
		if err != nil {
			return nil, err
		}
		iface.Ethtool = et

		addresses, err := n.readAddresses(link, vishnetlink.FAMILY_V4)
		if err != nil {
			return nil, err
		}
		if len(addresses) > 0 {
			iface.IPv4.Enabled = pointer.P(true)
			iface.IPv4.Address = addresses
		}

		addresses, err = n.readAddresses(link, vishnetlink.FAMILY_V6)
		if err != nil {
			return nil, err
		}
		if len(addresses) > 0 {
			iface.IPv6.Enabled = pointer.P(true)
			iface.IPv6.Address = addresses
		}

		linuxStack, err := n.readLinuxStackByLink(link)
		if err != nil {
			return nil, err
		}
		iface.LinuxStack = linuxStack

		ifaces = append(ifaces, iface)
	}

	return ifaces, nil
}

func (n NMState) readLinuxStackByLink(link vishnetlink.Link) (LinuxIfaceStack, error) {
	ip4RouteLocalNet, err := n.adapter.IPv4GetRouteLocalNet(link.Attrs().Name)
	if err != nil {
		return LinuxIfaceStack{}, err
	}

	// The generic link reader unfortunately is not populating the protinfo data,
	// therefore an explicit read for it is required.
	// On some links (e.g. lo), the read returns an error. Such an error is ignored.
	protInfo, err := n.adapter.LinkGetProtinfo(link)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return LinuxIfaceStack{}, err
	}

	return LinuxIfaceStack{
		IP4RouteLocalNet: &ip4RouteLocalNet,
		PortLearning:     &protInfo.Learning,
	}, nil
}

func (n NMState) readEthtool(name string) (Ethtool, error) {
	txChecksumEnabled, err := n.adapter.ReadTXChecksum(name)
	if err != nil {
		return Ethtool{}, err
	}
	return Ethtool{Feature: Feature{TxChecksum: &txChecksumEnabled}}, nil
}

func (n NMState) readAddresses(link vishnetlink.Link, family int) ([]IPAddress, error) {
	addresses, err := n.adapter.AddrList(link, family)
	if err != nil {
		return nil, err
	}

	var ipAddrs []IPAddress
	for _, address := range addresses {
		ip := address.IP.String()
		prefixLen, _ := address.Mask.Size()
		ipAddrs = append(ipAddrs, IPAddress{IP: ip, PrefixLen: prefixLen})
	}
	return ipAddrs, nil
}

func (n NMState) readRoutes() ([]Route, error) {
	routesState, err := n.readRoutesFamily(vishnetlink.FAMILY_V4)
	if err != nil {
		return nil, err
	}

	routesState6, err := n.readRoutesFamily(vishnetlink.FAMILY_V6)
	if err != nil {
		return nil, err
	}

	routesState = append(routesState, routesState6...)
	return routesState, nil
}

func (n NMState) readRoutesFamily(family int) ([]Route, error) {
	routes, err := n.adapter.RouteList(nil, family)
	if err != nil {
		return nil, err
	}

	defaultDst := DefaultDestinationRoute(family)

	var routesState []Route
	for _, route := range routes {
		link, err := n.adapter.LinkByIndex(route.LinkIndex)
		if err != nil {
			return nil, err
		}

		if route.Dst == nil {
			route.Dst = defaultDst
		}

		var nextHopAddress string
		if route.Gw != nil {
			nextHopAddress = route.Gw.String()
		}

		routesState = append(routesState, Route{
			Destination:      route.Dst.String(),
			NextHopInterface: link.Attrs().Name,
			NextHopAddress:   nextHopAddress,
			TableID:          route.Table,
		})
	}
	return routesState, nil
}

func DefaultDestinationRoute(family int) *net.IPNet {
	if family == vishnetlink.FAMILY_V4 {
		return &net.IPNet{IP: net.IPv4zero, Mask: net.IPv4Mask(0, 0, 0, 0)}
	}
	if family == vishnetlink.FAMILY_V6 {
		return &net.IPNet{IP: net.IPv6zero, Mask: net.CIDRMask(0, 8*net.IPv6len)}
	}
	return nil
}

func (n NMState) readLinuxStack() (LinuxStack, error) {
	arpIgnore, err := n.adapter.IPv4GetArpIgnore("all")
	if err != nil {
		return LinuxStack{}, err
	}
	ip4Forwarding, err := n.adapter.IPv4GetForwarding()
	if err != nil {
		return LinuxStack{}, err
	}
	pgrFrom, pgrTo, err := n.adapter.IPv4GetPingGroupRange()
	if err != nil {
		return LinuxStack{}, err
	}
	unprvPortStart, err := n.adapter.IPv4GetUnprivilegedPortStart()
	if err != nil {
		return LinuxStack{}, err
	}
	ip6Forwarding, err := n.adapter.IPv6GetForwarding()
	if err != nil {
		return LinuxStack{}, err
	}

	return LinuxStack{
		IPv4: LinuxStackIP4{
			ArpIgnore:             &arpIgnore,
			Forwarding:            &ip4Forwarding,
			PingGroupRange:        []int{pgrFrom, pgrTo},
			UnprivilegedPortStart: &unprvPortStart,
		},
		IPv6: LinuxStackIP6{
			Forwarding: &ip6Forwarding,
		},
	}, nil
}
