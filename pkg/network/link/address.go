/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package link

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

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
