//go:build linux

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

package admission

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// defaultCgroupV2Root is the unified cgroup v2 mount used by modern Kubernetes nodes.
	// Cgroup v1 hierarchies (/sys/fs/cgroup/cpu, /sys/fs/cgroup/memory, etc.) are not
	// supported; reads fail there, Utilization() returns ok=false, and soft utilization
	// admission intentionally fails open (transfer-count limits still apply).
	defaultCgroupV2Root  = "/sys/fs/cgroup"
	minCPUSampleInterval = 250 * time.Millisecond
)

func newPlatformUtilizationReader() UtilizationReader {
	return &cgroupUtilizationReader{}
}

type cgroupUtilizationReader struct {
	mu                  sync.Mutex
	lastCPUSampleTime   time.Time
	lastCPUUsageUsec    uint64
	lastCPUPercent      float64
	lastCPUPercentValid bool
}

func (r *cgroupUtilizationReader) Utilization() (cpuPercent, memoryPercent float64, ok bool) {
	cpu, cpuOK := r.cpuUtilization()
	mem, memOK := r.memoryUtilization()
	if !cpuOK && !memOK {
		return 0, 0, false
	}
	return cpu, mem, true
}

func (r *cgroupUtilizationReader) cpuUtilization() (float64, bool) {
	usage, err := readCPUUsageUsec()
	if err != nil {
		return 0, false
	}
	quota, period, unlimited, err := readCPUMax()
	if err != nil || unlimited || quota == 0 || period == 0 {
		return 0, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if r.lastCPUSampleTime.IsZero() {
		r.lastCPUSampleTime = now
		r.lastCPUUsageUsec = usage
		r.lastCPUPercent = 0
		r.lastCPUPercentValid = true
		return 0, true
	}

	elapsed := now.Sub(r.lastCPUSampleTime)
	if elapsed < minCPUSampleInterval {
		if r.lastCPUPercentValid {
			return r.lastCPUPercent, true
		}
		return 0, true
	}

	if usage < r.lastCPUUsageUsec {
		r.lastCPUSampleTime = now
		r.lastCPUUsageUsec = usage
		r.lastCPUPercent = 0
		r.lastCPUPercentValid = true
		return 0, true
	}

	usageDelta := usage - r.lastCPUUsageUsec
	elapsedUsec := elapsed.Seconds() * 1_000_000
	percent := float64(usageDelta) * float64(period) / (float64(quota) * elapsedUsec) * 100

	r.lastCPUSampleTime = now
	r.lastCPUUsageUsec = usage
	r.lastCPUPercent = percent
	r.lastCPUPercentValid = true

	return percent, true
}

func (r *cgroupUtilizationReader) memoryUtilization() (float64, bool) {
	current, err := readCgroupUint("memory.current")
	if err != nil {
		return 0, false
	}
	limit, unlimited, err := readMemoryMax()
	if err != nil || unlimited || limit == 0 {
		return 0, false
	}
	return float64(current) / float64(limit) * 100, true
}

func readCPUUsageUsec() (uint64, error) {
	file, err := os.Open(filepath.Join(defaultCgroupV2Root, "cpu.stat"))
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "usage_usec ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return 0, fmt.Errorf("unexpected cpu.stat line %q", line)
		}
		return strconv.ParseUint(fields[1], 10, 64)
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, fmt.Errorf("usage_usec not found in cpu.stat")
}

func readCPUMax() (quota, period uint64, unlimited bool, err error) {
	raw, err := readCgroupFile("cpu.max")
	if err != nil {
		return 0, 0, false, err
	}
	fields := strings.Fields(raw)
	if len(fields) != 2 {
		return 0, 0, false, fmt.Errorf("unexpected cpu.max value %q", raw)
	}
	if fields[0] == "max" {
		return 0, 0, true, nil
	}
	quota, err = strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return 0, 0, false, err
	}
	period, err = strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0, 0, false, err
	}
	return quota, period, false, nil
}

func readMemoryMax() (limit uint64, unlimited bool, err error) {
	raw, err := readCgroupFile("memory.max")
	if err != nil {
		return 0, false, err
	}
	if raw == "max" {
		return 0, true, nil
	}
	limit, err = strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return limit, false, nil
}

func readCgroupFile(name string) (string, error) {
	data, err := os.ReadFile(filepath.Join(defaultCgroupV2Root, name))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func readCgroupUint(name string) (uint64, error) {
	raw, err := readCgroupFile(name)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(raw, 10, 64)
}
