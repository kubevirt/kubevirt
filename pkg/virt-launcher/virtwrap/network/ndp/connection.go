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

package ndp

import (
	"fmt"
	"net"
	"os"

	"github.com/ftrvxmtrx/fd"
	"github.com/mdlayher/ndp"
	"golang.org/x/net/ipv6"
)

const (
	chkOff  = 2
	maxHops = 255
)

// A NDPConnection instruments a system.Conn and adds retry functionality for
// receiving / sending NDP messages on a given interface.
type NDPConnection struct {
	iface      *net.Interface
	rawConn    *net.IPConn
	conn       *ipv6.PacketConn
	controlMsg *ipv6.ControlMessage
}

// Return an NDPConnection bound to the chosen interface.
func NewNDPConnection(ifaceName string) (*NDPConnection, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("could not find interface %s: %v", ifaceName, err)
	}

	listenAddr := &net.IPAddr{
		IP:   net.IPv6unspecified,
		Zone: ifaceName,
	}
	icmpListener, err := net.ListenIP("ip6:ipv6-icmp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("could not listen to ip6:ipv6-icmp on addr %s: %v", listenAddr.String(), err)
	}

	ipv6Conn := ipv6.NewPacketConn(icmpListener)

	// Calculate and place ICMPv6 checksum at correct offset in all messages.
	if err := ipv6Conn.SetChecksum(true, chkOff); err != nil {
		return nil, fmt.Errorf("could not enable ICMPv6 checksum processing: %v", err)
	}

	routersMulticastGroup := &net.IPAddr{
		IP:   net.IPv6linklocalallrouters,
		Zone: ifaceName,
	}
	if err := ipv6Conn.JoinGroup(iface, routersMulticastGroup); err != nil {
		return nil, fmt.Errorf("failed to join %s multicast group: %v", routersMulticastGroup.String(), err)
	}

	listener := &NDPConnection{
		iface:      iface,
		conn:       ipv6Conn,
		rawConn:    icmpListener,
		controlMsg: getIPv6ControlMsg(),
	}

	if err := listener.Filter(ipv6.ICMPTypeRouterSolicitation); err != nil {
		return nil, fmt.Errorf("failed to set an ICMP filter for RouterSolicitations")
	}

	return listener, nil
}

func importNDPConnection(openedFD *os.File, iface *net.Interface) (*NDPConnection, error) {
	conn, err := net.FilePacketConn(openedFD)
	if err != nil {
		return nil, fmt.Errorf("could not get a PacketConnection from the bloody filer: %v", err)
	}
	ipv6Conn := ipv6.NewPacketConn(conn)
	ndpConn := &NDPConnection{
		iface:      iface,
		conn:       ipv6Conn,
		controlMsg: getIPv6ControlMsg(),
	}

	return ndpConn, nil
}

func getIPv6ControlMsg() *ipv6.ControlMessage {
	return &ipv6.ControlMessage{
		HopLimit: maxHops,
	}
}

func (l *NDPConnection) Export(socketPath string) error {
	socketListener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("could not create a UNIX domain socket: %v", err)
	}

	defer socketListener.Close()
	f, err := l.GetFD()
	if err != nil {
		return fmt.Errorf("could not get an opened file descriptor for out stupid ass server: %v", err)
	}

	socketTransferConnection, err := socketListener.Accept()
	if err != nil {
		return fmt.Errorf("could not listen on the UNIX domain socket: %v", err)
	}
	defer socketTransferConnection.Close()
	listenConn := socketTransferConnection.(*net.UnixConn)
	if err = fd.Put(listenConn, f); err != nil {
		return fmt.Errorf("could not send the opened file descriptor across: %v", err)
	}

	defer func() {
		_ = l.rawConn.Close()
		_ = l.conn.Close()
	}()
	return nil
}

func (l *NDPConnection) GetFD() (*os.File, error) {
	return l.rawConn.File()
}

func (l *NDPConnection) ReadFrom() (ndp.Message, error) {
	buf := make([]byte, 512)
	n, _, _, err := l.conn.ReadFrom(buf)
	if err != nil || n == 0 {
		return nil, fmt.Errorf("failed to read NDP. n bytes: %d, err: %v", n, err)
	}

	msg, err := ndp.ParseMessage(buf[:n])
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall NDP msg: %v", err)
	}
	return msg, err
}

func (l *NDPConnection) WriteTo(msg ndp.Message, dst net.IP) error {
	msgBytes, err := ndp.MarshalMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to marshall the NDP msg: %v", err)
	}
	dstAddr := &net.IPAddr{
		IP:   dst,
		Zone: l.iface.Name,
	}

	n, err := l.conn.WriteTo(msgBytes, l.controlMsg, dstAddr)
	if err != nil || n == 0 {
		return fmt.Errorf("failed to send the NDP msg to %s. Error: %v, n bytes: %d", dst.String(), err, n)
	}
	return nil
}

func (l *NDPConnection) Filter(icmpType ipv6.ICMPType) error {
	var filter ipv6.ICMPFilter
	filter.SetAll(true)
	filter.Accept(icmpType)

	if err := l.conn.SetICMPFilter(&filter); err != nil {
		return err
	}

	return nil
}
