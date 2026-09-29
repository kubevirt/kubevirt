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

package thp

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	// minCollapseRegionSize is a strict lower bound (exclusive) on VMA size.
	// Small device/firmware RAMBlocks are also THPeligible (vga.vram default
	// is exactly 16Mi; ACPI tables reservation is 2Mi). Guest RAM is always
	// larger even when split between NUMA nodes.
	minCollapseRegionSize = 16 * 1024 * 1024
)

// Region is a THPeligible guest-RAM VMA with smaps THP accounting.
type Region struct {
	Start          uintptr
	Size           uint64 // bytes
	AnonHugePages  uint64 // bytes
	ShmemPmdMapped uint64 // bytes
}

// Coverage returns (AnonHugePages+ShmemPmdMapped)/Size for this region
func (r Region) Coverage() float64 {
	if r.Size == 0 {
		return 0
	}
	return float64(r.AnonHugePages+r.ShmemPmdMapped) / float64(r.Size)
}

// FullyBacked reports whether this region's coverage meets GuaranteedCoverageThreshold.
func (r Region) FullyBacked() bool {
	return r.Coverage() >= GuaranteedCoverageThreshold
}

// ParseTHPEligibleRegions returns VMAs from /proc/<pid>/smaps that are
// THPeligible and larger than minCollapseRegionSize.
func ParseTHPEligibleRegions(smaps string) ([]Region, error) {
	var (
		regions     []Region
		start, end  uint64
		haveHeader  bool
		thpEligible bool
		sizeKiB     uint64
		anonHugeKiB uint64
		shmemPmdKiB uint64
	)

	flush := func() {
		if !haveHeader || !thpEligible {
			return
		}
		span := end - start
		if span <= minCollapseRegionSize {
			return
		}
		size := span
		if sizeKiB > 0 {
			size = sizeKiB * 1024
		}
		regions = append(regions, Region{
			Start:          uintptr(start),
			Size:           size,
			AnonHugePages:  anonHugeKiB * 1024,
			ShmemPmdMapped: shmemPmdKiB * 1024,
		})
	}

	scanner := bufio.NewScanner(strings.NewReader(smaps))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if fields := strings.Fields(line); len(fields) >= 5 && strings.Contains(fields[0], "-") {
			flush()
			haveHeader = false
			thpEligible = false
			sizeKiB = 0
			anonHugeKiB = 0
			shmemPmdKiB = 0
			addrRange := strings.Split(fields[0], "-")
			if len(addrRange) != 2 {
				continue
			}
			s, err1 := strconv.ParseUint(addrRange[0], 16, 64)
			e, err2 := strconv.ParseUint(addrRange[1], 16, 64)
			if err1 != nil || err2 != nil || e <= s {
				continue
			}
			start, end = s, e
			haveHeader = true
			continue
		}
		if !haveHeader {
			continue
		}
		switch {
		case strings.HasPrefix(line, "THPeligible:"):
			fields := strings.Fields(line)
			thpEligible = len(fields) >= 2 && fields[1] == "1"
		case strings.HasPrefix(line, "Size:"):
			sizeKiB = parseSmapsKiB(line)
		case strings.HasPrefix(line, "AnonHugePages:"):
			anonHugeKiB = parseSmapsKiB(line)
		case strings.HasPrefix(line, "ShmemPmdMapped:"):
			shmemPmdKiB = parseSmapsKiB(line)
		}
	}
	flush()
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan smaps: %w", err)
	}
	return regions, nil
}

func parseSmapsKiB(line string) uint64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	v, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func readProcessSmaps(pid int) (string, error) {
	return readProcessSmapsFunc(pid)
}

// readProcessSmapsFunc is replaced in tests.
var readProcessSmapsFunc = func(pid int) (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/smaps", pid))
	if err != nil {
		return "", err
	}
	return string(data), nil
}
