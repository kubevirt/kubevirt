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

package dra

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	drautil "kubevirt.io/kubevirt/pkg/dra"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

const (
	failedCreateGenericHostDevicesFmt = "failed to create dra generic host-devices: %v"
	DRAHostDeviceAliasPrefix          = "dra-hostdevice-"
)

func CreateDRAHostDevices(vmi *v1.VirtualMachineInstance) ([]api.HostDevice, error) {
	var hostDevices []api.HostDevice
	if !hasHostDevicesWithDRA(vmi) {
		return hostDevices, nil
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

	if err := validateCreationOfDRAHostDevices(vmi.Spec.Domain.Devices.HostDevices, hostDevices); err != nil {
		return nil, fmt.Errorf(failedCreateGenericHostDevicesFmt, err)
	}

	return hostDevices, nil
}

func getDRAPCIHostDevices(vmi *v1.VirtualMachineInstance) ([]api.HostDevice, error) {
	hostDevices := []api.HostDevice{}
	if vmi.Status.DeviceStatus == nil {
		return hostDevices, fmt.Errorf("vmi has dra host-devices devices but no device status found")
	}

	for _, hdStatus := range vmi.Status.DeviceStatus.HostDeviceStatuses {
		hdStatus := hdStatus.DeepCopy()
		if hdStatus.DeviceResourceClaimStatus != nil && hdStatus.DeviceResourceClaimStatus.Attributes != nil {
			if hdStatus.DeviceResourceClaimStatus.Attributes.PCIAddress != nil {
				hostAddr, err := device.NewPciAddressField(*hdStatus.DeviceResourceClaimStatus.Attributes.PCIAddress)
				if err != nil {
					return nil, fmt.Errorf("failed to create PCI device for %s: %v", hdStatus.Name, err)
				}
				hostDevices = append(hostDevices, api.HostDevice{
					Alias:   api.NewUserDefinedAlias(DRAHostDeviceAliasPrefix + hdStatus.Name),
					Source:  api.HostDeviceSource{Address: hostAddr},
					Type:    api.HostDevicePCI,
					Managed: "no",
				})
			}
		}
	}
	return hostDevices, nil
}

func getDRAMDEVHostDevices(vmi *v1.VirtualMachineInstance) ([]api.HostDevice, error) {
	hostDevices := []api.HostDevice{}
	if vmi.Status.DeviceStatus == nil {
		return hostDevices, fmt.Errorf("vmi has dra host-devices devices but no device status found")
	}

	for _, hdStatus := range vmi.Status.DeviceStatus.HostDeviceStatuses {
		hdStatus := hdStatus.DeepCopy()
		if hdStatus.DeviceResourceClaimStatus != nil && hdStatus.DeviceResourceClaimStatus.Attributes != nil {
			if hdStatus.DeviceResourceClaimStatus.Attributes.PCIAddress != nil {
				continue
			}
			if hdStatus.DeviceResourceClaimStatus.Attributes.MDevUUID != nil {
				hostDevices = append(hostDevices, api.HostDevice{
					Alias:  api.NewUserDefinedAlias(DRAHostDeviceAliasPrefix + hdStatus.Name),
					Source: api.HostDeviceSource{Address: &api.Address{UUID: *hdStatus.DeviceResourceClaimStatus.Attributes.MDevUUID}},
					Type:   api.HostDeviceMDev,
					Mode:   "subsystem",
					Model:  "vfio-pci",
				})
			}
		}
	}
	return hostDevices, nil
}

func validateCreationOfDRAHostDevices(genericHostDevices []v1.HostDevice, hostDevices []api.HostDevice) error {
	var hostDevsWithDRA []v1.HostDevice
	for _, hd := range genericHostDevices {
		if drautil.IsHostDeviceDRA(hd) {
			hostDevsWithDRA = append(hostDevsWithDRA, hd)
		}
	}

	if len(hostDevsWithDRA) > 0 && len(hostDevsWithDRA) != len(hostDevices) {
		return fmt.Errorf("the number of DRA HostDevice/s do not match the number of devices:\nHostDevice: %v\nDevice: %v", hostDevsWithDRA, hostDevices)
	}
	return nil
}

func hasHostDevicesWithDRA(vmi *v1.VirtualMachineInstance) bool {
	for _, hd := range vmi.Spec.Domain.Devices.HostDevices {
		if drautil.IsHostDeviceDRA(hd) {
			return true
		}
	}
	return false
}
