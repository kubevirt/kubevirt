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
 * Copyright the KubeVirt Authors.
 *
 */

package sriov

import (
	"encoding/json"
	"fmt"
)

type PCIAddressWithNetworkStatusPool struct {
	networkPCIMap map[string]string
}

// NewPCIAddressPoolWithNetworkStatus creates a PCI address pool based on the networkPciMapPath volume
func NewPCIAddressPoolWithNetworkStatus(networkPCIMapBytes []byte) (*PCIAddressWithNetworkStatusPool, error) {
	pool := &PCIAddressWithNetworkStatusPool{}

	var networkPciMap map[string]string
	err := json.Unmarshal(networkPCIMapBytes, &networkPciMap)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal network-pci map %w", err)
	}

	pool.networkPCIMap = networkPciMap
	return pool, nil
}

// Len returns the length of the pool.
func (p *PCIAddressWithNetworkStatusPool) Len() int {
	if p == nil {
		return 0
	}
	return len(p.networkPCIMap)
}

// Pop gets the next PCI address available to a particular SR-IOV network. The
// function makes sure that the allocated address is not allocated to other networks.
func (p *PCIAddressWithNetworkStatusPool) Pop(networkName string) (string, error) {
	pciAddress, exists := p.networkPCIMap[networkName]
	if !exists {
		return "", fmt.Errorf("PCI-Address for SR-IOV network %q not found", networkName)
	}

	if pciAddress == "" {
		return "", fmt.Errorf("failed to allocate SR-IOV PCI address for network %q", networkName)
	}
	delete(p.networkPCIMap, networkName)
	return pciAddress, nil
}
