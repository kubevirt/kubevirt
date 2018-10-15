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

	"kubevirt.io/kubevirt/pkg/util/types"

	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

const pvcBaseDir = "/var/run/kubevirt-private/vmi-disks"

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

			volumeSource.HostDisk = &v1.HostDisk{
				Path:     getPVCDiskImgPath(vmi.Spec.Volumes[i].Name),
				Type:     v1.HostDiskExistsOrCreate,
				Capacity: pvc.Status.Capacity[k8sv1.ResourceStorage],
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
	return (stat.Bavail * uint64(stat.Bsize)), nil
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

func CreateHostDisks(vmi *v1.VirtualMachineInstance) error {
	for _, volume := range vmi.Spec.Volumes {
		if hostDisk := volume.VolumeSource.HostDisk; hostDisk != nil && hostDisk.Type == v1.HostDiskExistsOrCreate && hostDisk.Path != "" {

			if _, err := os.Stat(hostDisk.Path); os.IsNotExist(err) {
				availableSpace, err := dirBytesAvailable(path.Dir(hostDisk.Path))
				if err != nil {
					return err
				}
				size, _ := hostDisk.Capacity.AsInt64()
				if uint64(size) > availableSpace {
					return fmt.Errorf("Unable to create %s with size %s - not enough space on the cluster", hostDisk.Path, hostDisk.Capacity.String())
				}
				err = createSparseRaw(hostDisk.Path, size)
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
