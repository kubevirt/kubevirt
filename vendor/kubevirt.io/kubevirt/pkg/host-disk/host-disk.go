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
	"syscall"

	"kubevirt.io/kubevirt/pkg/log"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/util/types"
)

const (
	pvcBaseDir                  = "/var/run/kubevirt-private/vmi-disks"
	EventReasonToleratedSmallPV = "ToleratedSmallPV"
	EventTypeToleratedSmallPV   = k8sv1.EventTypeNormal
)

func ReplacePVCByHostDisk(vmi *v1.VirtualMachineInstance, clientset kubecli.KubevirtClient) error {
	// If PVC is defined and it's not a BlockMode PVC, then it is replaced by HostDisk
	// Filesystem PersistenVolumeClaim is mounted into pod as directory from node filesystem
	for i := range vmi.Spec.Volumes {
		if volumeSource := &vmi.Spec.Volumes[i].VolumeSource; volumeSource.PersistentVolumeClaim != nil {

			pvc, exists, isBlockVolumePVC, err := types.IsPVCBlockFromClient(clientset, vmi.Namespace, volumeSource.PersistentVolumeClaim.ClaimName)
			if err != nil {
				return err
			} else if !exists {
				return fmt.Errorf("persistentvolumeclaim %v not found", volumeSource.PersistentVolumeClaim.ClaimName)
			} else if isBlockVolumePVC {
				continue
			}
			isSharedPvc := types.IsPVCShared(pvc)

			volumeSource.HostDisk = &v1.HostDisk{
				Path:     getPVCDiskImgPath(vmi.Spec.Volumes[i].Name),
				Type:     v1.HostDiskExistsOrCreate,
				Capacity: pvc.Status.Capacity[k8sv1.ResourceStorage],
				Shared:   &isSharedPvc,
			}
			// PersistenVolumeClaim is replaced by HostDisk
			volumeSource.PersistentVolumeClaim = nil
		}
	}
	return nil
}

func dirBytesAvailable(path string) (uint64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}

func createSparseRaw(fullPath string, size int64) error {
	offset := size - 1
	f, _ := os.Create(fullPath)
	defer f.Close()
	_, err := f.WriteAt([]byte{0}, offset)
	if err != nil {
		return err
	}
	return nil
}

func getPVCDiskImgPath(volumeName string) string {
	return path.Join(pvcBaseDir, volumeName, "disk.img")
}

type DiskImgCreator struct {
	dirBytesAvailableFunc  func(path string) (uint64, error)
	notifier               k8sNotifier
	lessPVCSpaceToleration int
}

type k8sNotifier interface {
	SendK8sEvent(vmi *v1.VirtualMachineInstance, severity string, reason string, message string) error
}

func NewHostDiskCreator(notifier k8sNotifier, lessPVCSpaceToleration int) DiskImgCreator {
	return DiskImgCreator{
		dirBytesAvailableFunc:  dirBytesAvailable,
		notifier:               notifier,
		lessPVCSpaceToleration: lessPVCSpaceToleration,
	}
}

func (hdc *DiskImgCreator) setlessPVCSpaceToleration(toleration int) {
	hdc.lessPVCSpaceToleration = toleration
}

func (hdc DiskImgCreator) Create(vmi *v1.VirtualMachineInstance) error {
	for _, volume := range vmi.Spec.Volumes {
		if hostDisk := volume.VolumeSource.HostDisk; hostDisk != nil && hostDisk.Type == v1.HostDiskExistsOrCreate && hostDisk.Path != "" {
			if _, err := os.Stat(hostDisk.Path); os.IsNotExist(err) {
				availableSize, err := hdc.dirBytesAvailableFunc(path.Dir(hostDisk.Path))
				if err != nil {
					return err
				}
				requestedSize, _ := hostDisk.Capacity.AsInt64()
				diskSize := requestedSize
				if uint64(requestedSize) > availableSize {
					// Some storage provisioners provision less space than requested, due to filesystem overhead etc.
					// We tolerate some difference in requested and available capacity up to some degree.
					// This can be configured with the "pvc-tolerate-less-space-up-to-percent" parameter in the kubevirt-config ConfigMap.
					// It is provided as argument to virt-launcher.
					toleratedSize := requestedSize * (100 - int64(hdc.lessPVCSpaceToleration)) / 100
					if uint64(toleratedSize) > availableSize {
						return fmt.Errorf("unable to create %s, not enough space, demanded size %d B is bigger than available space %d B, also after taking %v %% toleration into account",
							hostDisk.Path, uint64(requestedSize), availableSize, hdc.lessPVCSpaceToleration)
					}
					diskSize = int64(availableSize)

					msg := fmt.Sprintf("PV size too small: expected %v B, found %v B. Using it anyway, it is within %v %% toleration", requestedSize, availableSize, hdc.lessPVCSpaceToleration)
					log.Log.Info(msg)
					err = hdc.notifier.SendK8sEvent(vmi, EventTypeToleratedSmallPV, EventReasonToleratedSmallPV, msg)
					if err != nil {
						log.Log.Reason(err).Warningf("Couldn't send k8s event for tolerated PV size: %v", err)
					}
				}
				err = createSparseRaw(hostDisk.Path, int64(diskSize))
				if err != nil {
					return err
				}
			} else if err != nil {
				return err
			}

		}
	}
	return nil
}
