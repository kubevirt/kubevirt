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

package common

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	// parse CPU Mask expressions
	cpuRangeRegex  = regexp.MustCompile(`^(\d+)-(\d+)$`)
	negateCPURegex = regexp.MustCompile(`^\^(\d+)$`)
	singleCPURegex = regexp.MustCompile(`^(\d+)$`)
)

type MaskType bool

type CPUMask struct {
	Mask map[string]MaskType
}

const (
	Enabled  MaskType = true
	Disabled MaskType = false
)

func IsVCPU(comm []byte, vcpuRegex *regexp.Regexp) (string, bool) {
	if !vcpuRegex.MatchString(string(comm)) {
		return "", false
	}
	v := vcpuRegex.FindSubmatch(comm)
	return string(v[1]), true
}
func GetVCPUThreadIDs(pid int, vcpuRegex *regexp.Regexp) (map[string]string, error) {

	p := filepath.Join(string(os.PathSeparator), "proc", strconv.Itoa(pid), "task")
	d, err := os.ReadDir(p)
	if err != nil {
		return nil, err
	}
	ret := map[string]string{}
	for _, f := range d {
		if f.IsDir() {
			c, err := os.ReadFile(filepath.Join(p, f.Name(), "comm"))
			if err != nil {
				return nil, err
			}
			if v, ok := IsVCPU(c, vcpuRegex); ok {
				ret[v] = f.Name()
			}
		}
	}
	return ret, nil
}

// ParseCPUMask parses the mask and maps the results into a structure that contains which
// CPUs are enabled or disabled for the scheduling and priority changes.
// This implementation reimplements the libvirt parsing logic defined here:
// https://github.com/libvirt/libvirt/blob/56de80cb793aa7aedc45572f8b6ec3fc32c99309/src/util/virbitmap.c#L382
// except that in this case it uses a map[string]MaskType instead of a bit array.
func ParseCPUMask(mask string) (*CPUMask, error) {

	vcpus := CPUMask{}
	if len(mask) == 0 {
		return &vcpus, nil
	}
	vcpus.Mask = make(map[string]MaskType)

	masks := strings.Split(mask, ",")
	for _, i := range masks {
		m := strings.TrimSpace(i)
		switch {
		case cpuRangeRegex.MatchString(m):
			match := cpuRangeRegex.FindSubmatch([]byte(m))
			startID, err := strconv.Atoi(string(match[1]))
			if err != nil {
				return nil, err
			}
			endID, err := strconv.Atoi(string(match[2]))
			if err != nil {
				return nil, err
			}
			if startID < 0 {
				return nil, fmt.Errorf("invalid vcpu mask start index `%d`", startID)
			}
			if endID < 0 {
				return nil, fmt.Errorf("invalid vcpu mask end index `%d`", endID)
			}
			if startID > endID {
				return nil, fmt.Errorf("invalid mask range `%d-%d`", startID, endID)
			}
			for id := startID; id <= endID; id++ {
				vid := strconv.Itoa(id)
				if !vcpus.has(vid) {
					vcpus.set(vid, Enabled)
				}
			}
		case singleCPURegex.MatchString(m):
			match := singleCPURegex.FindSubmatch([]byte(m))
			vid, err := strconv.Atoi(string(match[1]))
			if err != nil {
				return nil, err
			}
			if vid < 0 {
				return nil, fmt.Errorf("invalid vcpu index `%d`", vid)
			}
			if !vcpus.has(string(match[1])) {
				vcpus.set(string(match[1]), Enabled)
			}
		case negateCPURegex.MatchString(m):
			match := negateCPURegex.FindSubmatch([]byte(m))
			vid, err := strconv.Atoi(string(match[1]))
			if err != nil {
				return nil, err
			}
			if vid < 0 {
				return nil, fmt.Errorf("invalid vcpu index `%d`", vid)
			}
			vcpus.set(string(match[1]), Disabled)
		default:
			return nil, fmt.Errorf("invalid mask value '%s' in '%s'", i, mask)
		}
	}
	return &vcpus, nil
}

func (c CPUMask) IsEnabled(vcpuID string) bool {
	if len(c.Mask) == 0 {
		return true
	}
	if t, ok := c.Mask[vcpuID]; ok {
		return t == Enabled
	}
	return false
}

func (c *CPUMask) has(vcpuID string) bool {
	_, ok := c.Mask[vcpuID]
	return ok
}

func (c *CPUMask) set(vcpuID string, mtype MaskType) {
	c.Mask[vcpuID] = mtype
}
