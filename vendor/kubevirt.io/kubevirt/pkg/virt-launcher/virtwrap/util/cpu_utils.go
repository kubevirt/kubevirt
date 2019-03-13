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

package util

import (
	"bufio"
	"fmt"
	"os"

	"kubevirt.io/kubevirt/pkg/util/hardware"
)

func GetPodCPUSet() ([]int, error) {
	var cpuset string
	file, err := os.Open(hardware.CPUSET_PATH)
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
	cpusList, err := hardware.ParseCPUSetLine(cpuset)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cpuset file: %v", err)
	}
	return cpusList, nil
}
