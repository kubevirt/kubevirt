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

package pci_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/pci"
)

var _ = Describe("DisableHotplugOnOccupiedRootPorts", func() {
	pciAddr := func(bus string) *api.Address {
		return &api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: bus, Slot: "0x00", Function: "0x0"}
	}

	uint32Ptr := func(v uint32) *uint32 { return &v }

	newRootPortController := func(index string, busNr uint32) api.Controller {
		return api.Controller{
			Type:   api.ControllerTypePCI,
			Index:  index,
			Model:  api.ControllerModelPCIeRootPort,
			Target: &api.ControllerTarget{BusNr: uint32Ptr(busNr)},
		}
	}

	expectHotplugOff := func(controller api.Controller, desc string) {
		ExpectWithOffset(1, controller.Target).NotTo(BeNil(), desc)
		ExpectWithOffset(1, controller.Target.Hotplug).To(Equal("off"), desc)
	}

	expectHotplugUnchanged := func(controller api.Controller, desc string) {
		if controller.Target == nil {
			return
		}
		ExpectWithOffset(1, controller.Target.Hotplug).To(BeEmpty(), desc)
	}

	It("should disable hotplug on occupied ports and leave empty ones unchanged", func() {
		spec := &api.DomainSpec{
			Devices: api.Devices{
				Controllers: []api.Controller{
					{Type: api.ControllerTypePCI, Index: "0", Model: api.ControllerModelPCIeRoot},
					newRootPortController("1", 1),
					newRootPortController("2", 2),
					newRootPortController("3", 3),
				},
				Interfaces: []api.Interface{{Address: pciAddr("0x01")}},
				Ballooning: &api.MemBalloon{Address: pciAddr("0x02")},
			},
		}

		pci.DisableHotplugOnOccupiedRootPorts(spec)

		expectHotplugUnchanged(spec.Devices.Controllers[1], "port with NIC stays hotpluggable")
		expectHotplugOff(spec.Devices.Controllers[2], "port with balloon")
		expectHotplugUnchanged(spec.Devices.Controllers[3], "empty port")
	})

	It("should detect all non-interface PCI device types", func() {
		spec := &api.DomainSpec{
			Devices: api.Devices{
				Controllers: []api.Controller{
					{Type: api.ControllerTypePCI, Index: "0", Model: api.ControllerModelPCIeRoot},
					newRootPortController("1", 1), newRootPortController("2", 2), newRootPortController("3", 3), newRootPortController("4", 4),
					newRootPortController("5", 5), newRootPortController("6", 6), newRootPortController("7", 7), newRootPortController("8", 8),
					newRootPortController("9", 9),
				},
				Interfaces:  []api.Interface{{Address: pciAddr("0x01")}},
				Disks:       []api.Disk{{Target: api.DiskTarget{Bus: v1.DiskBusVirtio}, Address: pciAddr("0x02")}},
				Inputs:      []api.Input{{Bus: v1.VirtIO, Address: pciAddr("0x03")}},
				Watchdogs:   []api.Watchdog{{Address: pciAddr("0x04")}},
				HostDevices: []api.HostDevice{{Type: api.HostDevicePCI, Address: pciAddr("0x05")}},
				Ballooning:  &api.MemBalloon{Address: pciAddr("0x06")},
				Rng:         &api.Rng{Address: pciAddr("0x07")},
				Memory:      &api.MemoryDevice{Address: pciAddr("0x08")},
			},
		}

		pci.DisableHotplugOnOccupiedRootPorts(spec)

		expectHotplugUnchanged(spec.Devices.Controllers[1], "NIC port stays hotpluggable")
		expectHotplugOff(spec.Devices.Controllers[2], "virtio disk on bus 2")
		expectHotplugOff(spec.Devices.Controllers[3], "virtio input on bus 3")
		expectHotplugOff(spec.Devices.Controllers[4], "watchdog on bus 4")
		expectHotplugOff(spec.Devices.Controllers[5], "PCI host device on bus 5")
		expectHotplugOff(spec.Devices.Controllers[6], "memory balloon on bus 6")
		expectHotplugOff(spec.Devices.Controllers[7], "RNG on bus 7")
		expectHotplugOff(spec.Devices.Controllers[8], "memory device on bus 8")
		expectHotplugUnchanged(spec.Devices.Controllers[9], "empty port")
	})

	It("should not modify non-pcie-root-port controllers", func() {
		spec := &api.DomainSpec{
			Devices: api.Devices{
				Controllers: []api.Controller{
					{Type: api.ControllerTypePCI, Index: "0", Model: api.ControllerModelPCIeRoot},
					{Type: "usb", Index: "0", Model: "none"},
					{Type: "scsi", Index: "0", Model: "virtio-scsi", Address: pciAddr("0x05")},
				},
			},
		}

		pci.DisableHotplugOnOccupiedRootPorts(spec)

		for _, ctrl := range spec.Devices.Controllers {
			Expect(ctrl.Target).To(BeNil())
		}
	})

	It("should skip root ports with nil Target (no BusNr available)", func() {
		spec := &api.DomainSpec{
			Devices: api.Devices{
				Controllers: []api.Controller{
					{Type: api.ControllerTypePCI, Index: "1", Model: api.ControllerModelPCIeRootPort},
				},
				Ballooning: &api.MemBalloon{Address: pciAddr("0x01")},
			},
		}

		pci.DisableHotplugOnOccupiedRootPorts(spec)

		expectHotplugUnchanged(spec.Devices.Controllers[0], "nil Target means no BusNr, port skipped")
	})

	It("should preserve existing Target fields when setting hotplug off", func() {
		busNr := uint32(1)
		numaNode := uint32(0)
		spec := &api.DomainSpec{
			Devices: api.Devices{
				Controllers: []api.Controller{
					{
						Type:  api.ControllerTypePCI,
						Index: "1",
						Model: api.ControllerModelPCIeRootPort,
						Target: &api.ControllerTarget{
							BusNr:    &busNr,
							NUMANode: &numaNode,
						},
					},
				},
				Ballooning: &api.MemBalloon{Address: pciAddr("0x01")},
			},
		}

		pci.DisableHotplugOnOccupiedRootPorts(spec)

		target := spec.Devices.Controllers[0].Target
		Expect(target).NotTo(BeNil())
		Expect(target.Hotplug).To(Equal("off"))
		Expect(target.BusNr).To(Equal(&busNr))
		Expect(target.NUMANode).To(Equal(&numaNode))
	})

	It("should mark a root port as occupied when a child controller sits on its bus", func() {
		spec := &api.DomainSpec{
			Devices: api.Devices{
				Controllers: []api.Controller{
					{Type: api.ControllerTypePCI, Index: "0", Model: api.ControllerModelPCIeRoot},
					newRootPortController("1", 1),
					newRootPortController("5", 5),
					{Type: "scsi", Index: "0", Model: "virtio-scsi", Address: pciAddr("0x05")},
				},
				Ballooning: &api.MemBalloon{Address: pciAddr("0x01")},
			},
		}

		pci.DisableHotplugOnOccupiedRootPorts(spec)

		expectHotplugOff(spec.Devices.Controllers[1], "balloon on bus 1")
		expectHotplugOff(spec.Devices.Controllers[2], "SCSI controller on bus 5")
	})

	It("should not mark interface ports as occupied to preserve NIC hot-unplug", func() {
		spec := &api.DomainSpec{
			Devices: api.Devices{
				Controllers: []api.Controller{
					{Type: api.ControllerTypePCI, Index: "0", Model: api.ControllerModelPCIeRoot},
					newRootPortController("1", 1),
					newRootPortController("2", 2),
					newRootPortController("3", 3),
				},
				Interfaces: []api.Interface{
					{Address: pciAddr("0x01")},
					{Address: pciAddr("0x02")},
					{Address: pciAddr("0x03")},
				},
			},
		}

		pci.DisableHotplugOnOccupiedRootPorts(spec)

		expectHotplugUnchanged(spec.Devices.Controllers[1], "NIC on bus 1")
		expectHotplugUnchanged(spec.Devices.Controllers[2], "NIC on bus 2")
		expectHotplugUnchanged(spec.Devices.Controllers[3], "NIC on bus 3")
	})

	It("should ignore non-PCI device addresses (SATA disks, USB inputs)", func() {
		driveAddr := &api.Address{Type: "drive", Bus: "0", Controller: "0", Target: "0", Unit: "0"}
		usbAddr := &api.Address{Type: "usb", Bus: "0"}

		spec := &api.DomainSpec{
			Devices: api.Devices{
				Controllers: []api.Controller{
					newRootPortController("0", 0),
					newRootPortController("1", 1),
				},
				Disks:  []api.Disk{{Target: api.DiskTarget{Bus: "sata"}, Address: driveAddr}},
				Inputs: []api.Input{{Type: "tablet", Bus: "usb", Address: usbAddr}},
			},
		}

		pci.DisableHotplugOnOccupiedRootPorts(spec)

		expectHotplugUnchanged(spec.Devices.Controllers[0], "SATA disk drive address must not match")
		expectHotplugUnchanged(spec.Devices.Controllers[1], "USB input address must not match")
	})
})
