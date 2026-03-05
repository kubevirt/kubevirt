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
 * Copyright The KubeVirt Authors.
 *
 */

package virthandler

import (
	"strconv"
	"strings"

	v1 "kubevirt.io/api/core/v1"
	"libvirt.org/go/libvirtxml"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
)

func virtualMachineOptions(
	smbios *v1.SMBiosConfiguration,
	period uint32,
	preallocatedVolumes []string,
	capabilities *libvirtxml.Caps,
	clusterConfig *virtconfig.ClusterConfig,
) *cmdv1.VirtualMachineOptions {
	options := &cmdv1.VirtualMachineOptions{
		MemBalloonStatsPeriod: period,
		PreallocatedVolumes:   preallocatedVolumes,
		Topology:              capabilitiesToTopology(capabilities),
		// New virt-launcher images no longer use this value, it's kept empty for backward compatibility.
		DisksInfo: map[string]*cmdv1.DiskInfo{},
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
		bochsDisplay := true
		if clusterConfig.VGADisplayForEFIGuestsEnabled() {
			bochsDisplay = false
		}
		options.ExpandDisksEnabled = clusterConfig.ExpandDisksEnabled()
		options.ClusterConfig = &cmdv1.ClusterConfig{
			ExpandDisksEnabled:          clusterConfig.ExpandDisksEnabled(),
			FreePageReportingDisabled:   clusterConfig.IsFreePageReportingDisabled(),
			BochsDisplayForEFIGuests:    bochsDisplay,
			SerialConsoleLogDisabled:    clusterConfig.IsSerialConsoleLogDisabled(),
			PCINUMAAwareTopologyEnabled: clusterConfig.PCINUMAAwareTopologyEnabled(),
		}
	}

	return options
}

func capabilitiesToTopology(capabilities *libvirtxml.Caps) *cmdv1.Topology {
	topology := &cmdv1.Topology{}
	if capabilities == nil {
		return topology
	}

	for _, cell := range capabilities.Host.NUMA.Cells.Cells {
		topology.NumaCells = append(topology.NumaCells, cellToCell(cell))
	}
	return topology
}

func cellToCell(cell libvirtxml.CapsHostNUMACell) *cmdv1.Cell {
	c := &cmdv1.Cell{
		Id: uint32(cell.ID),
	}

	if cell.Memory != nil {
		c.Memory = &cmdv1.Memory{
			Amount: cell.Memory.Size,
			Unit:   cell.Memory.Unit,
		}
	}

	for _, page := range cell.PageInfo {
		c.Pages = append(c.Pages, pageToPage(page))
	}
	if cell.Distances != nil {
		for _, distance := range cell.Distances.Siblings {
			c.Distances = append(c.Distances, distanceToDistance(distance))
		}
	}
	if cell.CPUS != nil {
		for _, cpu := range cell.CPUS.CPUs {
			c.Cpus = append(c.Cpus, cpuToCPU(cpu))
		}
	}
	return c
}

func pageToPage(pages libvirtxml.CapsHostNUMAPageInfo) *cmdv1.Pages {
	return &cmdv1.Pages{
		Count: pages.Count,
		Unit:  pages.Unit,
		Size:  uint32(pages.Size),
	}
}

func distanceToDistance(distance libvirtxml.CapsHostNUMASibling) *cmdv1.Sibling {
	return &cmdv1.Sibling{
		Id:    uint32(distance.ID),
		Value: uint64(distance.Value),
	}
}

func cpuToCPU(cpu libvirtxml.CapsHostNUMACPU) *cmdv1.CPU {
	return &cmdv1.CPU{
		Id:       uint32(cpu.ID),
		Siblings: convertListOfIntStringToSlice(cpu.Siblings),
	}
}

func convertListOfIntStringToSlice(siblings string) []uint32 {
	var convertedSiblings []uint32
	for _, sibling := range strings.Split(siblings, ",") {
		num, err := strconv.ParseUint(sibling, 10, 32)
		if err != nil {
			// Sibling must be int, otherwise skip
			continue
		}
		convertedSiblings = append(convertedSiblings, uint32(num))
	}
	return convertedSiblings
}
