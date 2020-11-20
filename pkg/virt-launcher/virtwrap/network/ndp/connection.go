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

	"github.com/mdlayher/ndp"
	"golang.org/x/net/ipv6"
)

// A NDPConnection instruments a system.Conn and adds retry functionality for
// receiving / sending NDP messages on a given interface.
type NDPConnection struct {
	iface string
	conn  *ndp.Conn
}

// Return an NDPConnection bound to the chosen interface.
func NewNDPConnection(ifaceName string) (*NDPConnection, error) {
	iface, err := findIface(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("could not find interface %s: %v", ifaceName, err)
	}

	c, _, err := ndp.Dial(iface, ndp.Unspecified)
	if err != nil {
		return nil, fmt.Errorf("could not start NDP ndpConn on %s: %v", ifaceName, err)
	}

	// join the routers multicast group
	if err := c.JoinGroup(net.IPv6linklocalallrouters); err != nil {
		return nil, fmt.Errorf("failed to join multicast group: %v", err)
	}

	listener := &NDPConnection{
		iface: ifaceName,
		conn:  c,
	}

	return listener, nil
}

func (l *NDPConnection) ReadFrom() (ndp.Message, *ipv6.ControlMessage, net.IP, error) {
	return l.conn.ReadFrom()
}

func (l *NDPConnection) WriteTo(m ndp.Message, cm *ipv6.ControlMessage, dst net.IP) error {
	return l.conn.WriteTo(m, cm, dst)
}

func findIface(ifaceName string) (*net.Interface, error) {
	ifi, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("could not find interface %s: %v", ifaceName, err)
	}

	return ifi, nil
}
