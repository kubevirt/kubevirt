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
	Version   int                           `json:"version"`
	Ethernets map[string]CloudInitInterface `json:"ethernets,omitempty"`
}

type CloudInitInterface struct {
	AcceptRA       *bool                `json:"accept-ra,omitempty"`
	Addresses      []string             `json:"addresses,omitempty"`
	DHCP4          *bool                `json:"dhcp4,omitempty"`
	DHCP6          *bool                `json:"dhcp6,omitempty"`
	DHCPIdentifier string               `json:"dhcp-identifier,omitempty"` // "duid" or  "mac"
	Gateway4       string               `json:"gateway4,omitempty"`
	Gateway6       string               `json:"gateway6,omitempty"`
	Nameservers    CloudInitNameservers `json:"nameservers,omitempty"`
	MACAddress     string               `json:"macaddress,omitempty"`
	MTU            int                  `json:"mtu,omitempty"`
	Routes         []CloudInitRoute     `json:"routes,omitempty"`
}

type CloudInitNameservers struct {
	Search    []string `json:"search,omitempty,flow"`
	Addresses []string `json:"addresses,omitempty,flow"`
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
	DefaultIPv6Address = "fd10:0:2::2/120"
	DefaultIPv6Gateway = "fd10:0:2::1"
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
					Addresses: []string{DefaultIPv6Address},
					DHCP4:     &enabled,
					Gateway6:  DefaultIPv6Gateway,
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
