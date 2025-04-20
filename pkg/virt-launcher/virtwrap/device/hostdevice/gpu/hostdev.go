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

	if err := validateCreationOfAllDevices(vmiGPUs, hostDevices); err != nil {
		return nil, fmt.Errorf(failedCreateGPUHostDeviceFmt, err)
	}

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

// validateCreationOfAllDevices validates that all specified GPU/s have a matching host-device.
// On validation failure, an error is returned.
// The validation assumes that the assignment of a device to a specified GPU is correct,
// therefore a simple quantity check is sufficient.
func validateCreationOfAllDevices(gpus []v1.GPU, hostDevices []api.HostDevice) error {
	if len(gpus) != len(hostDevices) {
		return fmt.Errorf(
			"the number of GPU/s do not match the number of devices:\nGPU: %v\nDevice: %v", gpus, hostDevices,
		)
	}
	return nil
}
