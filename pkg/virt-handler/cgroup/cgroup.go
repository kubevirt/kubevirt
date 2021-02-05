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
 * Copyright 2021
 *
 */

package cgroup

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/opencontainers/runc/libcontainer/cgroups"
)

const (
	procMountPoint   = "/proc"
	cgroupMountPoint = "/sys/fs/cgroup"
)

func ControllerPath(controller string) string {
	return controllerPath(cgroups.IsCgroup2UnifiedMode(), cgroupMountPoint, controller)
}

func controllerPath(isCgroup2UnifiedMode bool, cgroupMount, controller string) string {
	if isCgroup2UnifiedMode {
		return cgroupMount
	}
	return filepath.Join(cgroupMount, controller)
}

func CPUSetPath() string {
	return cpuSetPath(cgroups.IsCgroup2UnifiedMode(), cgroupMountPoint)
}

func cpuSetPath(isCgroup2UnifiedMode bool, cgroupMount string) string {
	if isCgroup2UnifiedMode {
		return filepath.Join(cgroupMount, "cpuset.cpus.effective")
	}
	return filepath.Join(cgroupMount, "cpuset", "cpuset.cpus")
}

type Parser interface {
	// Parse retrieves the cgroup data for the given process id and returns a
	// map of controllers to slice paths.
	Parse(pid int) (map[string]string, error)
}

type v1Parser struct {
	procMount string
}

func (v1 *v1Parser) Parse(pid int) (map[string]string, error) {
	return cgroups.ParseCgroupFile(filepath.Join(v1.procMount, strconv.Itoa(pid), "cgroup"))
}

type v2Parser struct {
	procMount   string
	cgroupMount string
}

func (v2 *v2Parser) Parse(pid int) (map[string]string, error) {
	slices, err := cgroups.ParseCgroupFile(filepath.Join(v2.procMount, strconv.Itoa(pid), "cgroup"))
	if err != nil {
		return nil, err
	}

	slice, ok := slices[""]
	if !ok {
		return nil, fmt.Errorf("Slice not found for PID %d", pid)
	}

	availableControllers, err := v2.getAvailableControllers(slice)
	if err != nil {
		return nil, err
	}

	// For cgroup v2 there are no per-controller paths.
	slices = make(map[string]string)
	for _, c := range availableControllers {
		slices[c] = slice
	}

	return slices, nil
}

// getAvailableControllers returns all controllers available for the cgroup.
// Based on GetAllSubsystems from
//  https://github.com/opencontainers/runc/blob/ff819c7e9184c13b7c2607fe6c30ae19403a7aff/libcontainer/cgroups/utils.go#L80
func (v2 *v2Parser) getAvailableControllers(slice string) ([]string, error) {
	// "pseudo" controllers do not appear in /sys/fs/cgroup/.../cgroup.controllers.
	// - devices: implemented in kernel 4.15
	// - freezer: implemented in kernel 5.2
	// We assume these are always available, as it is hard to detect availability.
	pseudo := []string{"devices", "freezer"}
	data, err := ioutil.ReadFile(filepath.Join(v2.cgroupMount, slice, "cgroup.controllers"))
	if err != nil {
		return nil, err
	}
	subsystems := append(pseudo, strings.Fields(string(data))...)
	return subsystems, nil
}

func NewParser() Parser {
	return newParser(cgroups.IsCgroup2UnifiedMode(), procMountPoint, cgroupMountPoint)
}

func newParser(isCgroup2UnifiedMode bool, procMount, cgroupMount string) Parser {
	if isCgroup2UnifiedMode {
		return &v2Parser{
			procMount:   procMount,
			cgroupMount: cgroupMount,
		}
	}
	return &v1Parser{
		procMount: procMount,
	}
}
