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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package dhcp

import (
	"bytes"
	"net"
	"time"

	dhcp "github.com/krolaw/dhcp4"
	dhcpConn "github.com/krolaw/dhcp4/conn"

	"kubevirt.io/kubevirt/pkg/log"
)

const infiniteLease = 999 * 24 * time.Hour

func SingleClientDHCPServer(
	clientMAC net.HardwareAddr,
	clientIP net.IP,
	clientMask net.IPMask,
	serverIface string,
	serverIP net.IP,
	routerIP net.IP,
	dnsIP net.IP) error {

	log.Log.Info("Starting SingleClientDHCPServer")

	handler := &DHCPHandler{
		clientIP:      clientIP,
		clientMAC:     clientMAC,
		serverIP:      serverIP,
		leaseDuration: infiniteLease,
		options: dhcp.Options{
			dhcp.OptionSubnetMask:       []byte(clientMask),
			dhcp.OptionRouter:           []byte(routerIP),
			dhcp.OptionDomainNameServer: []byte(dnsIP),
		},
	}

	l, err := dhcpConn.NewUDP4BoundListener(serverIface, ":67")
	if err != nil {
		return err
	}
	defer l.Close()
	err = dhcp.Serve(l, handler)
	if err != nil {
		return err
	}
	return nil
}

type DHCPHandler struct {
	serverIP      net.IP
	clientIP      net.IP
	clientMAC     net.HardwareAddr
	leaseDuration time.Duration
	options       dhcp.Options
}

func (h *DHCPHandler) ServeDHCP(p dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) (d dhcp.Packet) {
	log.Log.Debug("Serving a new request")
	if mac := p.CHAddr(); !bytes.Equal(mac, h.clientMAC) {
		log.Log.Debug("The request is not from our client")
		return nil // Is not our client
	}

	switch msgType {

	case dhcp.Discover:
		log.Log.Debug("The request has message type DISCOVER")
		return dhcp.ReplyPacket(p, dhcp.Offer, h.serverIP, h.clientIP, h.leaseDuration,
			h.options.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]))

	case dhcp.Request:
		log.Log.Debug("The request has message type REQUEST")
		return dhcp.ReplyPacket(p, dhcp.ACK, h.serverIP, h.clientIP, h.leaseDuration,
			h.options.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]))

	default:
		log.Log.Debug("The request has unhandled message type")
		return nil // Ignored message type

	}
}
