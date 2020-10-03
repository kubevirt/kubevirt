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
	"sigs.k8s.io/yaml"
)

type CloudInitNetworkData struct {
	Version   int                           `yaml:"version"`
	Ethernets map[string]CloudInitInterface `yaml:"ethernets,omitempty"`
}

type CloudInitInterface struct {
	AcceptRA       *bool                `yaml:"accept-ra,omitempty"`
	Addresses      []string             `yaml:"addresses,omitempty"`
	DHCP4          *bool                `yaml:"dhcp4,omitempty"`
	DHCP6          *bool                `yaml:"dhcp6,omitempty"`
	DHCPIdentifier string               `yaml:"dhcp-identifier,omitempty"` // "duid" or  "mac"
	Gateway4       string               `yaml:"gateway4,omitempty"`
	Gateway6       string               `yaml:"gateway6,omitempty"`
	Nameservers    CloudInitNameservers `yaml:"nameservers,omitempty"`
	MACAddress     string               `yaml:"macaddress,omitempty"`
	MTU            int                  `yaml:"mtu,omitempty"`
	Routes         []CloudInitRoute     `yaml:"routes,omitempty"`
}

type CloudInitNameservers struct {
	Search    []string `yaml:"search,omitempty,flow"`
	Addresses []string `yaml:"addresses,omitempty,flow"`
}

type CloudInitRoute struct {
	From   string `yaml:"from,omitempty"`
	OnLink *bool  `yaml:"on-link,omitempty"`
	Scope  string `yaml:"scope,omitempty"`
	Table  *int   `yaml:"table,omitempty"`
	To     string `yaml:"to,omitempty"`
	Type   string `yaml:"type,omitempty"`
	Via    string `yaml:"via,omitempty"`
	Metric *int   `yaml:"metric,omitempty"`
}

const (
	ipv6MasqueradeAddress = "fd10:0:2::2/120"
	ipv6MasqueradeGateway = "fd10:0:2::1"
)

// CreateDefaultCloudInitNetworkData generates a default configuration
// for the Cloud-Init Network Data, in version 2 format.
// The default configuration sets dynamic IPv4 (DHCP) and static IPv6 addresses,
// inclusing DNS settings of the cluster nameserver IP and search domains.
func CreateDefaultCloudInitNetworkData() (string, error) {
	dnsServerIP, err := ClusterDNSServiceIP()
	if err != nil {
		return "", err
	}

	enabled := true
	networkData, err := CreateCloudInitNetworkData(
		&CloudInitNetworkData{
			Version: 2,
			Ethernets: map[string]CloudInitInterface{
				"eth0": {
					Addresses: []string{ipv6MasqueradeAddress},
					DHCP4:     &enabled,
					Gateway6:  ipv6MasqueradeGateway,
					Nameservers: CloudInitNameservers{
						Addresses: []string{dnsServerIP},
						Search:    SearchDomains(),
					},
				},
			},
		},
	)
	if err != nil {
		return "", err
	}

	return string(networkData), nil
}

// CreateCloudInitNetworkData generates a configuration for the Cloud-Init Network Data version 2 format
// based on the inputed data.
func CreateCloudInitNetworkData(networkData *CloudInitNetworkData) ([]byte, error) {
	return yaml.Marshal(networkData)
}
