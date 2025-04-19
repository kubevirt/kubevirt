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
package server

import (
	"context"
	"net"
	"syscall"

	dhcpConn "github.com/krolaw/dhcp4/conn"

	"golang.org/x/net/ipv4"
)

// Creates listener on all interfaces and then filters packets not received by interfaceName
func NewUDP4FilterListener(interfaceName, laddr string) (c ServeIfConn, e error) {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, err
	}
	lc := CreateListenConfig()
	l, err := lc.ListenPacket(context.Background(), "udp4", laddr)

	if err != nil {
		return nil, err
	}
	defer func() {
		if e != nil {
			closeDHCPServerIgnoringError(l)
		}
	}()
	p := ipv4.NewPacketConn(l)
	if err := p.SetControlMessage(ipv4.FlagInterface, true); err != nil {
		return nil, err
	}
	return dhcpConn.NewServeIf(iface.Index, p), nil
}

func CreateListenConfig() net.ListenConfig {
	return net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var opErr error
			err := c.Control(func(fd uintptr) {
				opErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
			})
			if err != nil {
				return err
			}
			return opErr
		},
	}
}

type ServeIfConn interface {
	ReadFrom(b []byte) (n int, addr net.Addr, err error)
	WriteTo(b []byte, addr net.Addr) (n int, err error)
	Close() error
}
