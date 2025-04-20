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

package sriov

import (
	"fmt"
	"os"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
)

type PCIAddressPool struct {
	pool              *hostdevice.AddressPool
	networkToResource map[string]string
}

const resourcePrefix = "PCIDEVICE"

// NewPCIAddressPool creates a PCI address pool based on the provided list of interfaces and
// the environment variables that describe the SRIOV devices.
func NewPCIAddressPool(ifaces []v1.Interface) *PCIAddressPool {
	pool := &PCIAddressPool{
		networkToResource: make(map[string]string),
	}
	pool.loadResourcesNames(ifaces)
	pool.loadResourcesAddresses()
	return pool
}

func (p *PCIAddressPool) loadResourcesNames(ifaces []v1.Interface) {
	for _, iface := range ifaces {
		resourceEnvVarName := fmt.Sprintf("KUBEVIRT_RESOURCE_NAME_%s", iface.Name)
		resource, isSet := os.LookupEnv(resourceEnvVarName)
		if !isSet {
			log.Log.Warningf("%s not set for SR-IOV interface %s", resourceEnvVarName, iface.Name)
			continue
		}
		p.networkToResource[iface.Name] = resource
	}
}

func (p *PCIAddressPool) loadResourcesAddresses() {
	var resources []string
	for _, resource := range p.networkToResource {
		resources = append(resources, resource)
	}
	p.pool = hostdevice.NewAddressPool(resourcePrefix, resources)
}

// Pop gets the next PCI address available to a particular SR-IOV network. The
// function makes sure that the allocated address is not allocated to next
// callers, whether they request an address for the same network or another
// network that is backed by the same resourceName.
func (p *PCIAddressPool) Pop(networkName string) (string, error) {
	resource, exists := p.networkToResource[networkName]
	if !exists {
		return "", fmt.Errorf("resource for SR-IOV network %s does not exist", networkName)
	}

	pciAddress, err := p.pool.Pop(resource)
	if err != nil {
		return "", fmt.Errorf("failed to allocate SR-IOV PCI address for network %s: %v", networkName, err)
	}
	return pciAddress, nil
}
