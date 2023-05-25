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
 * Copyright 2018 Red Hat, Inc.
 */

package stats

// stats Package wraps the libvirt bulk stats data types.
//
// The libvirt.DomainStats type is not a POD type, it includes a pointer to the
// generating libvirt.Domain. But consumers of the virtwrap package should not deal
// with libvirt types directly.
// Furthermore, we need to close gaps about the bulk stats API. We care about memory
// stats, but these aren't yet delivered as bulk stats output; again the client code
// should not worry about this.
// The solution is to handle all the low-level details in the cli subpackage (see its docs).
// Doing so, however, prevents us to add the types and the converter in the canonical places,
// namely the virtwrap main package and the api subpackage, to avoid dependency cycles.
// The cleanest approach to untangle the mess is thus to create a separate subpackage.
//
// We choose to replicate the libvirt structs here, kinda like the rest of the virtwrap
// package tree does, again to be independent with respec to the libvirt data types,
// and to avoid that client code (e.g. anything that uses cmd-client) needs to import it,
// dragging the libvirt dependency in the final binary.

// For versioning of the virt-handler and -launcher communication,
// you need to increase the Version const when making changes,
// and make necessary changes in the cmd rpc implementation!
const (
	DomainStatsVersion = "v1"

	// VIR_VCPU_OFFLINE    = 0,    /* the virtual CPU is offline */
	VCPUOffline = 0
	//  VIR_VCPU_RUNNING    = 1,    /* the virtual CPU is running */
	VCPURunning = 1
	//  VIR_VCPU_BLOCKED    = 2,    /* the virtual CPU is blocked on resource */
	VCPUBlocked = 2
)

type DomainStats struct {
	// the following aren't really needed for stats, but it's practical to report
	// OTOH, the whole "Domain" is too much data to be unconditionally reported
	Name string
	UUID string
	// omitted from libvirt-go: Domain
	// omitted from libvirt-go: State
	Cpu *DomainStatsCPU
	// new, see below
	Memory *DomainStatsMemory
	// omitted from libvirt-go: DomainJobInfo
	MigrateDomainJobInfo *DomainJobInfo
	// omitted from libvirt-go: Balloon
	Vcpu  []DomainStatsVcpu
	Net   []DomainStatsNet
	Block []DomainStatsBlock
	// omitted from libvirt-go: Perf
	// extra stats
	CPUMapSet bool
	CPUMap    [][]bool
	NrVirtCpu uint
}

type DomainStatsCPU struct {
	TimeSet   bool
	Time      uint64
	UserSet   bool
	User      uint64
	SystemSet bool
	System    uint64
}

type DomainStatsVcpu struct {
	StateSet bool
	State    int // VcpuState
	TimeSet  bool
	Time     uint64
	WaitSet  bool
	Wait     uint64
}

type DomainStatsNet struct {
	NameSet    bool
	Name       string
	AliasSet   bool
	Alias      string
	RxBytesSet bool
	RxBytes    uint64
	RxPktsSet  bool
	RxPkts     uint64
	RxErrsSet  bool
	RxErrs     uint64
	RxDropSet  bool
	RxDrop     uint64
	TxBytesSet bool
	TxBytes    uint64
	TxPktsSet  bool
	TxPkts     uint64
	TxErrsSet  bool
	TxErrs     uint64
	TxDropSet  bool
	TxDrop     uint64
}

type DomainStatsBlock struct {
	NameSet         bool
	Name            string
	Alias           string
	BackingIndexSet bool
	BackingIndex    uint
	PathSet         bool
	Path            string
	RdReqsSet       bool
	RdReqs          uint64
	RdBytesSet      bool
	RdBytes         uint64
	RdTimesSet      bool
	RdTimes         uint64
	WrReqsSet       bool
	WrReqs          uint64
	WrBytesSet      bool
	WrBytes         uint64
	WrTimesSet      bool
	WrTimes         uint64
	FlReqsSet       bool
	FlReqs          uint64
	FlTimesSet      bool
	FlTimes         uint64
	ErrorsSet       bool
	Errors          uint64
	AllocationSet   bool
	Allocation      uint64
	CapacitySet     bool
	Capacity        uint64
	PhysicalSet     bool
	Physical        uint64
}

// mimic existing structs, but data is taken from
// DomainMemoryStat
type DomainStatsMemory struct {
	UnusedSet        bool
	Unused           uint64
	AvailableSet     bool
	Available        uint64
	ActualBalloonSet bool
	ActualBalloon    uint64
	RSSSet           bool
	RSS              uint64
	SwapInSet        bool
	SwapIn           uint64
	SwapOutSet       bool
	SwapOut          uint64
	MajorFaultSet    bool
	MajorFault       uint64
	MinorFaultSet    bool
	MinorFault       uint64
	UsableSet        bool
	Usable           uint64
	TotalSet         bool
	Total            uint64
}

// mimic existing structs, but data is taken from
// DomainJobInfo
type DomainJobInfo struct {
	DataProcessedSet bool
	DataProcessed    uint64
	MemoryBpsSet     bool
	MemoryBps        uint64
	DiskBpsSet       bool
	DiskBps          uint64
	DataRemainingSet bool
	DataRemaining    uint64
	MemDirtyRateSet  bool
	MemDirtyRate     uint64
}
