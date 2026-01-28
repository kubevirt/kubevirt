package converter

import (
	"fmt"

	"k8s.io/utils/ptr"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	maxExpanderBusNr = 254
)

// iteratePCIAddresses invokes the callback function for each PCI device specified in the domain
func iteratePCIAddresses(spec *api.DomainSpec, callback func(address *api.Address) (*api.Address, error)) (err error) {
	fn := func(address *api.Address) (*api.Address, error) {
		if address == nil || address.Type == "" || address.Type == api.AddressPCI {
			return callback(address)
		}
		return address, nil
	}
	for i, iface := range spec.Devices.Interfaces {
		spec.Devices.Interfaces[i].Address, err = fn(iface.Address)
		if err != nil {
			return err
		}
	}
	for i, hostDev := range spec.Devices.HostDevices {
		if hostDev.Type != api.HostDevicePCI {
			continue
		}
		spec.Devices.HostDevices[i].Address, err = fn(hostDev.Address)
		if err != nil {
			return err
		}
	}
	for i, controller := range spec.Devices.Controllers {
		// pci-root and pcie-root devices can by definition hot have a pci address on its own
		if controller.Model == "pci-root" || controller.Model == api.ControllerModelPCIeRoot || controller.Model == api.ControllerModelPCIeExpanderBus {
			continue
		}
		spec.Devices.Controllers[i].Address, err = fn(controller.Address)
		if err != nil {
			return err
		}
	}
	for i, disk := range spec.Devices.Disks {
		if disk.Target.Bus != v1.DiskBusVirtio {
			continue
		}
		spec.Devices.Disks[i].Address, err = fn(disk.Address)
		if err != nil {
			return err
		}
	}
	for i, input := range spec.Devices.Inputs {
		if input.Bus != v1.VirtIO {
			continue
		}
		spec.Devices.Inputs[i].Address, err = fn(input.Address)
		if err != nil {
			return err
		}
	}
	for i, watchdog := range spec.Devices.Watchdogs {
		spec.Devices.Watchdogs[i].Address, err = fn(watchdog.Address)
		if err != nil {
			return err
		}
	}
	if spec.Devices.Rng != nil {
		spec.Devices.Rng.Address, err = fn(spec.Devices.Rng.Address)
		if err != nil {
			return err
		}
	}
	if spec.Devices.Ballooning != nil {
		spec.Devices.Ballooning.Address, err = fn(spec.Devices.Ballooning.Address)
		if err != nil {
			return err
		}
	}
	return nil
}

func CountPCIDevices(spec *api.DomainSpec) (count int, err error) {
	err = iteratePCIAddresses(spec, func(address *api.Address) (*api.Address, error) {
		count++
		return address, nil
	})
	return count, err
}

func PlacePCIDevicesOnRootComplex(spec *api.DomainSpec) (err error) {
	assigner := newRootSlotAssigner()
	return iteratePCIAddresses(spec, assigner.PlacePCIDeviceAtNextSlot)
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

// newPCIAddress creates a PCI address with the specified bus and slot.
func newPCIAddress(bus string, slot string) *api.Address {
	return &api.Address{
		Type:     api.AddressPCI,
		Domain:   "0x0000",
		Bus:      bus,
		Slot:     slot,
		Function: "0x0",
	}
}

func (p *pciRootSlotAssigner) PlacePCIDeviceAtNextSlot(address *api.Address) (*api.Address, error) {
	if address == nil {
		address = &api.Address{}
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

// numaKey represents a unique NUMA node for expander bus placement.
type numaKey struct {
	numaNode uint32
}

// numaAwareTopology represents the PCIe topology for a specific NUMA node.
type numaAwareTopology struct {
	expanderBus           *api.Controller
	rootPorts             []*api.Controller
	addressPerDeviceAlias map[string]*api.Address
}

// expanderBusAssigner manages the assignment of PCIe expander buses and
// NUMA aligned device placement.
type expanderBusAssigner struct {
	domainSpec  *api.DomainSpec
	index       uint32
	topologyMap map[numaKey]*numaAwareTopology
	devices     []*api.HostDevice
}

// NewExpanderBusAssigner creates a new PCIe expander bus assigner.
func NewExpanderBusAssigner(domainSpec *api.DomainSpec) *expanderBusAssigner {
	assigner := &expanderBusAssigner{
		domainSpec:  domainSpec,
		topologyMap: make(map[numaKey]*numaAwareTopology),
		devices:     []*api.HostDevice{},
		index:       1,
	}

	return assigner
}

func (a *expanderBusAssigner) createController(model string, parentBus string, slot uint32, numaNode *uint32) *api.Controller {
	a.index++

	controller := &api.Controller{
		Type:  api.ControllerTypePCI,
		Index: fmt.Sprint(a.index),
		Model: model,
	}

	// PCIe expander bus doesn't have a PCI address and has a NUMA target
	if model == api.ControllerModelPCIeExpanderBus {
		controller.Target = &api.ControllerTarget{
			NUMANode: numaNode,
		}
		return controller
	}

	// All other controllers have PCI addresses
	slotStr := "0x00"
	if slot > 0 {
		slotStr = fmt.Sprintf("%#02x", slot)
	}

	controller.Address = newPCIAddress(parentBus, slotStr)

	return controller
}

// AddDevices queues host devices for NUMA aligned placement.
// Call PlaceDevices() after adding all devices.
func (a *expanderBusAssigner) AddDevices(devices []api.HostDevice) {
	for i := range devices {
		if devices[i].Type != api.HostDevicePCI {
			continue
		}

		// Skip devices without source address
		if devices[i].Source.Address == nil {
			log.Log.Infof("device has no source address, skipping for pcie-expander-bus assignment")
			continue
		}

		guestOSNumaNode := hardware.LookupDeviceVCPUNumaNode(
			devices[i].Source.Address,
			a.domainSpec,
		)

		if guestOSNumaNode == nil {
			log.Log.Infof("device %s has no NUMA affinity information, skipping for pcie-expander-bus assignment",
				hardware.PCIAddressToString(devices[i].Source.Address))
			continue
		}

		devices[i].NUMANode = guestOSNumaNode

		a.devices = append(a.devices, &devices[i])
	}
}

// numaDeviceGroups represents a mapping of NUMA nodes to host devices.
type numaDeviceGroups map[numaKey][]*api.HostDevice

// numaDeviceGroups groups devices by their NUMA node.
func (a *expanderBusAssigner) groupDevicesByNUMA() (numaDeviceGroups, error) {
	groups := make(numaDeviceGroups)

	for _, device := range a.devices {
		numaNode := device.NUMANode

		key := numaKey{
			numaNode: *numaNode,
		}

		groups[key] = append(groups[key], device)
	}

	return groups, nil
}

// numaAwareTopology handles NUMA aware topology creation/retrieval from the
// topology map. It creates an expander bus if the topology for that NUMA node
// doesn't exist and returns that topology.
func (a *expanderBusAssigner) numaAwareTopology(numaKey numaKey) *numaAwareTopology {
	topology, exists := a.topologyMap[numaKey]
	if !exists {
		topology = &numaAwareTopology{
			expanderBus:           a.createController(api.ControllerModelPCIeExpanderBus, "", 0, &numaKey.numaNode),
			addressPerDeviceAlias: make(map[string]*api.Address),
		}
		a.topologyMap[numaKey] = topology
	}
	return topology
}

// addRootPort creates a PCIe root port and adds it to the topology.
func (a *expanderBusAssigner) addRootPort(topology *numaAwareTopology, parentBus string) *api.Controller {
	slot := uint32(len(topology.rootPorts))
	rootPort := a.createController(api.ControllerModelPCIeRootPort, parentBus, slot, nil)
	topology.rootPorts = append(topology.rootPorts, rootPort)
	return rootPort
}

// placeDevice creates a root port and assigns the device address directly to it.
func (a *expanderBusAssigner) placeDevice(topology *numaAwareTopology, device *api.HostDevice) {
	rootPort := a.addRootPort(topology, topology.expanderBus.Index)
	if device.Alias != nil {
		topology.addressPerDeviceAlias[device.Alias.GetName()] = newPCIAddress(rootPort.Index, "0x00")
	}
}

// buildTopology groups devices by NUMA node by using a pcie-expander-bus per
// NUMA node. Within a pcie-expander-bus one pcie-root-port per device is created.
// Each device is then placed behind its respective root port.
//
// pcie-expander-bus (one per NUMA node) -> pcie-root-port (one per device) -> device
//
// It modifies the topology per NUMA node in place by creating the necessary controllers
// and updating the addresses of the devices.
func (a *expanderBusAssigner) buildTopology() error {
	numaDeviceGroups, err := a.groupDevicesByNUMA()
	if err != nil {
		return fmt.Errorf("failed to generate device groups per NUMA node and PCIe root: %w", err)
	}

	for numaKey, devices := range numaDeviceGroups {
		topology := a.numaAwareTopology(numaKey)

		for _, device := range devices {
			a.placeDevice(topology, device)
		}

		// Set the busNr of the expander bus so that it has enough space for all its children.
		// We start from 254 and go downwards to leave space for system controllers and other expander buses.
		busNr := maxExpanderBusNr - a.index
		topology.expanderBus.Target.BusNr = ptr.To(uint32(busNr))
	}

	return nil
}

// PlaceDevices places the added devices into a PCIe topology aligned to their
// NUMA node. It modifies the domainSpec in place or leaves it unchanged in case
// of an error.
func (a *expanderBusAssigner) PlaceDevices() error {
	if err := a.buildTopology(); err != nil {
		return fmt.Errorf("failed to create PCIe topology with NUMA alignment: %w", err)
	}

	for _, topology := range a.topologyMap {
		a.domainSpec.Devices.Controllers = append(a.domainSpec.Devices.Controllers, *topology.expanderBus)

		for _, rootPort := range topology.rootPorts {
			a.domainSpec.Devices.Controllers = append(a.domainSpec.Devices.Controllers, *rootPort)
		}
		for _, device := range a.devices {
			if device.Alias != nil {
				if address, exists := topology.addressPerDeviceAlias[device.Alias.GetName()]; exists {
					device.Address = address
				}
			}
			// If the device was not placed in the topology (e.g. missing CPU
			// affinity information), we leave it unmodified so that it can be
			// placed by the root slot assigner.
		}
	}

	return nil
}
