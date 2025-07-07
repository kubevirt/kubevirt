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

package serverv6

import (
	"fmt"
	"net"
	"time"

	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/dhcpv6/server6"
	"github.com/insomniacslk/dhcp/iana"

	"kubevirt.io/client-go/log"
)

const (
	infiniteLease = 999 * 24 * time.Hour
)

type DHCPv6Handler struct {
	clientIP  net.IP
	modifiers []dhcpv6.Modifier
}

func SingleClientDHCPv6Server(clientIP net.IP, serverIfaceName string) error {
	log.Log.Info("Starting SingleClientDHCPv6Server")

	iface, err := net.InterfaceByName(serverIfaceName)
	if err != nil {
		return fmt.Errorf("couldn't create DHCPv6 server, couldn't get the dhcp6 server interface: %v", err)
	}

	modifiers := prepareDHCPv6Modifiers(clientIP, iface.HardwareAddr)

	handler := &DHCPv6Handler{
		clientIP:  clientIP,
		modifiers: modifiers,
	}

	conn, err := NewConnection(iface)
	if err != nil {
		return fmt.Errorf("couldn't create DHCPv6 server: %v", err)
	}

	s, err := server6.NewServer("", nil, handler.ServeDHCPv6, server6.WithConn(conn))
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
	log.Log.V(4).Info("DHCPv6 serving a new request")

	// TODO if we extend the server to support bridge binding, we need to filter out non-vm requests

	response, err := h.buildResponse(m)
	if err != nil {
		log.Log.Reason(err).Error("DHCPv6 failed building a response to the client")
		return
	}

	if _, err := conn.WriteTo(response.ToBytes(), peer); err != nil {
		log.Log.Reason(err).Error("DHCPv6 failed sending a response to the client")
	}
}

func (h *DHCPv6Handler) buildResponse(msg dhcpv6.DHCPv6) (*dhcpv6.Message, error) {
	var response *dhcpv6.Message
	var err error

	dhcpv6Msg := msg.(*dhcpv6.Message)
	switch dhcpv6Msg.Type() {
	case dhcpv6.MessageTypeSolicit:
		log.Log.V(4).Info("DHCPv6 - the request has message type Solicit")
		if dhcpv6Msg.GetOneOption(dhcpv6.OptionRapidCommit) == nil {
			response, err = dhcpv6.NewAdvertiseFromSolicit(dhcpv6Msg, h.modifiers...)
		} else {
			log.Log.V(4).Info("DHCPv6 - replying with rapid commit")
			response, err = dhcpv6.NewReplyFromMessage(dhcpv6Msg, h.modifiers...)
		}
	default:
		log.Log.V(4).Info("DHCPv6 - non Solicit request received")
		response, err = dhcpv6.NewReplyFromMessage(dhcpv6Msg, h.modifiers...)
	}

	if err != nil {
		return nil, err
	}

	ianaRequest := dhcpv6Msg.Options.OneIANA()
	if ianaRequest != nil {
		ianaResponse := response.Options.OneIANA()
		ianaResponse.IaId = ianaRequest.IaId
		response.UpdateOption(ianaResponse)
	}
	return response, nil
}

func prepareDHCPv6Modifiers(clientIP net.IP, serverInterfaceMac net.HardwareAddr) []dhcpv6.Modifier {
	optIAAddress := dhcpv6.OptIAAddress{IPv6Addr: clientIP, PreferredLifetime: infiniteLease, ValidLifetime: infiniteLease}
	duid := &dhcpv6.DUIDLL{HWType: iana.HWTypeEthernet, LinkLayerAddr: serverInterfaceMac}

	return []dhcpv6.Modifier{dhcpv6.WithIANA(optIAAddress), dhcpv6.WithServerID(duid)}
}
