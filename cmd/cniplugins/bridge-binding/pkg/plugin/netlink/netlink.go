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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package netlink

import (
	"fmt"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"net"
	"syscall"
)

const (
	tapOwnerUID = 107
	tapOwnerGID
)

const (
	primaryPodInterfaceName = "eth0"
	bridgeInterfaceName     = "k6t-eth0"
	bridgeIpAddress         = "169.254.75.10"
	dummyInterfaceName      = "dummy0"
	tapInterfaceName        = "tap0"
)

type NetLink struct {
	PodNs     ns.NetNS
	PodLink   netlink.Link
	BrLink    *netlink.Bridge
	DummyLink *netlink.Dummy
	TapLink   *netlink.Tuntap
}

func New(netns ns.NetNS) *NetLink {
	return &NetLink{
		PodNs: netns,
	}
}

func (m *NetLink) DummyInterface() *current.Interface {
	dummy := &current.Interface{
		Name:    m.DummyLink.Attrs().Name,
		Mac:     m.DummyLink.Attrs().HardwareAddr.String(),
		Sandbox: m.PodNs.Path(),
	}
	return dummy
}

func (m *NetLink) BridgeInterface() *current.Interface {
	bridge := &current.Interface{
		Name:    m.BrLink.Attrs().Name,
		Mac:     m.BrLink.Attrs().HardwareAddr.String(),
		Sandbox: m.PodNs.Path(),
	}
	return bridge
}

func (m *NetLink) TapInterface() *current.Interface {
	tap := &current.Interface{
		Name:    m.TapLink.Attrs().Name,
		Mac:     m.TapLink.Attrs().HardwareAddr.String(),
		Sandbox: m.PodNs.Path(),
	}

	return tap
}

func (m *NetLink) ReadLink(name string) (netlink.Link, error) {
	return netlink.LinkByName(name)
}

func (m *NetLink) ConfigurePodNetworks() error {
	var err error

	if m.PodLink, err = m.ReadLink(primaryPodInterfaceName); err != nil {
		return err
	}

	addrs, err := netlink.AddrList(m.PodLink, netlink.FAMILY_V4)

	for _, a := range addrs {
		if err = m.deleteAddr(m.PodLink, a.IPNet); err != nil {
			return err
		}
		if err = m.ensureAddr(m.DummyLink, netlink.FAMILY_V4, a.IPNet); err != nil {
			return err
		}
	}

	// add link-local address to bridge interface
	addr := &netlink.Addr{IPNet: &net.IPNet{IP: net.ParseIP(bridgeIpAddress), Mask: net.CIDRMask(32, 32)}}
	if err = netlink.AddrAdd(m.BrLink, addr); err != nil {
		return err
	}

	// swap names between PrimaryPodInterface and DummyInterface
	extNicName := fmt.Sprintf("%s-nic", primaryPodInterfaceName)

	if err = m.renameNic(primaryPodInterfaceName, extNicName); err != nil {
		return err
	}
	if err = m.renameNic(dummyInterfaceName, primaryPodInterfaceName); err != nil {
		return err
	}

	// connect primaryPodInterface and tapInterface over the bridge
	m.PodLink, err = m.ReadLink(extNicName)

	if err = netlink.LinkSetMaster(m.PodLink, m.BrLink); err != nil {
		return fmt.Errorf("failed to connect %q to bridge %v: %v", m.PodLink.Attrs().Name, m.BrLink.Attrs().Name, err)
	}
	if err = netlink.LinkSetMaster(m.TapLink, m.BrLink); err != nil {
		return fmt.Errorf("failed to connect %q to bridge %v: %v", m.TapLink.Attrs().Name, m.BrLink.Attrs().Name, err)
	}

	return err
}

func (m *NetLink) EnsureDummyLink() error {
	m.DummyLink = &netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{
			Name:  dummyInterfaceName,
			Flags: unix.IFF_NOARP | unix.IFF_BROADCAST,
		},
	}
	if err := netlink.LinkAdd(m.DummyLink); err != nil {
		return fmt.Errorf("failed to create dummy interface: %v", err)
	}

	// bring the dummy down, as we use it only for keeping the origin pod's IP
	if err := netlink.LinkSetDown(m.DummyLink); err != nil {
		return fmt.Errorf("could not set dummy interface down: %v", err)
	}

	return nil
}

func (m *NetLink) EnsureBridgeLink() error {
	m.BrLink = &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: bridgeInterfaceName,
			MTU:  1480,
			// Let kernel use default txqueuelen; leaving it unset
			// means 0, and a zero-length TX queue messes up FIFO
			// traffic shapers which use TX queue length as the
			// default packet limit
			TxQLen: -1,
		},
	}

	if err := netlink.LinkAdd(m.BrLink); err != nil && err != syscall.EEXIST {
		return fmt.Errorf("failed to create bridge: %v", err)
	}

	if err := netlink.LinkSetUp(m.BrLink); err != nil {
		return fmt.Errorf("failed to set bridge interface up: %v", err)
	}

	return nil
}

func (m *NetLink) EnsureTapLink() error {
	m.TapLink = &netlink.Tuntap{
		LinkAttrs: netlink.LinkAttrs{
			Name:      tapInterfaceName,
			Namespace: netlink.NsFd(int(m.PodNs.Fd())),
			MTU:       1480,
		},
		NonPersist: false,
		Queues:     1,
		Mode:       unix.IFF_TAP,
		Flags:      netlink.TUNTAP_DEFAULTS,
		Owner:      uint32(tapOwnerGID),
		Group:      uint32(tapOwnerUID),
	}

	if err := netlink.LinkAdd(m.TapLink); err != nil {
		return fmt.Errorf("failed to create tap: %v", err)
	}

	if err := netlink.LinkSetUp(m.TapLink); err != nil {
		return fmt.Errorf("failed to set tap interface up: %v", err)
	}

	return nil
}

func (m *NetLink) renameNic(ifName string, newName string) error {
	link, err := netlink.LinkByName(ifName)
	if err != nil {
		return fmt.Errorf("could not find interface %q: %v", ifName, err)
	}
	if err = netlink.LinkSetDown(link); err != nil {
		return fmt.Errorf("could not set interface %q down: %v", ifName, link)
	}
	if err := netlink.LinkSetName(link, newName); err != nil {
		return fmt.Errorf("could not rename interface %q to %q: %v", ifName, newName, err)
	}
	if err = netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("could not set interface %q up: %v", ifName, link)
	}
	return nil
}

func (m *NetLink) deleteAddr(link netlink.Link, ipn *net.IPNet) error {
	addr := &netlink.Addr{IPNet: ipn, Label: ""}

	if err := netlink.AddrDel(link, addr); err != nil {
		return fmt.Errorf("could not remove IP address from %q: %v", link.Attrs().Name, err)
	}

	return nil
}

func (m *NetLink) ensureAddr(link netlink.Link, family int, ipn *net.IPNet) error {
	addrs, err := netlink.AddrList(link, family)
	if err != nil && err != syscall.ENOENT {
		return fmt.Errorf("could not get list of IP addresses: %v", err)
	}

	ipnStr := ipn.String()
	for _, a := range addrs {

		// string comp is actually easiest for doing IPNet comps
		if a.IPNet.String() == ipnStr {
			return nil
		}

		if family == netlink.FAMILY_V4 || a.IPNet.Contains(ipn.IP) || ipn.Contains(a.IPNet.IP) {
			return fmt.Errorf("%q already has an IP address different from %v", link.Attrs().Name, ipnStr)

		}
	}

	addr := &netlink.Addr{IPNet: ipn, Label: ""}
	if err := netlink.AddrAdd(link, addr); err != nil && err != syscall.EEXIST {
		return fmt.Errorf("could not add IP address to %q: %v", link.Attrs().Name, err)
	}

	return nil
}
