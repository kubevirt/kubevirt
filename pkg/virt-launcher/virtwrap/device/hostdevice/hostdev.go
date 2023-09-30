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
	"strings"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

const failedCreateHostDeviceFmt = "failed to create hostdevice for %s: %v"

type HostDeviceMetaData struct {
	AliasPrefix       string
	Name              string
	ResourceName      string
	VirtualGPUOptions *v1.VGPUOptions
	// DecorateHook is a function pointer that may be used to mutate the domain host-device
	// with additional specific parameters. E.g. guest PCI address.
	DecorateHook func(hostDevice *api.HostDevice) error
}

type createHostDevice func(HostDeviceMetaData, string) (*api.HostDevice, error)

type AddressPooler interface {
	Pop(key string) (value string, err error)
}

func CreatePCIHostDevices(hostDevicesData []HostDeviceMetaData, pciAddrPool AddressPooler) ([]api.HostDevice, error) {
	return createHostDevices(hostDevicesData, pciAddrPool, createPCIHostDevice)
}

func CreateMDEVHostDevices(hostDevicesData []HostDeviceMetaData, mdevAddrPool AddressPooler, enableDefaultDisplay bool) ([]api.HostDevice, error) {
	if enableDefaultDisplay {
		return createHostDevices(hostDevicesData, mdevAddrPool, createMDEVHostDeviceWithDisplay)
	}
	return createHostDevices(hostDevicesData, mdevAddrPool, createMDEVHostDevice)
}

func CreateUSBHostDevices(hostDevicesData []HostDeviceMetaData, usbAddrPool AddressPooler) ([]api.HostDevice, error) {
	return createHostDevices(hostDevicesData, usbAddrPool, createUSBHostDevice)
}

func createHostDevices(hostDevicesData []HostDeviceMetaData, addrPool AddressPooler, createHostDev createHostDevice) ([]api.HostDevice, error) {
	var (
		hostDevices          []api.HostDevice
		hostDevicesAddresses []string
	)

	for _, hostDeviceData := range hostDevicesData {
		address, err := addrPool.Pop(hostDeviceData.ResourceName)
		if err != nil {
			return nil, fmt.Errorf(failedCreateHostDeviceFmt, hostDeviceData.Name, err)
		}

		// if pop succeeded but the address is empty, ignore the device and let the caller
		// decide if this is acceptable or not.
		if address == "" {
			continue
		}

		hostDevice, err := createHostDev(hostDeviceData, address)
		if err != nil {
			return nil, fmt.Errorf(failedCreateHostDeviceFmt, hostDeviceData.Name, err)
		}
		if hostDeviceData.DecorateHook != nil {
			if err := hostDeviceData.DecorateHook(hostDevice); err != nil {
				return nil, fmt.Errorf(failedCreateHostDeviceFmt, hostDeviceData.Name, err)
			}
		}
		hostDevices = append(hostDevices, *hostDevice)
		hostDevicesAddresses = append(hostDevicesAddresses, address)
	}

	if len(hostDevices) > 0 {
		log.Log.Infof("host-devices created: [%s]", strings.Join(hostDevicesAddresses, ", "))
	}

	return hostDevices, nil
}

func createPCIHostDevice(hostDeviceData HostDeviceMetaData, hostPCIAddress string) (*api.HostDevice, error) {
	hostAddr, err := device.NewPciAddressField(hostPCIAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create PCI device for %s: %v", hostDeviceData.Name, err)
	}
	domainHostDevice := &api.HostDevice{
		Alias:   api.NewUserDefinedAlias(hostDeviceData.AliasPrefix + hostDeviceData.Name),
		Source:  api.HostDeviceSource{Address: hostAddr},
		Type:    api.HostDevicePCI,
		Managed: "no",
	}
	return domainHostDevice, nil
}

func createMDEVHostDeviceWithDisplay(hostDeviceData HostDeviceMetaData, mdevUUID string) (*api.HostDevice, error) {
	mdev, err := createMDEVHostDevice(hostDeviceData, mdevUUID)
	if err != nil {
		return mdev, err
	}
	if hostDeviceData.VirtualGPUOptions != nil {
		if hostDeviceData.VirtualGPUOptions.Display != nil {
			displayEnabled := hostDeviceData.VirtualGPUOptions.Display.Enabled
			if displayEnabled == nil || *displayEnabled {
				mdev.Display = "on"
				if hostDeviceData.VirtualGPUOptions.Display.RamFB == nil || *hostDeviceData.VirtualGPUOptions.Display.RamFB.Enabled {
					mdev.RamFB = "on"
				}
			}
		}
	} else {
		mdev.Display = "on"
		mdev.RamFB = "on"
	}
	return mdev, nil
}

func createMDEVHostDevice(hostDeviceData HostDeviceMetaData, mdevUUID string) (*api.HostDevice, error) {
	domainHostDevice := &api.HostDevice{
		Alias: api.NewUserDefinedAlias(hostDeviceData.AliasPrefix + hostDeviceData.Name),
		Source: api.HostDeviceSource{
			Address: &api.Address{
				UUID: mdevUUID,
			},
		},
		Type:  api.HostDeviceMDev,
		Mode:  "subsystem",
		Model: "vfio-pci",
	}
	return domainHostDevice, nil
}

func createUSBHostDevice(device HostDeviceMetaData, usbAddress string) (*api.HostDevice, error) {
	strs := strings.Split(usbAddress, ":")
	if len(strs) != 2 {
		return nil, fmt.Errorf("Bad value: %s", usbAddress)
	}
	bus, deviceNumber := strs[0], strs[1]

	return &api.HostDevice{
		Type:  "usb",
		Mode:  "subsystem",
		Alias: api.NewUserDefinedAlias("usb-host-" + device.Name),
		Source: api.HostDeviceSource{
			Address: &api.Address{
				Bus:    bus,
				Device: deviceNumber,
			},
		},
	}, nil
}
