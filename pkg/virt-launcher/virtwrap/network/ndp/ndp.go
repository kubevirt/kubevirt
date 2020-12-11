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
	"time"

	"github.com/mdlayher/ndp"

	"kubevirt.io/client-go/log"
)

const (
	routerAdvertisementMaxLifetime = 65535 * time.Second // check RFC 4861, section 4.2; 16 bit integer.
	routerAdvertisementPeriod      = 5 * time.Minute
)

type RouterAdvertisementDaemon struct {
	raOptions []ndp.Option
	ndpConn   *NDPConnection
}

func RouterAdvertisementDaemonFromFD(openedFD *os.File, ifaceName string, ipv6CIDR string, routerMACAddr net.HardwareAddr) (*RouterAdvertisementDaemon, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("could not find interface %s: %v", ifaceName, err)
	}

	ndpConnection, err := importNDPConnection(openedFD, iface)
	if err != nil {
		return nil, fmt.Errorf("failed to import the NDP connection from the opened file descriptor")
	}

	prefix, network, err := net.ParseCIDR(ipv6CIDR)
	if err != nil {
		return nil, fmt.Errorf("could not compute prefix / prefix length from %s. Reason: %v", ipv6CIDR, err)
	}
	prefixLength, _ := network.Mask.Size()
	rad := &RouterAdvertisementDaemon{
		ndpConn:   ndpConnection,
		raOptions: prepareRAOptions(prefix, uint8(prefixLength), routerMACAddr),
	}
	return rad, nil
}

func (rad *RouterAdvertisementDaemon) Serve() error {
	for {
		msg, _, err := rad.ndpConn.ReadFrom()
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

func (rad *RouterAdvertisementDaemon) PeriodicallySendRAs() {
	doneChannel := make(chan struct{})
	ticker := time.NewTicker(routerAdvertisementPeriod)

	for {
		select {
		case <-doneChannel:
			ticker.Stop()
			return

		case <-ticker.C:
			if err := rad.SendRouterAdvertisement(); err != nil {
				log.Log.Warningf("failed to send periodic RouterAdvertisement: %v", err)
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

	if err := rad.ndpConn.WriteTo(ra, net.IPv6linklocalallnodes); err != nil {
		return fmt.Errorf("failed to send router advertisement: %v", err)
	}
	return nil
}

func prepareRAOptions(prefix net.IP, prefixLength uint8, routerMACAddr net.HardwareAddr) []ndp.Option {
	prefixInfo := &ndp.PrefixInformation{
		PrefixLength:                   prefixLength,
		OnLink:                         true,
		AutonomousAddressConfiguration: false,
		ValidLifetime:                  ndp.Infinity,
		PreferredLifetime:              ndp.Infinity,
		Prefix:                         prefix,
	}

	sourceLinkLayerAddr := &ndp.LinkLayerAddress{
		Addr:      routerMACAddr,
		Direction: ndp.Source,
	}

	return []ndp.Option{prefixInfo, sourceLinkLayerAddr}
}
