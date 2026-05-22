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

package iommu

import (
	"fmt"
	"strconv"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	iommupci "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/iommu-pci"
)

// HandleIOMMU configures IOMMU settings for the domain spec on systems
// that require special NUMA node handling for device passthrough.
//
// On ARM64 systems with SMMUv3, certain devices (like NVIDIA GPUs) require
// fake NUMA nodes to be created for proper memory mapping and DMA isolation.
// This function orchestrates the IOMMU configuration by calling appropriate
// handlers based on the system capabilities.
//
// Parameters:
//   - domain: The libvirt domain specification to modify
//   - iommu: IOMMU configuration object containing system capabilities
//
// The function is a no-op if IOMMU is not initialized or SMMUv3 is not enabled.
func HandleIOMMU(domain *api.DomainSpec, iommu *iommupci.IommuPCI) {
	// FakeNumaNodes and Iommu handling is required only in SMMUv3 enabled systems
	if iommu == nil || !iommu.SMMUEnabled {
		return
	}
	handleFakeNumaNodes(domain)
	applyNUMADistances(domain)
}

// handleFakeNumaNodes creates fake NUMA nodes for host devices that require them.
//
// This function is specifically needed for NVIDIA GPU passthrough on ARM64 systems
// with SMMUv3. The NVIDIA vGPU driver requires a set of 8 contiguous NUMA nodes to
// properly map device memory regions and handle DMA operations.
//
// For each host device marked with ACPI NodeSet "tofill":
//  1. Verifies NUMA configuration exists
//  2. Finds the highest existing NUMA cell ID
//  3. Creates 8 new zero-memory NUMA cells with sequential IDs
//  4. Updates the device's ACPI NodeSet to reference the new cells
//
// If NUMA is not configured or an error occurs, the ACPI configuration is cleared
// to allow fallback to non-NUMA device placement.
//
// Parameters:
//   - domain: The libvirt domain specification to modify
func handleFakeNumaNodes(domain *api.DomainSpec) {
	for i := range domain.Devices.HostDevices {
		hostDev := &domain.Devices.HostDevices[i]
		if hostDev.ACPI != nil && hostDev.ACPI.NodeSet == "tofill" {
			// Skip if NUMA is not configured on the VM
			if domain.CPU.NUMA == nil {
				hostDev.ACPI = nil
				continue
			}
			if len(domain.CPU.NUMA.Cells) == 0 {
				hostDev.ACPI = nil
				continue
			}
			// Find the highest existing NUMA cell ID
			initialNumaCellId, err := getMaxNumaCellId(domain.CPU.NUMA.Cells)
			if err != nil {
				hostDev.ACPI = nil
				continue
			}
			// Start new fake cells after the highest existing cell
			initialNumaCellId++
			finalNumaCellId := initialNumaCellId

			// Create 8 fake NUMA nodes with zero memory
			// These nodes are used by the NVIDIA driver for address mapping
			for count := range 8 {
				domain.CPU.NUMA.Cells = append(domain.CPU.NUMA.Cells, api.NUMACell{
					ID:     fmt.Sprintf("%d", initialNumaCellId+count),
					Memory: pointer.P(uint64(0)),
					Unit:   "KiB",
				})
				finalNumaCellId = initialNumaCellId + count
			}

			// Set the device's ACPI NodeSet to span the new fake nodes
			hostDev.ACPI.NodeSet = fmt.Sprintf("%d-%d", initialNumaCellId, finalNumaCellId)
		}
	}
}

// getMaxNumaCellId finds the highest NUMA cell ID among the provided cells.
//
// This is used to determine where to start allocating new fake NUMA nodes,
// ensuring they don't conflict with existing NUMA configuration.
//
// Parameters:
//   - cells: Slice of existing NUMA cells to search
//
// Returns:
//   - int: The highest cell ID found
//   - error: Error if a cell ID cannot be parsed as an integer
func getMaxNumaCellId(cells []api.NUMACell) (int, error) {
	result := 0
	for _, cell := range cells {
		cellID, err := strconv.Atoi(cell.ID)
		if err != nil {
			return 0, err
		}
		if cellID > result {
			result = cellID
		}
	}
	return result, nil
}
