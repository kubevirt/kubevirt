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
	"strconv"
	"strings"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
)

const CPUSET_PATH = "/sys/fs/cgroup/cpuset/cpuset.cpus"

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
