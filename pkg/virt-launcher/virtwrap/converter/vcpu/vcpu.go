package vcpu

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/client-go/log"

	k8sv1 "k8s.io/api/core/v1"

	v12 "kubevirt.io/api/core/v1"

	v1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type VCPUPool interface {
	FitCores() (tune *api.CPUTune, err error)
	FitThread() (thread uint32, err error)
}

func CalculateRequestedVCPUs(cpuTopology *api.CPUTopology) uint32 {
	return cpuTopology.Cores * cpuTopology.Sockets * cpuTopology.Threads
}

type cell struct {
	fullCoresList       [][]uint32
	fragmentedCoresList []uint32
	threadsPerCore      int
}

// GetNotFragmentedThreads consumes the amount of threadsPerCore from the numa cell
// or none if it can't be fit on the numa cell
func (c *cell) GetNotFragmentedThreads() []uint32 {
	if len(c.fullCoresList) > 0 {
		selected := c.fullCoresList[0][:c.threadsPerCore]
		remaining := c.fullCoresList[0][c.threadsPerCore:]
		if len(remaining) >= c.threadsPerCore {
			c.fullCoresList[0] = remaining
		} else {
			c.fullCoresList = c.fullCoresList[1:]
			c.fragmentedCoresList = append(c.fragmentedCoresList, remaining...)
		}
		return selected
	}
	return nil
}

// GetFragmentedThreads will allocate as many threadsPerCore out of the request
func (c *cell) GetFragmentedThreads() []uint32 {
	if c.threadsPerCore <= len(c.fragmentedCoresList) {
		selected := c.fragmentedCoresList[:c.threadsPerCore]
		c.fragmentedCoresList = c.fragmentedCoresList[c.threadsPerCore:]
		return selected
	}
	return nil
}

// GetFragmentedThreads will allocate as many threads as possible
// and return them, even if it can only satisfy parts of the request.
func (c *cell) GetFragmentedThreadsUpTo(threads int) []uint32 {
	selector := threads
	if threads > len(c.fragmentedCoresList) {
		selector = len(c.fragmentedCoresList)
	}
	selected := c.fragmentedCoresList[:selector]
	c.fragmentedCoresList = c.fragmentedCoresList[selector:]
	return selected
}

// GetThread will first try to allocate a thread from fragmented cores
// but fall back to not fragmented cores if the request can't be satisfied otherwise
func (c *cell) GetThread() *uint32 {
	if len(c.fragmentedCoresList) > 0 {
		thread := c.fragmentedCoresList[0]
		c.fragmentedCoresList = c.fragmentedCoresList[1:]
		return &thread
	} else if len(c.fullCoresList) > 0 {
		thread := c.fullCoresList[0][0]
		remaining := c.fullCoresList[0][1:]
		if len(remaining) >= c.threadsPerCore {
			c.fullCoresList[0] = remaining
		} else {
			c.fullCoresList = c.fullCoresList[1:]
			c.fragmentedCoresList = append(c.fragmentedCoresList, remaining...)
		}
		return &thread
	}
	return nil
}

func (c *cell) IsEmpty() bool {
	return len(c.fragmentedCoresList) == 0 && len(c.fullCoresList) == 0
}

type cpuPool struct {
	// cells contains a host thread mapping of host threads to their cores
	// and cores to their numa cells
	cells []*cell
	// threadsPerCore is the amount of vcpu threads per vcpu core
	threadsPerCore int
	// cores is the amount of vcpu cores requested by the VMI
	cores int
	// allowCellCrossing allows inefficient cpu mapping where a single
	// core can have threads on different host numa cells
	allowCellCrossing bool
	// availableThreads is the amount of all threads assigned to the pod
	availableThreads int
}

func NewStrictCPUPool(requestedToplogy *api.CPUTopology, nodeTopology *v1.Topology, cpuSet []int) VCPUPool {
	return newCPUPool(requestedToplogy, nodeTopology, cpuSet, false)
}

func NewRelaxedCPUPool(requestedToplogy *api.CPUTopology, nodeTopology *v1.Topology, cpuSet []int) VCPUPool {
	return newCPUPool(requestedToplogy, nodeTopology, cpuSet, true)
}

func newCPUPool(requestedToplogy *api.CPUTopology, nodeTopology *v1.Topology, cpuSet []int, allowCellCrossing bool) *cpuPool {
	pool := &cpuPool{threadsPerCore: int(requestedToplogy.Threads), cores: int(requestedToplogy.Cores * requestedToplogy.Sockets), allowCellCrossing: allowCellCrossing, availableThreads: len(cpuSet)}
	cores := cpuChunksToCells(cpuSet, nodeTopology)

	for _, coresOnCell := range cores {
		c := cell{threadsPerCore: int(requestedToplogy.Threads)}
		for j, core := range coresOnCell {
			if len(core) >= c.threadsPerCore {
				c.fullCoresList = append(c.fullCoresList, coresOnCell[j])
			} else {
				c.fragmentedCoresList = append(c.fragmentedCoresList, coresOnCell[j]...)
			}
		}
		pool.cells = append(pool.cells, &c)
	}
	return pool
}

// cpuChunksToCells takes the allocated cpuset, determines which of the threads belongs to which cpu and which numa
// cell and returns an aggregated view. The first dimension of the returned array represents the numa nodes. The next
// level the cores of the corresponding numa node and the inner most array contains the available threads of the core.
func cpuChunksToCells(cpuSet []int, nodeTopology *v1.Topology) (cores [][][]uint32) {
	threads := map[int]struct{}{}
	visited := map[uint32]struct{}{}
	cores = [][][]uint32{}
	for _, cpu := range cpuSet {
		threads[cpu] = struct{}{}
	}
	for _, cell := range nodeTopology.NumaCells {
		var coresOnCell [][]uint32
		for _, cpu := range cell.Cpus {
			if _, exists := visited[cpu.Id]; exists {
				continue
			}
			core := []uint32{}
			if len(cpu.Siblings) == 0 {
				visited[cpu.Id] = struct{}{}
				if _, exists := threads[int(cpu.Id)]; exists {
					core = append(core, cpu.Id)
				}
			} else {
				for _, thread := range cpu.Siblings {
					visited[thread] = struct{}{}
					if _, exists := threads[int(thread)]; exists {
						core = append(core, thread)
					}
				}
			}
			coresOnCell = append(coresOnCell, core)
		}
		cores = append(cores, coresOnCell)
	}
	return cores
}

func (p *cpuPool) FitCores() (cpuTune *api.CPUTune, err error) {
	threads, remaining := p.fitCores(p.cores)

	if remaining > 0 {
		if p.allowCellCrossing || p.availableThreads < p.cores*p.threadsPerCore {
			return nil, fmt.Errorf("not enough exclusive threads provided, could not fit %v core(s)", remaining)
		} else {
			return nil, fmt.Errorf("could not fit %v core(s) without crossing numa cell boundaries for individual cores", remaining)
		}
	}
	cpuTune = &api.CPUTune{}
	for idx, hostThread := range threads {
		vcpupin := api.CPUTuneVCPUPin{}
		vcpupin.VCPU = uint32(idx)
		vcpupin.CPUSet = strconv.Itoa(int(hostThread))
		cpuTune.VCPUPin = append(cpuTune.VCPUPin, vcpupin)
	}
	return cpuTune, nil
}

func (p *cpuPool) fitCores(coreCount int) (threads []uint32, remainingCores int) {
	remainingCores = coreCount
	assignedThreads, remainingCores := p.fitCPUBound(remainingCores)
	threads = append(threads, assignedThreads...)
	if remainingCores == 0 {
		return threads, 0
	}
	assignedThreads, remainingCores = p.fitCellBound(remainingCores)
	threads = append(threads, assignedThreads...)
	if remainingCores == 0 {
		return threads, 0
	}
	if p.allowCellCrossing {
		assignedThreads, remainingCores = p.fitUnbound(remainingCores)
		threads = append(threads, assignedThreads...)
		if remainingCores == 0 {
			return threads, 0
		}
	}
	return threads, remainingCores
}

func (p *cpuPool) FitThread() (thread uint32, err error) {
	t := p.fitThread()
	if t == nil {
		return 0, fmt.Errorf("no remaining unassigned threads")
	}
	return *t, nil
}

func fitChunk(cells []*cell, requested int, allocator func(cells []*cell, idx int) []uint32) (threads []uint32, remainingCores int) {
	for idx := range cells {
		for {
			chunk := allocator(cells, idx)
			if len(chunk) == 0 {
				// go to the next cell
				break
			}
			threads = append(threads, chunk...)
			requested--
			if requested == 0 {
				return threads, 0
			}
		}
	}
	return threads, requested
}

func (p *cpuPool) fitCPUBound(requested int) (threads []uint32, remainingCores int) {
	allocator := func(cell []*cell, idx int) []uint32 {
		return cell[idx].GetNotFragmentedThreads()
	}
	return fitChunk(p.cells, requested, allocator)
}

func (p *cpuPool) fitCellBound(requested int) (threads []uint32, remainingCores int) {
	allocator := func(cell []*cell, idx int) []uint32 {
		return cell[idx].GetFragmentedThreads()
	}
	return fitChunk(p.cells, requested, allocator)
}

func (p *cpuPool) fitUnbound(requested int) (threads []uint32, remainingCores int) {
	remainingThreads := p.threadsPerCore * requested
	for _, cell := range p.cells {
		for {
			chunk := cell.GetFragmentedThreadsUpTo(remainingThreads)
			if len(chunk) == 0 {
				// go to the next cell
				break
			}
			threads = append(threads, chunk...)
			remainingThreads -= len(chunk)
			if remainingThreads < 0 {
				panic(fmt.Errorf("this is a bug, remainingCores must never be below 0 but it is %v", remainingThreads))
			}
			if remainingThreads == 0 {
				return threads, 0
			}
		}
	}
	return threads, int(float64(remainingThreads)+0.5) / p.threadsPerCore
}

func (p *cpuPool) fitThread() (thread *uint32) {
	for _, cell := range p.cells {
		if cell.IsEmpty() {
			continue
		}
		return cell.GetThread()
	}
	return nil
}

func GetCPUTopology(vmi *v12.VirtualMachineInstance) *api.CPUTopology {
	cores := uint32(1)
	threads := uint32(1)
	sockets := uint32(1)
	vmiCPU := vmi.Spec.Domain.CPU
	if vmiCPU != nil {
		if vmiCPU.Cores != 0 {
			cores = vmiCPU.Cores
		}

		if vmiCPU.Threads != 0 {
			threads = vmiCPU.Threads
		}

		if vmiCPU.Sockets != 0 {
			sockets = vmiCPU.Sockets
		}
	}
	// A default guest CPU topology is being set in API mutator webhook, if nothing provided by a user.
	// However this setting is still required to handle situations when the webhook fails to set a default topology.
	if vmiCPU == nil || (vmiCPU.Cores == 0 && vmiCPU.Sockets == 0 && vmiCPU.Threads == 0) {
		//if cores, sockets, threads are not set, take value from domain resources request or limits and
		//set value into sockets, which have best performance (https://bugzilla.redhat.com/show_bug.cgi?id=1653453)
		resources := vmi.Spec.Domain.Resources
		if cpuLimit, ok := resources.Limits[k8sv1.ResourceCPU]; ok {
			sockets = uint32(cpuLimit.Value())
		} else if cpuRequests, ok := resources.Requests[k8sv1.ResourceCPU]; ok {
			sockets = uint32(cpuRequests.Value())
		}
	}

	return &api.CPUTopology{
		Sockets: sockets,
		Cores:   cores,
		Threads: threads,
	}
}

func QuantityToByte(quantity resource.Quantity) (api.Memory, error) {
	memorySize, isInt := quantity.AsInt64()
	if !isInt {
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

func isNumaPassthrough(vmi *v12.VirtualMachineInstance) bool {
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

func FormatDomainIOThreadPin(vmi *v12.VirtualMachineInstance, domain *api.Domain, emulatorThread uint32, cpuset []int) error {
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

func AdjustDomainForTopologyAndCPUSet(domain *api.Domain, vmi *v12.VirtualMachineInstance, topology *v1.Topology, cpuset []int, useIOThreads bool) error {
	var cpuPool VCPUPool
	requestedToplogy := &api.CPUTopology{
		Sockets: domain.Spec.CPU.Topology.Sockets,
		Cores:   domain.Spec.CPU.Topology.Cores,
		Threads: domain.Spec.CPU.Topology.Threads,
	}

	if vmi.Spec.Domain.CPU.MaxSockets != 0 {
		disabledVCPUs := 0
		for _, vcpu := range domain.Spec.VCPUs.VCPU {
			if vcpu.Enabled != "yes" {
				disabledVCPUs += 1
			}
		}
		disabledSockets := uint32(disabledVCPUs) / (requestedToplogy.Cores * requestedToplogy.Threads)
		requestedToplogy.Sockets -= uint32(disabledSockets)
	}

	if isNumaPassthrough(vmi) {
		cpuPool = NewStrictCPUPool(requestedToplogy, topology, cpuset)
	} else {
		cpuPool = NewRelaxedCPUPool(requestedToplogy, topology, cpuset)
	}
	cpuTune, err := cpuPool.FitCores()
	if err != nil {
		log.Log.Reason(err).Error("failed to format domain cputune.")
		return err
	}
	domain.Spec.CPUTune = cpuTune

	// Add the hint-dedicated feature when dedicatedCPUs are requested for AMD64 architecture.
	if util.IsAMD64VMI(vmi) {
		if domain.Spec.Features == nil {
			domain.Spec.Features = &api.Features{}
		}
		if domain.Spec.Features.KVM == nil {
			domain.Spec.Features.KVM = &api.FeatureKVM{}
		}
		domain.Spec.Features.KVM.HintDedicated = &api.FeatureState{
			State: "on",
		}
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

		// Changes to the vcpu scheduling and priorities are performed by the virt-handler to allow
		// workloads that run without CAP_SYS_NICE to work as well as with CAP_SYS_NICE.
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

func cpuToCell(topology *v1.Topology) map[uint32]*v1.Cell {
	cpumap := map[uint32]*v1.Cell{}
	for i, cell := range topology.NumaCells {
		for _, cpu := range cell.Cpus {
			cpumap[cpu.Id] = topology.NumaCells[i]
		}
	}
	return cpumap
}

func involvedCells(cpumap map[uint32]*v1.Cell, cpuTune *api.CPUTune) (map[uint32][]uint32, error) {
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

func GetVirtualMemory(vmi *v12.VirtualMachineInstance) *resource.Quantity {
	// In case that guest memory is explicitly set, return it
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Guest != nil {
		return vmi.Spec.Domain.Memory.Guest
	}

	// Get the requested memory
	reqMemory, isReqMemSet := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]

	// Otherwise, take memory from the memory-limit, if set and requested Memory not set
	if v, ok := vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceMemory]; ok && !isReqMemSet {
		return &v
	}

	// Otherwise, take memory from the requested memory
	return &reqMemory
}

// numaMapping maps numa nodes based on already applied VCPU pinning. The sort result is stable compared to the order
// of provided host numa nodes.
func numaMapping(vmi *v12.VirtualMachineInstance, domain *api.DomainSpec, topology *v1.Topology) error {
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

func hugePagesInfo(vmi *v12.VirtualMachineInstance, domain *api.DomainSpec) (size uint64, unit string, enabled bool, err error) {
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
