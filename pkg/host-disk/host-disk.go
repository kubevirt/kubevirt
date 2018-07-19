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
	"os"
	"os/exec"
	"path"
	"strconv"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/precond"
)

const pvcBaseDir = "/var/run/kubevirt-private/vmi-disks"

func getDiskImgPath(volumeName string) string {
	return path.Join(pvcBaseDir, volumeName, "disk.img")
}

func calculateRawImgSize(quantity resource.Quantity) int64 {
	// TODO: take fs overhead into account
	size, _ := quantity.AsInt64()
	return size
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
	// TODO: add checks:
	// - if there is enough space
	for _, volume := range vmi.Spec.Volumes {
		if hostDisk := volume.VolumeSource.HostDisk; hostDisk.PersistentVolumeClaim != nil {
			size := strconv.FormatInt(calculateRawImgSize(hostDisk.Capacity), 10)
			file := getDiskImgPath(volume.Name)

			if _, err := os.Stat(file); os.IsNotExist(err) {
				if err := exec.Command("qemu-img", "create", "-f", "raw", file, size).Run(); err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
		}
	}
	return nil
}
