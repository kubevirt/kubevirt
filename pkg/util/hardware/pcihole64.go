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

package hardware

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// Linux exposes IORESOURCE_* bits in the sysfs PCI resource file along
	// with PCI BAR memory type bits in the low nibble.
	ioResourceMem                = uint64(0x00000200)
	pciBaseAddressMemoryTypeMask = uint64(0x0000000f)
	pciBaseAddressMemoryType64   = uint64(0x00000004)
	pciBaseAddressMemoryPrefetch = uint64(0x00000008)

	// pciStdNumBars is PCI_STD_NUM_BARS in the Linux kernel: the sysfs
	// "resource" file has one line per PCI_NUM_RESOURCES slot, and only the
	// first 6 are the device's own standard BARs. The remaining lines are
	// expansion ROM, bridge windows and, on an SR-IOV physical function, the
	// PCI_IOV_RESOURCES VF BAR aggregate reservations (sriov_totalvfs times
	// the per-VF BAR size). A passthrough PF is never handed its whole VF
	// pool, so those aggregate lines must not be summed into its aperture.
	pciStdNumBars = 6
)

// PCI64BitPrefetchableBARBytes sums the 64-bit prefetchable standard PCI BARs
// of the device at bdf under devicesPath, as reported by its sysfs
// "resource" file.
func PCI64BitPrefetchableBARBytes(devicesPath, bdf string) (uint64, error) {
	resourcePath := filepath.Join(devicesPath, bdf, "resource")
	file, err := os.Open(resourcePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open %s: %w", resourcePath, err)
	}
	defer file.Close()

	var total uint64
	scanner := bufio.NewScanner(file)
	for line := 0; line < pciStdNumBars && scanner.Scan(); line++ {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 {
			continue
		}
		start, err := strconv.ParseUint(strings.TrimPrefix(fields[0], "0x"), 16, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse start address on %s line %d (%q): %w", resourcePath, line, fields[0], err)
		}
		end, err := strconv.ParseUint(strings.TrimPrefix(fields[1], "0x"), 16, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse end address on %s line %d (%q): %w", resourcePath, line, fields[1], err)
		}
		flags, err := strconv.ParseUint(strings.TrimPrefix(fields[2], "0x"), 16, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse flags on %s line %d (%q): %w", resourcePath, line, fields[2], err)
		}
		if start == 0 || end == 0 || start > end {
			continue
		}
		if flags&ioResourceMem != ioResourceMem ||
			flags&pciBaseAddressMemoryTypeMask != pciBaseAddressMemoryType64|pciBaseAddressMemoryPrefetch {
			continue
		}
		total += end - start + 1
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("failed to read %s: %w", resourcePath, err)
	}
	return total, nil
}

// PCIHole64KiB adds marginKiB to sizeBytes (rounded up to the next KiB) and
// rounds the result up to the next power of two. It returns (0, nil) when
// sizeBytes is 0, and an error if the margin addition or the rounding would
// overflow a uint64.
func PCIHole64KiB(sizeBytes, marginKiB uint64) (uint64, error) {
	if sizeBytes == 0 {
		return 0, nil
	}
	sizeKiB := sizeBytes / 1024
	if sizeBytes%1024 != 0 {
		sizeKiB++
	}
	if sizeKiB > math.MaxUint64-marginKiB {
		return 0, fmt.Errorf("pcihole64 size %d KiB plus margin %d KiB overflows uint64", sizeKiB, marginKiB)
	}
	sizeKiB += marginKiB

	var rounded uint64 = 1
	for rounded < sizeKiB {
		if rounded > math.MaxUint64/2 {
			return 0, fmt.Errorf("pcihole64 size %d KiB rounded to a power of two overflows uint64", sizeKiB)
		}
		rounded <<= 1
	}
	return rounded, nil
}

// CPUInfoPath is overridable in tests.
var CPUInfoPath = "/proc/cpuinfo"

// HostCPUAddressInfo returns the host's physical address width, in bits, and
// its CPU vendor ID, parsed from the first processor entry in /proc/cpuinfo.
// A guest using CPU mode host-passthrough is given this same address width
// and vendor by QEMU.
func HostCPUAddressInfo() (physBits uint64, vendorID string, err error) {
	file, err := os.Open(CPUInfoPath)
	if err != nil {
		return 0, "", fmt.Errorf("failed to open %s: %w", CPUInfoPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, found := strings.Cut(scanner.Text(), ":")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case "vendor_id":
			vendorID = value
		case "address sizes":
			// e.g. "46 bits physical, 48 bits virtual"
			fields := strings.Fields(value)
			if len(fields) < 2 {
				continue
			}
			physBits, err = strconv.ParseUint(fields[0], 10, 64)
			if err != nil {
				return 0, "", fmt.Errorf("failed to parse physical address bits from %s (%q): %w", CPUInfoPath, value, err)
			}
		}
		if physBits != 0 && vendorID != "" {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, "", fmt.Errorf("failed to read %s: %w", CPUInfoPath, err)
	}
	if physBits == 0 {
		return 0, "", fmt.Errorf("no \"address sizes\" line found in %s", CPUInfoPath)
	}
	return physBits, vendorID, nil
}
