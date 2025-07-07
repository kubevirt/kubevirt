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

	vishnetlink "github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/network/driver/ethtool"
	"kubevirt.io/kubevirt/pkg/network/driver/netlink"
	"kubevirt.io/kubevirt/pkg/network/driver/procsys"
	"kubevirt.io/kubevirt/pkg/network/driver/virtchroot"
)

type Spec struct {
	Interfaces []Interface `json:"interfaces,omitempty"`
	LinuxStack LinuxStack  `json:"linux-stack,omitempty"`
}

type Status struct {
	Interfaces []Interface `json:"interfaces"`
	Routes     Routes      `json:"routes"`
	LinuxStack LinuxStack  `json:"linux-stack,omitempty"`
}

type Interface struct {
	Name        string  `json:"name"`
	Index       int     `json:"index,omitempty"`
	TypeName    string  `json:"type,omitempty"`
	State       string  `json:"state,omitempty"`
	MacAddress  string  `json:"mac-address,omitempty"`
	CopyMacFrom string  `json:"copy-mac-from,omitempty"`
	MTU         int     `json:"mtu,omitempty"`
	Controller  string  `json:"controller,omitempty"`
	Ethtool     Ethtool `json:"ethtool,omitempty"`

	Tap *TapDevice `json:"tap,omitempty"`

	IPv4 IP `json:"IPv4,omitempty"`
	IPv6 IP `json:"IPv6,omitempty"`

	LinuxStack LinuxIfaceStack `json:"linux-stack,omitempty"`

	Metadata *IfaceMetadata
}

type IP struct {
	Enabled *bool       `json:"enabled,omitempty"`
	Address []IPAddress `json:"address,omitempty"`
}

type IPAddress struct {
	IP        string `json:"IP,omitempty"`
	PrefixLen int    `json:"prefix-length,omitempty"`
}

type Routes struct {
	Running []Route `json:"running,omitempty"`
}

type Route struct {
	Destination      string `json:"destination"`
	NextHopInterface string `json:"next-hop-interface,omitempty"`
	NextHopAddress   string `json:"next-hop-address,omitempty"`
	TableID          int    `json:"table-id,omitempty"`
}

type Ethtool struct {
	Feature Feature `json:"feature,omitempty"`
}

type Feature struct {
	TxChecksum *bool `json:"tx-checksum,omitempty"`
}

type TapDevice struct {
	Queues int `json:"queues,omitempty"`
	UID    int `json:"UID,omitempty"`
	GID    int `json:"GID,omitempty"`
}

type LinuxIfaceStack struct {
	IP4RouteLocalNet *bool `json:"ip4-route-local-net,omitempty"`
	PortLearning     *bool `json:"port-learning,omitempty"`
}

type LinuxStack struct {
	IPv4 LinuxStackIP4 `json:"ipv4,omitempty"`
	IPv6 LinuxStackIP6 `json:"ipv6,omitempty"`
}

type LinuxStackIP4 struct {
	ArpIgnore             *procsys.ArpReplyMode `json:"arp-ignore,omitempty"`
	Forwarding            *bool                 `json:"forwarding,omitempty"`
	PingGroupRange        []int                 `json:"ping-group-range,omitempty"`
	UnprivilegedPortStart *int                  `json:"unprivileged-port-start,omitempty"`
}

type LinuxStackIP6 struct {
	Forwarding *bool `json:"forwarding,omitempty"`
}

// IfaceMetadata includes extra data which is piggyback on the nmstate object.
// Users of the nmstate object can use it to store data and use it in some scenarios (e.g. creating the tap device).
type IfaceMetadata struct {
	// Pid refers to the process ID of the virt-launcher.
	Pid int
	// NetworkName refers to the logical network interface name which is associated with the interface spec.
	NetworkName string
}

type adapter interface {
	LinkList() ([]vishnetlink.Link, error)
	LinkByIndex(int) (vishnetlink.Link, error)
	LinkByName(string) (vishnetlink.Link, error)
	LinkAdd(vishnetlink.Link) error
	LinkDel(vishnetlink.Link) error
	LinkSetUp(link vishnetlink.Link) error
	LinkSetDown(link vishnetlink.Link) error
	LinkSetHardwareAddr(vishnetlink.Link, net.HardwareAddr) error
	LinkSetMTU(vishnetlink.Link, int) error
	LinkSetMaster(vishnetlink.Link, *vishnetlink.Bridge) error
	LinkSetName(vishnetlink.Link, string) error
	LinkSetLearningOff(vishnetlink.Link) error
	ReadTXChecksum(string) (bool, error)
	TXChecksumOff(name string) error

	AddrList(vishnetlink.Link, int) ([]vishnetlink.Addr, error)
	AddrAdd(vishnetlink.Link, *vishnetlink.Addr) error
	AddrDel(vishnetlink.Link, *vishnetlink.Addr) error
	ParseAddr(string) (*vishnetlink.Addr, error)
	RouteList(vishnetlink.Link, int) ([]vishnetlink.Route, error)

	IPv4GetForwarding() (bool, error)
	IPv4EnableForwarding() error
	IPv6GetForwarding() (bool, error)
	IPv6EnableForwarding() error
	IPv4GetPingGroupRange() (int, int, error)
	IPv4SetPingGroupRange(int, int) error
	IPv4GetUnprivilegedPortStart() (int, error)
	IPv4SetUnprivilegedPortStart(int) error

	IPv4GetArpIgnore(string) (procsys.ArpReplyMode, error)
	IPv4SetArpIgnore(string, procsys.ArpReplyMode) error
	IPv4GetRouteLocalNet(string) (bool, error)
	IPv4EnableRouteLocalNet(string) error
	LinkGetProtinfo(vishnetlink.Link) (vishnetlink.Protinfo, error)

	AddTapDeviceWithSELinuxLabel(name string, mtu int, queueCount int, ownerID int, pid int) error
}

type NMState struct {
	adapter adapter
}

type option func(state *NMState)

func New(opts ...option) NMState {
	n := NMState{adapter: defaultHandler{}}
	for _, opt := range opts {
		opt(&n)
	}
	return n
}

func WithAdapter(adapter adapter) option {
	return func(n *NMState) {
		n.adapter = adapter
	}
}

type defaultHandler struct {
	netlink.NetLink
	ethtool.Ethtool
	procsys.ProcSys
	virtchroot.VirtCHRoot
}

const (
	TypeVETH   = "veth"
	TypeBridge = "bridge"
	TypeDummy  = "dummy"
	TypeTap    = "tap"
)

const (
	IfaceStateUnknown = "unknown"
	IfaceStateUp      = "up"
	IfaceStateDown    = "down"
	IfaceStateAbsent  = "absent"
)

func AnyInterface(ifaces []Interface, predicate func(Interface) bool) bool {
	return LookupInterface(ifaces, predicate) != nil
}

func LookupInterface(ifaces []Interface, predicate func(Interface) bool) *Interface {
	for i, iface := range ifaces {
		if predicate(iface) {
			return &ifaces[i]
		}
	}
	return nil
}

func normalizeLinkTypeName(link vishnetlink.Link) string {
	typeName := link.Type()
	if typeName == "tuntap" {
		tuntapLink := link.(*vishnetlink.Tuntap)
		if tuntapLink.Mode == vishnetlink.TUNTAP_MODE_TAP {
			typeName = TypeTap
		}
	}
	return typeName
}
