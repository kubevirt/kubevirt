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
)

// PCI64BitPrefetchableBARBytes sums the 64-bit prefetchable PCI BARs of the
// device at bdf under devicesPath, as reported by its sysfs "resource" file.
func PCI64BitPrefetchableBARBytes(devicesPath, bdf string) (uint64, error) {
	resourcePath := filepath.Join(devicesPath, bdf, "resource")
	file, err := os.Open(resourcePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var total uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 {
			continue
		}
		start, err := strconv.ParseUint(strings.TrimPrefix(fields[0], "0x"), 16, 64)
		if err != nil {
			return 0, err
		}
		end, err := strconv.ParseUint(strings.TrimPrefix(fields[1], "0x"), 16, 64)
		if err != nil {
			return 0, err
		}
		flags, err := strconv.ParseUint(strings.TrimPrefix(fields[2], "0x"), 16, 64)
		if err != nil {
			return 0, err
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
		return 0, err
	}
	return total, nil
}

// PCIHole64KiB adds marginKiB to sizeBytes (rounded up to the next KiB) and
// rounds the result up to the next power of two. It returns 0 on overflow or
// when sizeBytes is 0.
func PCIHole64KiB(sizeBytes, marginKiB uint64) uint64 {
	if sizeBytes == 0 {
		return 0
	}
	sizeKiB := sizeBytes / 1024
	if sizeBytes%1024 != 0 {
		sizeKiB++
	}
	if sizeKiB > math.MaxUint64-marginKiB {
		return 0
	}
	sizeKiB += marginKiB

	var rounded uint64 = 1
	for rounded < sizeKiB {
		if rounded > math.MaxUint64/2 {
			return 0
		}
		rounded <<= 1
	}
	return rounded
}
