package vcpu

import (
	"fmt"
	"strconv"

	v1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
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
