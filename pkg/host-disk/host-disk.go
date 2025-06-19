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
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"

	"kubevirt.io/client-go/log"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/safepath"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/util"
)

var pvcBaseDir = "/var/run/kubevirt-private/vmi-disks"

const (
	EventReasonToleratedSmallPV = "ToleratedSmallPV"
	EventTypeToleratedSmallPV   = k8sv1.EventTypeNormal
)

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
	capacity, capacityOk := volumeStatus.PersistentVolumeClaimInfo.Capacity[k8sv1.ResourceStorage]
	requested, requestedOk := volumeStatus.PersistentVolumeClaimInfo.Requests[k8sv1.ResourceStorage]

	if !capacityOk && !requestedOk {
		return fmt.Errorf("unable to determine capacity of HostDisk from PVC that provides no storage capacity or requests")
	}

	var size int64
	// Use the requested size if it is smaller than the overall capacity of the PVC to ensure the created disks are the size requested by the user
	if requestedOk && ((capacityOk && capacity.Value() > requested.Value()) || !capacityOk) {
		// The host-disk must be 1MiB-aligned. If the volume specifies a misaligned size, shrink it down to the nearest multiple of 1MiB
		size = util.AlignImageSizeTo1MiB(requested.Value(), log.Log)
	} else {
		size = util.AlignImageSizeTo1MiB(capacity.Value(), log.Log)
	}

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

func createSparseRaw(diskdir *safepath.Path, diskName string, size int64) (err error) {
	offset := size - 1
	if filepath.Base(diskName) != diskName {
		return fmt.Errorf("Disk name needs to be base")
	}

	err = safepath.TouchAtNoFollow(diskdir, filepath.Base(diskName), 0666)
	if err != nil {
		return err
	}

	diskPath, err := safepath.JoinNoFollow(diskdir, diskName)
	if err != nil {
		return err
	}

	sFile, err := safepath.OpenAtNoFollow(diskPath)
	if err != nil {
		return err
	}
	defer util.CloseIOAndCheckErr(sFile, &err)

	f, err := os.OpenFile(sFile.SafePath(), os.O_WRONLY, 0666)
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

func createQcow2(fullPath string, size int64) (err error) {
	log.Log.Infof("Create %s with qcow2 format", fullPath)
	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", fullPath, fmt.Sprintf("%db", size))
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("failed to create qcow2: %w", err)
	}
	return nil
}

func getPVCDiskImgPath(volumeName string, diskName string) string {
	return path.Join(pvcBaseDir, volumeName, diskName)
}

func GetMountedHostDiskPath(volumeName string, path string) string {
	return getPVCDiskImgPath(volumeName, filepath.Base(path))
}

func GetMountedHostDiskDir(volumeName string) string {
	return getPVCDiskImgPath(volumeName, "")
}

type HostDiskImgCreator struct {
	mountRoot *safepath.Path

	diskImgCreator diskImgCreator
}

func NewHostDiskImgCreator(recorder record.EventRecorder, lessPVCSpaceToleration int, minimumPVCReserveBytes uint64, mountRoot *safepath.Path) HostDiskImgCreator {
	return HostDiskImgCreator{
		mountRoot:      mountRoot,
		diskImgCreator: newDiskImgCreator(recorder, lessPVCSpaceToleration, minimumPVCReserveBytes),
	}
}

func (hdc *HostDiskImgCreator) setlessPVCSpaceToleration(toleration int) {
	hdc.diskImgCreator.lessPVCSpaceToleration = toleration
}

func (hdc *HostDiskImgCreator) Create(vmi *v1.VirtualMachineInstance) error {
	for _, volume := range vmi.Spec.Volumes {
		if hostDisk := volume.VolumeSource.HostDisk; shouldMountHostDisk(hostDisk) {
			diskPath := GetMountedHostDiskPath(volume.Name, hostDisk.Path)
			diskDir := GetMountedHostDiskDir(volume.Name)

			requestedSize, _ := hostDisk.Capacity.AsInt64()
			if err := hdc.diskImgCreator.CreateDiskAndSetOwnership(vmi, diskDir, diskPath, volume.Name, requestedSize); err != nil {
				return err
			}
		}
	}
	return nil
}

func shouldMountHostDisk(hostDisk *v1.HostDisk) bool {
	return hostDisk != nil && hostDisk.Type == v1.HostDiskExistsOrCreate && hostDisk.Path != ""
}
