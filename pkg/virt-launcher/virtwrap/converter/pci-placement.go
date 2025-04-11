package converter

import (
	"fmt"
	"strconv"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func PlacePCIDevicesOnRootComplex(spec *api.DomainSpec) (err error) {
	assigner := newRootSlotAssigner()
	for i, iface := range spec.Devices.Interfaces {
		spec.Devices.Interfaces[i].Address, err = assigner.PlacePCIDeviceAtNextSlot(iface.Address)
		if err != nil {
			return err
		}
	}
	for i, hostDev := range spec.Devices.HostDevices {
		if hostDev.Type != api.HostDevicePCI {
			continue
		}
		spec.Devices.HostDevices[i].Address, err = assigner.PlacePCIDeviceAtNextSlot(hostDev.Address)
		if err != nil {
			return err
		}
	}
	for i, controller := range spec.Devices.Controllers {
		// pci-root and pcie-root devices can by definition hot have a pci address on its own
		if controller.Model == "pci-root" || controller.Model == "pcie-root" {
			continue
		}
		spec.Devices.Controllers[i].Address, err = assigner.PlacePCIDeviceAtNextSlot(controller.Address)
		if err != nil {
			return err
		}
	}
	for i, disk := range spec.Devices.Disks {
		if disk.Target.Bus != v1.DiskBusVirtio {
			continue
		}
		spec.Devices.Disks[i].Address, err = assigner.PlacePCIDeviceAtNextSlot(disk.Address)
		if err != nil {
			return err
		}
	}
	for i, input := range spec.Devices.Inputs {
		if input.Bus != v1.VirtIO {
			continue
		}
		spec.Devices.Inputs[i].Address, err = assigner.PlacePCIDeviceAtNextSlot(input.Address)
		if err != nil {
			return err
		}
	}
	for i, watchdog := range spec.Devices.Watchdogs {
		spec.Devices.Watchdogs[i].Address, err = assigner.PlacePCIDeviceAtNextSlot(watchdog.Address)
		if err != nil {
			return err
		}
	}
	if spec.Devices.Rng != nil {
		spec.Devices.Rng.Address, err = assigner.PlacePCIDeviceAtNextSlot(spec.Devices.Rng.Address)
		if err != nil {
			return err
		}
	}
	if spec.Devices.Ballooning != nil {
		spec.Devices.Ballooning.Address, err = assigner.PlacePCIDeviceAtNextSlot(spec.Devices.Ballooning.Address)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *pciRootSlotAssigner) nextSlot() (int, error) {
	slot := p.slot + 1
	// reserved slots are:
	// slot 0
	// slot 1 for VGA
	// slot 0x1f for a sata controller from  qemu
	// slot 0x1b for the first ich9 sound card
	switch slot {
	case 0, 0x01:
		slot = 0x02
	case 0x1f, 0x1b:
		slot = slot + 1
	}

	if slot >= 0x20 {
		return slot, fmt.Errorf("No space left on the root PCI bus.")
	}
	p.slot = slot
	return slot, nil
}

func newRootSlotAssigner() *pciRootSlotAssigner {
	return &pciRootSlotAssigner{slot: -1}
}

type pciRootSlotAssigner struct {
	slot int
}

func (p *pciRootSlotAssigner) PlacePCIDeviceAtNextSlot(address *api.Address) (*api.Address, error) {
	if address == nil {
		address = &api.Address{}
	}
	if address.Type != api.AddressPCI && address.Type != "" {
		return address, nil
	}

	// keep explicit requests for pci addresses
	if address.Domain != "" {
		return address, nil
	}

	slot, err := p.nextSlot()
	if err != nil {
		return nil, err
	}
	address.Type = api.AddressPCI
	address.Domain = "0x0000"
	address.Bus = "0x00"
	address.Slot = fmt.Sprintf("%#02x", slot)
	address.Function = "0x0"
	return address, nil
}

type pcieExpanderBusAssigner struct {
	domainSpec  *api.DomainSpec
	hasPCIeRoot bool
	index       int
}

func NewPCIeExpanderBusAssigner(domainSpec *api.DomainSpec) *pcieExpanderBusAssigner {
	return &pcieExpanderBusAssigner{
		domainSpec:  domainSpec,
		hasPCIeRoot: false,
		index:       1,
	}
}

func (a *pcieExpanderBusAssigner) addPCIeRoot() {
	if !a.hasPCIeRoot {
		a.hasPCIeRoot = true

		for _, controller := range a.domainSpec.Devices.Controllers {
			if controller.Model == api.ControllerModelPCIeRoot {
				return
			}
		}

		a.domainSpec.Devices.Controllers = append(
			a.domainSpec.Devices.Controllers,
			api.Controller{
				Type:  api.ControllerTypePCI,
				Index: "0",
				Model: api.ControllerModelPCIeRoot,
			},
		)
	}
}

func (a *pcieExpanderBusAssigner) addPCIeExpanderBus(numaNode *uint32) int {
	pcieExpanderBusIndex := a.index
	a.domainSpec.Devices.Controllers = append(
		a.domainSpec.Devices.Controllers,
		api.Controller{
			Type:  api.ControllerTypePCI,
			Index: strconv.Itoa(pcieExpanderBusIndex),
			Model: api.ControllerModelPCIeExpanderBus,
			Target: &api.ControllerTarget{
				Node: numaNode,
			},
		},
	)
	a.index = a.index + 1
	return pcieExpanderBusIndex
}

func (a *pcieExpanderBusAssigner) addPCIeRootPort(pcieExpanderBusIndex int) int {
	pcieRootPortIndex := a.index
	a.domainSpec.Devices.Controllers = append(
		a.domainSpec.Devices.Controllers,
		api.Controller{
			Type:  api.ControllerTypePCI,
			Index: strconv.Itoa(pcieRootPortIndex),
			Model: api.ControllerModelPCIeRootPort,
			Address: &api.Address{
				Type:     api.AddressPCI,
				Domain:   "0x0000",
				Bus:      fmt.Sprintf("%#02x", pcieExpanderBusIndex),
				Slot:     "0x00",
				Function: "0x0",
			},
		},
	)
	a.index = a.index + 1
	return pcieRootPortIndex
}

func (a *pcieExpanderBusAssigner) AddDevice(hostDevice *api.HostDevice) {
	guestOSNumaNode := hardware.LookupDeviceVCPUNumaNode(
		hostDevice.Source.Address,
		a.domainSpec,
	)

	if guestOSNumaNode == nil {
		return
	}

	a.addPCIeRoot()
	pcieExpanderBusIndex := a.addPCIeExpanderBus(guestOSNumaNode)
	pcieRootPortIndex := a.addPCIeRootPort(pcieExpanderBusIndex)

	hostDevice.Address = &api.Address{
		Type:     api.AddressPCI,
		Domain:   "0x0000",
		Bus:      fmt.Sprintf("%#02x", pcieRootPortIndex),
		Slot:     "0x00",
		Function: "0x0",
	}
}
