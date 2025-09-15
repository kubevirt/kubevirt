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
 * Copyright The KubeVirt Authors.
 */

package dhcp

import (
	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
)

type cacheCreator interface {
	New(filePath string) *cache.Cache
}

type BridgeConfigGenerator struct {
	handler          netdriver.NetworkHandler
	podInterfaceName string
	cacheCreator     cacheCreator
	vmiSpecIfaces    []v1.Interface
	vmiSpecIface     *v1.Interface
	subdomain        string
}

func (d *BridgeConfigGenerator) Generate() (*cache.DHCPConfig, error) {
	const launcherPID = "self"
	dhcpConfig, err := cache.ReadDHCPInterfaceCache(d.cacheCreator, launcherPID, d.podInterfaceName)
	if err != nil {
		return nil, err
	}

	if dhcpConfig.IPAMDisabled {
		return dhcpConfig, nil
	}

	dhcpConfig.Name = d.podInterfaceName

	fakeBridgeIP := virtnetlink.GetFakeBridgeIP(d.vmiSpecIfaces, d.vmiSpecIface)
	fakeServerAddr, _ := netlink.ParseAddr(fakeBridgeIP)
	dhcpConfig.AdvertisingIPAddr = fakeServerAddr.IP

	newPodNicName := virtnetlink.GenerateNewBridgedVmiInterfaceName(d.podInterfaceName)
	podNicLink, err := d.handler.LinkByName(newPodNicName)
	if err != nil {
		return nil, err
	}
	dhcpConfig.Mtu = uint16(podNicLink.Attrs().MTU)
	dhcpConfig.Subdomain = d.subdomain

	return dhcpConfig, nil
}
