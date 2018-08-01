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

package virtwrap

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const cpuset_path = "/sys/fs/cgroup/cpuset/cpuset.cpus"

func getPodPinnedCpus() ([]int, error) {
	var cpuset string
	file, err := os.Open(cpuset_path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		cpuset = scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	cpusList, err := parseCPUSetLine(cpuset)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cpuset file: %v", err)
	}
	return cpusList, nil
}

// Parse linux cpuset into an array of ints
// See: http://man7.org/linux/man-pages/man7/cpuset.7.html#FORMATS
func parseCPUSetLine(cpusetLine string) (cpusList []int, err error) {
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

func formatDomainCPUTune(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	availableCpus, err := getPodPinnedCpus()
	if err != nil || len(availableCpus) == 0 {
		return fmt.Errorf("failed for get pods pinned cpus: %v", err)
	}

	cpuTune := api.CPUTune{}
	for idx := 0; idx < int(vmi.Spec.Domain.CPU.Cores); idx++ {
		vcpupin := api.CPUTuneVCPUPin{}
		vcpupin.VCPU = uint(idx)
		vcpupin.CPUSet = strconv.Itoa(availableCpus[idx])
		cpuTune.VCPUPin = append(cpuTune.VCPUPin, vcpupin)
	}
	domain.Spec.CPUTune = &cpuTune
	return nil
}
