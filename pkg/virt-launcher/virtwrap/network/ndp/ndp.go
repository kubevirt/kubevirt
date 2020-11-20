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
	"time"

	"github.com/mdlayher/ndp"
	"golang.org/x/net/ipv6"

	"kubevirt.io/client-go/log"
)

const (
	routerAdvertisementMaxLifetime = 65535 * time.Second
)

type RouterAdvertisementDaemon struct {
	raOptions []ndp.Option
	ndpConn   *NDPConnection
}

func SingleClientRouterAdvertisementDaemon(serverIface string, ipv6CIDR string) error {
	log.Log.Info("Starting RouterAdvertisement daemon")

	prefix, network, err := net.ParseCIDR(ipv6CIDR)
	if err != nil {
		return fmt.Errorf("could not compute prefix / prefix length from %s. Reason: %v", ipv6CIDR, err)
	}
	prefixLength, _ := network.Mask.Size()

	ndpConnection, err := NewNDPConnection(serverIface)
	if err != nil {
		return fmt.Errorf("could not listen to icmpv6 on interface %s. Reason: %v", serverIface, err)
	}

	handler := &RouterAdvertisementDaemon{
		raOptions: prepareRAOptions(prefix, uint8(prefixLength)),
		ndpConn:   ndpConnection,
	}

	return handler.serve()
}

func (rad *RouterAdvertisementDaemon) serve() error {
	if err := rad.filterRouterSolicitations(); err != nil {
		return fmt.Errorf("could not set a RouterSolicitation filter on the ICMPv6 listener: %v", err)
	}

	for {
		msg, _, _, err := rad.ndpConn.ReadFrom()
		if err != nil {
			return err
		}

		switch msg.(type) {
		case *ndp.RouterSolicitation:
			log.Log.V(4).Info("Received RouterSolicitation msg. Will reply w/ RA")
			err = rad.SendRouterAdvertisement()
			if err != nil {
				return err
			}
		}
	}
}

func (rad *RouterAdvertisementDaemon) SendRouterAdvertisement() error {
	ra := &ndp.RouterAdvertisement{
		ManagedConfiguration: true,
		OtherConfiguration:   true,
		RouterLifetime:       routerAdvertisementMaxLifetime,
		ReachableTime:        ndp.Infinity,
		Options:              rad.raOptions,
	}

	if err := rad.ndpConn.WriteTo(ra, nil, net.IPv6linklocalallnodes); err != nil {
		return fmt.Errorf("failed to send router advertisement: %v", err)
	}
	return nil
}

func (rad *RouterAdvertisementDaemon) filterRouterSolicitations() error {
	var filter ipv6.ICMPFilter
	filter.SetAll(true)
	filter.Accept(ipv6.ICMPTypeRouterSolicitation)
	if err := rad.listener.ConfigureFilter(&filter); err != nil {
		return err
	}
	return nil
}

func prepareRAOptions(prefix net.IP, prefixLength uint8) []ndp.Option {
	prefixInfo := &ndp.PrefixInformation{
		PrefixLength:                   prefixLength,
		OnLink:                         true,
		AutonomousAddressConfiguration: false,
		ValidLifetime:                  ndp.Infinity,
		PreferredLifetime:              ndp.Infinity,
		Prefix:                         prefix,
	}

	defaultRoute := &ndp.RouteInformation{
		PrefixLength:  prefixLength,
		Preference:    ndp.High,
		RouteLifetime: ndp.Infinity,
		Prefix:        prefix,
	}
	return []ndp.Option{prefixInfo, defaultRoute}
}
