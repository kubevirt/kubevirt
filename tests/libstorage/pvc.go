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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package libstorage

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/util/net/ip"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/cleanup"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/util"
)

const (
	DefaultPvcMountPath                = "/pvc"
	StorageClassHostPathSeparateDevice = "host-path-sd"
)

func RenderPodWithPVC(name string, cmd []string, args []string, pvc *k8sv1.PersistentVolumeClaim) *k8sv1.Pod {
	volumeName := "disk0"
	// Change to 'pod := RenderPod(name, cmd, args)' once we have a libpod package
	pod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: name,
			Namespace:    util.NamespaceTestDefault,
			Labels: map[string]string{
				v1.AppLabel: "test",
			},
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy: k8sv1.RestartPolicyNever,
			Containers: []k8sv1.Container{
				{
					Name:    name,
					Image:   fmt.Sprintf("%s/vm-killer:%s", flags.KubeVirtUtilityRepoPrefix, flags.KubeVirtUtilityVersionTag),
					Command: cmd,
					Args:    args,
				},
			},
			Volumes: []k8sv1.Volume{
				{
					Name: volumeName,
					VolumeSource: k8sv1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvc.GetName(),
						},
					},
				},
			},
		},
	}

	volumeMode := pvc.Spec.VolumeMode
	if volumeMode != nil && *volumeMode == k8sv1.PersistentVolumeBlock {
		pod.Spec.Containers[0].VolumeDevices = addVolumeDevices(volumeName)
	} else {
		pod.Spec.Containers[0].VolumeMounts = addVolumeMounts(volumeName)
	}

	return pod
}

// this is being called for pods using PV with block volume mode
func addVolumeDevices(volumeName string) []k8sv1.VolumeDevice {
	volumeDevices := []k8sv1.VolumeDevice{
		{
			Name:       volumeName,
			DevicePath: DefaultPvcMountPath,
		},
	}
	return volumeDevices
}

// this is being called for pods using PV with filesystem volume mode
func addVolumeMounts(volumeName string) []k8sv1.VolumeMount {
	volumeMounts := []k8sv1.VolumeMount{
		{
			Name:      volumeName,
			MountPath: DefaultPvcMountPath,
		},
	}
	return volumeMounts
}

func NewPVC(name, size, storageClass string) *k8sv1.PersistentVolumeClaim {
	quantity, err := resource.ParseQuantity(size)
	util.PanicOnError(err)

	return &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
			Resources: k8sv1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					"storage": quantity,
				},
			},
			StorageClassName: &storageClass,
		},
	}
}

func createPVC(pvc *k8sv1.PersistentVolumeClaim) *k8sv1.PersistentVolumeClaim {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	createdPvc, err := virtCli.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Create(context.Background(), pvc, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) {
		util.PanicOnError(err)
	}

	return createdPvc
}

func CreateFSPVC(name, size string) *k8sv1.PersistentVolumeClaim {
	sc, exists := GetRWOFileSystemStorageClass()
	if !exists {
		Skip("Skip test when RWOFileSystem storage class is not present")
	}
	pvc := NewPVC(name, size, sc)
	volumeMode := k8sv1.PersistentVolumeFilesystem
	pvc.Spec.VolumeMode = &volumeMode

	return createPVC(pvc)
}

func CreateBlockPVC(name, size string) *k8sv1.PersistentVolumeClaim {
	sc, exists := GetRWOBlockStorageClass()
	if !exists {
		Skip("Skip test when RWOBlock storage class is not present")
	}
	pvc := NewPVC(name, size, sc)
	volumeMode := k8sv1.PersistentVolumeBlock
	pvc.Spec.VolumeMode = &volumeMode

	return createPVC(pvc)
}

func CreateHostPathPVC(os, size string) {
	sc := "manual"
	CreatePVC(os, size, sc, false)
}

func CreatePVC(os, size, storageClass string, recycledPV bool) *k8sv1.PersistentVolumeClaim {
	pvcName := fmt.Sprintf("disk-%s", os)

	selector := map[string]string{
		util.KubevirtIoTest: os,
	}

	// If the PV is not recycled, it will have a namespace related test label which  we should match
	if !recycledPV {
		selector[cleanup.TestLabelForNamespace(util.NamespaceTestDefault)] = ""
	}

	pvc := NewPVC(pvcName, size, storageClass)
	pvc.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: selector,
	}
	return createPVC(pvc)
}

func DeleteAllSeparateDeviceHostPathPvs() {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	pvList, err := virtClient.CoreV1().PersistentVolumes().List(context.Background(), metav1.ListOptions{})
	util.PanicOnError(err)
	for _, pv := range pvList.Items {
		if pv.Spec.StorageClassName == StorageClassHostPathSeparateDevice {
			// ignore error we want to attempt to delete them all.
			_ = virtClient.CoreV1().PersistentVolumes().Delete(context.Background(), pv.Name, metav1.DeleteOptions{})
		}
	}

	DeleteStorageClass(StorageClassHostPathSeparateDevice)
}

func CreateAllSeparateDeviceHostPathPvs(osName string) {
	CreateStorageClass(StorageClassHostPathSeparateDevice, &wffc)
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	Eventually(func() int {
		nodes := libnode.GetAllSchedulableNodes(virtClient)
		if len(nodes.Items) > 0 {
			for _, node := range nodes.Items {
				createSeparateDeviceHostPathPv(osName, node.Name)
			}
		}
		return len(nodes.Items)
	}, 5*time.Minute, 10*time.Second).ShouldNot(BeZero(), "no schedulable nodes found")
}

func createSeparateDeviceHostPathPv(osName, nodeName string) {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	name := fmt.Sprintf("separate-device-%s-pv", nodeName)
	pv := &k8sv1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", name, util.NamespaceTestDefault),
			Labels: map[string]string{
				util.KubevirtIoTest: osName,
				cleanup.TestLabelForNamespace(util.NamespaceTestDefault): "",
			},
		},
		Spec: k8sv1.PersistentVolumeSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
			Capacity: k8sv1.ResourceList{
				"storage": resource.MustParse("3Gi"),
			},
			PersistentVolumeReclaimPolicy: k8sv1.PersistentVolumeReclaimRetain,
			PersistentVolumeSource: k8sv1.PersistentVolumeSource{
				HostPath: &k8sv1.HostPathVolumeSource{
					Path: "/tmp/hostImages/mount_hp/test",
				},
			},
			StorageClassName: StorageClassHostPathSeparateDevice,
			NodeAffinity: &k8sv1.VolumeNodeAffinity{
				Required: &k8sv1.NodeSelector{
					NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sv1.NodeSelectorRequirement{
								{
									Key:      util.KubernetesIoHostName,
									Operator: k8sv1.NodeSelectorOpIn,
									Values:   []string{nodeName},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = virtCli.CoreV1().PersistentVolumes().Create(context.Background(), pv, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) {
		util.PanicOnError(err)
	}
}

func CreateHostPathPv(osName, hostPath string) string {
	return createHostPathPvWithSize(osName, hostPath, "1Gi")
}

func createHostPathPvWithSize(osName, hostPath, size string) string {
	sc := "manual"
	return CreateHostPathPvWithSizeAndStorageClass(osName, hostPath, size, sc)
}

func CreateHostPathPvWithSizeAndStorageClass(osName, hostPath, size, sc string) string {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	quantity, err := resource.ParseQuantity(size)
	util.PanicOnError(err)

	hostPathType := k8sv1.HostPathDirectoryOrCreate

	name := fmt.Sprintf("%s-disk-for-tests-%s", osName, rand.String(12))
	pv := &k8sv1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", name, util.NamespaceTestDefault),
			Labels: map[string]string{
				util.KubevirtIoTest: osName,
				cleanup.TestLabelForNamespace(util.NamespaceTestDefault): "",
			},
		},
		Spec: k8sv1.PersistentVolumeSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
			Capacity: k8sv1.ResourceList{
				"storage": quantity,
			},
			PersistentVolumeReclaimPolicy: k8sv1.PersistentVolumeReclaimRetain,
			PersistentVolumeSource: k8sv1.PersistentVolumeSource{
				HostPath: &k8sv1.HostPathVolumeSource{
					Path: hostPath,
					Type: &hostPathType,
				},
			},
			StorageClassName: sc,
			NodeAffinity: &k8sv1.VolumeNodeAffinity{
				Required: &k8sv1.NodeSelector{
					NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sv1.NodeSelectorRequirement{
								{
									Key:      util.KubernetesIoHostName,
									Operator: k8sv1.NodeSelectorOpIn,
									Values:   []string{libnode.SchedulableNode},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = virtCli.CoreV1().PersistentVolumes().Create(context.Background(), pv, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) {
		util.PanicOnError(err)
	}
	return libnode.SchedulableNode
}

func DeletePVC(os string) {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	name := fmt.Sprintf("disk-%s", os)
	err = virtCli.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Delete(context.Background(), name, metav1.DeleteOptions{})
	if !errors.IsNotFound(err) {
		util.PanicOnError(err)
	}
}

func DeletePV(os string) {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	name := fmt.Sprintf("%s-disk-for-tests", os)
	err = virtCli.CoreV1().PersistentVolumes().Delete(context.Background(), name, metav1.DeleteOptions{})
	if !errors.IsNotFound(err) {
		util.PanicOnError(err)
	}
}

func CreateNFSPvAndPvc(name string, namespace string, size string, nfsTargetIP string, os string) {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	_, err = virtCli.CoreV1().PersistentVolumes().Create(context.Background(), newNFSPV(name, namespace, size, nfsTargetIP, os), metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) {
		util.PanicOnError(err)
	}

	_, err = virtCli.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), newNFSPVC(name, namespace, size, os), metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) {
		util.PanicOnError(err)
	}
}

func newNFSPV(name string, namespace string, size string, nfsTargetIP string, os string) *k8sv1.PersistentVolume {
	quantity := resource.MustParse(size)

	storageClass, exists := GetRWOFileSystemStorageClass()
	if !exists {
		Skip("Skip test when Filesystem storage is not present")
	}
	volumeMode := k8sv1.PersistentVolumeFilesystem

	nfsTargetIP = ip.NormalizeIPAddress(nfsTargetIP)

	return &k8sv1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				util.KubevirtIoTest:                      os,
				cleanup.TestLabelForNamespace(namespace): "",
			},
		},
		Spec: k8sv1.PersistentVolumeSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteMany},
			Capacity: k8sv1.ResourceList{
				"storage": quantity,
			},
			StorageClassName: storageClass,
			VolumeMode:       &volumeMode,
			PersistentVolumeSource: k8sv1.PersistentVolumeSource{
				NFS: &k8sv1.NFSVolumeSource{
					Server: nfsTargetIP,
					Path:   "/",
				},
			},
		},
	}
}

func newNFSPVC(name string, namespace string, size string, os string) *k8sv1.PersistentVolumeClaim {
	quantity, err := resource.ParseQuantity(size)
	util.PanicOnError(err)

	storageClass, exists := GetRWOFileSystemStorageClass()
	if !exists {
		Skip("Skip test when Filesystem storage is not present")
	}
	volumeMode := k8sv1.PersistentVolumeFilesystem

	return &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteMany},
			Resources: k8sv1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					"storage": quantity,
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					util.KubevirtIoTest:                      os,
					cleanup.TestLabelForNamespace(namespace): "",
				},
			},
			StorageClassName: &storageClass,
			VolumeMode:       &volumeMode,
		},
	}
}
