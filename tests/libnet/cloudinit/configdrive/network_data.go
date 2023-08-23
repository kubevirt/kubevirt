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

package configdrive

import "encoding/json"

// LinkType represents interface type
type LinkType string

const LinkTypePhy LinkType = "phy"

// Link represents L2 interface settings
type Link struct {
	Id                 string   `json:"id"`
	Type               LinkType `json:"type"`
	EthernetMacAddress string   `json:"ethernet_mac_address"`
	MTU                int      `json:"mtu,omitempty"`
}

// Route represents L3 IPv4 / IPv6 routing configuration item
type Route struct {
	Network string `json:"network"`
	Netmask string `json:"netmask"`
	Gateway string `json:"gateway"`
}

type NetworkType string

const (
	NetworkTypeIPv4     NetworkType = "ipv4"
	NetworkTypeIPv4DHCP NetworkType = "ipv4_dhcp"
	NetworkTypeIPv6     NetworkType = "ipv6"
	NetworkTypeIPv6DHCP NetworkType = "ipv6_dhcp"
)

// Network represents L3 network
type Network struct {
	Id        string      `json:"id"`
	Type      NetworkType `json:"type"`
	Link      string      `json:"link"`
	NetworkId string      `json:"network_id"`
	IPAddress string      `json:"ip_address,omitempty"`
	Netmask   string      `json:"netmask,omitempty"`
	Routes    []Route     `json:"routes,omitempty"`
}

type NetworkOption func(network *Network)

func NewNetwork(id string, Type NetworkType, link string, options ...NetworkOption) Network {
	network := Network{
		Id:   id,
		Type: Type,
		Link: link,
	}

	for _, option := range options {
		option(&network)
	}

	return network
}

func WithIPAddress(IPAddress string) NetworkOption {
	return func(network *Network) {
		network.IPAddress = IPAddress
	}
}

func WithNetmask(netmask string) NetworkOption {
	return func(network *Network) {
		network.Netmask = netmask
	}
}

// Service on IPv4 / IPv6 network
type Service struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

type NetworkDataOption func(networkData *NetworkData)

// NetworkData represents OpenStack Nova instance network configuration information
type NetworkData struct {
	Links    []Link    `json:"links"`
	Networks []Network `json:"networks"`
	Services []Service `json:"services"`
}

func NewNetworkData(options ...NetworkDataOption) NetworkData {
	networkData := NetworkData{
		Links:    make([]Link, 0),
		Networks: make([]Network, 0),
		Services: make([]Service, 0),
	}

	for _, option := range options {
		option(&networkData)
	}

	return networkData
}

func WithLink(link Link) NetworkDataOption {
	return func(networkData *NetworkData) {
		networkData.Links = append(networkData.Links, link)
	}
}

func WithNetwork(network Network) NetworkDataOption {
	return func(networkData *NetworkData) {
		networkData.Networks = append(networkData.Networks, network)
	}
}

func (nd NetworkData) String() string {
	jsonBytes, _ := json.Marshal(nd)
	return string(jsonBytes)
}
