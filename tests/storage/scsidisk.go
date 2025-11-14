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

package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/framework/k8s"
	"kubevirt.io/kubevirt/tests/libnode"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The tests using the function CreateErrorDisk need to be run serially as it relies on the kernel scsi_debug module
func CreateErrorDisk(nodeName string) (address string, device string) {
	By("Creating error disk")
	return CreateSCSIDisk(nodeName, []string{"opts=2", "every_nth=4", "dev_size_mb=8"})
}

// CreateSCSIDisk creates a SCSI disk using the scsi_debug module. This function should be used only to check SCSI disk functionalities and not for creating a filesystem or any data. The disk is stored in ram and it isn't suitable for storing large amount of data.
// If a test uses this function, it needs to be run serially. The device is created directly on the node and the addition and removal of the scsi_debug kernel module could create flakiness
func CreateSCSIDisk(nodeName string, opts []string) (address string, device string) {
	args := []string{"/usr/sbin/modprobe", "scsi_debug"}
	args = append(args, opts...)
	_, err := virtChrootExecuteCommandInVirtHandlerPod(nodeName, args)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to create faulty disk")

	EventuallyWithOffset(1, func() error {
		args = []string{"/bin/sh", "-c", "/bin/grep -l scsi_debug /sys/bus/scsi/devices/*/model"}
		stdout, err := virtChrootExecuteCommandInVirtHandlerPod(nodeName, args)
		if err != nil {
			return err
		}

		// Example output
		// /sys/bus/scsi/devices/0:0:0:0/model
		if !filepath.IsAbs(stdout) {
			return fmt.Errorf("Device path extracted from sysfs is not populated: %s", stdout)
		}

		pathname := strings.Split(stdout, "/")
		address = pathname[5]

		args = []string{"/bin/ls", "/sys/bus/scsi/devices/" + address + "/block"}
		stdout, err = virtChrootExecuteCommandInVirtHandlerPod(nodeName, args)
		if err != nil {
			return err
		}
		device = "/dev/" + strings.TrimSpace(stdout)

		return nil
	}, 20*time.Second, 5*time.Second).ShouldNot(HaveOccurred())

	return address, device
}

func RemoveSCSIDisk(nodeName, address string) {
	By("Removing scsi disk")
	args := []string{"/usr/bin/echo", "1", ">", fmt.Sprintf("/proc/1/root/sys/class/scsi_device/%s/device/delete", address)}
	_, err := libnode.ExecuteCommandInVirtHandlerPod(nodeName, args)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to disable scsi disk")

	args = []string{"/usr/sbin/modprobe", "-r", "scsi_debug"}
	_, err = virtChrootExecuteCommandInVirtHandlerPod(nodeName, args)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to disable scsi disk")
}

func FixErrorDevice(nodeName string) {
	args := []string{"/usr/bin/bash", "-c", "echo 0 > /proc/1/root/sys/bus/pseudo/drivers/scsi_debug/opts"}
	stdout, err := libnode.ExecuteCommandInVirtHandlerPod(nodeName, args)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to fix faulty disk, %s", stdout))

	args = []string{"/usr/bin/cat", "/proc/1/root/sys/bus/pseudo/drivers/scsi_debug/opts"}

	By("Checking opts of scsi_debug")
	stdout, err = libnode.ExecuteCommandInVirtHandlerPod(nodeName, args)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to fix faulty disk")
	ExpectWithOffset(1, strings.Contains(stdout, "0x0")).To(BeTrue(), fmt.Sprintf("Failed to fix faulty disk, opts don't contains 0x0, opts: %s", stdout))
	ExpectWithOffset(1, !strings.Contains(stdout, "0x02")).To(BeTrue(), fmt.Sprintf("Failed to fix faulty disk, opts contains 0x02, opts: %s", stdout))

}

func CreatePVandPVCwithFaultyDisk(nodeName, devicePath, namespace string) (*corev1.PersistentVolume, *corev1.PersistentVolumeClaim, error) {
	return CreatePVandPVCwithSCSIDisk(nodeName, devicePath, namespace, "faulty-disks", "ioerrorpvc", "ioerrorpvc")
}

func CreatePVwithSCSIDisk(storageClass, pvName, nodeName, devicePath string) (*corev1.PersistentVolume, error) {
	volumeMode := corev1.PersistentVolumeBlock
	size := resource.MustParse("8Mi")
	affinity := corev1.VolumeNodeAffinity{
		Required: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      k8sv1.LabelHostname,
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{nodeName},
						},
					},
				},
			},
		},
	}
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: pvName,
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity:         map[corev1.ResourceName]resource.Quantity{corev1.ResourceStorage: size},
			StorageClassName: storageClass,
			VolumeMode:       &volumeMode,
			NodeAffinity:     &affinity,
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				Local: &corev1.LocalVolumeSource{
					Path: devicePath,
				},
			},
		},
	}
	return k8s.Client().CoreV1().PersistentVolumes().Create(context.Background(), pv, metav1.CreateOptions{})
}

func CreatePVandPVCwithSCSIDisk(nodeName, devicePath, namespace, storageClass, pvName, pvcName string) (*corev1.PersistentVolume, *corev1.PersistentVolumeClaim, error) {
	pv, err := CreatePVwithSCSIDisk(storageClass, pvName, nodeName, devicePath)
	if err != nil {
		return nil, nil, err
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: pvcName,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeMode:       pv.Spec.VolumeMode,
			StorageClassName: &storageClass,
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{corev1.ResourceStorage: pv.Spec.Capacity["storage"]},
			},
		},
	}

	pvc, err = k8s.Client().CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if err != nil {
		return pv, nil, err
	}

	return pv, pvc, err
}

func virtChrootExecuteCommandInVirtHandlerPod(nodeName string, args []string) (stdout string, err error) {
	args = append([]string{"/usr/bin/virt-chroot", "--mount", "/proc/1/ns/mnt", "exec", "--"}, args...)
	return libnode.ExecuteCommandInVirtHandlerPod(nodeName, args)
}
