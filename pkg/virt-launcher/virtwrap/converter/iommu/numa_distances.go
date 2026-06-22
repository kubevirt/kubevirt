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
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	maxGINodesPerGPU = 16
	giNodesPerGPU    = 8
)

// Overridden in unit tests to point at a mock sysfs tree.
var (
	SysfsNodeBasePath = "/sys/devices/system/node"
	SysfsPCIBasePath  = "/sys/bus/pci/devices"
)

func applyNUMADistances(domain *api.DomainSpec) {
	if domain.CPU.NUMA == nil || len(domain.CPU.NUMA.Cells) == 0 {
		return
	}

	guestToHost, err := buildGuestToHostMapping(domain)
	if err != nil {
		log.Log.Reason(err).Warning("Failed to build guest-to-host NUMA mapping, skipping distance computation")
		return
	}
	if len(guestToHost) == 0 {
		return
	}

	hostNodeIDs := uniqueHostNodes(guestToHost)
	hostDistances, err := readHostNUMADistances(hostNodeIDs)
	if err != nil {
		log.Log.Reason(err).Warning("Failed to read host NUMA distances from sysfs, skipping distance computation")
		return
	}

	guestCellIDs := make([]int, 0, len(domain.CPU.NUMA.Cells))
	for _, cell := range domain.CPU.NUMA.Cells {
		id, err := strconv.Atoi(cell.ID)
		if err != nil {
			continue
		}
		guestCellIDs = append(guestCellIDs, id)
	}
	sort.Ints(guestCellIDs)

	for i := range domain.CPU.NUMA.Cells {
		srcID, err := strconv.Atoi(domain.CPU.NUMA.Cells[i].ID)
		if err != nil {
			continue
		}
		srcHostNode, ok := guestToHost[srcID]
		if !ok {
			continue
		}

		siblings := make([]api.NUMACellSibling, 0, len(guestCellIDs))
		for _, dstID := range guestCellIDs {
			dstHostNode, ok := guestToHost[dstID]
			if !ok {
				continue
			}

			distance := lookupHostDistance(hostDistances, srcHostNode, dstHostNode)
			if srcID != dstID && distance == 10 {
				distance = 11
			}
			siblings = append(siblings, api.NUMACellSibling{
				ID:    strconv.Itoa(dstID),
				Value: distance,
			})
		}

		if len(siblings) > 0 {
			domain.CPU.NUMA.Cells[i].Distances = &api.NUMACellDistances{
				Siblings: siblings,
			}
		}
	}
}

// buildGuestToHostMapping builds a mapping from guest NUMA cell IDs to host
// NUMA node IDs. CPU cells are mapped via NUMATune MemNodes. GI cells are
// mapped by discovering all memory-less/CPU-less host nodes and distributing
// them among GPUs in domain spec order (matching handleGraceVirtualizationNumaNodes ordering).
//
// On a real GB200, GPU PCI devices report the CPU socket NUMA node (e.g., 0 or
// 1), not a GI node. The GI nodes (memory-less, CPU-less) start after the CPU
// nodes (e.g., nodes 2-33 on a 4-GPU system). This function maps each GPU's
// first guest GI cell to the GPU's CPU socket node (for CPU<->GPU distances),
// and the remaining guest GI cells to the corresponding host GI nodes.
func buildGuestToHostMapping(domain *api.DomainSpec) (map[int]int, error) {
	mapping := make(map[int]int)

	if domain.NUMATune != nil {
		for _, memNode := range domain.NUMATune.MemNodes {
			hostNode, err := strconv.Atoi(memNode.NodeSet)
			if err != nil {
				continue
			}
			mapping[int(memNode.CellID)] = hostNode
		}
	}

	type gpuDevice struct {
		bdf        string
		guestStart int
		guestEnd   int
	}

	var gpus []gpuDevice
	for _, hostDev := range domain.Devices.HostDevices {
		if hostDev.ACPI == nil || hostDev.ACPI.NodeSet == "" || hostDev.ACPI.NodeSet == "tofill" {
			continue
		}
		if hostDev.Source.Address == nil {
			continue
		}

		guestStart, guestEnd, err := parseNodeSetRange(hostDev.ACPI.NodeSet)
		if err != nil {
			continue
		}

		bdf := hardware.PCIAddressToString(hostDev.Source.Address)
		gpus = append(gpus, gpuDevice{bdf: bdf, guestStart: guestStart, guestEnd: guestEnd})
	}

	if len(gpus) == 0 {
		return mapping, nil
	}

	allGINodes, err := discoverAllGINodes()
	if err != nil {
		log.Log.Reason(err).Warning("Failed to discover host GI nodes, skipping GI distance mapping")
		return mapping, nil
	}

	giChunkSize := giNodesPerGPU
	giOffset := 0

	for _, gpu := range gpus {
		primaryNode, err := readDeviceNUMANode(gpu.bdf)
		if err != nil {
			log.Log.Reason(err).Warningf("Failed to read NUMA node for device %s, skipping distance mapping", gpu.bdf)
			continue
		}

		mapping[gpu.guestStart] = primaryNode

		giChunk := giNodesForGPU(allGINodes, giOffset, giChunkSize)
		giOffset += giChunkSize

		for j, hostNode := range giChunk {
			guestCell := gpu.guestStart + 1 + j
			if guestCell > gpu.guestEnd {
				break
			}
			mapping[guestCell] = hostNode
		}
	}

	return mapping, nil
}

// readHostNUMADistances reads /sys/devices/system/node/nodeN/distance for each
// provided host node ID. Returns a nested map: hostDistances[src][dst] = value.
func readHostNUMADistances(hostNodeIDs []int) (map[int]map[int]uint64, error) {
	distances := make(map[int]map[int]uint64)

	onlineNodes, err := getOnlineNodeIDs()
	if err != nil {
		return nil, fmt.Errorf("failed to read online NUMA nodes: %w", err)
	}

	for _, nodeID := range hostNodeIDs {
		distPath := filepath.Join(SysfsNodeBasePath, fmt.Sprintf("node%d", nodeID), "distance")
		data, err := os.ReadFile(distPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read distance for node %d: %w", nodeID, err)
		}

		values := strings.Fields(strings.TrimSpace(string(data)))
		nodeDistances := make(map[int]uint64)
		for i, valStr := range values {
			if i >= len(onlineNodes) {
				break
			}
			val, err := strconv.ParseUint(valStr, 10, 64)
			if err != nil {
				continue
			}
			nodeDistances[onlineNodes[i]] = val
		}
		distances[nodeID] = nodeDistances
	}

	return distances, nil
}

// discoverAllGINodes returns all memory-less, CPU-less NUMA nodes on the host,
// sorted by node ID. These are the GI (Generic Initiator) / MIG nodes.
func discoverAllGINodes() ([]int, error) {
	onlineNodes, err := getOnlineNodeIDs()
	if err != nil {
		return nil, fmt.Errorf("failed to read online NUMA nodes: %w", err)
	}

	var giNodes []int
	for _, nodeID := range onlineNodes {
		if isMemoryLessNoCPUNode(nodeID) {
			giNodes = append(giNodes, nodeID)
		}
	}
	sort.Ints(giNodes)
	return giNodes, nil
}

// readDeviceNUMANode reads a PCI device's NUMA node from sysfs.
func readDeviceNUMANode(bdf string) (int, error) {
	numaPath := filepath.Join(SysfsPCIBasePath, bdf, "numa_node")
	data, err := os.ReadFile(numaPath)
	if err != nil {
		return -1, fmt.Errorf("failed to read numa_node for %s: %w", bdf, err)
	}
	node, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return -1, err
	}
	if node < 0 {
		return -1, fmt.Errorf("invalid NUMA node %d for device %s", node, bdf)
	}
	return node, nil
}

// giNodesForGPU returns up to chunkSize GI nodes starting at offset.
func giNodesForGPU(allGINodes []int, offset, chunkSize int) []int {
	if offset >= len(allGINodes) {
		return nil
	}
	end := offset + chunkSize
	if end > len(allGINodes) {
		end = len(allGINodes)
	}
	return allGINodes[offset:end]
}

func getOnlineNodeIDs() ([]int, error) {
	data, err := os.ReadFile(filepath.Join(SysfsNodeBasePath, "online"))
	if err != nil {
		return nil, err
	}
	nodes, err := hardware.ParseCPUSetLine(strings.TrimSpace(string(data)), math.MaxInt)
	if err != nil {
		return nil, err
	}
	sort.Ints(nodes)
	return nodes, nil
}

func isMemoryLessNoCPUNode(nodeID int) bool {
	memPath := filepath.Join(SysfsNodeBasePath, fmt.Sprintf("node%d", nodeID), "meminfo")
	memData, err := os.ReadFile(memPath)
	if err != nil {
		return false
	}
	if !strings.Contains(string(memData), "MemTotal:") {
		return false
	}
	for _, line := range strings.Split(string(memData), "\n") {
		if strings.Contains(line, "MemTotal:") {
			fields := strings.Fields(line)
			for i, f := range fields {
				if f == "MemTotal:" && i+1 < len(fields) {
					if fields[i+1] != "0" {
						return false
					}
				}
			}
		}
	}

	cpuPath := filepath.Join(SysfsNodeBasePath, fmt.Sprintf("node%d", nodeID), "cpulist")
	cpuData, err := os.ReadFile(cpuPath)
	if err != nil {
		return false
	}
	cpuStr := strings.TrimSpace(string(cpuData))
	return cpuStr == "" || cpuStr == "none"
}

func parseNodeSetRange(nodeSet string) (int, int, error) {
	parts := strings.SplitN(nodeSet, "-", 2)
	if len(parts) != 2 {
		n, err := strconv.Atoi(nodeSet)
		if err != nil {
			return 0, 0, err
		}
		return n, n, nil
	}
	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	end, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
}

func uniqueHostNodes(mapping map[int]int) []int {
	seen := make(map[int]bool)
	var result []int
	for _, hostNode := range mapping {
		if !seen[hostNode] {
			seen[hostNode] = true
			result = append(result, hostNode)
		}
	}
	sort.Ints(result)
	return result
}

func lookupHostDistance(distances map[int]map[int]uint64, src, dst int) uint64 {
	if srcDist, ok := distances[src]; ok {
		if val, ok := srcDist[dst]; ok {
			return val
		}
	}
	if src == dst {
		return 10
	}
	return 0
}
