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

package gpu

import (
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
)

const (
	pciResourcePrefix  = "PCI_RESOURCE"
	mdevResourcePrefix = "MDEV_PCI_RESOURCE"
)

// NewPCIAddressPool creates a PCI address pool based on the provided list of host-devices and
// the environment variables that describe the resource.
func NewPCIAddressPool(gpuDevices []v1.GPU) *hostdevice.AddressPool {
	return hostdevice.NewAddressPool(pciResourcePrefix, extractResources(gpuDevices))
}

// NewMDEVAddressPool creates a MDEV address pool based on the provided list of host-devices and
// the environment variables that describe the resource.
func NewMDEVAddressPool(gpuDevices []v1.GPU) *hostdevice.AddressPool {
	return hostdevice.NewAddressPool(mdevResourcePrefix, extractResources(gpuDevices))
}

func extractResources(gpuDevices []v1.GPU) []string {
	var resourceSet = make(map[string]struct{})
	for _, gpuDevice := range gpuDevices {
		resourceSet[gpuDevice.DeviceName] = struct{}{}
	}

	var resources []string
	for resource, _ := range resourceSet {
		resources = append(resources, resource)
	}
	return resources
}
