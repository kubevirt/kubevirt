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

package generic

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
)

const (
	failedCreateGenericHostDevicesFmt = "failed to create generic host-devices: %v"
	AliasPrefix                       = "hostdevice-"
	DefaultDisplayOff                 = false
)

func CreateHostDevices(vmi *v1.VirtualMachineInstance) ([]api.HostDevice, error) {
	vmiHostDevices := vmi.Spec.Domain.Devices.HostDevices

	hostDevices, err := CreateHostDevicesFromPools(vmiHostDevices,
		NewPCIAddressPool(vmiHostDevices), NewMDEVAddressPool(vmiHostDevices), NewUSBAddressPool(vmiHostDevices))
	if err != nil {
		return nil, err
	}

	draPCIHostDevices, err := getDRAPCIHostDevices(vmi)
	if err != nil {
		return nil, fmt.Errorf(failedCreateGenericHostDevicesFmt, err)
	}
	draMDEVHostDevices, err := getDRAMDEVHostDevices(vmi)
	if err != nil {
		return nil, fmt.Errorf(failedCreateGenericHostDevicesFmt, err)
	}

	hostDevices = append(hostDevices, draPCIHostDevices...)
	hostDevices = append(hostDevices, draMDEVHostDevices...)

	if err := validateCreationOfAllDevices(vmiHostDevices, hostDevices); err != nil {
		return nil, fmt.Errorf(failedCreateGenericHostDevicesFmt, err)
	}

	return hostDevices, nil
}

func hasHostDevicesWithDRA(vmi *v1.VirtualMachineInstance) bool {
	for _, hd := range vmi.Spec.Domain.Devices.HostDevices {
		if hd.ClaimRequest != nil {
			return true
		}
	}
	return false
}

func getDRAPCIHostDevices(vmi *v1.VirtualMachineInstance) ([]api.HostDevice, error) {
	hostDevices := []api.HostDevice{}
	if !hasHostDevicesWithDRA(vmi) {
		return hostDevices, nil
	}

	if vmi.Status.DeviceStatus != nil {
		for _, hdStatus := range vmi.Status.DeviceStatus.HostDeviceStatuses {
			if hdStatus.DeviceResourceClaimStatus != nil && hdStatus.DeviceResourceClaimStatus.Attributes != nil {
				if hdStatus.DeviceResourceClaimStatus.Attributes.PCIAddress != nil {
					hostAddr, err := device.NewPciAddressField(*hdStatus.DeviceResourceClaimStatus.Attributes.PCIAddress)
					if err != nil {
						return nil, fmt.Errorf("failed to create PCI device for %s: %v", hdStatus.Name, err)
					}
					hostDevices = append(hostDevices, api.HostDevice{
						Alias:   api.NewUserDefinedAlias(AliasPrefix + hdStatus.Name),
						Source:  api.HostDeviceSource{Address: hostAddr},
						Type:    api.HostDevicePCI,
						Managed: "no",
					})
				}
			}
		}
	}
	return hostDevices, nil
}

func getDRAMDEVHostDevices(vmi *v1.VirtualMachineInstance) ([]api.HostDevice, error) {
	hostDevices := []api.HostDevice{}
	if !hasHostDevicesWithDRA(vmi) {
		return hostDevices, nil
	}

	if vmi.Status.DeviceStatus != nil {
		for _, hdStatus := range vmi.Status.DeviceStatus.HostDeviceStatuses {
			if hdStatus.DeviceResourceClaimStatus != nil && hdStatus.DeviceResourceClaimStatus.Attributes != nil {
				if hdStatus.DeviceResourceClaimStatus.Attributes.PCIAddress != nil {
					continue
				}
				if hdStatus.DeviceResourceClaimStatus.Attributes.MDevUUID != nil {
					hostDevices = append(hostDevices, api.HostDevice{
						Alias:  api.NewUserDefinedAlias(AliasPrefix + hdStatus.Name),
						Source: api.HostDeviceSource{Address: &api.Address{UUID: *hdStatus.DeviceResourceClaimStatus.Attributes.MDevUUID}},
						Type:   api.HostDeviceMDev,
						Mode:   "subsystem",
						Model:  "vfio-pci",
					})
				}
			}
		}
	}
	return hostDevices, nil
}

func CreateHostDevicesFromPools(vmiHostDevices []v1.HostDevice, pciAddressPool, mdevAddressPool, usbAddressPool hostdevice.AddressPooler) ([]api.HostDevice, error) {
	pciPool := hostdevice.NewBestEffortAddressPool(pciAddressPool)
	mdevPool := hostdevice.NewBestEffortAddressPool(mdevAddressPool)
	usbPool := hostdevice.NewBestEffortAddressPool(usbAddressPool)

	hostDevicesMetaData := createHostDevicesMetadata(vmiHostDevices)
	pciHostDevices, err := hostdevice.CreatePCIHostDevices(hostDevicesMetaData, pciPool)
	if err != nil {
		return nil, fmt.Errorf(failedCreateGenericHostDevicesFmt, err)
	}
	mdevHostDevices, err := hostdevice.CreateMDEVHostDevices(hostDevicesMetaData, mdevPool, DefaultDisplayOff)
	if err != nil {
		return nil, fmt.Errorf(failedCreateGenericHostDevicesFmt, err)
	}

	hostDevices := append(pciHostDevices, mdevHostDevices...)

	usbHostDevices, err := hostdevice.CreateUSBHostDevices(hostDevicesMetaData, usbPool)
	if err != nil {
		return nil, err
	}

	return append(hostDevices, usbHostDevices...), nil
}

func createHostDevicesMetadata(vmiHostDevices []v1.HostDevice) []hostdevice.HostDeviceMetaData {
	var hostDevicesMetaData []hostdevice.HostDeviceMetaData
	for _, dev := range vmiHostDevices {
		hostDevicesMetaData = append(hostDevicesMetaData, hostdevice.HostDeviceMetaData{
			AliasPrefix:  AliasPrefix,
			Name:         dev.Name,
			ResourceName: dev.DeviceName,
		})
	}
	return hostDevicesMetaData
}

func validateCreationOfAllDevices(genericHostDevices []v1.HostDevice, hostDevices []api.HostDevice) error {
	hostDevsWithDP := []v1.HostDevice{}
	hostDevsWithDRA := []v1.HostDevice{}

	for _, hd := range genericHostDevices {
		if hd.ClaimRequest != nil {
			hostDevsWithDRA = append(hostDevsWithDRA, hd)
		} else {
			hostDevsWithDP = append(hostDevsWithDP, hd)
		}
	}

	if len(hostDevsWithDP) > 0 && len(hostDevsWithDP) != len(hostDevices) {
		return fmt.Errorf("the number of device plugin HostDevice/s do not match the number of devices:\nHostDevice: %v\nDevice: %v", hostDevsWithDP, hostDevices)
	}
	if len(hostDevsWithDRA) > 0 && len(hostDevsWithDRA) != len(hostDevices) {
		return fmt.Errorf("the number of DRA HostDevice/s do not match the number of devices:\nHostDevice: %v\nDevice: %v", hostDevsWithDRA, hostDevices)
	}
	return nil
}
