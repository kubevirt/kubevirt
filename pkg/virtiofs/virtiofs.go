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

package virtiofs

import (
	"fmt"
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/config"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/util"
)

const (
	PlaceholderSocketVolumeMountPoint = "/var/run/sockets"
	PlaceholderSocketVolumeName       = "virtiofs-sockets"
)

// This is empty dir
var VirtioFSContainers = "virtiofs-containers"
var VirtioFSContainersMountBaseDir = filepath.Join(util.VirtShareDir, VirtioFSContainers)

func VirtioFSSocketPath(volumeName string) string {
	socketName := fmt.Sprintf("%s.sock", volumeName)
	return filepath.Join(VirtioFSContainersMountBaseDir, socketName)
}

func PlaceholderSocketName(volumeName string) string {
	return fmt.Sprintf("%s.sock", volumeName)
}

func PlaceholderSocketPath(volumeName string) string {
	return filepath.Join(PlaceholderSocketVolumeMountPoint, PlaceholderSocketName(volumeName))
}

func GetFilesystemPersistentVolumes(vmi *v1.VirtualMachineInstance) []v1.Volume {
	var vols []v1.Volume
	fss := storagetypes.GetFilesystemsFromVolumes(vmi)
	for _, volume := range vmi.Spec.Volumes {
		if _, ok := fss[volume.Name]; !ok {
			continue
		}
		if volume.VolumeSource.PersistentVolumeClaim != nil ||
			volume.VolumeSource.DataVolume != nil {
			vols = append(vols, volume)
		}
	}

	return vols
}

func HasFilesystemPersistentVolumes(vmi *v1.VirtualMachineInstance) bool {
	return len(GetFilesystemPersistentVolumes(vmi)) > 0
}

func FSMountPoint(volume *v1.Volume) string {
	switch {
	case volume.ConfigMap != nil:
		return config.GetConfigMapSourcePath(volume.Name)
	case volume.Secret != nil:
		return config.GetSecretSourcePath(volume.Name)
	case volume.ServiceAccount != nil:
		return config.ServiceAccountSourceDir
	case volume.DownwardAPI != nil:
		return config.GetDownwardAPISourcePath(volume.Name)
	default:
		return fmt.Sprintf("/%s", volume.Name)
	}
}
