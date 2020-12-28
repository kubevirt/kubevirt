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

package sriov_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/sriov"
)

const (
	netname1 = "net1"
	netname2 = "net2"
)

var _ = Describe("SRIOV HostDevice", func() {
	It("creates no device given no interfaces", func() {
		vmi := &v1.VirtualMachineInstance{}

		Expect(sriov.CreateHostDevices(vmi)).To(BeEmpty())
	})

	It("creates no device given no SRIOV interfaces", func() {
		iface := v1.Interface{}
		iface.Masquerade = &v1.InterfaceMasquerade{}
		vmi := &v1.VirtualMachineInstance{}
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface}

		Expect(sriov.CreateHostDevices(vmi)).To(BeEmpty())
	})

	It("fails to create device given no available host PCI", func() {
		iface := newSRIOVInterface("test")
		vmi := &v1.VirtualMachineInstance{}
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface}

		_, err := sriov.CreateHostDevices(vmi)

		Expect(err).To(HaveOccurred())
	})

	It("fails to create a device given bad host PCI address", func() {
		ifaces := []v1.Interface{newSRIOVInterface("net1")}
		pool := newPCIAddressPoolStub("0bad0pci0address0")

		_, err := sriov.CreateHostDevicesFromIfacesAndPool(ifaces, pool)

		Expect(err).To(HaveOccurred())
	})

	It("fails to create a device given bad guest PCI address", func() {
		iface := newSRIOVInterface("net1")
		iface.PciAddress = "0bad0pci0address0"
		pool := newPCIAddressPoolStub("0000:81:01.0")

		_, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface}, pool)

		Expect(err).To(HaveOccurred())
	})

	It("fails to create a device given two interfaces but only one host PCI", func() {
		iface1 := newSRIOVInterface(netname1)
		iface2 := newSRIOVInterface(netname1)
		pool := newPCIAddressPoolStub("0000:81:01.0")

		_, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface1, iface2}, pool)

		Expect(err).To(HaveOccurred())
	})

	It("creates 2 devices that are connected to the same network", func() {
		iface1 := newSRIOVInterface(netname1)
		iface2 := newSRIOVInterface(netname1)
		pool := newPCIAddressPoolStub("0000:81:01.0", "0000:81:01.1")

		devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface1, iface2}, pool)

		hostPCIAddress1 := api.Address{Type: "pci", Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
		expectHostDevice1 := api.HostDevice{
			Alias:   newSRIOVAlias(netname1),
			Source:  api.HostDeviceSource{Address: &hostPCIAddress1},
			Type:    "pci",
			Managed: "no",
		}
		hostPCIAddress2 := api.Address{Type: "pci", Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x1"}
		expectHostDevice2 := api.HostDevice{
			Alias:   newSRIOVAlias(netname1),
			Source:  api.HostDeviceSource{Address: &hostPCIAddress2},
			Type:    "pci",
			Managed: "no",
		}
		Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
	})

	It("creates 2 devices that are connected to different networks", func() {
		iface1 := newSRIOVInterface(netname1)
		iface2 := newSRIOVInterface(netname2)
		pool := newPCIAddressPoolStub("0000:81:01.0", "0000:81:02.0")

		devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface1, iface2}, pool)

		hostPCIAddress1 := api.Address{Type: "pci", Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
		expectHostDevice1 := api.HostDevice{
			Alias:   newSRIOVAlias(netname1),
			Source:  api.HostDeviceSource{Address: &hostPCIAddress1},
			Type:    "pci",
			Managed: "no",
		}
		hostPCIAddress2 := api.Address{Type: "pci", Domain: "0x0000", Bus: "0x81", Slot: "0x02", Function: "0x0"}
		expectHostDevice2 := api.HostDevice{
			Alias:   newSRIOVAlias(netname2),
			Source:  api.HostDeviceSource{Address: &hostPCIAddress2},
			Type:    "pci",
			Managed: "no",
		}
		Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
	})

	It("creates 1 device that includes guest PCI addresses", func() {
		iface := newSRIOVInterface(netname1)
		iface.PciAddress = "0000:01:01.0"
		pool := newPCIAddressPoolStub("0000:81:01.0", "0000:81:02.0")

		devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface}, pool)

		hostPCIAddress1 := api.Address{Type: "pci", Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
		guestPCIAddress1 := api.Address{Type: "pci", Domain: "0x0000", Bus: "0x01", Slot: "0x01", Function: "0x0"}
		expectHostDevice1 := api.HostDevice{
			Alias:   newSRIOVAlias(netname1),
			Source:  api.HostDeviceSource{Address: &hostPCIAddress1},
			Type:    "pci",
			Managed: "no",
			Address: &guestPCIAddress1,
		}
		Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1}))
	})

	It("creates 1 device that includes boot-order", func() {
		iface := newSRIOVInterface(netname1)
		val := uint(1)
		iface.BootOrder = &val
		pool := newPCIAddressPoolStub("0000:81:01.0", "0000:81:02.0")

		devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface}, pool)

		hostPCIAddress1 := api.Address{Type: "pci", Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
		expectHostDevice1 := api.HostDevice{
			Alias:     newSRIOVAlias(netname1),
			Source:    api.HostDeviceSource{Address: &hostPCIAddress1},
			Type:      "pci",
			Managed:   "no",
			BootOrder: &api.BootOrder{Order: *iface.BootOrder},
		}
		Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1}))
	})
})

func newSRIOVAlias(netName string) *api.Alias {
	return api.NewUserDefinedAlias(sriov.AliasPrefix + netName)
}

type stubPCIAddressPool struct {
	pciAddresses []string
}

func newPCIAddressPoolStub(PCIAddresses ...string) *stubPCIAddressPool {
	return &stubPCIAddressPool{PCIAddresses}
}

func (p *stubPCIAddressPool) Pop(_ string) (string, error) {
	if len(p.pciAddresses) == 0 {
		return "", fmt.Errorf("pool is empty")
	}

	address := p.pciAddresses[0]
	p.pciAddresses = p.pciAddresses[1:]

	return address, nil
}
