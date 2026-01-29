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

package gpu

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	drautil "kubevirt.io/kubevirt/pkg/dra"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
)

const (
	failedCreateGPUHostDeviceFmt = "failed to create GPU host-devices: %v"
	AliasPrefix                  = "gpu-"
	DefaultDisplayOn             = true
)

func CreateHostDevices(vmiGPUs []v1.GPU) ([]api.HostDevice, error) {
	return CreateHostDevicesFromPools(vmiGPUs, NewPCIAddressPool(vmiGPUs), NewMDEVAddressPool(vmiGPUs))
}

func CreateHostDevicesFromPools(vmiGPUs []v1.GPU, pciAddressPool, mdevAddressPool hostdevice.AddressPooler) ([]api.HostDevice, error) {
	pciPool := hostdevice.NewBestEffortAddressPool(pciAddressPool)
	mdevPool := hostdevice.NewBestEffortAddressPool(mdevAddressPool)

	hostDevicesMetaData := createHostDevicesMetadata(vmiGPUs)
	pciHostDevices, err := hostdevice.CreatePCIHostDevices(hostDevicesMetaData, pciPool)
	if err != nil {
		return nil, fmt.Errorf(failedCreateGPUHostDeviceFmt, err)
	}
	mdevHostDevices, err := hostdevice.CreateMDEVHostDevices(hostDevicesMetaData, mdevPool, DefaultDisplayOn)
	if err != nil {
		return nil, fmt.Errorf(failedCreateGPUHostDeviceFmt, err)
	}

	hostDevices := append(pciHostDevices, mdevHostDevices...)

	if err := validateCreationOfDevicePluginsDevices(vmiGPUs, pciHostDevices, mdevHostDevices); err != nil {
		return nil, fmt.Errorf(failedCreateGPUHostDeviceFmt, err)
	}

	// Create host devices for remaining IOMMU companion devices.
	// When GPUs have multiple devices in their IOMMU group (e.g., GPU + audio controller),
	// the device plugin provides all addresses but we only consumed one per requested GPU above.
	// We need to passthrough all remaining devices in the IOMMU groups for proper operation.
	resources := extractUniqueResources(vmiGPUs)
	iommuCompanionDevices, err := hostdevice.CreatePCIHostDevicesFromRemainingAddresses(AliasPrefix, resources, pciPool)
	if err != nil {
		return nil, fmt.Errorf(failedCreateGPUHostDeviceFmt, err)
	}
	hostDevices = append(hostDevices, iommuCompanionDevices...)

	return hostDevices, nil
}

func createHostDevicesMetadata(vmiGPUs []v1.GPU) []hostdevice.HostDeviceMetaData {
	var hostDevicesMetaData []hostdevice.HostDeviceMetaData
	for _, dev := range vmiGPUs {
		hostDevicesMetaData = append(hostDevicesMetaData, hostdevice.HostDeviceMetaData{
			AliasPrefix:       AliasPrefix,
			Name:              dev.Name,
			ResourceName:      dev.DeviceName,
			VirtualGPUOptions: dev.VirtualGPUOptions,
		})
	}
	return hostDevicesMetaData
}

// validateCreationOfDevicePluginsDevices validates that all specified GPU/s have a matching host-device.
// On validation failure, an error is returned.
// The validation assumes that the assignment of a device to a specified GPU is correct,
// therefore a simple quantity check is sufficient.
// Note: This validates the primary GPU devices only, not IOMMU companion devices.
func validateCreationOfDevicePluginsDevices(gpus []v1.GPU, pciHostDevices, mdevHostDevices []api.HostDevice) error {
	var gpusWithDP []v1.GPU
	for _, gpu := range gpus {
		if !drautil.IsGPUDRA(gpu) {
			gpusWithDP = append(gpusWithDP, gpu)
		}
	}

	primaryHostDevices := append(pciHostDevices, mdevHostDevices...)
	if len(gpusWithDP) > 0 && len(gpusWithDP) != len(primaryHostDevices) {
		return fmt.Errorf(
			"the number of device plugin GPU/s do not match the number of devices:\nGPU: %v\nDevice: %v", gpusWithDP, primaryHostDevices,
		)
	}
	return nil
}

// extractUniqueResources returns a deduplicated list of resource names from the GPU specs
func extractUniqueResources(gpus []v1.GPU) []string {
	resourceSet := make(map[string]struct{})
	for _, gpu := range gpus {
		resourceSet[gpu.DeviceName] = struct{}{}
	}

	var resources []string
	for resource := range resourceSet {
		resources = append(resources, resource)
	}
	return resources
}
