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
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/client-go/apis/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests/flags"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/util"
)

func NodeNameWithHandler() string {
	listOptions := metav1.ListOptions{LabelSelector: v1.AppLabel + "=virt-handler"}
	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
	virtHandlerPods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), listOptions)
	Expect(err).ToNot(HaveOccurred())
	node, err := virtClient.CoreV1().Nodes().Get(context.Background(), virtHandlerPods.Items[0].Spec.NodeName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return node.ObjectMeta.Name
}

func ExecuteCommandInVirtHandlerPod(nodeName string, args []string) (stdout string, err error) {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return stdout, err
	}

	pod, err := kubecli.NewVirtHandlerClient(virtClient).Namespace(flags.KubeVirtInstallNamespace).ForNode(nodeName).Pod()
	if err != nil {
		return stdout, err
	}

	stdout, stderr, err := ExecuteCommandOnPodV2(virtClient, pod, "virt-handler", args)
	if err != nil {
		return stdout, fmt.Errorf("Failed excuting command=%v, error=%v, stdout=%s, stderr=%s", args, err, stdout, stderr)
	}
	return stdout, nil
}

func CreateErrorDisk(nodeName string) (address string, device string) {
	By("Creating error disk")
	args := []string{"/usr/bin/virt-chroot", "--mount", "/proc/1/ns/mnt", "exec", "--", "/usr/sbin/modprobe", "scsi_debug", "opts=2", "every_nth=4"}
	_, err := ExecuteCommandInVirtHandlerPod(nodeName, args)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to create faulty disk")

	args = []string{"/usr/bin/virt-chroot", "--mount", "/proc/1/ns/mnt", "exec", "--", "/usr/bin/lsscsi"}
	stdout, err := ExecuteCommandInVirtHandlerPod(nodeName, args)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to find out address of  faulty disk")

	// Example output
	// [2:0:0:0]    cd/dvd  QEMU     QEMU DVD-ROM     2.5+  /dev/sr0
	// [6:0:0:0]    disk    Linux    scsi_debug       0190  /dev/sda
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		if strings.Contains(line, "scsi_debug") {
			line = strings.TrimSpace(line)
			disk := strings.Split(line, " ")
			address = disk[0]
			address = address[1 : len(address)-1]
			device = disk[len(disk)-1]
			break
		}
	}

	return address, device
}

func RemoveErrorDisk(nodeName, address string) {
	By("Removing error disk")
	args := []string{"/usr/bin/echo", "1", ">", fmt.Sprintf("/proc/1/root/sys/class/scsi_device/%s/device/delete", address)}
	_, err := ExecuteCommandInVirtHandlerPod(nodeName, args)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to disable faulty disk")

	args = []string{"/usr/bin/virt-chroot", "--mount", "/proc/1/ns/mnt", "exec", "--", "/usr/sbin/modprobe", "-r", "scsi_debug"}
	_, err = ExecuteCommandInVirtHandlerPod(nodeName, args)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to disable faulty disk")
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
	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())

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
	pod, err = virtClient.CoreV1().Pods(util.NamespaceTestDefault).Create(context.Background(), pod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	Eventually(ThisPod(pod), 30).Should(HaveSucceeded())
}

func CreateFaultyDisk(nodeName, deviceName string) {
	By(fmt.Sprintf("Creating faulty disk %s on %s node", deviceName, nodeName))
	args := []string{"dmsetup", "create", deviceName, "--table", "0 204791 error"}
	executeDeviceMapperOnNode(nodeName, args)
}

func CreatePVandPVCwithFaultyDisk(nodeName, devicePath, namespace string) (*corev1.PersistentVolume, *corev1.PersistentVolumeClaim, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return nil, nil, err
	}

	size := resource.MustParse("1Gi")
	volumeMode := corev1.PersistentVolumeBlock
	storageClass := "faulty-disks"

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
			GenerateName: "ioerrorpv",
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
	pv, err = virtClient.CoreV1().PersistentVolumes().Create(context.Background(), pv, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, err
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "ioerrorpvc",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeMode:       &volumeMode,
			StorageClassName: &storageClass,
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{corev1.ResourceStorage: size},
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
