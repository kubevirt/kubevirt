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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package backendstorage

import (
	"context"
	"fmt"
	"sort"

	"kubevirt.io/kubevirt/pkg/storage/types"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	corev1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	PVCPrefix = "persistent-state-for-"
	PVCSize   = "10Mi"

	BlockVolumeDevicePath = "/dev/vm-state"
	PodVMStatePath        = "/var/lib/libvirt/vm-state"
	PodNVRAMPath          = "/var/lib/libvirt/qemu/nvram"
	PodSwtpmPath          = "/var/lib/libvirt/swtpm"
	PodSwtpmLocalcaPath   = "/var/lib/swtpm-localca"
)

func PVCForVMI(vmi *corev1.VirtualMachineInstance) string {
	return PVCPrefix + vmi.Name
}

func HasPersistentTPMDevice(vmiSpec *corev1.VirtualMachineInstanceSpec) bool {
	return vmiSpec.Domain.Devices.TPM != nil &&
		vmiSpec.Domain.Devices.TPM.Persistent != nil &&
		*vmiSpec.Domain.Devices.TPM.Persistent
}

func HasPersistentEFI(vmiSpec *corev1.VirtualMachineInstanceSpec) bool {
	return vmiSpec.Domain.Firmware != nil &&
		vmiSpec.Domain.Firmware.Bootloader != nil &&
		vmiSpec.Domain.Firmware.Bootloader.EFI != nil &&
		vmiSpec.Domain.Firmware.Bootloader.EFI.Persistent != nil &&
		*vmiSpec.Domain.Firmware.Bootloader.EFI.Persistent
}

func IsBackendStorageNeededForVMI(vmiSpec *corev1.VirtualMachineInstanceSpec) bool {
	return HasPersistentTPMDevice(vmiSpec) || HasPersistentEFI(vmiSpec)
}

func IsBackendStorageNeededForVM(vm *corev1.VirtualMachine) bool {
	if vm.Spec.Template == nil {
		return false
	}
	return HasPersistentTPMDevice(&vm.Spec.Template.Spec)
}

func CreateIfNeeded(vmi *corev1.VirtualMachineInstance, clusterConfig *virtconfig.ClusterConfig, client kubecli.KubevirtClient, pvcStore cache.Store) error {
	if !IsBackendStorageNeededForVMI(&vmi.Spec) {
		return nil
	}

	_, err := client.CoreV1().PersistentVolumeClaims(vmi.Namespace).Get(context.Background(), PVCForVMI(vmi), metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	storageClass, volumeMode, err := DetermineStorageClassAndVolumeMode(vmi, clusterConfig, pvcStore)
	if err != nil {
		return fmt.Errorf("failed to determine the backend storage class and volume mode: %w", err)
	}
	ownerReferences := vmi.OwnerReferences
	if len(vmi.OwnerReferences) == 0 {
		// If the VMI has no owner, then it did not originate from a VM.
		// In that case, we tie the PVC to the VMI, rendering it quite useless since it wont actually persist.
		// The alternative is to remove this `if` block, allowing the PVC to persist after the VMI is deleted.
		// However, that would pose security and littering concerns.
		ownerReferences = []metav1.OwnerReference{
			*metav1.NewControllerRef(vmi, corev1.VirtualMachineInstanceGroupVersionKind),
		}
	}
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:            PVCForVMI(vmi),
			OwnerReferences: ownerReferences,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{v1.ResourceStorage: resource.MustParse(PVCSize)},
			},
			StorageClassName: storageClass,
			VolumeMode:       volumeMode,
		},
	}

	_, err = client.CoreV1().PersistentVolumeClaims(vmi.Namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}

	return err
}

func PodVolumeName(vmiName string) string {
	return vmiName + "-state"
}

func DetermineStorageClassAndVolumeMode(vmi *corev1.VirtualMachineInstance, clusterConfig *virtconfig.ClusterConfig, pvcStore cache.Store) (*string, *v1.PersistentVolumeMode, error) {
	storageClass := clusterConfig.GetVMStateStorageClass()
	volumeMode := clusterConfig.GetVMStateVolumeMode()
	if len(storageClass) != 0 && volumeMode != nil {
		return &storageClass, volumeMode, nil
	}
	disksCopy := make([]corev1.Disk, len(vmi.Spec.Domain.Devices.Disks))
	for i, d := range vmi.Spec.Domain.Devices.Disks {
		d.DeepCopyInto(&disksCopy[i])
	}
	sort.Slice(disksCopy, func(i, j int) bool {
		a, b := disksCopy[i], disksCopy[j]
		if a.BootOrder != nil && b.BootOrder != nil {
			return *a.BootOrder < *b.BootOrder
		}
		return a.BootOrder != nil
	})
	for _, d := range disksCopy {
		if d.Disk == nil {
			continue
		}
		for _, v := range vmi.Spec.Volumes {
			if v.Name != d.Name {
				continue
			}
			var pvcName string
			if v.DataVolume != nil {
				pvcName = v.DataVolume.Name
			} else if v.PersistentVolumeClaim != nil {
				pvcName = v.PersistentVolumeClaim.ClaimName
			}
			if len(pvcName) > 0 {
				pvc, exists, _, err := types.IsPVCBlockFromStore(pvcStore, vmi.Namespace, pvcName)
				if err != nil {
					return nil, nil, err
				}
				if !exists {
					return nil, nil, fmt.Errorf("could not find the PVC %s", pvcName)
				}
				s, v := determineStorageClassAndVolumeModeFromPVC(storageClass, volumeMode, pvc)
				return s, v, nil
			}
		}
	}
	// If we cannot detect the storage class and volume mode, we will just
	// return NILs, and Kubernetes will use the default configuration in the
	// cluster to create the backend PVC.
	return nil, nil, nil
}

// determineStorageClassAndVolumeModeFromPVC returns the final storage class and
// the volume mode to be used, based on the provided default configuration and
// the configuration of the VM main disk PVC.
//
// The webhook will block the case where volume mode is provided but the
// storage class is not. Therefore, in the following we only care about the
// case where 1) storage class is provided but volume mode is not, and 2) both
// storage class and volume mode are not provided.
func determineStorageClassAndVolumeModeFromPVC(defaultStorageClass string, defaultVolumeMode *v1.PersistentVolumeMode, pvc *v1.PersistentVolumeClaim) (*string, *v1.PersistentVolumeMode) {
	// If no default storage class was provided, we use the storage class and the
	// volume mode from the PVC.
	if len(defaultStorageClass) == 0 {
		return pvc.Spec.StorageClassName, pvc.Spec.VolumeMode
	}
	// If no default volume mode was provided, we use the volume mode from the PVC,
	// ONLY when the PVC storage class matches the default storage class.
	if defaultVolumeMode == nil && (pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName == defaultStorageClass) {
		return &defaultStorageClass, pvc.Spec.VolumeMode
	}
	// In all the other cases, we just use the default configuration provided by
	// the user.
	return &defaultStorageClass, defaultVolumeMode
}
