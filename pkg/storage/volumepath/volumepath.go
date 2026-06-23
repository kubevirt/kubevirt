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

package volumepath

import (
	"fmt"
	"path/filepath"
)

func Filesystem(volumeName string) string {
	return filepath.Join(string(filepath.Separator), "var", "run", "kubevirt-private", "vmi-disks", volumeName, "disk.img")
}

func HotplugFilesystem(volumeName string) string {
	return filepath.Join(string(filepath.Separator), "var", "run", "kubevirt", "hotplug-disks", fmt.Sprintf("%s.img", volumeName))
}

func BlockDevice(volumeName string) string {
	return filepath.Join(string(filepath.Separator), "dev", volumeName)
}

func HotplugBlockDevice(volumeName string) string {
	return filepath.Join(string(filepath.Separator), "var", "run", "kubevirt", "hotplug-disks", volumeName)
}

func Image(volumeName string, isBlock, isHotplug bool) string {
	if isBlock {
		if isHotplug {
			return HotplugBlockDevice(volumeName)
		}
		return BlockDevice(volumeName)
	}

	if isHotplug {
		return HotplugFilesystem(volumeName)
	}
	return Filesystem(volumeName)
}
