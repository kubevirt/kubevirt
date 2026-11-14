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
 * Copyright 2023 The KubeVirt Authors.
 *
 */

package libvirtxml

import (
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
