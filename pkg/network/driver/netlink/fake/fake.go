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

package fake

import (
	"net"

	vishnetlink "github.com/vishvananda/netlink"
)

type NetLink struct {
	links                  []vishnetlink.Link
	ip4AddressesByLinkName map[string][]vishnetlink.Addr
	ip6AddressesByLinkName map[string][]vishnetlink.Addr
	routes4                []vishnetlink.Route
	routes6                []vishnetlink.Route
}

func New() *NetLink {
	return &NetLink{
		ip4AddressesByLinkName: map[string][]vishnetlink.Addr{},
		ip6AddressesByLinkName: map[string][]vishnetlink.Addr{},
	}
}

func (n *NetLink) LinkList() ([]vishnetlink.Link, error) {
	return n.links, nil
}

func (n *NetLink) LinkByIndex(index int) (vishnetlink.Link, error) {
	link := n.lookupLinkByIndex(index)
	if link == nil {
		return nil, vishnetlink.LinkNotFoundError{}
	}
	return link, nil
}

func (n *NetLink) LinkByName(name string) (vishnetlink.Link, error) {
	link := n.lookupLinkByName(name)
	if link == nil {
		return nil, vishnetlink.LinkNotFoundError{}
	}
	return link, nil
}

func (n *NetLink) LinkAdd(link vishnetlink.Link) error {
	// Indexes should start from 1.
	link.Attrs().Index = len(n.links) + 1
	n.links = append(n.links, link)
	n.ip4AddressesByLinkName[link.Attrs().Name] = nil
	n.ip6AddressesByLinkName[link.Attrs().Name] = nil
	return nil
}

func (n *NetLink) LinkDel(link vishnetlink.Link) error {
	var links []vishnetlink.Link
	for _, l := range n.links {
		if l.Attrs().Name != link.Attrs().Name {
			links = append(links, l)
		}
	}
	if len(n.links) == len(links) {
		return vishnetlink.LinkNotFoundError{}
	}
	n.links = links
	return nil
}

func (n *NetLink) LinkSetUp(link vishnetlink.Link) error {
	l := n.lookupLinkByName(link.Attrs().Name)
	if l == nil {
		return vishnetlink.LinkNotFoundError{}
	}
	l.Attrs().OperState = vishnetlink.OperUp
	return nil
}

func (n *NetLink) LinkSetDown(link vishnetlink.Link) error {
	l := n.lookupLinkByName(link.Attrs().Name)
	if l == nil {
		return vishnetlink.LinkNotFoundError{}
	}
	l.Attrs().OperState = vishnetlink.OperDown
	return nil
}

func (n *NetLink) LinkSetHardwareAddr(link vishnetlink.Link, hwAddress net.HardwareAddr) error {
	l := n.lookupLinkByName(link.Attrs().Name)
	if l == nil {
		return vishnetlink.LinkNotFoundError{}
	}
	l.Attrs().HardwareAddr = hwAddress
	return nil
}

func (n *NetLink) LinkSetMTU(link vishnetlink.Link, mtu int) error {
	l := n.lookupLinkByName(link.Attrs().Name)
	if l == nil {
		return vishnetlink.LinkNotFoundError{}
	}
	l.Attrs().MTU = mtu
	return nil
}

func (n *NetLink) LinkSetMaster(link vishnetlink.Link, bridge *vishnetlink.Bridge) error {
	l := n.lookupLinkByName(link.Attrs().Name)
	b := n.lookupLinkByName(bridge.LinkAttrs.Name)
	if l == nil || b == nil {
		return vishnetlink.LinkNotFoundError{}
	}
	l.Attrs().MasterIndex = b.Attrs().Index
	return nil
}

func (n *NetLink) LinkSetName(link vishnetlink.Link, name string) error {
	l := n.lookupLinkByName(link.Attrs().Name)
	l.Attrs().Name = name
	return nil
}

func (n *NetLink) LinkSetLearningOff(link vishnetlink.Link) error {
	l := n.lookupLinkByName(link.Attrs().Name)
	if l == nil {
		return vishnetlink.LinkNotFoundError{}
	}
	if l.Attrs().Protinfo == nil {
		l.Attrs().Protinfo = &vishnetlink.Protinfo{}
	}
	l.Attrs().Protinfo.Learning = false
	return nil
}

func (n *NetLink) LinkGetProtinfo(link vishnetlink.Link) (vishnetlink.Protinfo, error) {
	l := n.lookupLinkByName(link.Attrs().Name)
	if l == nil {
		return vishnetlink.Protinfo{}, vishnetlink.LinkNotFoundError{}
	}
	if protinfo := l.Attrs().Protinfo; protinfo != nil {
		return *protinfo, nil
	}
	return vishnetlink.Protinfo{}, nil
}

func (n *NetLink) AddrList(link vishnetlink.Link, family int) ([]vishnetlink.Addr, error) {
	linkName := link.Attrs().Name
	if l := n.lookupLinkByName(linkName); l == nil {
		return nil, vishnetlink.LinkNotFoundError{}
	}
	return n.ipAddresses(linkName, family), nil
}

func (n *NetLink) AddrAdd(link vishnetlink.Link, addr *vishnetlink.Addr) error {
	linkName := link.Attrs().Name
	if l := n.lookupLinkByName(linkName); l == nil {
		return vishnetlink.LinkNotFoundError{}
	}

	var ipAddressesByLinkName map[string][]vishnetlink.Addr
	switch ipFamily(addr.IP) {
	case vishnetlink.FAMILY_V4:
		ipAddressesByLinkName = n.ip4AddressesByLinkName
	case vishnetlink.FAMILY_V6:
		ipAddressesByLinkName = n.ip6AddressesByLinkName
	}
	ipAddressesByLinkName[linkName] = append(ipAddressesByLinkName[linkName], *addr)

	return nil
}

func (n *NetLink) AddrDel(link vishnetlink.Link, addr *vishnetlink.Addr) error {
	linkName := link.Attrs().Name
	if l := n.lookupLinkByName(linkName); l == nil {
		return vishnetlink.LinkNotFoundError{}
	}

	var ipAddressesByLinkName map[string][]vishnetlink.Addr
	switch ipFamily(addr.IP) {
	case vishnetlink.FAMILY_V4:
		ipAddressesByLinkName = n.ip4AddressesByLinkName
	case vishnetlink.FAMILY_V6:
		ipAddressesByLinkName = n.ip6AddressesByLinkName
	}

	var newAddresses []vishnetlink.Addr
	for _, address := range ipAddressesByLinkName[linkName] {
		if !address.IP.Equal(addr.IP) {
			newAddresses = append(newAddresses, address)
		}
	}
	ipAddressesByLinkName[linkName] = newAddresses
	return nil
}

func (n *NetLink) ParseAddr(s string) (*vishnetlink.Addr, error) {
	return vishnetlink.ParseAddr(s)
}

func (n *NetLink) RouteList(_ vishnetlink.Link, family int) ([]vishnetlink.Route, error) {
	switch family {
	case vishnetlink.FAMILY_V4:
		return n.routes4, nil
	case vishnetlink.FAMILY_V6:
		return n.routes6, nil
	}
	return nil, nil
}

func (n *NetLink) RouteAdd(route *vishnetlink.Route) error {
	switch ipFamily(route.Gw) {
	case vishnetlink.FAMILY_V4:
		n.routes4 = append(n.routes4, *route)
	case vishnetlink.FAMILY_V6:
		n.routes6 = append(n.routes6, *route)
	}
	return nil
}

func (n *NetLink) lookupLinkByName(name string) vishnetlink.Link {
	for i, l := range n.links {
		if l.Attrs().Name == name {
			return n.links[i]
		}
	}
	return nil
}

func (n *NetLink) lookupLinkByIndex(index int) vishnetlink.Link {
	for i, l := range n.links {
		if l.Attrs().Index == index {
			return n.links[i]
		}
	}
	return nil
}

func (n *NetLink) ipAddresses(linkName string, family int) []vishnetlink.Addr {
	switch family {
	case vishnetlink.FAMILY_V4:
		return n.ip4AddressesByLinkName[linkName]
	case vishnetlink.FAMILY_V6:
		return n.ip6AddressesByLinkName[linkName]
	case vishnetlink.FAMILY_ALL:
		ip4Addrs := n.ip4AddressesByLinkName[linkName]
		ip6Addrs := n.ip6AddressesByLinkName[linkName]
		ipAddresses := append([]vishnetlink.Addr{}, ip4Addrs...)
		ipAddresses = append(ipAddresses, ip6Addrs...)
		return ipAddresses
	}
	panic("illegal family")
}

func ipFamily(ip net.IP) int {
	if len(ip) <= net.IPv4len {
		return vishnetlink.FAMILY_V4
	}
	if ip.To4() != nil {
		return vishnetlink.FAMILY_V4
	}
	return vishnetlink.FAMILY_V6
}
