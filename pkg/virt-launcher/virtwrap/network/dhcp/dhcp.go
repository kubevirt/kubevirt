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
	"github.com/vishvananda/netlink"

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
	dnsIPs [][]byte,
	routes *[]netlink.Route) error {

	log.Log.Info("Starting SingleClientDHCPServer")

	dhcpOptions := dhcp.Options{
		dhcp.OptionSubnetMask:       []byte(clientMask),
		dhcp.OptionRouter:           []byte(routerIP),
		dhcp.OptionDomainNameServer: bytes.Join(dnsIPs, nil),
	}

	netRoutes := FormClasslessRoutes(routes, routerIP)

	if netRoutes != nil {
		dhcpOptions[dhcp.OptionClasslessRouteFormat] = netRoutes
	}

	handler := &DHCPHandler{
		clientIP:      clientIP,
		clientMAC:     clientMAC,
		serverIP:      serverIP.To4(),
		leaseDuration: infiniteLease,
		options:       dhcpOptions,
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

func FormClasslessRoutes(routes *[]netlink.Route, routerIP net.IP) (formattedRoutes []byte) {
	// See RFC4332 for additional information
	// (https://tools.ietf.org/html/rfc3442)
	// For example:
	// 		routes:
	//				10.0.0.0/8 ,  gateway: 10.1.2.3
	//              192.168.1/24, gateway: 192.168.2.3
	//		would result in the following structure:
	//      []byte{8, 10, 10, 1, 2, 3, 24, 192, 168, 1, 192, 168, 2, 3}

	for _, route := range *routes {
		if route.Dst == nil {
			continue
		}
		ip := route.Dst.IP.To4()
		width, _ := route.Dst.Mask.Size()
		octets := (width-1)/8 + 1
		newRoute := append([]byte{byte(width)}, ip[0:octets]...)
		gateway := route.Gw.To4()
		if gateway == nil {
			gateway = []byte{0, 0, 0, 0}
		}
		newRoute = append(newRoute, gateway...)
		formattedRoutes = append(formattedRoutes, newRoute...)
	}
	return
}
