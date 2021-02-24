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
	chkOff       = 2
	maxHops      = 255
	raBufferSize = 128
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

	_ = ipv6Conn.SetHopLimit(maxHops)          // as per RFC 4861, section 4.1
	_ = ipv6Conn.SetMulticastHopLimit(maxHops) // as per RFC 4861, section 4.1

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
		controlMsg: getIPv6ControlMsg(listenAddr.IP, iface),
	}

	return listener, nil
}

func importNDPConnection(openedFD *os.File, iface *net.Interface) (*NDPConnection, error) {
	conn, err := net.FilePacketConn(openedFD)
	if err != nil {
		return nil, fmt.Errorf("could not get a PacketConnection from the opened file descriptor: %v", err)
	}
	ipv6Conn := ipv6.NewPacketConn(conn)
	controlMsg := getIPv6ControlMsg(net.IPv6unspecified, iface)

	ndpConn := &NDPConnection{
		iface:      iface,
		conn:       ipv6Conn,
		controlMsg: controlMsg,
	}

	return ndpConn, nil
}

func getIPv6ControlMsg(listenAddr net.IP, iface *net.Interface) *ipv6.ControlMessage {
	return &ipv6.ControlMessage{
		HopLimit: maxHops,
		Src:      listenAddr,
		IfIndex:  iface.Index,
	}
}

func ImportConnection(socketPath string) (*os.File, error) {
	c, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("error dialing to unix domain socket at %s: %v", socketPath, err)
	}
	defer c.Close()
	fdConn := c.(*net.UnixConn)

	expectedFDNumber := 1
	fds, err := fd.Get(fdConn, expectedFDNumber, []string{})
	if err != nil || len(fds) == 0 {
		return nil, fmt.Errorf("error importing the NDP connection: %v", err)
	}

	openedFD := fds[expectedFDNumber-1]
	return openedFD, nil
}

func (l *NDPConnection) Export(socketPath string) error {
	socketListener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("could not create a UNIX domain socket: %v", err)
	}

	defer socketListener.Close()
	icmpListenerFD, err := l.GetFD()
	if err != nil {
		return fmt.Errorf("could not get an opened file descriptor from the icmp listener: %v", err)
	}

	socketTransferConnection, err := socketListener.Accept()
	if err != nil {
		return fmt.Errorf("could not listen on the UNIX domain socket: %v", err)
	}
	defer socketTransferConnection.Close()
	listenConn := socketTransferConnection.(*net.UnixConn)
	if err = fd.Put(listenConn, icmpListenerFD); err != nil {
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

func (l *NDPConnection) ReadFrom() (ndp.Message, *ipv6.ControlMessage, error) {
	buf := make([]byte, raBufferSize)
	n, cm, _, err := l.conn.ReadFrom(buf)
	if err != nil || n == 0 {
		return nil, nil, fmt.Errorf("failed to read NDP. n bytes: %d, err: %v", n, err)
	}

	msg, err := ndp.ParseMessage(buf[:n])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshall NDP msg: %v", err)
	}
	return msg, cm, err
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
