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
	"fmt"
	"net"

	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/dhcpv6/server6"

	"kubevirt.io/client-go/log"
)

type DHCPv6Handler struct {
	clientIP net.IP
}

func SingleClientDHCPv6Server(clientIP net.IP, serverIface string) error {
	log.Log.Info("Starting SingleClientDHCPv6Server")

	handler := &DHCPv6Handler{
		clientIP: clientIP,
	}

	s, err := server6.NewServer(serverIface, nil, handler.ServeDHCPv6)
	if err != nil {
		return fmt.Errorf("couldn't create DHCPv6 server: %v", err)
	}

	err = s.Serve()
	if err != nil {
		return fmt.Errorf("failed to run DHCPv6 server: %v", err)
	}

	return nil
}

func (h *DHCPv6Handler) ServeDHCPv6(conn net.PacketConn, peer net.Addr, m dhcpv6.DHCPv6) {
	log.Log.V(4).Info("Serving a new request")

	// TODO if we extend the server to support bridge binding, we need to filter out non-vm requests

	msg := m.(*dhcpv6.Message)
	adv, err := dhcpv6.NewAdvertiseFromSolicit(msg)
	if err != nil {
		log.Log.V(4).Errorf("NewAdvertiseFromSolicit failed: %v", err)
		return
	}
	if _, err := conn.WriteTo(adv.ToBytes(), peer); err != nil {
		log.Log.V(4).Errorf("Cannot reply to client: %v", err)
	}
}
