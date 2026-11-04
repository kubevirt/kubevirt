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

package hostdisk

import (
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/storage/diskimg"
	"kubevirt.io/kubevirt/pkg/storage/volumepath"
)

func GetMountedHostDiskPath(volumeName string, path string) string {
	return filepath.Join(volumepath.FilesystemDir(volumeName), filepath.Base(path))
}

func GetMountedHostDiskDir(volumeName string) string {
	return volumepath.FilesystemDir(volumeName)
}

func Images(vmi *v1.VirtualMachineInstance) []diskimg.Image {
	var images []diskimg.Image
	for _, volume := range vmi.Spec.Volumes {
		if hostDisk := volume.HostDisk; shouldCreateImage(hostDisk) {
			size, _ := hostDisk.Capacity.AsInt64()
			images = append(images, diskimg.Image{
				Dir:  GetMountedHostDiskDir(volume.Name),
				Name: filepath.Base(hostDisk.Path),
				Size: size,
			})
		}
	}
	return images
}

func shouldCreateImage(hostDisk *v1.HostDisk) bool {
	return hostDisk != nil && hostDisk.Type == v1.HostDiskExistsOrCreate && hostDisk.Path != ""
}
