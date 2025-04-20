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
package serverv6

import (
	"fmt"
	"net"
	"time"

	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/dhcpv6/server6"

	"golang.org/x/net/ipv6"
)

const errFmt = "%s: %v"

type FilteredConn struct {
	ifIndex    int
	packetConn *ipv6.PacketConn
	cm         *ipv6.ControlMessage
}

func (fc *FilteredConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	for { // Filter all other interfaces
		n, fc.cm, addr, err = fc.packetConn.ReadFrom(b)
		if err != nil || fc.cm == nil || fc.cm.IfIndex == fc.ifIndex {
			break
		}
	}
	return
}

func (fc *FilteredConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	fc.cm.Src = nil
	return fc.packetConn.WriteTo(b, fc.cm, addr)
}

func (fc *FilteredConn) Close() error {
	return fc.packetConn.Close()
}

func (fc *FilteredConn) LocalAddr() net.Addr {
	return fc.packetConn.LocalAddr()
}

func (fc *FilteredConn) SetDeadline(t time.Time) error {
	return fc.packetConn.SetDeadline(t)
}

func (fc *FilteredConn) SetReadDeadline(t time.Time) error {
	return fc.packetConn.SetReadDeadline(t)
}

func (fc *FilteredConn) SetWriteDeadline(t time.Time) error {
	return fc.packetConn.SetWriteDeadline(t)
}

func NewConnection(serverIface *net.Interface) (*FilteredConn, error) {
	const errorString = "Failed creating connection for dhcpv6 server"
	addr := &net.UDPAddr{
		IP:   net.IPv6unspecified,
		Port: dhcpv6.DefaultServerPort,
	}
	udpConn, err := server6.NewIPv6UDPConn("", addr)
	if err != nil {
		return nil, fmt.Errorf(errFmt, errorString, err)
	}

	packetConn := ipv6.NewPacketConn(udpConn)
	if err := packetConn.SetControlMessage(ipv6.FlagInterface, true); err != nil {
		return nil, fmt.Errorf(errFmt, errorString, err)
	}

	group := net.UDPAddr{
		IP:   dhcpv6.AllDHCPRelayAgentsAndServers,
		Port: dhcpv6.DefaultServerPort}
	if err := packetConn.JoinGroup(serverIface, &group); err != nil {
		return nil, fmt.Errorf(errFmt, errorString, err)
	}

	return &FilteredConn{packetConn: packetConn, ifIndex: serverIface.Index}, nil
}
