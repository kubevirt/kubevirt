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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests/flags"
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

func ExecuteCommandInVirtHandlerPod(nodeName string, args []string) error {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return err
	}

	pod, err := kubecli.NewVirtHandlerClient(virtClient).Namespace(flags.KubeVirtInstallNamespace).ForNode(nodeName).Pod()
	if err != nil {
		return err
	}

	stdout, stderr, err := ExecuteCommandOnPodV2(virtClient, pod, "virt-handler", args)
	if err != nil {
		return fmt.Errorf("Failed excuting command=%v, error=%v, stdout=%s, stderr=%s", args, err, stdout, stderr)
	}
	return nil
}

func CreateFaultyDisk(nodeName, deviceName string) {
	By(fmt.Sprintf("Creating faulty disk %s on %s node", deviceName, nodeName))
	args := []string{"dmsetup", "create", deviceName, "--table", "0 204791 error"}
	err := ExecuteCommandInVirtHandlerPod(nodeName, args)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to create faulty disk")
}

func CreatePVandPVCwithFaultyDisk(nodeName, deviceName, namespace string) (*corev1.PersistentVolume, *corev1.PersistentVolumeClaim, error) {
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
					Path: "/dev/mapper/" + deviceName,
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
	EventuallyWithOffset(1, func() error {
		return ExecuteCommandInVirtHandlerPod(nodeName, args)
	}, 30*time.Second, 5*time.Second).ShouldNot(HaveOccurred(), "Failed to remove faulty disk")
}
