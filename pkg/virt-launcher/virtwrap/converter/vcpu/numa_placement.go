package vcpu

import (
	"fmt"
	"strconv"
	"strings"

	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/client-go/log"

	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func QuantityToByte(quantity resource.Quantity) (api.Memory, error) {
	memorySize, int := quantity.AsInt64()
	if !int {
		memorySize = quantity.Value() - 1
	}

	if memorySize < 0 {
		return api.Memory{Unit: "b"}, fmt.Errorf("Memory size '%s' must be greater than or equal to 0", quantity.String())
	}
	return api.Memory{
		Value: uint64(memorySize),
		Unit:  "b",
	}, nil
}

func QuantityToMebiByte(quantity resource.Quantity) (uint64, error) {
	bytes, err := QuantityToByte(quantity)
	if err != nil {
		return 0, err
	}
	if bytes.Value == 0 {
		return 0, nil
	} else if bytes.Value < 1048576 {
		return 1, nil
	}
	return uint64(float64(bytes.Value)/1048576 + 0.5), nil
}

func isNumaPassthrough(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.CPU.NUMA != nil && vmi.Spec.Domain.CPU.NUMA.GuestMappingPassthrough != nil
}

func appendDomainEmulatorThreadPin(domain *api.Domain, allocatedCpu uint32) {
	emulatorThread := api.CPUEmulatorPin{
		CPUSet: strconv.Itoa(int(allocatedCpu)),
	}
	domain.Spec.CPUTune.EmulatorPin = &emulatorThread
}

func appendDomainIOThreadPin(domain *api.Domain, thread uint32, cpuset string) {
	iothreadPin := api.CPUTuneIOThreadPin{}
	iothreadPin.IOThread = thread
	iothreadPin.CPUSet = cpuset
	domain.Spec.CPUTune.IOThreadPin = append(domain.Spec.CPUTune.IOThreadPin, iothreadPin)
}

func FormatDomainIOThreadPin(vmi *v1.VirtualMachineInstance, domain *api.Domain, emulatorThread uint32, cpuset []int) error {
	iothreads := int(domain.Spec.IOThreads.IOThreads)
	vcpus := int(CalculateRequestedVCPUs(domain.Spec.CPU.Topology))

	if vmi.IsCPUDedicated() && vmi.Spec.Domain.CPU.IsolateEmulatorThread {
		// pin the IOThread on the same pCPU as the emulator thread
		cpuset := strconv.Itoa(int(emulatorThread))
		appendDomainIOThreadPin(domain, uint32(1), cpuset)
	} else if iothreads >= vcpus {
		// pin an IOThread on a CPU
		for thread := 1; thread <= iothreads; thread++ {
			cpuset := fmt.Sprintf("%d", cpuset[thread%vcpus])
			appendDomainIOThreadPin(domain, uint32(thread), cpuset)
		}
	} else {
		// the following will pin IOThreads to a set of cpus of a balanced size
		// for example, for 3 threads and 8 cpus the output will look like:
		// thread cpus
		//   1    0,1,2
		//   2    3,4,5
		//   3    6,7
		series := vcpus % iothreads
		curr := 0
		for thread := 1; thread <= iothreads; thread++ {
			remainder := vcpus/iothreads - 1
			if thread <= series {
				remainder += 1
			}
			end := curr + remainder
			slice := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(cpuset[curr:end+1])), ","), "[]")
			appendDomainIOThreadPin(domain, uint32(thread), slice)
			curr = end + 1
		}
	}
	return nil
}

func AdjustDomainForTopologyAndCPUSet(domain *api.Domain, vmi *v1.VirtualMachineInstance, topology *cmdv1.Topology, cpuset []int, useIOThreads bool) error {
	var cpuPool VCPUPool
	if isNumaPassthrough(vmi) {
		cpuPool = NewStrictCPUPool(domain.Spec.CPU.Topology, topology, cpuset)
	} else {
		cpuPool = NewRelaxedCPUPool(domain.Spec.CPU.Topology, topology, cpuset)
	}
	cpuTune, err := cpuPool.FitCores()
	if err != nil {
		log.Log.Reason(err).Error("failed to format domain cputune.")
		return err
	}
	domain.Spec.CPUTune = cpuTune

	// always add the hint-dedicated feature when dedicatedCPUs are requested.
	if domain.Spec.Features == nil {
		domain.Spec.Features = &api.Features{}
	}
	if domain.Spec.Features.KVM == nil {
		domain.Spec.Features.KVM = &api.FeatureKVM{}
	}
	domain.Spec.Features.KVM.HintDedicated = &api.FeatureState{
		State: "on",
	}

	var emulatorThread uint32
	if vmi.Spec.Domain.CPU.IsolateEmulatorThread {
		emulatorThread, err = cpuPool.FitThread()
		if err != nil {
			e := fmt.Errorf("no CPU allocated for the emulation thread: %v", err)
			log.Log.Reason(e).Error("failed to format emulation thread pin")
			return e
		}
		appendDomainEmulatorThreadPin(domain, emulatorThread)
	}
	if useIOThreads {
		if err := FormatDomainIOThreadPin(vmi, domain, emulatorThread, cpuset); err != nil {
			log.Log.Reason(err).Error("failed to format domain iothread pinning.")
			return err
		}
	}
	if vmi.IsRealtimeEnabled() {
		// RT settings
		// To be configured by manifest
		// - CPU Model: Host Passthrough
		// - VCPU (placement type and number)
		// - VCPU Pin (DedicatedCPUPlacement)
		// - USB controller should be disabled if no input type usb is found
		// - Memballoning can be disabled when setting 'autoattachMemBalloon' to false
		formatVCPUScheduler(domain, vmi)
		domain.Spec.Features.PMU = &api.FeatureState{State: "off"}
	}

	if isNumaPassthrough(vmi) {
		if err := numaMapping(vmi, &domain.Spec, topology); err != nil {
			log.Log.Reason(err).Error("failed to calculate passed through NUMA topology.")
			return err
		}
	}

	return nil
}

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

func GetVirtualMemory(vmi *v1.VirtualMachineInstance) *resource.Quantity {
	// In case that guest memory is explicitly set, return it
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Guest != nil {
		return vmi.Spec.Domain.Memory.Guest
	}

	// Otherwise, take memory from the memory-limit, if set
	if v, ok := vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceMemory]; ok {
		return &v
	}

	// Otherwise, take memory from the requested memory
	v, _ := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]
	return &v
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

	memory, err := QuantityToByte(*GetVirtualMemory(vmi))
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
