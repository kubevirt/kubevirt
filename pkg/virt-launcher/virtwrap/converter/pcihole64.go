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

	// pciHole64StartMarginBytes accounts for the guest-visible address
	// space QEMU reserves below the 64-bit PCI hole (e.g. the 32-bit PCI
	// hole and various fixed ranges) when checking the hole against
	// maxSafePhysBits.
	pciHole64StartMarginBytes = uint64(4) << 30 // 4 GiB
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

	pciHoleKiB := hardware.PCIHole64KiB(totalBARBytes, passthroughPCIHole64MarginKiB)
	if pciHoleKiB <= q35DefaultPCIHole64KiB {
		return 0
	}

	if !fitsGuestPhysAddressSpace(vmi, pciHoleKiB) {
		log.Log.Warningf(
			"Skipping pcihole64 expansion: VMI needs a %d KiB PCI hole for its passthrough devices, "+
				"but its CPU model is not host-passthrough so the guest's physical address width cannot "+
				"be assumed large enough; set spec.domain.cpu.model to host-passthrough to use these devices", pciHoleKiB)
		return 0
	}

	return uint(pciHoleKiB)
}

// fitsGuestPhysAddressSpace reports whether a pcihole64 of pciHoleKiB is safe
// given the VMI's guest memory and CPU model. KubeVirt never sets an explicit
// <maxphysaddr>, so a named or host-model CPU gets QEMU's default phys-bits
// (maxSafePhysBits); only host-passthrough reliably exposes the host's real,
// typically much larger, physical address width.
func fitsGuestPhysAddressSpace(vmi *v1.VirtualMachineInstance, pciHoleKiB uint64) bool {
	if vmi.Spec.Domain.CPU != nil && vmi.Spec.Domain.CPU.Model == v1.CPUModeHostPassthrough {
		return true
	}

	guestMemory := guestMemoryBytes(vmi)
	topOfHoleBytes := guestMemory + pciHole64StartMarginBytes + pciHoleKiB<<10
	return topOfHoleBytes <= uint64(1)<<maxSafePhysBits
}

func guestMemoryBytes(vmi *v1.VirtualMachineInstance) uint64 {
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.MaxGuest != nil {
		return uint64(vmi.Spec.Domain.Memory.MaxGuest.Value())
	}
	return uint64(vcpu.GetVirtualMemory(vmi).Value())
}
