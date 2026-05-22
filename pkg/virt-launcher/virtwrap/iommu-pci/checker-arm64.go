//go:build arm64

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

package iommu_pci

/*
#include <stdint.h>
uint64_t read_id_aa64mmfr0() {
    uint64_t id;
    // #nosec G103
    __asm__ volatile ("mrs %0, ID_AA64MMFR0_EL1" : "=r" (id));
    return id;
}
*/
import "C"
import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"

	"kubevirt.io/client-go/log"
)

// IOMMU and VFIO ioctl constants for interacting with the kernel's IOMMU subsystem.
const (
	// VFIO_DEVICE_BIND_IOMMUFD binds a VFIO device to an IOMMUFD
	VFIO_DEVICE_BIND_IOMMUFD = 0x3B76
	// IOMMU_GET_HW_INFO retrieves hardware-specific IOMMU information
	IOMMU_GET_HW_INFO = 0x3b8a
	// IOMMU_HW_INFO_FLAG_INPUT_TYPE indicates InOutDataType specifies the requested info type
	IOMMU_HW_INFO_FLAG_INPUT_TYPE = 1 << 0
	// IOMMU_HW_INFO_TYPE_DEFAULT requests generic IOMMU information
	IOMMU_HW_INFO_TYPE_DEFAULT = 0
	// IOMMU_HW_INFO_TYPE_ARM_SMMUV3 requests ARM SMMUv3-specific information
	IOMMU_HW_INFO_TYPE_ARM_SMMUV3 = 2
	// IOMMU_HW_CAP_PCI_ATS_NOT_SUPPORTED capability bit when PCI ATS is not supported
	IOMMU_HW_CAP_PCI_ATS_NOT_SUPPORTED = 1 << 3
)

// vfioDeviceBindIommufd is the argument structure for VFIO_DEVICE_BIND_IOMMUFD ioctl.
// It binds a VFIO device to an IOMMU file descriptor for modern IOMMU management.
type vfioDeviceBindIommufd struct {
	Argsz        uint32
	Flags        uint32
	Iommufd      int32
	OutDevid     uint32
	TokenUuidPtr uint64 // Optional pointer to token UUID for secure binding
}

// iommuHwInfo is the argument structure for IOMMU_GET_HW_INFO ioctl.
// It queries hardware-specific IOMMU capabilities and features.
type iommuHwInfo struct {
	Size            uint32   // Size of this structure
	Flags           uint32   // Input flags (e.g., IOMMU_HW_INFO_FLAG_INPUT_TYPE)
	DevID           uint32   // Device ID to query capabilities for
	DataLen         uint32   // Length of hardware-specific data buffer
	DataUptr        uint64   // Pointer to hardware-specific data (e.g., iommuHwInfoArmSmmuv3)
	InOutDataType   uint32   // Input: requested info type; Output: actual type returned
	OutMaxPasidLog2 uint8    // Output: log2 of maximum PASID size (0 if not supported)
	Reserved        [3]uint8 // Reserved for alignment
	OutCapabilities uint64   // Output: capability flags (e.g., ATS support)
}

// iommuHwInfoArmSmmuv3 contains ARM SMMUv3-specific hardware registers and information.
// This structure is populated when querying with IOMMU_HW_INFO_TYPE_ARM_SMMUV3.
type iommuHwInfoArmSmmuv3 struct {
	Flags    uint32    // SMMUv3-specific flags
	Reserved uint32    // Reserved for future use
	IDR      [6]uint32 // ID registers (IDR0-IDR5), including OAS (Output Address Size) in IDR5
	IIDR     uint32    // Implementation Identification Register
	AIDR     uint32    // Architecture Identification Register
}

// IOMMUCapabilities represents the parsed IOMMU capabilities for a PCI device.
// This provides a device-agnostic abstraction of IOMMU features.
type IOMMUCapabilities struct {
	SSIDSize       int  // SubStream ID size (same as log2 of max PASID)
	PASIDSupported bool // Whether Process Address Space ID (PASID) is supported
	ATSSupported   bool // Whether Address Translation Services (ATS) are supported
	OASBits        int  // Output Address Size in bits (typically 48 or 52 for ARM)
}

// IommufdEnabled checks if the IOMMUFD interface is available on the system.
// IOMMUFD (/dev/iommu) is the modern kernel interface for IOMMU management,
// required for features like GPU passthrough with PASID and ATS support on ARM64.
func IommufdEnabled() (bool, error) {
	// Check if /dev/iommu device node exists
	_, err := os.Stat("/dev/iommu")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Device doesn't exist - IOMMUFD not available
			return false, nil
		}
		// Other error (e.g., permission issue)
		return false, err
	}
	// Device exists - IOMMUFD is available
	return true, nil
}

// parseConfigHybrid is the main entry point for checking IOMMU capabilities of a PCI device.
// It uses the modern IOMMUFD interface with IOMMU_GET_HW_INFO to query device capabilities.
func parseConfigHybrid(bdf string) (atsSupported, atsEnabled, pasidSupported bool, ssidSize, oasBits int, err error) {
	// Find the VFIO character device path for this PCI device
	cdevPath := getCdevPath(bdf)
	if cdevPath == "" {
		return false, false, false, 0, 0, fmt.Errorf("no vfio device node found for %s", bdf)
	}

	// Bind the device to IOMMUFD and query its capabilities
	caps, err := bindDevice(cdevPath)
	if err != nil {
		return false, false, false, 0, 0, err
	}

	// Return successfully with the discovered capabilities
	return caps.ATSSupported, caps.ATSSupported, caps.PASIDSupported, caps.SSIDSize, caps.OASBits, nil
}

// bindDevice binds a VFIO device to the IOMMUFD subsystem and retrieves its capabilities.
// This is required for modern IOMMU management on ARM64 systems with SMMUv3.
func bindDevice(cdevPath string) (IOMMUCapabilities, error) {
	// Open the IOMMU file descriptor - this is the modern interface for IOMMU management
	iommuFd, err := unix.Open("/dev/iommu", unix.O_RDWR, 0)
	if err != nil {
		return IOMMUCapabilities{}, fmt.Errorf("SMMUv3 requires --device /dev/iommu (iommufd): %v", err)
	}
	defer unix.Close(iommuFd)

	// Open the VFIO device file descriptor
	devFd, err := unix.Open(cdevPath, unix.O_RDWR, 0)
	if err != nil {
		return IOMMUCapabilities{}, fmt.Errorf("open device failed: %v", err)
	}
	defer unix.Close(devFd)

	// Prepare the bind request structure
	var args vfioDeviceBindIommufd
	args.Argsz = uint32(unsafe.Sizeof(args))
	args.Iommufd = int32(iommuFd)

	// Issue the VFIO_DEVICE_BIND_IOMMUFD ioctl to bind the device to IOMMUFD
	// This returns a device ID (OutDevid) that we'll use to query capabilities
	_, _, e := unix.Syscall(unix.SYS_IOCTL, uintptr(devFd), uintptr(VFIO_DEVICE_BIND_IOMMUFD), uintptr(unsafe.Pointer(&args)))
	if e != 0 {
		return IOMMUCapabilities{}, fmt.Errorf("iommufd bind failed: %v", e)
	}

	// Query the IOMMU capabilities using the device ID returned from binding
	return getIOMMUCapabilities(iommuFd, args.OutDevid)
}

// getIOMMUCapabilities queries the IOMMU hardware capabilities for a bound device.
// It first attempts to get ARM SMMUv3-specific information, then falls back to
// generic IOMMU information if SMMUv3-specific query fails.
func getIOMMUCapabilities(iommuFd int, devID uint32) (IOMMUCapabilities, error) {
	// Prepare the hardware info request structure
	var info iommuHwInfo
	info.Size = uint32(unsafe.Sizeof(info))
	info.DevID = devID
	info.Flags = IOMMU_HW_INFO_FLAG_INPUT_TYPE
	info.InOutDataType = IOMMU_HW_INFO_TYPE_ARM_SMMUV3

	// Allocate buffer for ARM SMMUv3-specific hardware information
	var armInfo iommuHwInfoArmSmmuv3
	info.DataLen = uint32(unsafe.Sizeof(armInfo))
	info.DataUptr = uint64(uintptr(unsafe.Pointer(&armInfo)))

	// Attempt to query ARM SMMUv3-specific capabilities
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(iommuFd), uintptr(IOMMU_GET_HW_INFO), uintptr(unsafe.Pointer(&info)))
	if errno != 0 {
		// ARM SMMUv3 query failed, fall back to generic IOMMU information
		// This might happen on systems with different IOMMU implementations
		info.InOutDataType = IOMMU_HW_INFO_TYPE_DEFAULT
		info.DataLen = 0
		_, _, errno = unix.Syscall(unix.SYS_IOCTL, uintptr(iommuFd), uintptr(IOMMU_GET_HW_INFO), uintptr(unsafe.Pointer(&info)))
		if errno != 0 {
			// Even generic query failed - return default capabilities with error
			return IOMMUCapabilities{OASBits: 48}, fmt.Errorf("IOMMU_GET_HW_INFO failed: %v", errno)
		}
	}

	// Parse the common capabilities from the ioctl response
	caps := IOMMUCapabilities{
		SSIDSize:       int(info.OutMaxPasidLog2),
		PASIDSupported: info.OutMaxPasidLog2 > 0,
		// ATS is supported if the "not supported" bit is NOT set
		ATSSupported: (info.OutCapabilities & IOMMU_HW_CAP_PCI_ATS_NOT_SUPPORTED) == 0,
		OASBits:      48, // Default to 48-bit output address space
	}

	// If we successfully queried ARM SMMUv3 info, extract the actual OAS from IDR5
	if info.InOutDataType == IOMMU_HW_INFO_TYPE_ARM_SMMUV3 {
		// Extract the OAS field from IDR5 register (bits 3:0)
		// The encoding is: 0=32bit, 1=36bit, 2=40bit, 3=42bit, 4=44bit, 5=48bit, 6=52bit
		oasField := armInfo.IDR[5] & 0xF
		if oasField <= 5 {
			// Calculate actual OAS: 32 + (field * 4) gives the bit width
			caps.OASBits = 32 + int(oasField*4)
			log.Log.V(3).Infof("[IOMMU] Raw IDR5.OAS field = %d -> OAS: %d bits", oasField, caps.OASBits)
		}
	}
	return caps, nil
}

// getCdevPath finds the VFIO character device path for a given PCI device.
// It searches in the sysfs vfio-dev directory for the device's VFIO node.
func getCdevPath(bdf string) string {
	// Construct path to the device's vfio-dev directory in sysfs
	vfioDevDir := fmt.Sprintf("/sys/bus/pci/devices/%s/vfio-dev", bdf)
	entries, err := os.ReadDir(vfioDevDir)
	if err != nil {
		return ""
	}
	// Look for an entry starting with "vfio" (e.g., "vfio0", "vfio1")
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "vfio") {
			// Return the corresponding character device path
			return filepath.Join("/dev/vfio/devices", entry.Name())
		}
	}
	return ""
}

// isSMMUv3Enabled checks if ARM SMMUv3 (System Memory Management Unit v3) is enabled.
// It searches /sys/class/iommu for symlinks containing "arm-smmu-v3".
func isSMMUv3Enabled() (bool, error) {
	dir := "/sys/class/iommu"
	files, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, f := range files {
		link := filepath.Join(dir, f.Name())
		target, err := os.Readlink(link)
		if err == nil && strings.Contains(target, "arm-smmu-v3") {
			return true, nil
		}
	}
	return false, nil
}

// readIDAA64MMFR0 reads the ARM64 ID_AA64MMFR0_EL1 system register using inline assembly.
// This register provides information about the memory model and address translation features.
func readIDAA64MMFR0() uint64 { return uint64(C.read_id_aa64mmfr0()) }

// CalculatePCIHole64Size computes the total size of 64-bit PCI memory regions for a device.
// It reads the /sys/bus/pci/devices/<bdf>/resource file which lists all BARs.
//
// Each line in the resource file contains:
//   - start address (hex)
//   - end address (hex)
//   - flags (hex)
//
// This function only counts 64-bit prefetchable memory regions:
//   - flags & 0x200 must be set (64-bit region)
//   - flags & 0xf must equal 0xc (prefetchable memory)
//
// Returns the total size in bytes.
func CalculatePCIHole64Size(bdf string) (uint64, error) {
	var totalSize uint64
	resourcePath := fmt.Sprintf("/sys/bus/pci/devices/%s/resource", bdf)
	file, err := os.Open(resourcePath)
	if err != nil {
		return 0, fmt.Errorf("open resource failed: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 {
			continue
		}
		// Remove "0x" prefix for parsing
		fields[0] = strings.TrimPrefix(fields[0], "0x")
		fields[1] = strings.TrimPrefix(fields[1], "0x")
		fields[2] = strings.TrimPrefix(fields[2], "0x")

		start, _ := strconv.ParseUint(fields[0], 16, 64)
		end, _ := strconv.ParseUint(fields[1], 16, 64)
		flags, _ := strconv.ParseUint(fields[2], 16, 64)

		// Skip invalid or empty regions
		if start == 0 || end == 0 || start > end {
			continue
		}

		// Only include 64-bit prefetchable memory regions
		// 0x200: 64-bit region flag
		// 0xc: prefetchable memory type
		if (flags&0x00000200) != 0x00000200 || (flags&0xf) != 0xc {
			continue
		}

		totalSize += end - start + 1
	}

	minSize := uint64(549755813888) // 512 GiB in bytes
	if totalSize < minSize {
		totalSize = minSize
	}

	return totalSize, nil
}

// CalculateTotalPCIHole64Size computes the total PCI hole size with margin and rounds up to
// the next power of 2. This ensures proper alignment for memory mapping.
//
// Parameters:
//   - bdfComputedSize: total size in bytes from all devices
//   - marginKiB: safety margin in KiB to add
//
// Returns the final hole size in KiB, rounded up to the nearest power of 2.
func CalculateTotalPCIHole64Size(bdfComputedSize uint64, marginKiB uint64) uint64 {
	// Convert bytes to KiB and add margin
	totalSizeKiB := bdfComputedSize / 1024
	totalSizeKiB += marginKiB
	mod := bdfComputedSize % 1024

	if totalSizeKiB == 0 && mod == 0 {
		return 0
	}

	// Round up to next power of 2 for proper memory alignment
	holeSize := uint64(1)
	for holeSize < totalSizeKiB {
		holeSize <<= 1
	}

	// Round up if there were a missing margin
	if holeSize == totalSizeKiB && mod > 0 {
		holeSize <<= 1
	}

	return holeSize
}

// InferExtraNUMANodes determines the NUMA node configuration needed for proper IOMMU setup.
// For devices with SMMUv3 and PASID support, additional virtual NUMA nodes may be required
// to satisfy memory topology constraints.
//
// The function handles two scenarios:
//  1. Main node has CPUs: Return all memory-less nodes as extra nodes
//  2. Main node has no CPUs: Find consecutive memory-less nodes starting from mainNode+1
//
// This is important because IOMMU page tables need to be allocated on specific NUMA nodes
// to ensure proper memory affinity and performance.
//
// Returns:
//   - extraNodes: list of additional NUMA node IDs needed
//   - mainNode: the primary NUMA node for the device
//   - err: any error encountered reading NUMA information
func InferExtraNUMANodes(bdf string) (extraNodes []int, mainNode int, err error) {
	// Read the device's primary NUMA node from sysfs
	numaPath := fmt.Sprintf("/sys/bus/pci/devices/%s/numa_node", bdf)
	numaData, err := os.ReadFile(numaPath)
	if err != nil {
		return nil, -1, err
	}
	mainNode, err = strconv.Atoi(strings.TrimSpace(string(numaData)))
	if err != nil {
		return nil, -1, err
	}
	if mainNode < 0 {
		return nil, mainNode, fmt.Errorf("invalid main NUMA node: %d", mainNode)
	}

	// Get system NUMA topology information
	onlineNodes, _ := getOnlineNodes()
	allMemoryLess, _ := getMemoryLessNoCPUNodes(onlineNodes)
	hasCPUs, _ := nodeHasCPUs(mainNode)

	if hasCPUs {
		// Scenario 1: Main node has CPUs - use all memory-less nodes as extras
		sort.Ints(allMemoryLess)
		return allMemoryLess, mainNode, nil
	} else {
		// Scenario 2: Main node has no CPUs - find consecutive memory-less nodes
		extraNodes = []int{}
		candidate := mainNode + 1
		maxExtra := 16

		for len(extraNodes) < maxExtra {
			// Stop if we've reached a non-existent node
			if !slices.Contains(onlineNodes, candidate) {
				break
			}
			// Stop if we've reached a node with memory or CPUs
			isMemoryLess, _ := isMemoryLessNoCPU(candidate)
			if !isMemoryLess {
				break
			}
			extraNodes = append(extraNodes, candidate)
			candidate++
		}
		return extraNodes, mainNode, nil
	}
}

// getOnlineNodes parses /sys/devices/system/node/online to get a list of online NUMA nodes.
// The file format can be:
//   - Single nodes: "0,2,4"
//   - Ranges: "0-3,8-11"
//   - Mixed: "0-3,5,8-11"
//
// Returns a sorted list of all online node IDs.
func getOnlineNodes() ([]int, error) {
	data, err := os.ReadFile("/sys/devices/system/node/online")
	if err != nil {
		return nil, err
	}
	var nodes []int
	// Parse comma-separated list of nodes and ranges
	for _, part := range strings.Split(strings.TrimSpace(string(data)), ",") {
		if strings.Contains(part, "-") {
			// Handle range format (e.g., "0-3")
			rangeParts := strings.Split(part, "-")
			start, _ := strconv.Atoi(rangeParts[0])
			end, _ := strconv.Atoi(rangeParts[1])
			for i := start; i <= end; i++ {
				nodes = append(nodes, i)
			}
		} else {
			// Handle single node (e.g., "5")
			node, _ := strconv.Atoi(part)
			nodes = append(nodes, node)
		}
	}
	sort.Ints(nodes)
	return nodes, nil
}

// getMemoryLessNoCPUNodes filters the given list of nodes to return only those that have
// no memory and no CPUs.
func getMemoryLessNoCPUNodes(nodes []int) ([]int, error) {
	var memoryLess []int
	for _, node := range nodes {
		is, _ := isMemoryLessNoCPU(node)
		if is {
			memoryLess = append(memoryLess, node)
		}
	}
	return memoryLess, nil
}

// isMemoryLessNoCPU checks if a NUMA node has zero memory and no CPUs assigned.
// Such nodes are typically used for I/O devices or as placeholder nodes in the topology.
//
// Returns true if the node has MemTotal=0 and an empty/none cpulist.
func isMemoryLessNoCPU(node int) (bool, error) {
	// Check if node has zero memory
	memData, err := os.ReadFile(fmt.Sprintf("/sys/devices/system/node/node%d/meminfo", node))
	if err != nil {
		return false, err
	}
	memTotalRe := regexp.MustCompile(`MemTotal:\s+(\d+)\s+kB`)
	match := memTotalRe.FindStringSubmatch(string(memData))
	if len(match) != 2 || match[1] != "0" {
		return false, nil
	}

	// Check if node has no CPUs
	cpuData, err := os.ReadFile(fmt.Sprintf("/sys/devices/system/node/node%d/cpulist", node))
	if err != nil {
		return false, err
	}
	cpuStr := strings.TrimSpace(string(cpuData))
	if cpuStr != "" && cpuStr != "none" {
		return false, nil
	}

	return true, nil
}

// nodeHasCPUs checks if a NUMA node has any CPUs assigned to it.
// Returns true if the cpulist is non-empty and not "none".
func nodeHasCPUs(node int) (bool, error) {
	cpuData, err := os.ReadFile(fmt.Sprintf("/sys/devices/system/node/node%d/cpulist", node))
	if err != nil {
		return false, err
	}
	cpuStr := strings.TrimSpace(string(cpuData))
	return cpuStr != "" && cpuStr != "none", nil
}
