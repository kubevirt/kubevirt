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

package hostdevice

import (
	"fmt"
	"os"
	"strings"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
)

type AddressPool struct {
	addressesByResource map[string][]string
}

// NewAddressPool creates an address pool based on the provided list of resources and
// the environment variables that correspond to it.
func NewAddressPool(resourcePrefix string, resources []string) *AddressPool {
	pool := &AddressPool{
		addressesByResource: make(map[string][]string),
	}
	pool.load(resourcePrefix, resources)
	return pool
}

func (p *AddressPool) load(resourcePrefix string, resources []string) {
	for _, resource := range resources {
		addressEnvVarName := util.ResourceNameToEnvVar(resourcePrefix, resource)
		addressString, isSet := os.LookupEnv(addressEnvVarName)
		if !isSet {
			log.Log.Warningf("%s not set for resource %s", addressEnvVarName, resource)
			continue
		}

		addressString = strings.TrimSuffix(addressString, ",")
		if addressString != "" {
			p.addressesByResource[resource] = strings.Split(addressString, ",")
		} else {
			p.addressesByResource[resource] = nil
		}
	}
}

// Pop gets the next address available to a particular resource. The
// function makes sure that the allocated address is not allocated to next
// callers, whether they request an address for the same resource or another
// resource (covering cases of addresses that are share by multiple resources).
func (p *AddressPool) Pop(resource string) (string, error) {
	addresses, exists := p.addressesByResource[resource]
	if !exists {
		return "", fmt.Errorf("resource %s does not exist", resource)
	}

	if len(addresses) > 0 {
		selectedAddress := addresses[0]

		for resourceName, resourceAddresses := range p.addressesByResource {
			p.addressesByResource[resourceName] = filterOutAddress(resourceAddresses, selectedAddress)
		}

		return selectedAddress, nil
	}
	return "", fmt.Errorf("no more addresses to allocate for resource %s", resource)
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

type BestEffortAddressPool struct {
	pool AddressPooler
}

// NewBestEffortAddressPool creates a pool that wraps a provided pool
// and allows `Pop` calls to always succeed (even when a resource is missing).
func NewBestEffortAddressPool(pool AddressPooler) *BestEffortAddressPool {
	return &BestEffortAddressPool{pool}
}

func (p *BestEffortAddressPool) Pop(resource string) (string, error) {
	address, _ := p.pool.Pop(resource)
	return address, nil
}
