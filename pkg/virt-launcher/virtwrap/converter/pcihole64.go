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

package converter

import (
	"math"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
)

const (
	passthroughPCIHole64MarginKiB = uint64(1024 * 1024) // 1 GiB

	// q35DefaultPCIHole64KiB is QEMU's default 64-bit PCI MMIO hole on the
	// q35 machine type. Sizes at or below this already fit without an
	// explicit pcihole64 override.
	q35DefaultPCIHole64KiB = uint64(32) << 20

	// maxSafePhysBits is the phys-bits QEMU assumes for a named/host-model
	// CPU when no explicit value is configured. A CPU mode other than
	// host-passthrough cannot be trusted to expose more than this, so a
	// pcihole64 override that would push the top of guest address space
	// beyond it is skipped rather than risk a VM that fails to start.
	maxSafePhysBits = uint64(40)

	// above4GStartBytes is where QEMU's q35 machine normally starts mapping
	// guest RAM above the 32-bit PCI hole.
	above4GStartBytes = uint64(4) << 30 // 4 GiB

	// amdHTReservedStartBytes is the start of the HyperTransport reserved
	// range on AMD platforms. QEMU's pc_q35 code relocates the above-4G RAM
	// region from above4GStartBytes to amdRelocatedStartBytes when, with an
	// AMD guest CPU, RAM would otherwise overlap this range.
	amdHTReservedStartBytes = uint64(0xfd00000000)

	// amdRelocatedStartBytes is where QEMU restarts the above-4G RAM region
	// once relocated; it is also the end of the HyperTransport reserved range.
	amdRelocatedStartBytes = uint64(1) << 40 // 1 TiB

	// pciHole64AlignmentBytes is the minimum alignment QEMU enforces for the
	// 64-bit PCI hole's start address.
	pciHole64AlignmentBytes = uint64(1) << 30 // 1 GiB
)

// passthroughPCIDevicesPath is overridable in tests, mirroring
// graceRuntimeInfo's injectable sysfs path in grace.go.
var passthroughPCIDevicesPath = hardware.PciBasePath

// passthroughPCIHole64KiB returns the 64-bit PCI MMIO hole size, in KiB,
// needed for the VMI's assigned GPUs and host devices to have their PCI BARs
// mapped, or 0 if QEMU's q35 default hole is already sufficient.
//
// The hole is sized from the real BARs of the assigned devices (summed,
// margined and rounded up to a power of two, as Grace IO virtualization does
// for arm64 in grace.go) because QEMU only aligns the hole's start address to
// 1 GiB: a power-of-two window is what guarantees a naturally aligned BAR of
// that size fits inside it.
func passthroughPCIHole64KiB(vmi *v1.VirtualMachineInstance, hostDeviceLists ...[]api.HostDevice) uint {
	var totalBARBytes uint64
	for _, hostDevices := range hostDeviceLists {
		for i := range hostDevices {
			hostDevice := &hostDevices[i]
			if hostDevice.Type != api.HostDevicePCI {
				continue
			}

			bdf := hardware.PCIAddressToString(hostDevice.Source.Address)
			barBytes, err := hardware.PCI64BitPrefetchableBARBytes(passthroughPCIDevicesPath, bdf)
			if err != nil {
				// Skip the override for the whole VMI rather than just this
				// device: a hole sized from only the readable devices could
				// be too small for the one that failed, which is worse than
				// falling back to today's default hole.
				log.Log.Reason(err).Warningf(
					"Failed to read PCI BARs for passthrough device %s, skipping pcihole64 sizing", bdf)
				return 0
			}

			if barBytes > math.MaxUint64-totalBARBytes {
				log.Log.Warningf("Passthrough PCI BAR total overflowed while sizing pcihole64, skipping")
				return 0
			}
			totalBARBytes += barBytes
		}
	}

	pciHoleKiB, err := hardware.PCIHole64KiB(totalBARBytes, passthroughPCIHole64MarginKiB)
	if err != nil {
		log.Log.Reason(err).Warningf("Failed to size pcihole64 for passthrough devices, skipping")
		return 0
	}
	if pciHoleKiB <= q35DefaultPCIHole64KiB {
		return 0
	}

	if fits, effectivePhysBits := fitsGuestPhysAddressSpace(vmi, pciHoleKiB); !fits {
		log.Log.Warningf(
			"Skipping pcihole64 expansion: VMI needs a %d KiB PCI hole for its passthrough devices, "+
				"but that does not fit under the %d-bit guest physical address width available to its CPU "+
				"model; set spec.domain.cpu.model to host-passthrough to use these devices", pciHoleKiB, effectivePhysBits)
		return 0
	}

	return uint(pciHoleKiB)
}

// fitsGuestPhysAddressSpace reports whether a pcihole64 of pciHoleKiB fits
// under the guest's physical address width, and reports that width. KubeVirt
// never sets an explicit <maxphysaddr>, so a named or host-model CPU gets
// QEMU's default phys-bits (maxSafePhysBits); only host-passthrough reliably
// exposes the host's real, typically much larger, physical address width.
func fitsGuestPhysAddressSpace(vmi *v1.VirtualMachineInstance, pciHoleKiB uint64) (bool, uint64) {
	effectivePhysBits := maxSafePhysBits
	// Conservative default: without reading the host CPU we cannot tell
	// whether a named or host-model CPU is emulating AMD, so assume the
	// AMD HyperTransport reserved range could apply. That only ever makes
	// the check stricter (a higher required top-of-hole), never lets an
	// unsafe hole through.
	assumeAMDReservedRange := true

	if vmi.Spec.Domain.CPU != nil && vmi.Spec.Domain.CPU.Model == v1.CPUModeHostPassthrough {
		hostPhysBits, vendorID, err := hardware.HostCPUAddressInfo()
		if err != nil {
			log.Log.Reason(err).Warningf("Failed to read host CPU address width, assuming %d bits", maxSafePhysBits)
		} else {
			effectivePhysBits = hostPhysBits
			assumeAMDReservedRange = vendorID == "AuthenticAMD"
		}
	}

	guestMemory := guestMemoryBytes(vmi)
	var ramAbove4G uint64
	if guestMemory > above4GStartBytes {
		ramAbove4G = guestMemory - above4GStartBytes
	}

	above4GStart := above4GStartBytes
	if assumeAMDReservedRange && above4GStart+ramAbove4G > amdHTReservedStartBytes {
		above4GStart = amdRelocatedStartBytes
	}

	// pciHoleKiB is only guaranteed to fit in a uint64 KiB value; converting
	// it to bytes below would silently wrap around if it does not also fit
	// after the <<10 shift, so treat that as not fitting.
	if pciHoleKiB > math.MaxUint64>>10 {
		return false, effectivePhysBits
	}

	holeAlignment := pciHole64AlignmentBytes
	if pciHoleKiB<<10 > holeAlignment {
		holeAlignment = pciHoleKiB << 10
	}
	holeStart := roundUpToMultiple(above4GStart+ramAbove4G, holeAlignment)

	topOfHoleBytes := holeStart + pciHoleKiB<<10
	return topOfHoleBytes <= uint64(1)<<effectivePhysBits, effectivePhysBits
}

func roundUpToMultiple(value, multiple uint64) uint64 {
	remainder := value % multiple
	if remainder == 0 {
		return value
	}
	return value + (multiple - remainder)
}

func guestMemoryBytes(vmi *v1.VirtualMachineInstance) uint64 {
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.MaxGuest != nil {
		return uint64(vmi.Spec.Domain.Memory.MaxGuest.Value())
	}
	return uint64(vcpu.GetVirtualMemory(vmi).Value())
}
