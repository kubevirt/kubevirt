/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
