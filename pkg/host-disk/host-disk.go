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

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/types"
)

var pvcBaseDir = "/var/run/kubevirt-private/vmi-disks"

const (
	EventReasonToleratedSmallPV = "ToleratedSmallPV"
	EventTypeToleratedSmallPV   = k8sv1.EventTypeNormal
)

// Used by tests.
func setDiskDirectory(dir string) error {
	pvcBaseDir = dir
	return os.MkdirAll(dir, 0755)
}

func ReplacePVCByHostDisk(vmi *v1.VirtualMachineInstance, clientset kubecli.KubevirtClient) error {
	// If PVC is defined and it's not a BlockMode PVC, then it is replaced by HostDisk
	// Filesystem PersistenVolumeClaim is mounted into pod as directory from node filesystem
	passthoughFSVolumes := make(map[string]struct{})
	for i := range vmi.Spec.Domain.Devices.Filesystems {
		passthoughFSVolumes[vmi.Spec.Domain.Devices.Filesystems[i].Name] = struct{}{}
	}

	for i := range vmi.Spec.Volumes {
		volume := vmi.Spec.Volumes[i]
		if volumeSource := &vmi.Spec.Volumes[i].VolumeSource; volumeSource.PersistentVolumeClaim != nil {
			// If a PVC is used in a Filesystem (passthough), it should not be mapped as a HostDisk and a image file should
			// not be created.
			if _, isPassthoughFSVolume := passthoughFSVolumes[volume.Name]; isPassthoughFSVolume {
				log.Log.V(4).Infof("this volume %s is mapped as a filesystem passthrough, will not be replaced by HostDisk", volume.Name)
				continue
			}

			pvc, exists, isBlockVolumePVC, err := types.IsPVCBlockFromClient(clientset, vmi.Namespace, volumeSource.PersistentVolumeClaim.ClaimName)
			if err != nil {
				return err
			} else if !exists {
				return fmt.Errorf("persistentvolumeclaim %v not found", volumeSource.PersistentVolumeClaim.ClaimName)
			} else if isBlockVolumePVC {
				continue
			}
			isSharedPvc := types.IsPVCShared(pvc)

			file := getPVCDiskImgPath(vmi.Spec.Volumes[i].Name, "disk.img")
			volumeSource.HostDisk = &v1.HostDisk{
				Path:     file,
				Type:     v1.HostDiskExistsOrCreate,
				Capacity: pvc.Status.Capacity[k8sv1.ResourceStorage],
				Shared:   &isSharedPvc,
			}
			// PersistenVolumeClaim is replaced by HostDisk
			volumeSource.PersistentVolumeClaim = nil
			// Set ownership of the disk.img to qemu
			if err := ephemeraldiskutils.DefaultOwnershipManager.SetFileOwnership(file); err != nil && !os.IsNotExist(err) {
				log.Log.Reason(err).Errorf("Couldn't set Ownership on %s: %v", file, err)
				return err
			}
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

func GetMountedHostDiskPath(volumeName string, path string) string {
	return getPVCDiskImgPath(volumeName, filepath.Base(path))
}

func GetMountedHostDiskDir(volumeName string) string {
	return getPVCDiskImgPath(volumeName, "")
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
			diskPath := GetMountedHostDiskPath(volume.Name, hostDisk.Path)
			diskDir := GetMountedHostDiskDir(volume.Name)
			if _, err := os.Stat(diskPath); os.IsNotExist(err) {
				availableSize, err := hdc.dirBytesAvailableFunc(diskDir)
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
				err = createSparseRaw(diskPath, int64(diskSize))
				if err != nil {
					log.Log.Reason(err).Errorf("Couldn't create a sparse raw file for disk path: %s, error: %v", diskPath, err)
					return err
				}
			} else if err != nil {
				return err
			}
			// Change file ownership to the qemu user.
			if err := ephemeraldiskutils.DefaultOwnershipManager.SetFileOwnership(diskPath); err != nil {
				log.Log.Reason(err).Errorf("Couldn't set Ownership on %s: %v", diskPath, err)
				return err
			}
		}
	}
	return nil
}
