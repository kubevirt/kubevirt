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

package diskimg

import (
	"fmt"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/storage/disksize"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/storage/volumepath"
)

func ImageBackedPVCs(vmi *v1.VirtualMachineInstance) map[string]*v1.PersistentVolumeClaimInfo {
	filesystems := storagetypes.GetFilesystemsFromVolumes(vmi)

	pvcInfo := make(map[string]*v1.PersistentVolumeClaimInfo)
	for _, status := range vmi.Status.VolumeStatus {
		if status.HotplugVolume == nil && status.PersistentVolumeClaimInfo != nil {
			pvcInfo[status.Name] = status.PersistentVolumeClaimInfo
		}
	}

	fsPVCs := make(map[string]*v1.PersistentVolumeClaimInfo)
	for _, volume := range vmi.Spec.Volumes {
		if volume.PersistentVolumeClaim == nil && (volume.DataVolume == nil || volume.DataVolume.Name == "") {
			continue
		}
		if _, ok := filesystems[volume.Name]; ok {
			continue
		}
		info, ok := pvcInfo[volume.Name]
		if !ok || storagetypes.IsPVCBlock(info.VolumeMode) {
			continue
		}
		fsPVCs[volume.Name] = info
	}
	return fsPVCs
}

func ImageBackedPVCImages(vmi *v1.VirtualMachineInstance) ([]Image, error) {
	pvcs := ImageBackedPVCs(vmi)
	var images []Image
	for _, volume := range vmi.Spec.Volumes {
		info, ok := pvcs[volume.Name]
		if !ok {
			continue
		}
		size, err := imageSize(volume.Name, info)
		if err != nil {
			return nil, err
		}
		images = append(images, Image{
			Dir:  volumepath.FilesystemDir(volume.Name),
			Name: volumepath.FilesystemImageName,
			Size: size,
		})
	}
	return images, nil
}

func imageSize(volumeName string, info *v1.PersistentVolumeClaimInfo) (int64, error) {
	capacity, capacityOk := info.Capacity[k8sv1.ResourceStorage]
	requested, requestedOk := info.Requests[k8sv1.ResourceStorage]
	if !capacityOk && !requestedOk {
		return 0, fmt.Errorf("unable to determine disk image size of volume %s from PVC that provides no storage capacity or requests", volumeName)
	}

	var size int64
	if requestedOk && (!capacityOk || capacity.Value() > requested.Value()) {
		size = disksize.AlignImageSizeTo1MiB(requested.Value(), log.Log)
	} else {
		size = disksize.AlignImageSizeTo1MiB(capacity.Value(), log.Log)
	}
	if size == 0 {
		return 0, fmt.Errorf("the size for volume %s is too low, must be at least 1MiB", volumeName)
	}
	return size, nil
}
