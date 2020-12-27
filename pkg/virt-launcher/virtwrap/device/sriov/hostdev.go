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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package sriov

import (
	"fmt"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

type pool interface {
	Pop(key string) (value string, err error)
}

func CreateHostDevices(vmi *v1.VirtualMachineInstance) ([]api.HostDevice, error) {
	SRIOVInterfaces := filterVMISRIOVInterfaces(vmi)
	return CreateHostDevicesFromIfacesAndPool(SRIOVInterfaces, NewPCIAddressPool(SRIOVInterfaces))
}

func CreateHostDevicesFromIfacesAndPool(SRIOVInterfaces []v1.Interface, pciAddrPool pool) ([]api.HostDevice, error) {
	var hostDevices []api.HostDevice

	for _, iface := range SRIOVInterfaces {
		pciAddress, err := pciAddrPool.Pop(iface.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to create SRIOV hostdevice for %s: %v", iface.Name, err)
		}

		hostDevice, err := createHostDevice(iface, pciAddress)
		if err != nil {
			return nil, err
		}
		hostDevices = append(hostDevices, *hostDevice)
		log.Log.Infof("SR-IOV PCI device created: %s", pciAddress)
	}
	return hostDevices, nil
}

func createHostDevice(iface v1.Interface, hostPCIAddress string) (*api.HostDevice, error) {
	hostAddr, err := device.NewPciAddressField(hostPCIAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create SRIOV device for %s, host PCI: %v", iface.Name, err)
	}
	hostDev := &api.HostDevice{
		Source:  api.HostDeviceSource{Address: hostAddr},
		Type:    "pci",
		Managed: "no",
	}

	guestPCIAddress := iface.PciAddress
	if guestPCIAddress != "" {
		addr, err := device.NewPciAddressField(guestPCIAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to create SRIOV device for %s, guest PCI: %v", iface.Name, err)
		}
		hostDev.Address = addr
	}

	if iface.BootOrder != nil {
		hostDev.BootOrder = &api.BootOrder{Order: *iface.BootOrder}
	}

	return hostDev, nil
}
