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
 * Copyright 2023 The KubeVirt Authors
 *
 */

package libvirtxml

import (
	"strconv"

	"libvirt.org/go/libvirtxml"

	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// This package provides the methods to convert the types defined in
// kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api to the types defined in
// libvirt.org/go/libvirtxml package.
//
// The libvirtxml package provides 100% coverage to Libvirt XML schemas and we
// can rely on the Marshal/Unmarsh functions from this package.
//
// TODO: remove all the convert functions once we switch using the types of libvirtxml.
// See: https://github.com/kubevirt/kubevirt/issues/10844

func ConvertKubeVirtCPUTopologyToDomainCPUTopology(cpuTopology *api.CPUTopology) *libvirtxml.DomainCPUTopology {
	if cpuTopology == nil {
		return nil
	}
	return &libvirtxml.DomainCPUTopology{
		Sockets: int(cpuTopology.Sockets),
		Cores:   int(cpuTopology.Cores),
		Threads: int(cpuTopology.Threads),
	}
}

func ConvertKubeVirtNUMACellToDomainDomainCell(cell []api.NUMACell) ([]libvirtxml.DomainCell, error) {
	ret := []libvirtxml.DomainCell{}
	for _, c := range cell {
		v, err := strconv.ParseUint(c.ID, 10, 32)
		if err != nil {
			return nil, err
		}
		id := uint(v)
		ret = append(ret, libvirtxml.DomainCell{
			ID:        &id,
			CPUs:      c.CPUs,
			Memory:    uint(c.Memory),
			Unit:      c.Unit,
			MemAccess: c.MemoryAccess,
		})
	}
	return ret, nil
}

func ConvertKubeVirtNUMAToDomainNUMA(numa *api.NUMA) (*libvirtxml.DomainNuma, error) {
	if numa == nil {
		return nil, nil
	}
	cell, err := ConvertKubeVirtNUMACellToDomainDomainCell(numa.Cells)
	if err != nil {
		return nil, err
	}
	return &libvirtxml.DomainNuma{
		Cell: cell,
	}, nil
}

func ConvertKubeVirtVCPUToDomainVCPU(vcpu *api.VCPU) *libvirtxml.DomainVCPU {
	if vcpu == nil {
		return nil
	}
	return &libvirtxml.DomainVCPU{
		Placement: vcpu.Placement,
		Value:     uint(vcpu.CPUs),
	}
}

func ConvertKubeVirtCPUTuneVCPUPinToDomainCPUTuneVCPUPin(vcpupin []api.CPUTuneVCPUPin) []libvirtxml.DomainCPUTuneVCPUPin {
	res := []libvirtxml.DomainCPUTuneVCPUPin{}
	for _, v := range vcpupin {
		res = append(res, libvirtxml.DomainCPUTuneVCPUPin{
			VCPU:   uint(v.VCPU),
			CPUSet: v.CPUSet,
		})
	}
	return res
}

func ConvertKubeVirtCPUTuneIOThreadPinToDomainCPUTuneIOThreadPin(iothreadpin []api.CPUTuneIOThreadPin) []libvirtxml.DomainCPUTuneIOThreadPin {
	res := []libvirtxml.DomainCPUTuneIOThreadPin{}
	for _, v := range iothreadpin {
		res = append(res, libvirtxml.DomainCPUTuneIOThreadPin{
			IOThread: uint(v.IOThread),
			CPUSet:   v.CPUSet,
		})
	}
	return res
}

func ConvertKubeVirtCPUTuneToDomainCPUTune(cputune *api.CPUTune) *libvirtxml.DomainCPUTune {
	var emulatorPin *libvirtxml.DomainCPUTuneEmulatorPin
	if cputune == nil {
		return nil
	}
	if cputune.EmulatorPin != nil {
		emulatorPin = &libvirtxml.DomainCPUTuneEmulatorPin{
			CPUSet: cputune.EmulatorPin.CPUSet,
		}
	}
	return &libvirtxml.DomainCPUTune{
		VCPUPin:     ConvertKubeVirtCPUTuneVCPUPinToDomainCPUTuneVCPUPin(cputune.VCPUPin),
		IOThreadPin: ConvertKubeVirtCPUTuneIOThreadPinToDomainCPUTuneIOThreadPin(cputune.IOThreadPin),
		EmulatorPin: emulatorPin,
	}
}

func ConvertKubeVirtMemNodeToDomainNUMATuneMemNode(memNodes []api.MemNode) []libvirtxml.DomainNUMATuneMemNode {
	res := []libvirtxml.DomainNUMATuneMemNode{}
	for _, v := range memNodes {
		res = append(res, libvirtxml.DomainNUMATuneMemNode{
			CellID:  uint(v.CellID),
			Mode:    v.Mode,
			Nodeset: v.NodeSet,
		})
	}
	return res
}

func ConvertKubeVirtNUMATuneToDomainNUMATune(numatune *api.NUMATune) *libvirtxml.DomainNUMATune {
	if numatune == nil {
		return nil
	}
	return &libvirtxml.DomainNUMATune{
		Memory: &libvirtxml.DomainNUMATuneMemory{
			Mode:    numatune.Memory.Mode,
			Nodeset: numatune.Memory.NodeSet,
		},
		MemNodes: ConvertKubeVirtMemNodeToDomainNUMATuneMemNode(numatune.MemNodes),
	}
}
func ConvertKubeVirtHugepageToDomainMemoryHugepages(hugepage *api.HugePages) (*libvirtxml.DomainMemoryHugepages, error) {
	var h *libvirtxml.DomainMemoryHugepages
	if hugepage == nil {
		return nil, nil
	}
	h = &libvirtxml.DomainMemoryHugepages{}
	for _, v := range hugepage.HugePage {
		s, err := strconv.ParseUint(v.Size, 10, 32)
		if err != nil {
			return nil, err
		}
		h.Hugepages = append(h.Hugepages, libvirtxml.DomainMemoryHugepage{
			Size:    uint(s),
			Unit:    v.Unit,
			Nodeset: v.NodeSet,
		})
	}
	return h, nil
}

func ConvertKubeVirtMemoryBackingToDomainMemoryBacking(memBack *api.MemoryBacking) (*libvirtxml.DomainMemoryBacking, error) {
	var err error
	domMemBack := &libvirtxml.DomainMemoryBacking{}
	if memBack == nil {
		return nil, nil
	}
	domMemBack.MemoryHugePages, err = ConvertKubeVirtHugepageToDomainMemoryHugepages(memBack.HugePages)
	if err != nil {
		return nil, err
	}
	if memBack.Source != nil {
		domMemBack.MemorySource = &libvirtxml.DomainMemorySource{
			Type: memBack.Source.Type,
		}
	}
	if memBack.Access != nil {
		domMemBack.MemoryAccess = &libvirtxml.DomainMemoryAccess{
			Mode: memBack.Access.Mode,
		}
	}
	if memBack.Allocation != nil {
		domMemBack.MemoryAllocation = &libvirtxml.DomainMemoryAllocation{
			Mode: string(memBack.Allocation.Mode),
		}
	}
	if memBack.NoSharePages != nil {
		domMemBack.MemoryNosharepages = &libvirtxml.DomainMemoryNosharepages{}
	}
	return domMemBack, nil
}

func setDomainFeatureState(fs *api.FeatureState) *libvirtxml.DomainFeatureState {
	if fs == nil {
		return nil
	}
	return &libvirtxml.DomainFeatureState{
		State: fs.State,
	}
}

func ConvertKubeVirtFeatureToDomainFeatureHyperVTLBFlush(fs *api.FeatureState) *libvirtxml.DomainFeatureHyperVTLBFlush {
	if fs == nil {
		return nil
	}
	return &libvirtxml.DomainFeatureHyperVTLBFlush{
		DomainFeatureState: libvirtxml.DomainFeatureState{
			State: fs.State,
		},
	}
}

func ConvertKubeVirtFeatureSpinlocksToDomainFeatureHyperVSpinlocks(s *api.FeatureSpinlocks) *libvirtxml.DomainFeatureHyperVSpinlocks {
	if s == nil {
		return nil
	}
	ds := &libvirtxml.DomainFeatureHyperVSpinlocks{
		DomainFeatureState: libvirtxml.DomainFeatureState{State: s.State},
	}
	if s.Retries != nil {
		ds.Retries = uint(*s.Retries)
	}
	return ds
}

func ConvertKubeVirtSyNICTimerToDomainFeatureHyperVSTimer(t *api.SyNICTimer) *libvirtxml.DomainFeatureHyperVSTimer {
	if t == nil {
		return nil
	}
	return &libvirtxml.DomainFeatureHyperVSTimer{
		DomainFeatureState: libvirtxml.DomainFeatureState{State: t.State},
		Direct:             setDomainFeatureState(t.Direct),
	}
}

func ConvertKubeVirtFeatureVendorIDToDomainFeatureHyperVVendorId(vid *api.FeatureVendorID) *libvirtxml.DomainFeatureHyperVVendorId {
	if vid == nil {
		return nil
	}
	return &libvirtxml.DomainFeatureHyperVVendorId{
		DomainFeatureState: libvirtxml.DomainFeatureState{State: vid.State},
		Value:              vid.Value,
	}
}

func ConvertKubeVirtFeatureHypervToDomainFeatureHyperV(hv *api.FeatureHyperv) *libvirtxml.DomainFeatureHyperV {
	if hv == nil {
		return nil
	}
	return &libvirtxml.DomainFeatureHyperV{
		Relaxed:         setDomainFeatureState(hv.Relaxed),
		Spinlocks:       ConvertKubeVirtFeatureSpinlocksToDomainFeatureHyperVSpinlocks(hv.Spinlocks),
		VAPIC:           setDomainFeatureState(hv.VAPIC),
		VPIndex:         setDomainFeatureState(hv.VPIndex),
		Runtime:         setDomainFeatureState(hv.Runtime),
		Synic:           setDomainFeatureState(hv.SyNIC),
		STimer:          ConvertKubeVirtSyNICTimerToDomainFeatureHyperVSTimer(hv.SyNICTimer),
		Reset:           setDomainFeatureState(hv.Reset),
		VendorId:        ConvertKubeVirtFeatureVendorIDToDomainFeatureHyperVVendorId(hv.VendorID),
		Frequencies:     setDomainFeatureState(hv.Frequencies),
		ReEnlightenment: setDomainFeatureState(hv.Reenlightenment),
		TLBFlush:        ConvertKubeVirtFeatureToDomainFeatureHyperVTLBFlush(hv.TLBFlush),
		IPI:             setDomainFeatureState(hv.IPI),
		EVMCS:           setDomainFeatureState(hv.EVMCS),
	}
}

func ConverKubeVirtFeatureKVMToDomainFeatureKVM(fkvm *api.FeatureKVM) *libvirtxml.DomainFeatureKVM {
	if fkvm == nil {
		return nil
	}
	return &libvirtxml.DomainFeatureKVM{
		Hidden:        setDomainFeatureState(fkvm.Hidden),
		HintDedicated: setDomainFeatureState(fkvm.HintDedicated),
	}
}

func ConvertKubeVirtFeaturesToDomainFeatureList(features *api.Features) *libvirtxml.DomainFeatureList {
	if features == nil {
		return nil
	}
	f := &libvirtxml.DomainFeatureList{}
	if features.ACPI != nil {
		f.ACPI = &libvirtxml.DomainFeature{}
	}
	if features.APIC != nil {
		f.APIC = &libvirtxml.DomainFeatureAPIC{}
	}
	if features.SMM != nil {
		f.SMM = &libvirtxml.DomainFeatureSMM{}
	}
	if features.PVSpinlock != nil {
		f.PVSpinlock = &libvirtxml.DomainFeatureState{
			State: features.PVSpinlock.State,
		}
	}
	f.PMU = setDomainFeatureState(features.PMU)
	f.HyperV = ConvertKubeVirtFeatureHypervToDomainFeatureHyperV(features.Hyperv)
	f.KVM = ConverKubeVirtFeatureKVMToDomainFeatureKVM(features.KVM)
	f.VMPort = setDomainFeatureState(features.VMPort)
	return f

}
