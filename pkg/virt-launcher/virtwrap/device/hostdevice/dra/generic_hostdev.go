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
	"kubevirt.io/client-go/log"

	drautil "kubevirt.io/kubevirt/pkg/dra"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

const (
	failedCreateGenericHostDevicesFmt = "failed to create dra generic host-devices: %v"
	DRAHostDeviceAliasPrefix          = "dra-hostdevice-"
)

// CreateDRAHostDevices creates host devices for HostDevices allocated via DRA.
func CreateDRAHostDevices(vmi *v1.VirtualMachineInstance, downwardAPIAttributes *drautil.DownwardAPIAttributes) ([]api.HostDevice, error) {
	var hostDevices []api.HostDevice
	if !hasHostDevicesWithDRA(vmi) {
		return hostDevices, nil
	}

	for _, hd := range vmi.Spec.Domain.Devices.HostDevices {
		if !drautil.IsHostDeviceDRA(hd) {
			continue
		}

		hostDevice, err := createHostDeviceForHostDevice(hd, downwardAPIAttributes)
		if err != nil {
			return nil, fmt.Errorf(failedCreateGenericHostDevicesFmt, err)
		}
		if hostDevice != nil {
			hostDevices = append(hostDevices, *hostDevice)
		}
	}

	if err := validateCreationOfDRAHostDevices(vmi.Spec.Domain.Devices.HostDevices, hostDevices); err != nil {
		return nil, fmt.Errorf(failedCreateGenericHostDevicesFmt, err)
	}

	return hostDevices, nil
}

func createHostDeviceForHostDevice(hd v1.HostDevice, downwardAPIAttributes *drautil.DownwardAPIAttributes) (*api.HostDevice, error) {
	if hd.ClaimRequest == nil || hd.ClaimRequest.ClaimName == nil || hd.ClaimRequest.RequestName == nil {
		return nil, fmt.Errorf("HostDevice %s has incomplete ClaimRequest", hd.Name)
	}

	claimName := *hd.ClaimRequest.ClaimName
	requestName := *hd.ClaimRequest.RequestName

	if pciAddr, err := downwardAPIAttributes.GetPCIAddressForClaim(claimName, requestName); err == nil {
		log.Log.V(2).Infof("Adding DRA PCI HostDevice for %s", hd.Name)
		hostAddr, err := device.NewPciAddressField(pciAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to create PCI device for %s: %v", hd.Name, err)
		}
		return &api.HostDevice{
			Alias:   api.NewUserDefinedAlias(DRAHostDeviceAliasPrefix + hd.Name),
			Source:  api.HostDeviceSource{Address: hostAddr},
			Type:    api.HostDevicePCI,
			Managed: "no",
		}, nil
	}

	if mdevUUID, err := downwardAPIAttributes.GetMDevUUIDForClaim(claimName, requestName); err == nil {
		log.Log.V(2).Infof("Adding DRA MDEV HostDevice for %s", hd.Name)
		return &api.HostDevice{
			Alias: api.NewUserDefinedAlias(DRAHostDeviceAliasPrefix + hd.Name),
			Source: api.HostDeviceSource{
				Address: &api.Address{
					UUID: mdevUUID,
				},
			},
			Type:  api.HostDeviceMDev,
			Mode:  "subsystem",
			Model: "vfio-pci",
		}, nil
	}

	return nil, fmt.Errorf("HostDevice %s has no pciBusID or mdevUUID in metadata for claim %s request %s", hd.Name, claimName, requestName)
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
