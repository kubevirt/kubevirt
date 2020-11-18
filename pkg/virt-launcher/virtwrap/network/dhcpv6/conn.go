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
 * Copyright 2020 Red Hat, Inc.
 *
 */
package dhcpv6

import (
	"net"
	"time"

	"golang.org/x/net/ipv6"
)

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
