package virthandler

import (
	v1 "kubevirt.io/api/core/v1"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/api"
)

func virtualMachineOptions(
	smbios *v1.SMBiosConfiguration,
	period uint32,
	preallocatedVolumes []string,
	capabilities *api.Capabilities,
	disksInfo map[string]*containerdisk.DiskInfo,
	clusterConfig *virtconfig.ClusterConfig,
) *cmdv1.VirtualMachineOptions {
	options := &cmdv1.VirtualMachineOptions{
		MemBalloonStatsPeriod: period,
		PreallocatedVolumes:   preallocatedVolumes,
		Topology:              capabilitiesToTopology(capabilities),
		DisksInfo:             disksInfoToDisksInfo(disksInfo),
	}
	if smbios != nil {
		options.VirtualMachineSMBios = &cmdv1.SMBios{
			Family:       smbios.Family,
			Product:      smbios.Product,
			Manufacturer: smbios.Manufacturer,
			Sku:          smbios.Sku,
			Version:      smbios.Version,
		}
	}

	if clusterConfig != nil {
		options.ExpandDisksEnabled = clusterConfig.ExpandDisksEnabled()
		options.ClusterConfig = &cmdv1.ClusterConfig{
			ExpandDisksEnabled:        clusterConfig.ExpandDisksEnabled(),
			FreePageReportingDisabled: clusterConfig.IsFreePageReportingDisabled(),
			BochsDisplayForEFIGuests:  clusterConfig.BochsDisplayForEFIGuestsEnabled(),
			SerialConsoleLogDisabled:  clusterConfig.IsSerialConsoleLogDisabled(),
		}
	}

	return options
}

func capabilitiesToTopology(capabilities *api.Capabilities) *cmdv1.Topology {
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

func cpuToCPU(cpu api.CPU) *cmdv1.CPU {
	return &cmdv1.CPU{
		Id:       cpu.ID,
		Siblings: cpu.Siblings,
	}
}

func disksInfoToDisksInfo(disksInfo map[string]*containerdisk.DiskInfo) map[string]*cmdv1.DiskInfo {
	info := map[string]*cmdv1.DiskInfo{}
	for k, v := range disksInfo {
		if v != nil {
			info[k] = &cmdv1.DiskInfo{
				Format:      v.Format,
				BackingFile: v.BackingFile,
				ActualSize:  uint64(v.ActualSize),
				VirtualSize: uint64(v.VirtualSize),
			}
		}
	}
	return info
}
