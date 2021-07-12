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

// Package legacy provides support for using GPU & vGPU devices through a specific
// (external) device plugin: https://github.com/NVIDIA/kubevirt-gpu-device-plugin
// The DP passes GPU &vGPU devices by setting env variables in the virt-launcher pod.
//
// This method of defining GPU/vGPU devices is considered legacy, prefer
// using the in-spec VMI definitions instead.
package legacy

import (
	"fmt"
	"os"
	"strings"
)

const (
	gpuEnvPrefix  = "GPU_PASSTHROUGH_DEVICES"
	vgpuEnvPrefix = "VGPU_PASSTHROUGH_DEVICES"
)

type AddressPool struct {
	addresses []string
}

// NewGPUPCIAddressPool creates a PCI address pool from the environment variables that describe the GPU devices.
func NewGPUPCIAddressPool() *AddressPool {
	pool := &AddressPool{}
	pool.load(gpuEnvPrefix)
	return pool
}

// NewVGPUMdevAddressPool creates a PCI address pool from the environment variables that describe the vGPU devices.
func NewVGPUMdevAddressPool() *AddressPool {
	pool := &AddressPool{}
	pool.load(vgpuEnvPrefix)
	return pool
}

// load processes the environment variables and populates the pool with the PCI addresses.
func (p *AddressPool) load(envPrefix string) {
	var addresses []string
	for _, env := range os.Environ() {
		keyval := strings.Split(env, "=")
		resourceName, resourceValue := keyval[0], keyval[1]
		if strings.HasPrefix(resourceName, envPrefix) {
			addresses = append(addresses, parseAddressesFromResourceValue(resourceValue)...)
		}
	}

	// Normalizing the address value is kept for backward compatibility.
	p.addresses = normalizeAddresses(addresses)
}

func (p *AddressPool) Pop() (string, error) {
	if len(p.addresses) > 0 {
		addr := p.addresses[0]
		p.addresses = p.addresses[1:]

		return addr, nil
	}
	return "", fmt.Errorf("no more PCI addresses to allocate")
}

func (p *AddressPool) Len() int {
	return len(p.addresses)
}

// Each resource environment variable may hold:
// "": for no address set
// "<address_1>,": for a single address
// "<address_1>,<address_2>[,...]": for multiple addresses
func parseAddressesFromResourceValue(pciAddrString string) []string {
	pciAddrString = strings.TrimSuffix(pciAddrString, ",")
	return strings.Split(pciAddrString, ",")
}

func normalizeAddresses(pciAddresses []string) []string {
	var addresses []string
	for _, element := range pciAddresses {
		addresses = append(addresses, strings.TrimSpace(element))
	}
	return pciAddresses
}
