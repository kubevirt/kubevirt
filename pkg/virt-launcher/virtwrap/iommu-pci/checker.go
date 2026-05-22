//go:build !arm64

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

// Package iommu_pci provides IOMMU and PCI capability detection for device
// passthrough on ARM64 systems with SMMUv3.
//
// This file contains stub implementations for non-ARM64 architectures.
// All functions return safe defaults (false/0/nil) since IOMMU features
// like SMMUv3, IOMMUFD, and PCI hole sizing are specific to ARM64.
//
// The actual implementations are in checker-arm64.go, which uses CGo to
// perform ioctl-based capability queries against /dev/iommu and VFIO
// device files.
package iommu_pci

// IommufdEnabled checks if the IOMMUFD interface is available on the system.
// On non-ARM64 architectures, this always returns false since IOMMUFD is
// currently only used for ARM64 SMMUv3 device passthrough.
func IommufdEnabled() (bool, error) {
	return false, nil
}

// parseConfigHybrid reads the PCI extended configuration space for a device
// and extracts IOMMU-related capabilities (ATS, PASID, OAS).
//
// On non-ARM64 architectures, this is a no-op stub that returns all
// capabilities as disabled/zero.
func parseConfigHybrid(_ string) (atsSupported, atsEnabled, pasidSupported bool, ssidSize, oasBits int, err error) {
	return false, false, false, 0, 0, nil
}

// isSMMUv3Enabled checks if SMMUv3 (ARM System MMU v3) is present in the
// system's IORT (IO Remapping Table) ACPI table.
//
// Returns false on non-ARM64 architectures since SMMUv3 is ARM-specific.
func isSMMUv3Enabled() (bool, error) {
	return false, nil
}

// CalculatePCIHole64Size computes the total 64-bit PCI memory hole size
// required by a device based on its BAR (Base Address Register) regions.
//
// Returns 0 since PCI hole size calculation is not implemented for this
// architecture.
func CalculatePCIHole64Size(_ string) (uint64, error) {
	return 0, nil
}

// CalculateTotalPCIHole64Size is a stub for non-ARM64 architectures.
// On ARM64, this computes the total PCI hole size with margin and rounds
// up to the next power of 2 for proper memory alignment.
//
// Returns 0 since this calculation is not applicable for this architecture.
func CalculateTotalPCIHole64Size(bdfComputedSize uint64, marginKiB uint64) uint64 {
	return 0
}

// InferExtraNUMANodes is a stub for non-ARM64 architectures.
// On ARM64, this determines the NUMA node configuration needed for proper
// IOMMU setup, inferring additional virtual NUMA nodes for devices with
// SMMUv3 and PASID support.
//
// Returns empty results since this is not applicable for this architecture.
func InferExtraNUMANodes(_ string) ([]int, int, error) {
	return []int{}, -1, nil
}
