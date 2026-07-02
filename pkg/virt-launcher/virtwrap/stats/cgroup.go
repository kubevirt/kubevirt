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
 */

package stats

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const defaultCgroupMemoryStatPath = "/sys/fs/cgroup/memory.stat"

type CgroupMemoryStatReader struct {
	path string
}

func NewCgroupMemoryStatReader() *CgroupMemoryStatReader {
	return &CgroupMemoryStatReader{path: defaultCgroupMemoryStatPath}
}

func NewCgroupMemoryStatReaderWithPath(path string) *CgroupMemoryStatReader {
	return &CgroupMemoryStatReader{path: path}
}

func (r *CgroupMemoryStatReader) Read() (*CgroupMemoryStats, error) {
	f, err := os.Open(r.path)
	if err != nil {
		return nil, fmt.Errorf("failed to open cgroup memory.stat: %w", err)
	}
	defer f.Close()

	result := &CgroupMemoryStats{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 {
			continue
		}
		val, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "anon":
			result.Anon = val
			result.AnonSet = true
		case "anon_thp":
			result.AnonTHP = val
			result.AnonTHPSet = true
		case "inactive_anon":
			result.InactiveAnon = val
			result.InactiveAnonSet = true
		case "active_anon":
			result.ActiveAnon = val
			result.ActiveAnonSet = true
		}
	}
	return result, scanner.Err()
}
