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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package cache

import (
	"fmt"
	"net"
	"path/filepath"

	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/util"
)

type DHCPInterfaceCache struct {
	cache *Cache
}

func ReadDHCPInterfaceCache(c cacheCreator, pid, ifaceName string) (*DHCPConfig, error) {
	dhcpCache, err := NewDHCPInterfaceCache(c, pid).IfaceEntry(ifaceName)
	if err != nil {
		return nil, err
	}
	return dhcpCache.Read()
}

func WriteDHCPInterfaceCache(c cacheCreator, pid, ifaceName string, dhcpConfig *DHCPConfig) error {
	dhcpCache, err := NewDHCPInterfaceCache(c, pid).IfaceEntry(ifaceName)
	if err != nil {
		return err
	}
	return dhcpCache.Write(dhcpConfig)
}

func DeleteDHCPInterfaceCache(c cacheCreator, pid, ifaceName string) error {
	dhcpCache, err := NewDHCPInterfaceCache(c, pid).IfaceEntry(ifaceName)
	if err != nil {
		return err
	}
	return dhcpCache.Delete()
}

func NewDHCPInterfaceCache(creator cacheCreator, pid string) DHCPInterfaceCache {
	podRootFilesystemPath := fmt.Sprintf("/proc/%s/root", pid)
	return DHCPInterfaceCache{creator.New(filepath.Join(podRootFilesystemPath, util.VirtPrivateDir))}
}

func (d DHCPInterfaceCache) IfaceEntry(ifaceName string) (DHCPInterfaceCache, error) {
	const dhcpConfigCacheFileFormat = "vif-cache-%s.json"
	cacheFileName := fmt.Sprintf(dhcpConfigCacheFileFormat, ifaceName)
	cache, err := d.cache.Entry(cacheFileName)
	if err != nil {
		return DHCPInterfaceCache{}, err
	}

	return DHCPInterfaceCache{&cache}, nil
}

func (d DHCPInterfaceCache) Read() (*DHCPConfig, error) {
	cachedIface := &DHCPConfig{}
	_, err := d.cache.Read(cachedIface)
	return cachedIface, err
}

func (d DHCPInterfaceCache) Write(dhcpConfig *DHCPConfig) error {
	return d.cache.Write(dhcpConfig)
}

func (d DHCPInterfaceCache) Delete() error {
	return d.cache.Delete()
}

type DHCPConfig struct {
	Name                string
	IP                  netlink.Addr
	IPv6                netlink.Addr
	MAC                 net.HardwareAddr
	AdvertisingIPAddr   net.IP
	AdvertisingIPv6Addr net.IP
	Routes              *[]netlink.Route
	Mtu                 uint16
	IPAMDisabled        bool
	Gateway             net.IP
	Subdomain           string
}

func (d DHCPConfig) String() string {
	return fmt.Sprintf(
		"DHCPConfig: { Name: %s, IPv4: %s, IPv6: %s, MAC: %s, AdvertisingIPAddr: %s, MTU: %d, Gateway: %s, IPAMDisabled: %t, Routes: %v}",
		d.Name,
		d.IP,
		d.IPv6,
		d.MAC,
		d.AdvertisingIPAddr,
		d.Mtu,
		d.Gateway,
		d.IPAMDisabled,
		d.Routes,
	)
}
