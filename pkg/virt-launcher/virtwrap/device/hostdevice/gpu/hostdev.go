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
	"strings"

	v1 "kubevirt.io/api/core/v1"

	drautil "kubevirt.io/kubevirt/pkg/dra"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
	iommupci "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/iommu-pci"
)

const (
	failedCreateGPUHostDeviceFmt = "failed to create GPU host-devices: %v"
	AliasPrefix                  = "gpu-"
	DefaultDisplayOn             = true
	acpiNodeSetPending           = "tofill"
)

func CreateHostDevices(vmiGPUs []v1.GPU, iommuPCI *iommupci.IommuPCI) ([]api.HostDevice, error) {
	return CreateHostDevicesFromPools(vmiGPUs, NewPCIAddressPool(vmiGPUs), NewMDEVAddressPool(vmiGPUs), iommuPCI)
}

func CreateHostDevicesFromPools(vmiGPUs []v1.GPU, pciAddressPool, mdevAddressPool hostdevice.AddressPooler, iommuPCI *iommupci.IommuPCI) ([]api.HostDevice, error) {
	pciPool := hostdevice.NewBestEffortAddressPool(pciAddressPool)
	mdevPool := hostdevice.NewBestEffortAddressPool(mdevAddressPool)

	hostDevicesMetaData := createHostDevicesMetadata(vmiGPUs, iommuPCI)
	pciHostDevices, err := hostdevice.CreatePCIHostDevices(hostDevicesMetaData, pciPool)
	if err != nil {
		return nil, fmt.Errorf(failedCreateGPUHostDeviceFmt, err)
	}
	mdevHostDevices, err := hostdevice.CreateMDEVHostDevices(hostDevicesMetaData, mdevPool, DefaultDisplayOn)
	if err != nil {
		return nil, fmt.Errorf(failedCreateGPUHostDeviceFmt, err)
	}

	hostDevices := append(pciHostDevices, mdevHostDevices...)

	if err := validateCreationOfDevicePluginsDevices(vmiGPUs, hostDevices); err != nil {
		return nil, fmt.Errorf(failedCreateGPUHostDeviceFmt, err)
	}

	return hostDevices, nil
}

func createHostDevicesMetadata(vmiGPUs []v1.GPU, iommuPCI *iommupci.IommuPCI) []hostdevice.HostDeviceMetaData {
	var hostDevicesMetaData []hostdevice.HostDeviceMetaData
	for _, dev := range vmiGPUs {
		hostDevicesMetaData = append(hostDevicesMetaData, hostdevice.HostDeviceMetaData{
			AliasPrefix:       AliasPrefix,
			Name:              dev.Name,
			ResourceName:      dev.DeviceName,
			VirtualGPUOptions: dev.VirtualGPUOptions,
			DecorateHook:      newDecorateHook(dev.DeviceName, iommuPCI),
		})
	}
	return hostDevicesMetaData
}

// newDecorateHook creates a decoration function that configures IOMMU settings
// for GPU host devices on systems that support advanced IOMMU features.
//
// This function is specifically designed for NVIDIA GPU passthrough on ARM64
// systems with SMMUv3. It performs two main configurations:
//
//  1. IOMMUFD driver setup: Detects if the modern IOMMUFD interface is available
//     and configures the host device to use it instead of legacy VFIO.
//
//  2. ACPI NodeSet marking: On SMMUv3-enabled systems, marks devices with
//     "tofill" so that fake NUMA nodes can be created later in the conversion
//     pipeline.
func newDecorateHook(name string, iommuPCI *iommupci.IommuPCI) func(hostDevice *api.HostDevice) error {
	return func(hostDevice *api.HostDevice) error {
		if iommuPCI == nil {
			return nil
		}

		if !strings.Contains(name, "nvidia.com") {
			return nil
		}

		if hostDevice.Source.Address == nil {
			return nil
		}

		if iommuPCI.IommufdEnabled == nil {
			return nil
		}

		if *iommuPCI.IommufdEnabled {
			hostDevice.Driver = &api.HostDevDriver{
				Iommufd: "yes",
			}
		}

		if iommuPCI.SMMUEnabled {
			hostDevice.ACPI = &api.ACPIHostDev{
				NodeSet: acpiNodeSetPending,
			}
		}

		return nil
	}
}

// validateCreationOfDevicePluginsDevices validates that all specified GPU/s have a matching host-device.
// On validation failure, an error is returned.
// The validation assumes that the assignment of a device to a specified GPU is correct,
// therefore a simple quantity check is sufficient.
func validateCreationOfDevicePluginsDevices(gpus []v1.GPU, hostDevices []api.HostDevice) error {
	var gpusWithDP []v1.GPU
	for _, gpu := range gpus {
		if !drautil.IsGPUDRA(gpu) {
			gpusWithDP = append(gpusWithDP, gpu)
		}
	}

	if len(gpusWithDP) > 0 && len(gpusWithDP) != len(hostDevices) {
		return fmt.Errorf(
			"the number of device plugin GPU/s do not match the number of devices:\nGPU: %v\nDevice: %v", gpusWithDP, hostDevices,
		)
	}
	return nil
}
