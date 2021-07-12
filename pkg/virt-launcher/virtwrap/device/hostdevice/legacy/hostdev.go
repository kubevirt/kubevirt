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

package legacy

import (
	"fmt"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

type CreateHostDevice func(string) (*api.HostDevice, error)

type addressPool interface {
	Len() int
	Pop() (value string, err error)
}

func CreateGPUHostDevices() ([]api.HostDevice, error) {
	return CreateGPUHostDevicesFromPool(NewGPUPCIAddressPool())
}

func CreateVGPUHostDevices() ([]api.HostDevice, error) {
	return CreateVGPUHostDevicesFromPool(NewVGPUMdevAddressPool())
}

func CreateGPUHostDevicesFromPool(pool addressPool) ([]api.HostDevice, error) {
	return createHostDevicesFromPool(pool, createPCIHostDevice)
}

func CreateVGPUHostDevicesFromPool(pool addressPool) ([]api.HostDevice, error) {
	return createHostDevicesFromPool(pool, createMDEVHostDevice)
}

func createHostDevicesFromPool(pool addressPool, createHostDevice CreateHostDevice) ([]api.HostDevice, error) {
	var hostDevices []api.HostDevice

	for pool.Len() > 0 {
		address, err := pool.Pop()
		if err != nil {
			return nil, fmt.Errorf("failed to create hostdevice: %v", err)
		}

		hostDevice, err := createHostDevice(address)
		if err != nil {
			return nil, err
		}
		hostDevices = append(hostDevices, *hostDevice)
		log.Log.Infof("host device created: %s", address)
	}
	return hostDevices, nil
}

func createPCIHostDevice(pciAddress string) (*api.HostDevice, error) {
	address, err := device.NewPciAddressField(pciAddress)
	if err != nil {
		return nil, err
	}

	return &api.HostDevice{
		Source: api.HostDeviceSource{
			Address: address,
		},
		Type:    "pci",
		Managed: "yes",
	}, nil
}

func createMDEVHostDevice(mdevUUID string) (*api.HostDevice, error) {
	return &api.HostDevice{
		Source: api.HostDeviceSource{
			Address: &api.Address{
				UUID: mdevUUID,
			},
		},
		Type:  "mdev",
		Mode:  "subsystem",
		Model: "vfio-pci",
	}, nil
}
