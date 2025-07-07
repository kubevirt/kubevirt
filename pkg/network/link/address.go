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

package link

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/netmachinery"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const bridgeFakeIP = "169.254.75.1%d/32"

func getMasqueradeGwAndHostAddressesFromCIDR(s string) (string, string, error) {
	ip, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return "", "", err
	}

	subnet, _ := ipnet.Mask.Size()
	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); netmachinery.NextIP(ip) {
		ips = append(ips, fmt.Sprintf("%s/%d", ip.String(), subnet))

		if len(ips) == 4 {
			// remove network address and broadcast address
			return ips[1], ips[2], nil
		}
	}

	return "", "", fmt.Errorf("less than 4 addresses on network")
}

func GenerateMasqueradeGatewayAndVmIPAddrs(vmiSpecNetwork *v1.Network, ipVersion netdriver.IPVersion) (*netlink.Addr, *netlink.Addr, error) {
	var cidrToConfigure string
	if ipVersion == netdriver.IPv4 {
		if vmiSpecNetwork.Pod.VMNetworkCIDR == "" {
			cidrToConfigure = api.DefaultVMCIDR
		} else {
			cidrToConfigure = vmiSpecNetwork.Pod.VMNetworkCIDR
		}

	}

	if ipVersion == netdriver.IPv6 {
		if vmiSpecNetwork.Pod.VMIPv6NetworkCIDR == "" {
			cidrToConfigure = api.DefaultVMIpv6CIDR
		} else {
			cidrToConfigure = vmiSpecNetwork.Pod.VMIPv6NetworkCIDR
		}

	}

	gatewayIP, vmIP, err := getMasqueradeGwAndHostAddressesFromCIDR(cidrToConfigure)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get gw and vm available addresses from CIDR %s", cidrToConfigure)
		return nil, nil, err
	}

	gatewayAddr, err := netlink.ParseAddr(gatewayIP)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse gateway address %s err %v", gatewayAddr, err)
	}
	vmAddr, err := netlink.ParseAddr(vmIP)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse vm address %s err %v", vmAddr, err)
	}
	return gatewayAddr, vmAddr, nil
}

func RetrieveMacAddressFromVMISpecIface(vmiSpecIface *v1.Interface) (*net.HardwareAddr, error) {
	if vmiSpecIface.MacAddress != "" {
		macAddress, err := net.ParseMAC(vmiSpecIface.MacAddress)
		if err != nil {
			return nil, err
		}
		return &macAddress, nil
	}
	return nil, nil
}

func GetFakeBridgeIP(vmiSpecIfaces []v1.Interface, vmiSpecIface *v1.Interface) string {
	for i, iface := range vmiSpecIfaces {
		if iface.Name == vmiSpecIface.Name {
			return fmt.Sprintf(bridgeFakeIP, i)
		}
	}
	return ""
}

// FilterPodNetworkRoutes filters out irrelevant routes
func FilterPodNetworkRoutes(routes []netlink.Route, nic *cache.DHCPConfig) (filteredRoutes []netlink.Route) {
	for _, route := range routes {
		log.Log.V(5).Infof("route: %s", route.String())
		// don't create empty static routes
		if route.Dst == nil && route.Src.Equal(nil) && route.Gw.Equal(nil) {
			continue
		}

		// don't create static route for src == nic
		if route.Src != nil && route.Src.Equal(nic.IP.IP) {
			continue
		}

		filteredRoutes = append(filteredRoutes, route)
	}
	return
}
