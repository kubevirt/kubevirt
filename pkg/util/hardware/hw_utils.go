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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	PCI_ADDRESS_PATTERN = `^([\da-fA-F]{4}):([\da-fA-F]{2}):([\da-fA-F]{2})\.([0-7]{1})$`
)

// Parse linux cpuset into an array of ints
// See: http://man7.org/linux/man-pages/man7/cpuset.7.html#FORMATS
func ParseCPUSetLine(cpusetLine string, limit int) (cpusList []int, err error) {
	elements := strings.Split(cpusetLine, ",")
	for _, item := range elements {
		cpuRange := strings.Split(item, "-")
		// provided a range: 1-3
		if len(cpuRange) > 1 {
			start, err := strconv.Atoi(cpuRange[0])
			if err != nil {
				return nil, err
			}
			end, err := strconv.Atoi(cpuRange[1])
			if err != nil {
				return nil, err
			}
			// Add cpus to the list. Assuming it's a valid range.
			for cpuNum := start; cpuNum <= end; cpuNum++ {
				if cpusList, err = safeAppend(cpusList, cpuNum, limit); err != nil {
					return nil, err
				}
			}
		} else {
			cpuNum, err := strconv.Atoi(cpuRange[0])
			if err != nil {
				return nil, err
			}
			if cpusList, err = safeAppend(cpusList, cpuNum, limit); err != nil {
				return nil, err
			}
		}
	}
	return
}

func safeAppend(cpusList []int, cpu int, limit int) ([]int, error) {
	if len(cpusList) > limit {
		return nil, fmt.Errorf("rejecting expanding CPU array for safety reasons, limit is %v", limit)
	}
	return append(cpusList, cpu), nil
}

// GetNumberOfVCPUs returns number of vCPUs
// It counts sockets*cores*threads
func GetNumberOfVCPUs(cpuSpec *v1.CPU) int64 {
	vCPUs := cpuSpec.Cores
	if cpuSpec.Sockets != 0 {
		if vCPUs == 0 {
			vCPUs = cpuSpec.Sockets
		} else {
			vCPUs *= cpuSpec.Sockets
		}
	}
	if cpuSpec.Threads != 0 {
		if vCPUs == 0 {
			vCPUs = cpuSpec.Threads
		} else {
			vCPUs *= cpuSpec.Threads
		}
	}
	return int64(vCPUs)
}

// ParsePciAddress returns an array of PCI DBSF fields (domain, bus, slot, function)
func ParsePciAddress(pciAddress string) ([]string, error) {
	pciAddrRegx, err := regexp.Compile(PCI_ADDRESS_PATTERN)
	if err != nil {
		return nil, fmt.Errorf("failed to compile pci address pattern, %v", err)
	}
	res := pciAddrRegx.FindStringSubmatch(pciAddress)
	if len(res) == 0 {
		return nil, fmt.Errorf("failed to parse pci address %s", pciAddress)
	}
	return res[1:], nil
}

var (
	PciBasePath  = "/sys/bus/pci/devices"
	NodeBasePath = "/sys/bus/node/devices"
)

func GetDeviceNumaNode(pciAddress string) (*uint32, error) {
	numaNodePath := filepath.Join(PciBasePath, pciAddress, "numa_node")
	// #nosec No risk for path injection. Reading static path of NUMA node info
	numaNodeStr, err := os.ReadFile(numaNodePath)
	if err != nil {
		return nil, err
	}
	numaNodeStr = bytes.TrimSpace(numaNodeStr)
	numaNodeInt, err := strconv.Atoi(string(numaNodeStr))
	if err != nil {
		return nil, err
	}
	numaNode := uint32(numaNodeInt)
	return &numaNode, nil
}

func GetDeviceAlignedCPUs(pciAddress string) ([]int, error) {
	numaNode, err := GetDeviceNumaNode(pciAddress)
	if err != nil {
		return nil, err
	}
	cpuList, err := GetNumaNodeCPUList(int(*numaNode))
	if err != nil {
		return nil, err
	}
	return cpuList, err
}

func GetNumaNodeCPUList(numaNode int) ([]int, error) {
	filePath := filepath.Join(NodeBasePath, fmt.Sprintf("node%d", numaNode), "cpulist")
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	content = bytes.TrimSpace(content)
	cpusList, err := ParseCPUSetLine(string(content[:]), 50000)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cpulist file: %v", err)
	}

	return cpusList, nil
}

func LookupDeviceVCPUAffinity(pciAddress string, domainSpec *api.DomainSpec) ([]uint32, error) {
	alignedVCPUList := []uint32{}
	p2vCPUMap := make(map[string]uint32)
	alignedPhysicalCPUs, err := GetDeviceAlignedCPUs(pciAddress)
	if err != nil {
		return nil, err
	}

	cpuTune := domainSpec.CPUTune.VCPUPin
	for _, vcpuPin := range cpuTune {
		p2vCPUMap[vcpuPin.CPUSet] = vcpuPin.VCPU
	}

	for _, pcpu := range alignedPhysicalCPUs {
		if vCPU, exist := p2vCPUMap[strconv.Itoa(int(pcpu))]; exist {
			alignedVCPUList = append(alignedVCPUList, uint32(vCPU))
		}
	}
	return alignedVCPUList, nil
}

func PCIAddressToString(pciBusID *api.Address) string {
	if pciBusID == nil {
		return ""
	}
	prefix := "0x"
	return fmt.Sprintf("%s:%s:%s.%s",
		strings.TrimPrefix(pciBusID.Domain, prefix),
		strings.TrimPrefix(pciBusID.Bus, prefix),
		strings.TrimPrefix(pciBusID.Slot, prefix),
		strings.TrimPrefix(pciBusID.Function, prefix))
}

// LookupDeviceVCPUNumaNode looks up the NUMA node of a device based on its PCI address
// and the domain specification of the virtual machine.
//
// It returns a pointer to the NUMA node ID if found, or nil if not found.
func LookupDeviceVCPUNumaNode(pciAddress *api.Address, domainSpec *api.DomainSpec) (numaNode *uint32) {
	if pciAddress == nil || domainSpec == nil ||
		domainSpec.CPU.NUMA == nil || domainSpec.CPUTune == nil {
		return
	}

	// vCPUS by device PCI address
	vCPUList, err := LookupDeviceVCPUAffinity(
		PCIAddressToString(pciAddress),
		domainSpec,
	)
	if err != nil || len(vCPUList) == 0 {
		return
	}

	// guest OS numa node by vCPU
	for i, cell := range domainSpec.CPU.NUMA.Cells {
		vcpusInCell, err := ParseCPUSetLine(cell.CPUs, 5000)
		if err != nil {
			continue
		}

		for _, vcpu := range vcpusInCell {
			if vcpu == int(vCPUList[0]) {
				id, err := strconv.Atoi(domainSpec.CPU.NUMA.Cells[i].ID)
				if err == nil {
					cellID := uint32(id)
					numaNode = &cellID
				}
			}
		}
	}
	return
}
