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

package libnet

import (
	"fmt"

	"sigs.k8s.io/yaml"
)

type NetworkDataOption func(*CloudInitNetworkData) error
type NetworkDataInterfaceOption func(*CloudInitInterface) error

func NewNetworkData(options ...NetworkDataOption) (string, error) {
	networkData := CloudInitNetworkData{
		Version: 2,
	}

	for _, option := range options {
		err := option(&networkData)
		if err != nil {
			return "", fmt.Errorf("failed defining network data when running options: %w", err)
		}
	}

	nd, err := yaml.Marshal(&networkData)
	if err != nil {
		return "", err
	}

	return string(nd), nil
}

func WithEthernet(name string, options ...NetworkDataInterfaceOption) NetworkDataOption {
	return func(networkData *CloudInitNetworkData) error {
		if networkData.Ethernets == nil {
			networkData.Ethernets = map[string]CloudInitInterface{}
		}

		networkDataInterface := CloudInitInterface{name: name}

		for _, option := range options {
			err := option(&networkDataInterface)
			if err != nil {
				return fmt.Errorf("failed defining network data ethernet device when running options: %w", err)
			}
		}

		networkData.Ethernets[name] = networkDataInterface
		return nil
	}
}

func WithAddresses(addresses ...string) NetworkDataInterfaceOption {
	return func(networkDataInterface *CloudInitInterface) error {
		networkDataInterface.Addresses = append(networkDataInterface.Addresses, addresses...)
		return nil
	}
}

func WithDHCP4Enabled() NetworkDataInterfaceOption {
	return func(networkDataInterface *CloudInitInterface) error {
		enabled := true
		networkDataInterface.DHCP4 = &enabled
		return nil
	}
}

func WithAcceptRA() NetworkDataInterfaceOption {
	return func(networkDataInterface *CloudInitInterface) error {
		accept := true
		networkDataInterface.AcceptRA = &accept
		return nil
	}
}

func WithDHCP6Enabled() NetworkDataInterfaceOption {
	return func(networkDataInterface *CloudInitInterface) error {
		enabled := true
		networkDataInterface.DHCP6 = &enabled
		return nil
	}
}

func WithGateway6(gateway6 string) NetworkDataInterfaceOption {
	return func(networkDataInterface *CloudInitInterface) error {
		networkDataInterface.Gateway6 = gateway6
		return nil
	}
}

func WithNameserverFromCluster() NetworkDataInterfaceOption {
	return func(networkDataInterface *CloudInitInterface) error {
		dnsServerIP, err := ClusterDNSServiceIP()
		if err != nil {
			return fmt.Errorf("failed defining network data nameservers when retrieving cluster DNS service IP: %w", err)
		}
		networkDataInterface.Nameservers = CloudInitNameservers{
			Addresses: []string{dnsServerIP},
			Search:    SearchDomains(),
		}
		return nil
	}
}

func WithMatchingMAC(macAddress string) NetworkDataInterfaceOption {
	return func(networkDataInterface *CloudInitInterface) error {
		networkDataInterface.Match = CloudInitMatch{
			MACAddress: macAddress,
		}
		networkDataInterface.SetName = networkDataInterface.name
		return nil
	}
}

type CloudInitNetworkData struct {
	Version   int                           `json:"version"`
	Ethernets map[string]CloudInitInterface `json:"ethernets,omitempty"`
}

type CloudInitInterface struct {
	name           string
	AcceptRA       *bool                `json:"accept-ra,omitempty"`
	Addresses      []string             `json:"addresses,omitempty"`
	DHCP4          *bool                `json:"dhcp4,omitempty"`
	DHCP6          *bool                `json:"dhcp6,omitempty"`
	DHCPIdentifier string               `json:"dhcp-identifier,omitempty"` // "duid" or  "mac"
	Gateway4       string               `json:"gateway4,omitempty"`
	Gateway6       string               `json:"gateway6,omitempty"`
	Nameservers    CloudInitNameservers `json:"nameservers,omitempty"`
	MACAddress     string               `json:"macaddress,omitempty"`
	Match          CloudInitMatch       `json:"match,omitempty"`
	MTU            int                  `json:"mtu,omitempty"`
	Routes         []CloudInitRoute     `json:"routes,omitempty"`
	SetName        string               `json:"set-name,omitempty"`
}

type CloudInitNameservers struct {
	Search    []string `json:"search,omitempty"`
	Addresses []string `json:"addresses,omitempty"`
}

type CloudInitMatch struct {
	Name       string `json:"name,omitempty"`
	MACAddress string `json:"macaddress,omitempty"`
	Driver     string `json:"driver,omitempty"`
}

type CloudInitRoute struct {
	From   string `json:"from,omitempty"`
	OnLink *bool  `json:"on-link,omitempty"`
	Scope  string `json:"scope,omitempty"`
	Table  *int   `json:"table,omitempty"`
	To     string `json:"to,omitempty"`
	Type   string `json:"type,omitempty"`
	Via    string `json:"via,omitempty"`
	Metric *int   `json:"metric,omitempty"`
}

const (
	DefaultIPv6Address = "fd10:0:2::2"
	DefaultIPv6CIDR    = DefaultIPv6Address + "/120"
	DefaultIPv6Gateway = "fd10:0:2::1"
)

// CreateDefaultCloudInitNetworkData generates a default configuration
// for the Cloud-Init Network Data, in version 2 format.
// The default configuration sets dynamic IPv4 (DHCP) and static IPv6 addresses,
// inclusing DNS settings of the cluster nameserver IP and search domains.
func CreateDefaultCloudInitNetworkData() (string, error) {
	return NewNetworkData(
		WithEthernet("eth0",
			WithDHCP4Enabled(),
			WithAddresses(DefaultIPv6CIDR),
			WithGateway6(DefaultIPv6Gateway),
			WithNameserverFromCluster(),
		),
	)
}
