/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package dhcp

import (
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
	dhcpConfig.Subdomain = d.subdomain
	dhcpConfig.Mtu = uint16(podNicLink.Attrs().MTU)

	ipv4Enabled, err := d.handler.HasIPv4GlobalUnicastAddress(d.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to verify whether ipv4 is configured on %s", d.podInterfaceName)
		return nil, err
	}
	if ipv4Enabled {
		ipv4Gateway, ipv4, err := virtnetlink.GenerateMasqueradeGatewayAndVmIPAddrs(d.vmiSpecNetwork, netdriver.IPv4)
		if err != nil {
			return nil, err
		}
		dhcpConfig.IP = *ipv4
		dhcpConfig.AdvertisingIPAddr = ipv4Gateway.IP.To4()
		dhcpConfig.Gateway = ipv4Gateway.IP.To4()
	}

	ipv6Enabled, err := d.handler.HasIPv6GlobalUnicastAddress(d.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to verify whether ipv6 is configured on %s", d.podInterfaceName)
		return nil, err
	}

	if ipv6Enabled {
		ipv6Gateway, ipv6, err := virtnetlink.GenerateMasqueradeGatewayAndVmIPAddrs(d.vmiSpecNetwork, netdriver.IPv6)
		if err != nil {
			return nil, err
		}
		dhcpConfig.IPv6 = *ipv6
		dhcpConfig.AdvertisingIPv6Addr = ipv6Gateway.IP.To16()
	}

	return dhcpConfig, nil
}
