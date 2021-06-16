package converter

import (
	"fmt"
	"strconv"
	"strings"

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
func numaMapping(domain *api.DomainSpec, topology *cmdv1.Topology) error {
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

	for _, cell := range topology.NumaCells {
		if vcpus, exists := numamap[cell.Id]; exists {
			var cpus []string
			for _, cpu := range vcpus {
				cpus = append(cpus, strconv.Itoa(int(cpu)))
			}

			domain.CPU.NUMA.Cells = append(domain.CPU.NUMA.Cells, api.NUMACell{
				ID:     strconv.Itoa(int(vcpus[0])),
				CPUs:   strings.Join(cpus, ","),
				Memory: domain.Memory.Value / uint64(len(numamap)),
				Unit:   domain.Memory.Unit,
			})
			domain.NUMATune.MemNodes = append(domain.NUMATune.MemNodes, api.MemNode{
				CellID:  vcpus[0],
				Mode:    "strict",
				NodeSet: strconv.Itoa(int(cell.Id)),
			})
		}
	}
	return nil
}
