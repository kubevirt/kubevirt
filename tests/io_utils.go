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
 * Copyright 2021 Red Hat, Inc.
 *
 */
package tests

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnode"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/flags"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	UsrBinVirtChroot = "/usr/bin/virt-chroot"
	Mount            = "--mount"
	Proc1NsMnt       = "/proc/1/ns/mnt"
)

func NodeNameWithHandler() string {
	listOptions := metav1.ListOptions{LabelSelector: v1.AppLabel + "=virt-handler"}
	virtClient := kubevirt.Client()
	virtHandlerPods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), listOptions)
	Expect(err).ToNot(HaveOccurred())
	node, err := virtClient.CoreV1().Nodes().Get(context.Background(), virtHandlerPods.Items[0].Spec.NodeName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return node.ObjectMeta.Name
}

func ExecuteCommandInVirtHandlerPod(nodeName string, args []string) (stdout string, err error) {
	virtClient := kubevirt.Client()

	pod, err := libnode.GetVirtHandlerPod(virtClient, nodeName)
	if err != nil {
		return stdout, err
	}

	stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, pod, "virt-handler", args)
	if err != nil {
		return stdout, fmt.Errorf("Failed excuting command=%v, error=%v, stdout=%s, stderr=%s", args, err, stdout, stderr)
	}
	return stdout, nil
}

// The tests using the function CreateErrorDisk need to be run serially as it relies on the kernel scsi_debug module
func CreateErrorDisk(nodeName string) (address string, device string) {
	By("Creating error disk")
	return CreateSCSIDisk(nodeName, []string{"opts=2", "every_nth=4", "dev_size_mb=8"})
}

// CreateSCSIDisk creates a SCSI disk using the scsi_debug module. This function should be used only to check SCSI disk functionalities and not for creating a filesystem or any data. The disk is stored in ram and it isn't suitable for storing large amount of data.
// If a test uses this function, it needs to be run serially. The device is created directly on the node and the addition and removal of the scsi_debug kernel module could create flakiness
func CreateSCSIDisk(nodeName string, opts []string) (address string, device string) {
	args := []string{UsrBinVirtChroot, Mount, Proc1NsMnt, "exec", "--", "/usr/sbin/modprobe", "scsi_debug"}
	args = append(args, opts...)
	_, err := ExecuteCommandInVirtHandlerPod(nodeName, args)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to create faulty disk")

	EventuallyWithOffset(1, func() error {
		args = []string{UsrBinVirtChroot, Mount, Proc1NsMnt, "exec", "--", "/bin/sh", "-c", "/bin/grep -l scsi_debug /sys/bus/scsi/devices/*/model"}
		stdout, err := ExecuteCommandInVirtHandlerPod(nodeName, args)
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

		args = []string{UsrBinVirtChroot, Mount, Proc1NsMnt, "exec", "--", "/bin/ls", "/sys/bus/scsi/devices/" + address + "/block"}
		stdout, err = ExecuteCommandInVirtHandlerPod(nodeName, args)
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
	_, err := ExecuteCommandInVirtHandlerPod(nodeName, args)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to disable scsi disk")

	args = []string{UsrBinVirtChroot, Mount, Proc1NsMnt, "exec", "--", "/usr/sbin/modprobe", "-r", "scsi_debug"}
	_, err = ExecuteCommandInVirtHandlerPod(nodeName, args)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to disable scsi disk")
}

func FixErrorDevice(nodeName string) {
	args := []string{"/usr/bin/bash", "-c", "echo 0 > /proc/1/root/sys/bus/pseudo/drivers/scsi_debug/opts"}
	stdout, err := ExecuteCommandInVirtHandlerPod(nodeName, args)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to fix faulty disk, %s", stdout))

	args = []string{"/usr/bin/cat", "/proc/1/root/sys/bus/pseudo/drivers/scsi_debug/opts"}

	By("Checking opts of scsi_debug")
	stdout, err = ExecuteCommandInVirtHandlerPod(nodeName, args)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to fix faulty disk")
	ExpectWithOffset(1, strings.Contains(stdout, "0x0")).To(BeTrue(), fmt.Sprintf("Failed to fix faulty disk, opts don't contains 0x0, opts: %s", stdout))
	ExpectWithOffset(1, !strings.Contains(stdout, "0x02")).To(BeTrue(), fmt.Sprintf("Failed to fix faulty disk, opts contains 0x02, opts: %s", stdout))

}

func executeDeviceMapperOnNode(nodeName string, cmd []string) {
	virtClient := kubevirt.Client()

	// Image that happens to have dmsetup
	image := fmt.Sprintf("%s/vm-killer:%s", flags.KubeVirtRepoPrefix, flags.KubeVirtVersionTag)
	pod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "device-mapper-pod-",
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy: k8sv1.RestartPolicyNever,
			Containers: []k8sv1.Container{
				{
					Name:    "launcher",
					Image:   image,
					Command: cmd,
					SecurityContext: &k8sv1.SecurityContext{
						Privileged: pointer.BoolPtr(true),
						RunAsUser:  pointer.Int64Ptr(0),
					},
				},
			},
			NodeSelector: map[string]string{
				"kubernetes.io/hostname": nodeName,
			},
		},
	}
	pod, err := virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Create(context.Background(), pod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	Eventually(ThisPod(pod), 30).Should(HaveSucceeded())
}

func CreateFaultyDisk(nodeName, deviceName string) {
	By(fmt.Sprintf("Creating faulty disk %s on %s node", deviceName, nodeName))
	args := []string{"dmsetup", "create", deviceName, "--table", "0 204791 error"}
	executeDeviceMapperOnNode(nodeName, args)
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
							Key:      "kubernetes.io/hostname",
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
	return kubevirt.Client().CoreV1().PersistentVolumes().Create(context.Background(), pv, metav1.CreateOptions{})
}

func CreatePVandPVCwithSCSIDisk(nodeName, devicePath, namespace, storageClass, pvName, pvcName string) (*corev1.PersistentVolume, *corev1.PersistentVolumeClaim, error) {
	virtClient := kubevirt.Client()

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
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{corev1.ResourceStorage: pv.Spec.Capacity["storage"]},
			},
		},
	}

	pvc, err = virtClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if err != nil {
		return pv, nil, err
	}

	return pv, pvc, err
}

func RemoveFaultyDisk(nodeName, deviceName string) {
	By(fmt.Sprintf("Removing faulty disk %s on %s node", deviceName, nodeName))
	args := []string{"dmsetup", "remove", deviceName}
	executeDeviceMapperOnNode(nodeName, args)
}
