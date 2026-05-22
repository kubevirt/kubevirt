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

// Package iommu_pci provides IOMMU (Input-Output Memory Management Unit) and
// PCI device capability checking for KubeVirt virtual machines.
//
// This package is primarily used on ARM64 systems with NVIDIA GPU virtualization
// to verify that devices and the system have the necessary IOMMU capabilities
// for safe and efficient device passthrough.
//
// Key functionality:
//   - Detect SMMUv3 (ARM System Memory Management Unit) availability
//   - Check PCI device capabilities (ATS, PASID)
//   - Calculate required PCI memory hole sizes for device BARs
//   - Determine CPU Output Address Size (OAS) for address translation
//   - Support both modern IOMMUFD and legacy VFIO device access interfaces
//
// On non-ARM64 architectures, the functions are no-ops but maintain the same
// API for cross-platform compatibility.
package iommu_pci

import (
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	// defaultOAS is the default Output Address Size in bits for the IOMMU
	// when the actual OAS cannot be determined from the CPU
	defaultOAS = 48
	// PCIHoleMarginKiB is the safety margin in KiB added to the calculated PCI hole size
	// to account for potential alignment and allocation overhead
	PCIHoleMarginKiB = uint64(1024 * 1024)
)

var (
	// ParseConfigHybridFn overrides the architecture-specific parseConfigHybrid
	// implementation when non-nil. This allows tests to stub out sysfs/ioctl
	// operations that are only available on real hardware.
	ParseConfigHybridFn func(string) (atsSupported, atsEnabled, pasidSupported bool, ssidSize, oasBits int, err error)

	// CalculatePCIHole64SizeFn overrides the architecture-specific
	// CalculatePCIHole64Size implementation when non-nil.
	CalculatePCIHole64SizeFn func(string) (uint64, error)
)

// IommuPCI represents the IOMMU configuration for PCI devices
// on ARM64 systems with SMMUv3 (System Memory Management Unit version 3)
type IommuPCI struct {
	// IommufdEnabled indicates whether Iommufd driver is enabled on the system
	IommufdEnabled *bool
	// SMMUEnabled indicates whether SMMUv3 is available on the system
	SMMUEnabled bool
	// PCIHoleSize is the total size needed for the 64-bit PCI hole in KiB
	PCIHoleSize uint64
}

// BDF represents a single PCI device identified by Bus:Device.Function
// and its IOMMU-related capabilities and requirements
type BDF struct {
	// ID is the full BDF identifier string (e.g., "0000:3b:00.0")
	ID string
	// ATSSupported indicates if Address Translation Services are supported
	ATSSupported bool
	// ATSEnabled indicates if ATS is currently enabled
	ATSEnabled bool
	// PASIDSupported indicates if Process Address Space ID is supported
	PASIDSupported bool
	// SSIDSize is the SubStream ID size for PASID
	SSIDSize int
	// OASBits is Output Address Size in bits (typically 48 or 52 for ARM)
	OASBits int
}

// NewIommuPCI creates a new IommuPCI checker that verifies IOMMU capabilities
// and system configuration for device passthrough.
//
// On ARM64 systems, this function:
//   - Detects if SMMUv3 (ARM IOMMU) is available in the kernel
//   - Reads the CPU's Output Address Size (OAS) from hardware registers
//   - Initializes tracking for PCI memory hole requirements
//
// On other architectures, the checks are skipped but the structure is still
// created for API consistency.
func NewIommuPCI(arch string) *IommuPCI {
	if arch != "arm64" {
		log.Log.V(3).Info("Skipping IOMMUFD/SMMUv3 check for non-arm64 architecture")
		return &IommuPCI{}
	}

	iommufdEnabled, err := IommufdEnabled()
	if err != nil {
		log.Log.Errorf("error checking Iommufd: %v", err)
	}
	smmuEnabled, err := isSMMUv3Enabled()
	if err != nil {
		log.Log.Errorf("error checking SMMUv3: %v", err)
	}

	log.Log.V(3).Infof("iommufd: %t", iommufdEnabled)
	log.Log.V(3).Infof("smmuv3: %t", smmuEnabled)

	return &IommuPCI{
		IommufdEnabled: pointer.P(iommufdEnabled),
		SMMUEnabled:    smmuEnabled,
		PCIHoleSize:    0,
	}
}

// NewBDFDevice creates a new BDF (Bus:Device.Function) device object
// for tracking PCI device IOMMU capabilities.
func NewBDFDevice(id string) *BDF {
	return &BDF{
		ID: id,
	}
}

// ParseConfigHybrid reads and parses the PCI extended configuration space
// for this device to determine its IOMMU-related capabilities.
//
// It extracts:
//   - IOMMUFD support (whether the modern interface is being used)
//   - ATS (Address Translation Services) support and enablement status
//   - PASID (Process Address Space ID) support and SSID size
//   - OAS (Output Address Size) in bits
//
// These capabilities are critical for determining if GPU passthrough with
// IOMMU will work correctly.
func (bdf *BDF) ParseConfigHybrid() (*BDF, error) {
	fn := parseConfigHybrid
	if ParseConfigHybridFn != nil {
		fn = ParseConfigHybridFn
	}
	atsSupported, atsEnabled, pasidSupported, ssidSize, oasBits, err := fn(bdf.ID)
	if err != nil {
		log.Log.Errorf("Failed to process %s: %v", bdf.ID, err)
		return bdf, err
	}
	log.Log.V(3).Infof("ATS supported: %v, enabled: %v\n", atsSupported, atsEnabled)
	log.Log.V(3).Infof("PASID supported: %v, ssidSize: %d\n", pasidSupported, ssidSize)
	log.Log.V(3).Infof("OAS bits: %d\n", oasBits)

	bdf.ATSSupported = atsSupported
	bdf.ATSEnabled = atsEnabled
	bdf.PASIDSupported = pasidSupported
	bdf.SSIDSize = ssidSize
	bdf.OASBits = oasBits
	return bdf, nil
}

// CalculatePCIHoleSize computes the total size of 64-bit PCI memory regions
// (BARs - Base Address Registers) required by this device.
//
// The "PCI hole" is a region of guest physical memory address space reserved
// for mapping device memory. This is needed for:
//   - GPU framebuffer memory
//   - Device registers and control structures
//   - PCIe configuration space
//
// The calculation only includes 64-bit prefetchable memory regions, which
// are the type used by modern GPUs for their large memory BARs.
func (bdf *BDF) CalculatePCIHoleSize() (uint64, error) {
	fn := CalculatePCIHole64Size
	if CalculatePCIHole64SizeFn != nil {
		fn = CalculatePCIHole64SizeFn
	}
	holeSize, err := fn(bdf.ID)
	if err != nil {
		log.Log.Errorf("Failed calculating pcihole64 size of %s: %v", bdf.ID, err)
		return 0, err
	}
	return holeSize, nil
}
