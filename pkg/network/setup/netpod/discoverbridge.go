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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package netpod

import (
	"fmt"
	"net"
	"strconv"

	vishnetlink "github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/driver/nmstate"
)

func (n NetPod) storeBridgeBindingDHCPInterfaceData(currentStatus *nmstate.Status, podIfaceStatus nmstate.Interface, vmiSpecIface v1.Interface, podIfaceName string) error {
	var dhcpConfig cache.DHCPConfig
	dhcpConfig.IPAMDisabled = true
	if ipAddress := firstIPGlobalUnicast(podIfaceStatus.IPv4); ipAddress != nil {
		dhcpConfig.IPAMDisabled = false

		addr, iperr := vishnetlink.ParseAddr(fmt.Sprintf("%s/%d", ipAddress.IP, ipAddress.PrefixLen))
		if iperr != nil {
			return iperr
		}
		dhcpConfig.IP = *addr

		mac, err := resolveMacAddress(podIfaceStatus.MacAddress, vmiSpecIface.MacAddress)
		if err != nil {
			return err
		}
		dhcpConfig.MAC = mac

		linkRoutes, err := filterIPv4RoutesByInterface(currentStatus, podIfaceName)
		if err != nil {
			return err
		}
		dhcpConfig.Gateway = net.ParseIP(linkRoutes[0].NextHopAddress)

		otherRoutes, err := filterRoutesByNonLocalDestination(linkRoutes, addr)
		if err != nil {
			return err
		}

		dhcpRoutes, err := translateNmstateToNetlinkRoutes(otherRoutes)
		if err != nil {
			return err
		}
		if len(dhcpRoutes) > 0 {
			dhcpConfig.Routes = &dhcpRoutes
		}
	}

	log.Log.V(4).Infof("The generated dhcpConfig: %s\nRoutes: %+v", dhcpConfig.String(), dhcpConfig.Routes)
	if err := cache.WriteDHCPInterfaceCache(n.cacheCreator, strconv.Itoa(n.podPID), podIfaceName, &dhcpConfig); err != nil {
		return fmt.Errorf("failed to save DHCP configuration: %v", err)
	}

	return nil
}

func (n NetPod) storeBridgeDomainInterfaceData(podIfaceStatus nmstate.Interface, vmiSpecIface v1.Interface) error {
	mac, err := resolveMacAddress(podIfaceStatus.MacAddress, vmiSpecIface.MacAddress)
	if err != nil {
		return err
	}

	domainIface := api.Interface{MAC: &api.MAC{MAC: mac.String()}}

	log.Log.V(4).Infof("The generated domain interface data: mac = %s", domainIface.MAC.MAC)
	if err := cache.WriteDomainInterfaceCache(n.cacheCreator, strconv.Itoa(n.podPID), vmiSpecIface.Name, &domainIface); err != nil {
		return fmt.Errorf("failed to save domain interface data: %v", err)
	}

	return nil
}

func translateNmstateToNetlinkRoutes(otherRoutes []nmstate.Route) ([]vishnetlink.Route, error) {
	var dhcpRoutes []vishnetlink.Route
	for _, nmstateRoute := range otherRoutes {
		isDefaultRoute := nmstateRoute.Destination == nmstate.DefaultDestinationRoute(vishnetlink.FAMILY_V4).String()
		var dstAddr *net.IPNet
		if !isDefaultRoute {
			_, ipNet, perr := net.ParseCIDR(nmstateRoute.Destination)
			if perr != nil {
				return nil, perr
			}
			dstAddr = ipNet
		}
		route := vishnetlink.Route{
			Dst: dstAddr,
			Gw:  net.ParseIP(nmstateRoute.NextHopAddress),
		}
		dhcpRoutes = append(dhcpRoutes, route)
	}
	return dhcpRoutes, nil
}

// filterRoutesByNonLocalDestination filters out local routes (the destination is of the local link).
// Default routes should not be filter out.
func filterRoutesByNonLocalDestination(linkRoutes []nmstate.Route, addr *vishnetlink.Addr) ([]nmstate.Route, error) {
	var otherRoutes []nmstate.Route
	for _, route := range linkRoutes {
		_, dstIPNet, perr := net.ParseCIDR(route.Destination)
		if perr != nil {
			return nil, perr
		}
		isDefaultRoute := route.Destination == nmstate.DefaultDestinationRoute(vishnetlink.FAMILY_V4).String()
		localDestination := !isDefaultRoute && dstIPNet.Contains(addr.IP)
		if !localDestination {
			otherRoutes = append(otherRoutes, route)
		}
	}
	return otherRoutes, nil
}

func filterIPv4RoutesByInterface(currentStatus *nmstate.Status, podIfaceName string) ([]nmstate.Route, error) {
	var linkRoutes []nmstate.Route
	for _, route := range currentStatus.Routes.Running {
		ip, _, err := net.ParseCIDR(route.Destination)
		if err != nil {
			return nil, err
		}
		if isIPv6Family(ip) || isIPv6Family(net.ParseIP(route.NextHopAddress)) {
			continue
		}
		if route.NextHopInterface == podIfaceName {
			linkRoutes = append(linkRoutes, route)
		}
	}
	if len(linkRoutes) == 0 {
		return nil, fmt.Errorf("no gateway address found in routes for %s", podIfaceName)
	}
	return linkRoutes, nil
}

func resolveMacAddress(macAddressFromCurrent string, macAddressFromVMISpec string) (net.HardwareAddr, error) {
	macAddress := macAddressFromCurrent
	if macAddressFromVMISpec != "" {
		macAddress = macAddressFromVMISpec
	}
	mac, merr := net.ParseMAC(macAddress)
	if merr != nil {
		return nil, merr
	}
	return mac, nil
}

func isIPv6Family(ip net.IP) bool {
	isIPv4 := len(ip) <= net.IPv4len || ip.To4() != nil
	return !isIPv4
}
