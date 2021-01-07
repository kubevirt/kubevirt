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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package hardware

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/util"
)

const (
	topoBaseDir           = "/sys/devices/system/cpu"
	threadSiblingsListFmt = "cpu%d/topology/thread_siblings_list" // deprecated
	coreCPUsListFmt       = "cpu%d/topology/core_cpus_list"

	PCI_ADDRESS_PATTERN = `^([\da-fA-F]{4}):([\da-fA-F]{2}):([\da-fA-F]{2})\.([0-7]{1})$`
)

type CPUSet map[int]bool

func NewCPUSet(cpus []int) CPUSet {
	cpuSet := CPUSet{}
	for _, cpu := range cpus {
		cpuSet.Add(cpu)
	}
	return cpuSet
}

func (s CPUSet) Has(cpu int) bool {
	return s[cpu]
}

func (s CPUSet) Add(cpu int) {
	s[cpu] = true
}

func (s CPUSet) Remove(cpu int) {
	delete(s, cpu)
}

func (s CPUSet) Empty() bool {
	return len(s) == 0
}

type CPUTopology interface {
	GetCPUs() ([]int, error)
	GetCPUSiblings(cpu int) ([]int, error)
}

// Parse linux cpuset into an array of ints
// See: http://man7.org/linux/man-pages/man7/cpuset.7.html#FORMATS
func ParseCPUSetLine(cpusetLine string) (cpusList []int, err error) {
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
				cpusList = append(cpusList, cpuNum)
			}
		} else {
			cpuNum, err := strconv.Atoi(cpuRange[0])
			if err != nil {
				return nil, err
			}
			cpusList = append(cpusList, cpuNum)
		}
	}
	return
}

func ParseCPUSetFile(file string) ([]int, error) {
	line, err := util.ScanLine(file)
	if err != nil {
		return nil, err
	}
	cpusList, err := ParseCPUSetLine(line)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cpuset file: %v", err)
	}
	return cpusList, nil
}

func GroupCPUThreads(topo CPUTopology) ([]int, error) {
	cpus, err := topo.GetCPUs()
	if err != nil {
		return nil, err
	}

	cpuSet := NewCPUSet(cpus)
	res := make([]int, 0, len(cpus))

	appendAndFinish := func(cpu int) bool {
		res = append(res, cpu)
		cpuSet.Remove(cpu)
		return cpuSet.Empty()
	}

	for _, cpu := range cpus {
		if !cpuSet.Has(cpu) {
			continue
		}

		if appendAndFinish(cpu) {
			return res, nil
		}

		siblings, err := topo.GetCPUSiblings(cpu)
		if err != nil {
			log.DefaultLogger().Reason(err).Error("Error while reading CPU topology")
			return cpus, nil
		}
		for _, sibling := range siblings {
			if !cpuSet.Has(sibling) {
				continue
			}

			if appendAndFinish(sibling) {
				return res, nil
			}
		}
	}
	return res, nil
}

//GetNumberOfVCPUs returns number of vCPUs
//It counts sockets*cores*threads
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

type sysCPUTopo struct {
	cpusetPath  string
	topoBaseDir string
}

func NewPodCPUTopo(cpusetPath string) CPUTopology {
	return NewPodCPUTopoWithBaseDir(cpusetPath, topoBaseDir)
}

func NewPodCPUTopoWithBaseDir(cpusetPath, topoBaseDir string) CPUTopology {
	return &sysCPUTopo{cpusetPath, topoBaseDir}
}

func (t *sysCPUTopo) GetCPUs() ([]int, error) {
	return ParseCPUSetFile(t.cpusetPath)
}

func (t *sysCPUTopo) GetCPUSiblings(cpu int) ([]int, error) {
	file := filepath.Join(t.topoBaseDir, fmt.Sprintf(coreCPUsListFmt, cpu))
	if _, err := os.Stat(file); os.IsNotExist(err) {
		file = filepath.Join(t.topoBaseDir, fmt.Sprintf(threadSiblingsListFmt, cpu))
	}
	return ParseCPUSetFile(file)
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
