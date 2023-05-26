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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package hostdisk

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"syscall"

	"kubevirt.io/client-go/log"

	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/unsafepath"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/util"
)

var pvcBaseDir = "/var/run/kubevirt-private/vmi-disks"

const (
	EventReasonToleratedSmallPV = "ToleratedSmallPV"
	EventTypeToleratedSmallPV   = k8sv1.EventTypeNormal
)

// Used by tests.
func setDiskDirectory(dir string) error {
	pvcBaseDir = dir
	return os.MkdirAll(dir, 0750)
}

func ReplacePVCByHostDisk(vmi *v1.VirtualMachineInstance) error {
	// If PVC is defined and it's not a BlockMode PVC, then it is replaced by HostDisk
	// Filesystem PersistenVolumeClaim is mounted into pod as directory from node filesystem
	passthoughFSVolumes := make(map[string]struct{})
	for i := range vmi.Spec.Domain.Devices.Filesystems {
		passthoughFSVolumes[vmi.Spec.Domain.Devices.Filesystems[i].Name] = struct{}{}
	}

	pvcVolume := make(map[string]v1.VolumeStatus)
	hotplugVolumes := make(map[string]bool)
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.HotplugVolume != nil {
			hotplugVolumes[volumeStatus.Name] = true
		}

		if volumeStatus.PersistentVolumeClaimInfo != nil {
			pvcVolume[volumeStatus.Name] = volumeStatus
		}
	}

	for i := range vmi.Spec.Volumes {
		volume := vmi.Spec.Volumes[i]
		volumeSource := &vmi.Spec.Volumes[i].VolumeSource
		if volumeSource.PersistentVolumeClaim != nil {
			if shouldSkipVolumeSource(passthoughFSVolumes, hotplugVolumes, pvcVolume, volume.Name) {
				continue
			}

			err := replaceForHostDisk(volumeSource, volume.Name, pvcVolume)
			if err != nil {
				return err
			}
			// PersistenVolumeClaim is replaced by HostDisk
			volumeSource.PersistentVolumeClaim = nil
		}
		if volumeSource.DataVolume != nil && volumeSource.DataVolume.Name != "" {
			if shouldSkipVolumeSource(passthoughFSVolumes, hotplugVolumes, pvcVolume, volume.Name) {
				continue
			}

			err := replaceForHostDisk(volumeSource, volume.Name, pvcVolume)
			if err != nil {
				return err
			}
			// PersistenVolumeClaim is replaced by HostDisk
			volumeSource.DataVolume = nil
		}
	}
	return nil
}

func replaceForHostDisk(volumeSource *v1.VolumeSource, volumeName string, pvcVolume map[string]v1.VolumeStatus) error {
	volumeStatus := pvcVolume[volumeName]
	isShared := types.HasSharedAccessMode(volumeStatus.PersistentVolumeClaimInfo.AccessModes)
	file := getPVCDiskImgPath(volumeName, "disk.img")
	capacity := volumeStatus.PersistentVolumeClaimInfo.Capacity[k8sv1.ResourceStorage]
	requested := volumeStatus.PersistentVolumeClaimInfo.Requests[k8sv1.ResourceStorage]
	// Use the requested size if it is smaller than the overall capacity of the PVC to ensure the created disks are the size requested by the user
	if capacity.Value() > requested.Value() {
		capacity = requested
	}
	// The host-disk must be 1MiB-aligned. If the volume specifies a misaligned size, shrink it down to the nearest multiple of 1MiB
	size := util.AlignImageSizeTo1MiB(capacity.Value(), log.Log)
	if size == 0 {
		return fmt.Errorf("the size for volume %s is too low, must be at least 1MiB", volumeName)
	}
	capacity.Set(size)
	volumeSource.HostDisk = &v1.HostDisk{
		Path:     file,
		Type:     v1.HostDiskExistsOrCreate,
		Capacity: capacity,
		Shared:   &isShared,
	}

	return nil
}

func shouldSkipVolumeSource(passthoughFSVolumes map[string]struct{}, hotplugVolumes map[string]bool, pvcVolume map[string]v1.VolumeStatus, volumeName string) bool {
	// If a PVC is used in a Filesystem (passthough), it should not be mapped as a HostDisk and a image file should
	// not be created.
	if _, isPassthoughFSVolume := passthoughFSVolumes[volumeName]; isPassthoughFSVolume {
		log.Log.V(4).Infof("this volume %s is mapped as a filesystem passthrough, will not be replaced by HostDisk", volumeName)
		return true
	}

	if hotplugVolumes[volumeName] {
		log.Log.V(4).Infof("this volume %s is hotplugged, will not be replaced by HostDisk", volumeName)
		return true
	}

	volumeStatus, ok := pvcVolume[volumeName]
	if !ok || types.IsPVCBlock(volumeStatus.PersistentVolumeClaimInfo.VolumeMode) {
		log.Log.V(4).Infof("this volume %s is block, will not be replaced by HostDisk", volumeName)
		// This is not a disk on a file system, so skip it.
		return true
	}
	return false
}

func dirBytesAvailable(path string, reserve uint64) (uint64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}
	return stat.Bavail*uint64(stat.Bsize) - reserve, nil
}

func createSparseRaw(fullPath string, size int64) (err error) {
	offset := size - 1
	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer util.CloseIOAndCheckErr(f, &err)
	_, err = f.WriteAt([]byte{0}, offset)
	if err != nil {
		return err
	}
	return nil
}

func getPVCDiskImgPath(volumeName string, diskName string) string {
	return path.Join(pvcBaseDir, volumeName, diskName)
}

func GetMountedHostDiskPathFromHandler(mountRoot, volumeName, path string) string {
	return filepath.Join(mountRoot, getPVCDiskImgPath(volumeName, filepath.Base(path)))
}

func GetMountedHostDiskDirFromHandler(mountRoot, volumeName string) string {
	return filepath.Join(mountRoot, getPVCDiskImgPath(volumeName, ""))
}

func GetMountedHostDiskPath(volumeName string, path string) string {
	return getPVCDiskImgPath(volumeName, filepath.Base(path))
}

func GetMountedHostDiskDir(volumeName string) string {
	return getPVCDiskImgPath(volumeName, "")
}

type DiskImgCreator struct {
	dirBytesAvailableFunc  func(path string, reserve uint64) (uint64, error)
	recorder               record.EventRecorder
	lessPVCSpaceToleration int
	minimumPVCReserveBytes uint64
	mountRoot              *safepath.Path
}

func NewHostDiskCreator(recorder record.EventRecorder, lessPVCSpaceToleration int, minimumPVCReserveBytes uint64, mountRoot *safepath.Path) DiskImgCreator {
	return DiskImgCreator{
		dirBytesAvailableFunc:  dirBytesAvailable,
		recorder:               recorder,
		lessPVCSpaceToleration: lessPVCSpaceToleration,
		minimumPVCReserveBytes: minimumPVCReserveBytes,
		mountRoot:              mountRoot,
	}
}

func (hdc *DiskImgCreator) setlessPVCSpaceToleration(toleration int) {
	hdc.lessPVCSpaceToleration = toleration
}

func (hdc DiskImgCreator) Create(vmi *v1.VirtualMachineInstance) error {
	for _, volume := range vmi.Spec.Volumes {
		if hostDisk := volume.VolumeSource.HostDisk; shouldMountHostDisk(hostDisk) {
			if err := hdc.mountHostDiskAndSetOwnership(vmi, volume.Name, hostDisk); err != nil {
				return err
			}
		}
	}
	return nil
}

func shouldMountHostDisk(hostDisk *v1.HostDisk) bool {
	return hostDisk != nil && hostDisk.Type == v1.HostDiskExistsOrCreate && hostDisk.Path != ""
}

func (hdc *DiskImgCreator) mountHostDiskAndSetOwnership(vmi *v1.VirtualMachineInstance, volumeName string, hostDisk *v1.HostDisk) error {
	diskPath := GetMountedHostDiskPathFromHandler(unsafepath.UnsafeAbsolute(hdc.mountRoot.Raw()), volumeName, hostDisk.Path)
	diskDir := GetMountedHostDiskDirFromHandler(unsafepath.UnsafeAbsolute(hdc.mountRoot.Raw()), volumeName)
	fileExists, err := ephemeraldiskutils.FileExists(diskPath)
	if err != nil {
		return err
	}
	if !fileExists {
		if err := hdc.handleRequestedSizeAndCreateSparseRaw(vmi, diskDir, diskPath, hostDisk); err != nil {
			return err
		}
	}
	// Change file ownership to the qemu user.
	if err := ephemeraldiskutils.DefaultOwnershipManager.UnsafeSetFileOwnership(diskPath); err != nil {
		log.Log.Reason(err).Errorf("Couldn't set Ownership on %s: %v", diskPath, err)
		return err
	}
	return nil
}

func (hdc *DiskImgCreator) handleRequestedSizeAndCreateSparseRaw(vmi *v1.VirtualMachineInstance, diskDir string, diskPath string, hostDisk *v1.HostDisk) error {
	size, err := hdc.dirBytesAvailableFunc(diskDir, hdc.minimumPVCReserveBytes)
	availableSize := int64(size)
	if err != nil {
		return err
	}
	requestedSize, _ := hostDisk.Capacity.AsInt64()
	if requestedSize > availableSize {
		requestedSize, err = hdc.shrinkRequestedSize(vmi, requestedSize, availableSize, hostDisk)
		if err != nil {
			return err
		}
	}
	err = createSparseRaw(diskPath, requestedSize)
	if err != nil {
		log.Log.Reason(err).Errorf("Couldn't create a sparse raw file for disk path: %s, error: %v", diskPath, err)
		return err
	}
	return nil
}

func (hdc *DiskImgCreator) shrinkRequestedSize(vmi *v1.VirtualMachineInstance, requestedSize int64, availableSize int64, hostDisk *v1.HostDisk) (int64, error) {
	// Some storage provisioners provide less space than requested, due to filesystem overhead etc.
	// We tolerate some difference in requested and available capacity up to some degree.
	// This can be configured with the "pvc-tolerate-less-space-up-to-percent" parameter in the kubevirt-config ConfigMap.
	// It is provided as argument to virt-launcher.
	toleratedSize := requestedSize * (100 - int64(hdc.lessPVCSpaceToleration)) / 100
	if toleratedSize > availableSize {
		return 0, fmt.Errorf("unable to create %s, not enough space, demanded size %d B is bigger than available space %d B, also after taking %v %% toleration into account",
			hostDisk.Path, uint64(requestedSize), availableSize, hdc.lessPVCSpaceToleration)
	}

	msg := fmt.Sprintf("PV size too small: expected %v B, found %v B. Using it anyway, it is within %v %% toleration", requestedSize, availableSize, hdc.lessPVCSpaceToleration)
	log.Log.Info(msg)
	hdc.recorder.Event(vmi, EventTypeToleratedSmallPV, EventReasonToleratedSmallPV, msg)
	return availableSize, nil
}
