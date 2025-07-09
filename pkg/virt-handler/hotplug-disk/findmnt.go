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
	"fmt"
	"os"
	"path"
	"strings"

	mount "github.com/moby/sys/mountinfo"
)

const (
	rhcosPrefix = "/ostree/deploy/rhcos"
)

var (
	findMntByPID = func(pid int) ([]*mount.Info, error) {
		return getMountInfo(pid)
	}

	findMntByOne = func() ([]*mount.Info, error) {
		return getMountInfo(1)
	}
)

func getMountInfo(pid int) ([]*mount.Info, error) {
	procMountInfo := fmt.Sprintf("/proc/%d/mountinfo", pid)
	f, err := os.Open(procMountInfo)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return mount.GetMountsFromReader(f, nil)
}

type FindmntInfo struct {
	Target  string `json:"target"`
	Source  string `json:"source"`
	Fstype  string `json:"fstype"`
	Options string `json:"options"`
}

func LookupFindmntInfoByVolume(volumeName string, pid int) ([]FindmntInfo, error) {
	mounts, err := findMntByPID(pid)
	if err != nil {
		return nil, err
	}

	var result []FindmntInfo

	mountPoint := path.Join("/", volumeName)
	for _, mountInfo := range mounts {
		if mountInfo.Mountpoint == mountPoint {
			result = append(result, FindmntInfo{
				Target:  mountInfo.Mountpoint,
				Source:  mountInfo.Source,
				Fstype:  mountInfo.FSType,
				Options: mountInfo.Options,
			})
		}
	}
	return result, nil
}

func LookupFindmntInfoByDevice(deviceName string) ([]FindmntInfo, error) {
	mounts, err := findMntByOne()
	if err != nil {
		return nil, err
	}

	var result []FindmntInfo

	for _, mountInfo := range mounts {
		if mountInfo.Source == deviceName {
			result = append(result, FindmntInfo{
				Target:  mountInfo.Mountpoint,
				Source:  mountInfo.Source,
				Fstype:  mountInfo.FSType,
				Options: mountInfo.Options,
			})
		}
	}
	return result, nil
}

func (f *FindmntInfo) GetSourcePath() string {
	return strings.TrimPrefix(f.Source, rhcosPrefix)
}
