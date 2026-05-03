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

package storage

import (
	"fmt"
	"path/filepath"
)

func GetFilesystemVolumePath(volumeName string) string {
	return filepath.Join(string(filepath.Separator), "var", "run", "kubevirt-private", "vmi-disks", volumeName, "disk.img")
}

// GetHotplugFilesystemVolumePath returns the path and file name of a hotplug disk image
func GetHotplugFilesystemVolumePath(volumeName string) string {
	return filepath.Join(string(filepath.Separator), "var", "run", "kubevirt", "hotplug-disks", fmt.Sprintf("%s.img", volumeName))
}

func GetBlockDeviceVolumePath(volumeName string) string {
	return filepath.Join(string(filepath.Separator), "dev", volumeName)
}

// GetHotplugBlockDeviceVolumePath returns the path and name of a hotplugged block device
func GetHotplugBlockDeviceVolumePath(volumeName string) string {
	return filepath.Join(string(filepath.Separator), "var", "run", "kubevirt", "hotplug-disks", volumeName)
}

// GetVolumeImagePath returns the backing image path for a volume, considering whether it's
// a hotplug volume and whether it's a block device
func GetVolumeImagePath(volumeName string, isBlock, isHotplug bool) string {
	if isBlock {
		if isHotplug {
			return GetHotplugBlockDeviceVolumePath(volumeName)
		}
		return GetBlockDeviceVolumePath(volumeName)
	}

	if isHotplug {
		return GetHotplugFilesystemVolumePath(volumeName)
	}
	return GetFilesystemVolumePath(volumeName)
}
