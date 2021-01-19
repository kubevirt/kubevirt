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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package sriov

import (
	"fmt"
	"os"
	"strings"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/util"
)

type PCIAddressPool struct {
	networkToAddresses map[string][]string
}

// NewPCIAddressPool creates a PCI address pool based on the provided list of interfaces and
// the environment variables that describe the SRIOV devices.
func NewPCIAddressPool(ifaces []v1.Interface) *PCIAddressPool {
	pool := &PCIAddressPool{
		networkToAddresses: make(map[string][]string),
	}
	pool.load(ifaces)
	return pool
}

func (p *PCIAddressPool) load(ifaces []v1.Interface) {
	for _, iface := range ifaces {
		p.networkToAddresses[iface.Name] = []string{}
		resourceEnvVarName := fmt.Sprintf("KUBEVIRT_RESOURCE_NAME_%s", iface.Name)
		resourceName, isSet := os.LookupEnv(resourceEnvVarName)
		if !isSet {
			log.Log.Warningf("%s not set for SR-IOV interface %s", resourceEnvVarName, iface.Name)
			continue
		}

		pciAddrEnvVarName := util.ResourceNameToEnvVar("PCIDEVICE", resourceName)
		pciAddrString, isSet := os.LookupEnv(pciAddrEnvVarName)
		if !isSet {
			log.Log.Warningf("%s not set for SR-IOV interface %s", pciAddrEnvVarName, iface.Name)
			continue
		}

		pciAddrString = strings.TrimSuffix(pciAddrString, ",")
		p.networkToAddresses[iface.Name] = strings.Split(pciAddrString, ",")
	}
}

// Pop gets the next PCI address available to a particular SR-IOV network. The
// function makes sure that the allocated address is not allocated to next
// callers, whether they request an address for the same network or another
// network that is backed by the same resourceName.
func (p *PCIAddressPool) Pop(networkName string) (string, error) {
	if len(p.networkToAddresses[networkName]) > 0 {
		addr := p.networkToAddresses[networkName][0]

		for networkName, addrs := range p.networkToAddresses {
			p.networkToAddresses[networkName] = filterOutAddress(addrs, addr)
		}

		return addr, nil
	}
	return "", fmt.Errorf("no more SR-IOV PCI addresses to allocate for network %s", networkName)
}

func filterOutAddress(addrs []string, addr string) []string {
	var res []string
	for _, a := range addrs {
		if a != addr {
			res = append(res, a)
		}
	}
	return res
}
