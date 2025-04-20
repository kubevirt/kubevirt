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
package hotplug_volume

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	rhcosPrefix = "/ostree/deploy/rhcos"
)

var (
	sourceRgx = regexp.MustCompile(`\[(.+)\]`)
	deviceRgx = regexp.MustCompile(`([^\[]+)\[.+\]`)

	findMntByVolume = func(volumeName string, pid int) ([]byte, error) {
		return exec.Command("/usr/bin/findmnt", "-T", fmt.Sprintf("/%s", volumeName), "-N", strconv.Itoa(pid), "-J").CombinedOutput()
	}

	findMntByDevice = func(deviceName string) ([]byte, error) {
		return exec.Command("/usr/bin/findmnt", "-S", deviceName, "-N", "1", "-J").CombinedOutput()
	}
)

type FindmntInfo struct {
	Target  string `json:"target"`
	Source  string `json:"source"`
	Fstype  string `json:"fstype"`
	Options string `json:"options"`
}

type FileSystems struct {
	Filesystems []FindmntInfo `json:"filesystems"`
}

func LookupFindmntInfoByVolume(volumeName string, pid int) ([]FindmntInfo, error) {
	mntInfoJson, err := findMntByVolume(volumeName, pid)
	if err != nil {
		return make([]FindmntInfo, 0), fmt.Errorf("Error running findmnt for volume %s: %w", volumeName, err)
	}
	return parseMntInfoJson(mntInfoJson)
}

func LookupFindmntInfoByDevice(deviceName string) ([]FindmntInfo, error) {
	mntInfoJson, err := findMntByDevice(deviceName)
	if err != nil {
		return make([]FindmntInfo, 0), fmt.Errorf("Error running findmnt for device %s: %w", deviceName, err)
	}
	return parseMntInfoJson(mntInfoJson)
}

func parseMntInfoJson(mntInfoJson []byte) ([]FindmntInfo, error) {
	mntinfo := FileSystems{}
	if err := json.Unmarshal(mntInfoJson, &mntinfo); err != nil {
		return mntinfo.Filesystems, fmt.Errorf("unable to unmarshal [%v]: %w", mntInfoJson, err)
	}
	return mntinfo.Filesystems, nil
}

func (f *FindmntInfo) GetSourcePath() string {
	match := sourceRgx.FindStringSubmatch(f.Source)
	if len(match) != 2 {
		return strings.TrimPrefix(f.Source, rhcosPrefix)
	}
	return strings.TrimPrefix(match[1], rhcosPrefix)
}

func (f *FindmntInfo) GetSourceDevice() string {
	match := deviceRgx.FindStringSubmatch(f.Source)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func (f *FindmntInfo) GetOptions() []string {
	return strings.Split(f.Options, ",")
}
