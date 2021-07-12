package virthandler

import (
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/api"
)

func (d *VirtualMachineController) virtualMachineOptions(preallocatedVolumes []string) *cmdv1.VirtualMachineOptions {
	smbios := d.clusterConfig.GetSMBIOS()
	period := d.clusterConfig.GetMemBalloonStatsPeriod()

	options := &cmdv1.VirtualMachineOptions{
		VirtualMachineSMBios: &cmdv1.SMBios{
			Family:       smbios.Family,
			Product:      smbios.Product,
			Manufacturer: smbios.Manufacturer,
			Sku:          smbios.Sku,
			Version:      smbios.Version,
		},
		MemBalloonStatsPeriod: period,
		PreallocatedVolumes:   preallocatedVolumes,
		Topology:              topologyToTopology(d.capabilities),
		DiskSizes:             d.diskSizes,
		ChangedDisks:          d.changedDisks,
	}
	return options
}

func topologyToTopology(capabilities *api.Capabilities) *cmdv1.Topology {
	topology := &cmdv1.Topology{}
	if capabilities == nil {
		return topology
	}

	for _, cell := range capabilities.Host.Topology.Cells.Cell {
		topology.NumaCells = append(topology.NumaCells, cellToCell(cell))
	}
	return topology
}

func cellToCell(cell api.Cell) *cmdv1.Cell {
	c := &cmdv1.Cell{
		Id: cell.ID,
		Memory: &cmdv1.Memory{
			Amount: cell.Memory.Amount,
			Unit:   cell.Memory.Unit,
		},
	}

	for _, page := range cell.Pages {
		c.Pages = append(c.Pages, pageToPage(page))
	}

	for _, distance := range cell.Distances.Sibling {
		c.Distances = append(c.Distances, distanceToDistance(distance))
	}

	for _, cpu := range cell.Cpus.CPU {
		c.Cpus = append(c.Cpus, cpuToCPU(cpu))
	}

	return c
}

func pageToPage(pages api.Pages) *cmdv1.Pages {
	return &cmdv1.Pages{
		Count: pages.Count,
		Unit:  pages.Unit,
		Size:  pages.Size,
	}
}

func distanceToDistance(distance api.Sibling) *cmdv1.Sibling {
	return &cmdv1.Sibling{
		Id:    distance.ID,
		Value: distance.Value,
	}
}

func cpuToCPU(distance api.CPU) *cmdv1.CPU {
	return &cmdv1.CPU{
		Id: distance.ID,
	}
}
