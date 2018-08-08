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
	"errors"
	"fmt"
	"os"
	"path"
	"syscall"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/precond"
)

const pvcBaseDir = "/var/run/kubevirt-private/vmi-disks"

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

func GetPVCDiskImgPath(volumeName string) string {
	return path.Join(pvcBaseDir, volumeName, "disk.img")
}

func GetPVCSize(pvcName string, namespace string, clientset kubecli.KubevirtClient) (resource.Quantity, error) {
	precond.CheckNotNil(pvcName)
	precond.CheckNotEmpty(namespace)
	precond.CheckNotNil(clientset)

	pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(pvcName, metav1.GetOptions{})
	if err != nil {
		return resource.Quantity{}, err
	}

	return pvc.Status.Capacity[k8sv1.ResourceStorage], nil
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
					return errors.New(fmt.Sprintf("Unable to create %s with size %s - not enough space on the cluster", hostDisk.Path, hostDisk.Capacity.String()))
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
