package converter

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func cpuToCell(topology *cmdv1.Topology) map[uint32]*cmdv1.Cell {
	cpumap := map[uint32]*cmdv1.Cell{}
	for i, cell := range topology.NumaCells {
		for _, cpu := range cell.Cpus {
			cpumap[cpu.Id] = topology.NumaCells[i]
		}
	}
	return cpumap
}

func involvedCells(cpumap map[uint32]*cmdv1.Cell, cpuTune *api.CPUTune) (map[uint32][]uint32, error) {
	numamap := map[uint32][]uint32{}
	for _, tune := range cpuTune.VCPUPin {
		cpu, err := strconv.ParseInt(tune.CPUSet, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("expected only full cpu to be mapped, but got %v: %v", tune.CPUSet, err)
		}
		if _, exists := cpumap[uint32(cpu)]; !exists {
			return nil, fmt.Errorf("vcpu %v is mapped to a not existing host cpu set %v", tune.VCPU, tune.CPUSet)
		}
		numamap[cpumap[uint32(cpu)].Id] = append(numamap[cpumap[uint32(cpu)].Id], tune.VCPU)
	}
	return numamap, nil
}

// numaMapping maps numa nodes based on already applied VCPU pinning. The sort result is stable compared to the order
// of provided host numa nodes.
func numaMapping(vmi *v1.VirtualMachineInstance, domain *api.DomainSpec, topology *cmdv1.Topology) error {
	if topology == nil || len(topology.NumaCells) == 0 {
		// If there is no numa topology reported, we don't do anything.
		// this also means that emulated numa for e.g. memfd will keep intact
		return nil
	}
	cpumap := cpuToCell(topology)
	numamap, err := involvedCells(cpumap, domain.CPUTune)
	if err != nil {
		return fmt.Errorf("failed to generate numa pinning information: %v", err)
	}

	var involvedCellIDs []string
	for _, cell := range topology.NumaCells {
		if _, exists := numamap[cell.Id]; exists {
			involvedCellIDs = append(involvedCellIDs, strconv.Itoa(int(cell.Id)))
		}
	}

	domain.CPU.NUMA = &api.NUMA{}
	domain.NUMATune = &api.NUMATune{
		Memory: api.NumaTuneMemory{
			Mode:    "strict",
			NodeSet: strings.Join(involvedCellIDs, ","),
		},
	}

	hugepagesSize, hugepagesUnit, hugepagesEnabled, err := hugePagesInfo(vmi, domain)
	if err != nil {
		return fmt.Errorf("failed to determine if hugepages are enabled: %v", err)
	} else if !hugepagesEnabled {
		return fmt.Errorf("passing through a numa topology is restricted to VMIs with hugepages enabled")
	}
	domain.MemoryBacking.Allocation = &api.MemoryAllocation{Mode: api.MemoryAllocationModeImmediate}

	memory, err := QuantityToByte(*getVirtualMemory(vmi))
	memoryBytes := memory.Value
	if err != nil {
		return fmt.Errorf("could not convert VMI memory to quantity: %v", err)
	}
	var mod uint64
	cellCount := uint64(len(involvedCellIDs))
	if memoryBytes < cellCount*hugepagesSize {
		return fmt.Errorf("not enough memory requested to allocate at least one hugepage per numa node: %v < %v", memory, cellCount*(hugepagesSize*1024*1024))
	} else if memoryBytes%hugepagesSize != 0 {
		return fmt.Errorf("requested memory can't be divided through the numa page size: %v mod %v != 0", memory, hugepagesSize)
	}
	mod = (memoryBytes % (hugepagesSize * cellCount) / hugepagesSize)
	if mod != 0 {
		memoryBytes = memoryBytes - mod*hugepagesSize
	}

	virtualCellID := -1
	for _, cell := range topology.NumaCells {
		if vcpus, exists := numamap[cell.Id]; exists {
			var cpus []string
			for _, cpu := range vcpus {
				cpus = append(cpus, strconv.Itoa(int(cpu)))
			}
			virtualCellID++

			domain.CPU.NUMA.Cells = append(domain.CPU.NUMA.Cells, api.NUMACell{
				ID:     strconv.Itoa(virtualCellID),
				CPUs:   strings.Join(cpus, ","),
				Memory: memoryBytes / uint64(len(numamap)),
				Unit:   memory.Unit,
			})
			domain.NUMATune.MemNodes = append(domain.NUMATune.MemNodes, api.MemNode{
				CellID:  uint32(virtualCellID),
				Mode:    "strict",
				NodeSet: strconv.Itoa(int(cell.Id)),
			})
			domain.MemoryBacking.HugePages.HugePage = append(domain.MemoryBacking.HugePages.HugePage, api.HugePage{
				Size:    strconv.Itoa(int(hugepagesSize)),
				Unit:    hugepagesUnit,
				NodeSet: strconv.Itoa(virtualCellID),
			})
		}
	}

	if hugepagesEnabled && mod > 0 {
		for i := range domain.CPU.NUMA.Cells[:mod] {
			domain.CPU.NUMA.Cells[i].Memory += hugepagesSize
		}
	}
	if vmi.IsRealtimeEnabled() {
		// RT settings when hugepages are enabled
		domain.MemoryBacking.NoSharePages = &api.NoSharePages{}
	}
	return nil
}

func hugePagesInfo(vmi *v1.VirtualMachineInstance, domain *api.DomainSpec) (size uint64, unit string, enabled bool, err error) {
	if domain.MemoryBacking != nil && domain.MemoryBacking.HugePages != nil {
		if vmi.Spec.Domain.Memory.Hugepages != nil {
			quantity, err := resource.ParseQuantity(vmi.Spec.Domain.Memory.Hugepages.PageSize)
			if err != nil {
				return 0, "", false, fmt.Errorf("could not parse hugepage value %v: %v", vmi.Spec.Domain.Memory.Hugepages.PageSize, err)
			}
			size, err := QuantityToByte(quantity)
			if err != nil {
				return 0, "b", false, fmt.Errorf("could not convert page size to MiB %v: %v", vmi.Spec.Domain.Memory.Hugepages.PageSize, err)
			}
			return size.Value, "b", true, nil
		}
	}
	return 0, "b", false, nil
}
