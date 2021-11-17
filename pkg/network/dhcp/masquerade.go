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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package dhcp

import (
	"github.com/coreos/go-iptables/iptables"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
)

type MasqueradeConfigGenerator struct {
	handler          netdriver.NetworkHandler
	vmiSpecIface     *v1.Interface
	vmiSpecNetwork   *v1.Network
	podInterfaceName string
	subdomain        string
}

func (d *MasqueradeConfigGenerator) Generate() (*cache.DHCPConfig, error) {
	dhcpConfig := &cache.DHCPConfig{}
	podNicLink, err := d.handler.LinkByName(d.podInterfaceName)
	if err != nil {
		return nil, err
	}

	dhcpConfig.Name = podNicLink.Attrs().Name
	dhcpConfig.Mtu = uint16(podNicLink.Attrs().MTU)

	ipv4Gateway, ipv4, err := virtnetlink.GenerateMasqueradeGatewayAndVmIPAddrs(d.vmiSpecNetwork, iptables.ProtocolIPv4)
	if err != nil {
		return nil, err
	}
	dhcpConfig.IP = *ipv4
	dhcpConfig.AdvertisingIPAddr = ipv4Gateway.IP.To4()
	dhcpConfig.Gateway = ipv4Gateway.IP.To4()

	ipv6Enabled, err := d.handler.IsIpv6Enabled(d.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to verify whether ipv6 is configured on %s", d.podInterfaceName)
		return nil, err
	}

	if ipv6Enabled {
		ipv6Gateway, ipv6, err := virtnetlink.GenerateMasqueradeGatewayAndVmIPAddrs(d.vmiSpecNetwork, iptables.ProtocolIPv6)
		if err != nil {
			return nil, err
		}
		dhcpConfig.IPv6 = *ipv6
		dhcpConfig.AdvertisingIPv6Addr = ipv6Gateway.IP.To16()
		dhcpConfig.Subdomain = d.subdomain
	}

	return dhcpConfig, nil
}
