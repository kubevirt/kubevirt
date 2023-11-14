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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package hotplugdisk

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/safepath"

	"kubevirt.io/kubevirt/pkg/util"
)

var mountBaseDir = filepath.Join(util.VirtShareDir, "/hotplug-disks")

const (
	hotplugDisksKubeletVolumePath = "volumes/kubernetes.io~empty-dir/hotplug-disks"
)

var (
	// visible for testing
	TargetPodBasePath = func(podBaseDir string, podUID types.UID) string {
		return filepath.Join(podBaseDir, string(podUID), hotplugDisksKubeletVolumePath)
	}
)

type HotplugDiskManagerInterface interface {
	GetHotplugTargetPodPathOnHost(virtlauncherPodUID types.UID) (*safepath.Path, error)
	GetFileSystemDiskTargetPathFromHostView(virtlauncherPodUID types.UID, volumeName string, create bool) (*safepath.Path, error)
	GetFileSystemDirectoryTargetPathFromHostView(virtlauncherPodUID types.UID, volumeName string, create bool) (*safepath.Path, error)
}

func NewHotplugDiskManager(kubeletPodsDir string) *hotplugDiskManager {
	return &hotplugDiskManager{
		podsBaseDir: filepath.Join(util.HostRootMount, kubeletPodsDir),
	}
}

func NewHotplugDiskWithOptions(podsBaseDir string) *hotplugDiskManager {
	return &hotplugDiskManager{
		podsBaseDir: podsBaseDir,
	}
}

type hotplugDiskManager struct {
	podsBaseDir string
}

// GetHotplugTargetPodPathOnHost retrieves the target pod (virt-launcher) path on the host.
func (h *hotplugDiskManager) GetHotplugTargetPodPathOnHost(virtlauncherPodUID types.UID) (*safepath.Path, error) {
	podpath := TargetPodBasePath(h.podsBaseDir, virtlauncherPodUID)
	return safepath.JoinAndResolveWithRelativeRoot("/", podpath)
}

// GetFileSystemDirectoryTargetPathFromHostView gets the directory path in the target pod (virt-launcher) on the host.
func (h *hotplugDiskManager) GetFileSystemDirectoryTargetPathFromHostView(virtlauncherPodUID types.UID, volumeName string, create bool) (*safepath.Path, error) {
	targetPath, err := h.GetHotplugTargetPodPathOnHost(virtlauncherPodUID)
	if err != nil {
		return nil, err
	}
	_, err = safepath.JoinNoFollow(targetPath, volumeName)
	if errors.Is(err, os.ErrNotExist) && create {
		if err := safepath.MkdirAtNoFollow(targetPath, volumeName, 0750); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return safepath.JoinNoFollow(targetPath, volumeName)
}

// GetFileSystemDiskTargetPathFromHostView gets the disk image file in the target pod (virt-launcher) on the host.
func (h *hotplugDiskManager) GetFileSystemDiskTargetPathFromHostView(virtlauncherPodUID types.UID, volumeName string, create bool) (*safepath.Path, error) {
	targetPath, err := h.GetHotplugTargetPodPathOnHost(virtlauncherPodUID)
	if err != nil {
		return targetPath, err
	}
	diskName := fmt.Sprintf("%s.img", volumeName)
	if err := safepath.TouchAtNoFollow(targetPath, diskName, 0666); err != nil && !os.IsExist(err) {
		return nil, err
	}
	return safepath.JoinNoFollow(targetPath, diskName)
}

// SetLocalDirectory creates the base directory where disk images will be mounted when hotplugged. File system volumes will be in
// a directory under this, that contains the volume name. block volumes will be in this directory as a block device.
func SetLocalDirectory(dir string) error {
	mountBaseDir = dir
	return os.MkdirAll(dir, 0755)
}

func GetVolumeMountDir(volumeName string) string {
	return filepath.Join(mountBaseDir, volumeName)
}
