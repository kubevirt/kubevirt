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
 *
 */

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

package driver

import (
	"github.com/vishvananda/netlink"
)

const (
	LibvirtUserAndGroupId = "0"
)

type IPVersion int

const (
	IPv4 IPVersion = 4
	IPv6 IPVersion = 6
)

type NetworkHandler interface {
	LinkByName(name string) (netlink.Link, error)
	HasIPv4GlobalUnicastAddress(interfaceName string) (bool, error)
	HasIPv6GlobalUnicastAddress(interfaceName string) (bool, error)
}

type NetworkUtilsHandler struct{}

func (h *NetworkUtilsHandler) LinkByName(name string) (netlink.Link, error) {
	return netlink.LinkByName(name)
}
func (h *NetworkUtilsHandler) HasIPv4GlobalUnicastAddress(interfaceName string) (bool, error) {
	link, err := h.LinkByName(interfaceName)
	if err != nil {
		return false, err
	}
	addrList, err := netlink.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		return false, err
	}

	for _, addr := range addrList {
		if addr.IP.IsGlobalUnicast() {
			return true, nil
		}
	}
	return false, nil
}

func (h *NetworkUtilsHandler) HasIPv6GlobalUnicastAddress(interfaceName string) (bool, error) {
	link, err := h.LinkByName(interfaceName)
	if err != nil {
		return false, err
	}
	addrList, err := netlink.AddrList(link, netlink.FAMILY_V6)
	if err != nil {
		return false, err
	}

	for _, addr := range addrList {
		if addr.IP.IsGlobalUnicast() {
			return true, nil
		}
	}
	return false, nil
}
