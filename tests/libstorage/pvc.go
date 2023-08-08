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

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"

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
	nonRootUser := int64(107)

	// Change to 'pod := RenderPod(name, cmd, args)' once we have a libpod package
	pod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: name,
			Namespace:    pvc.Namespace,
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
					SecurityContext: &k8sv1.SecurityContext{
						Capabilities: &k8sv1.Capabilities{
							Drop: []k8sv1.Capability{"ALL"},
						},
						Privileged:               pointer.Bool(false),
						RunAsUser:                &nonRootUser,
						RunAsNonRoot:             pointer.Bool(true),
						AllowPrivilegeEscalation: pointer.Bool(false),
						SeccompProfile: &k8sv1.SeccompProfile{
							Type: k8sv1.SeccompProfileTypeRuntimeDefault,
						},
					},
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
		if pod.Spec.SecurityContext == nil {
			pod.Spec.SecurityContext = &k8sv1.PodSecurityContext{}
		}
		pod.Spec.SecurityContext.FSGroup = &nonRootUser
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

func createPVC(pvc *k8sv1.PersistentVolumeClaim, namespace string) *k8sv1.PersistentVolumeClaim {
	virtCli := kubevirt.Client()

	createdPvc, err := virtCli.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) {
		util.PanicOnError(err)
	}

	return createdPvc
}

func CreateFSPVC(name, namespace, size string, labels map[string]string) *k8sv1.PersistentVolumeClaim {
	sc, exists := GetRWOFileSystemStorageClass()
	if !exists {
		Skip("Skip test when RWOFileSystem storage class is not present")
	}
	pvc := NewPVC(name, size, sc)
	volumeMode := k8sv1.PersistentVolumeFilesystem
	pvc.Spec.VolumeMode = &volumeMode
	if labels != nil && pvc.Labels == nil {
		pvc.Labels = map[string]string{}
	}

	for key, value := range labels {
		pvc.Labels[key] = value
	}

	return createPVC(pvc, namespace)
}

func CreateBlockPVC(name, namespace, size string) *k8sv1.PersistentVolumeClaim {
	sc, exists := GetRWOBlockStorageClass()
	if !exists {
		Skip("Skip test when RWOBlock storage class is not present")
	}
	pvc := NewPVC(name, size, sc)
	volumeMode := k8sv1.PersistentVolumeBlock
	pvc.Spec.VolumeMode = &volumeMode

	return createPVC(pvc, namespace)
}

func CreateHostPathPVC(os, namespace, size string) {
	sc := "manual"
	CreatePVC(os, namespace, size, sc, false)
}

func CreatePVC(os, namespace, size, storageClass string, recycledPV bool) *k8sv1.PersistentVolumeClaim {
	pvcName := fmt.Sprintf("disk-%s", os)

	selector := map[string]string{
		util.KubevirtIoTest: os,
	}

	// If the PV is not recycled, it will have a namespace related test label which  we should match
	if !recycledPV {
		selector[cleanup.TestLabelForNamespace(namespace)] = ""
	}

	pvc := NewPVC(pvcName, size, storageClass)
	pvc.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: selector,
	}
	return createPVC(pvc, namespace)
}

func DeleteAllSeparateDeviceHostPathPvs() {
	virtClient := kubevirt.Client()

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

func CreateAllSeparateDeviceHostPathPvs(osName, namespace string) {
	CreateStorageClass(StorageClassHostPathSeparateDevice, &wffc)
	virtClient := kubevirt.Client()
	Eventually(func() int {
		nodes := libnode.GetAllSchedulableNodes(virtClient)
		if len(nodes.Items) > 0 {
			for _, node := range nodes.Items {
				createSeparateDeviceHostPathPv(osName, namespace, node.Name)
			}
		}
		return len(nodes.Items)
	}, 5*time.Minute, 10*time.Second).ShouldNot(BeZero(), "no schedulable nodes found")
}

func createSeparateDeviceHostPathPv(osName, namespace, nodeName string) {
	virtCli := kubevirt.Client()
	name := fmt.Sprintf("separate-device-%s-pv", nodeName)
	pv := &k8sv1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", name, namespace),
			Labels: map[string]string{
				util.KubevirtIoTest:                      osName,
				cleanup.TestLabelForNamespace(namespace): "",
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

	_, err := virtCli.CoreV1().PersistentVolumes().Create(context.Background(), pv, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) {
		util.PanicOnError(err)
	}
}

func CreateHostPathPv(osName, namespace, hostPath string) string {
	return createHostPathPvWithSize(osName, namespace, hostPath, "1Gi")
}

func createHostPathPvWithSize(osName, namespace, hostPath, size string) string {
	sc := "manual"
	return CreateHostPathPvWithSizeAndStorageClass(osName, namespace, hostPath, size, sc)
}

func CreateHostPathPvWithSizeAndStorageClass(osName, namespace, hostPath, size, sc string) string {
	virtCli := kubevirt.Client()

	quantity, err := resource.ParseQuantity(size)
	util.PanicOnError(err)

	hostPathType := k8sv1.HostPathDirectoryOrCreate

	name := fmt.Sprintf("%s-disk-for-tests-%s", osName, rand.String(12))
	pv := &k8sv1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", name, namespace),
			Labels: map[string]string{
				util.KubevirtIoTest:                      osName,
				cleanup.TestLabelForNamespace(namespace): "",
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

func DeletePVC(os, namespace string) {
	virtCli := kubevirt.Client()

	name := fmt.Sprintf("disk-%s", os)
	err := virtCli.CoreV1().PersistentVolumeClaims(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if !errors.IsNotFound(err) {
		util.PanicOnError(err)
	}
}

func DeletePV(os string) {
	virtCli := kubevirt.Client()

	name := fmt.Sprintf("%s-disk-for-tests", os)
	err := virtCli.CoreV1().PersistentVolumes().Delete(context.Background(), name, metav1.DeleteOptions{})
	if !errors.IsNotFound(err) {
		util.PanicOnError(err)
	}
}

func CreateNFSPvAndPvc(name string, namespace string, size string, nfsTargetIP string, os string) {
	virtCli := kubevirt.Client()

	_, err := virtCli.CoreV1().PersistentVolumes().Create(context.Background(), newNFSPV(name, namespace, size, nfsTargetIP, os), metav1.CreateOptions{})
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
