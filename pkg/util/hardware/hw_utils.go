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
	"regexp"
	"strconv"
	"strings"

	v1 "kubevirt.io/client-go/apis/core/v1"
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
