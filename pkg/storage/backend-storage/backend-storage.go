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

	"kubevirt.io/client-go/log"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"k8s.io/apimachinery/pkg/api/errors"

	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"

	"kubevirt.io/kubevirt/pkg/storage/types"

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

type BackendStorage struct {
	client        kubecli.KubevirtClient
	clusterConfig *virtconfig.ClusterConfig
	scStore       cache.Store
	spStore       cache.Store
	pvcIndexer    cache.Indexer
}

func NewBackendStorage(client kubecli.KubevirtClient, clusterConfig *virtconfig.ClusterConfig, scStore cache.Store, spStore cache.Store, pvcIndexer cache.Indexer) *BackendStorage {
	return &BackendStorage{
		client:        client,
		clusterConfig: clusterConfig,
		scStore:       scStore,
		spStore:       spStore,
		pvcIndexer:    pvcIndexer,
	}
}

func (bs *BackendStorage) getStorageClass() (string, error) {
	storageClass := bs.clusterConfig.GetVMStateStorageClass()
	if storageClass != "" {
		return storageClass, nil
	}

	for _, obj := range bs.scStore.List() {
		sc := obj.(*storagev1.StorageClass)
		if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
			return sc.Name, nil
		}
	}

	return "", fmt.Errorf("no default storage class found")
}

func (bs *BackendStorage) getAccessMode(storageClass string, mode v1.PersistentVolumeMode) v1.PersistentVolumeAccessMode {
	// The default access mode should be RWX if the storage class was manually specified.
	// However, if we're using the cluster default storage class, default to access mode RWO.
	accessMode := v1.ReadWriteMany
	if bs.clusterConfig.GetVMStateStorageClass() == "" {
		accessMode = v1.ReadWriteOnce
	}

	// Storage profiles are guaranteed to have the same name as their storage class
	obj, exists, err := bs.spStore.GetByKey(storageClass)
	if err != nil {
		log.Log.Reason(err).Infof("couldn't access storage profiles, defaulting to %s", accessMode)
		return accessMode
	}
	if !exists {
		log.Log.Infof("no storage profile found for %s, defaulting to %s", storageClass, accessMode)
		return accessMode
	}
	storageProfile := obj.(*v1beta1.StorageProfile)

	if storageProfile.Status.ClaimPropertySets == nil || len(storageProfile.Status.ClaimPropertySets) == 0 {
		log.Log.Infof("no ClaimPropertySets in storage profile %s, defaulting to %s", storageProfile.Name, accessMode)
		return accessMode
	}

	foundrwo := false
	for _, property := range storageProfile.Status.ClaimPropertySets {
		if property.VolumeMode == nil || *property.VolumeMode != mode || property.AccessModes == nil {
			continue
		}
		for _, accessMode := range property.AccessModes {
			switch accessMode {
			case v1.ReadWriteMany:
				return v1.ReadWriteMany
			case v1.ReadWriteOnce:
				foundrwo = true
			}
		}
	}
	if foundrwo {
		return v1.ReadWriteOnce
	}

	return accessMode
}

func updateVolumeStatus(vmi *corev1.VirtualMachineInstance, accessMode v1.PersistentVolumeAccessMode) {
	if vmi.Status.VolumeStatus == nil {
		vmi.Status.VolumeStatus = []corev1.VolumeStatus{}
	}
	name := PVCForVMI(vmi)
	for i := range vmi.Status.VolumeStatus {
		if vmi.Status.VolumeStatus[i].Name == name {
			if vmi.Status.VolumeStatus[i].PersistentVolumeClaimInfo == nil {
				vmi.Status.VolumeStatus[i].PersistentVolumeClaimInfo = &corev1.PersistentVolumeClaimInfo{}
			}
			vmi.Status.VolumeStatus[i].PersistentVolumeClaimInfo.ClaimName = name
			vmi.Status.VolumeStatus[i].PersistentVolumeClaimInfo.AccessModes = []v1.PersistentVolumeAccessMode{accessMode}
			return
		}
	}
	vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, corev1.VolumeStatus{
		Name: name,
		PersistentVolumeClaimInfo: &corev1.PersistentVolumeClaimInfo{
			ClaimName:   name,
			AccessModes: []v1.PersistentVolumeAccessMode{accessMode},
		},
	})
}

func (bs *BackendStorage) CreateIfNeededAndUpdateVolumeStatus(vmi *corev1.VirtualMachineInstance) error {
	if !IsBackendStorageNeededForVMI(&vmi.Spec) {
		return nil
	}

	obj, exists, err := bs.pvcIndexer.GetByKey(vmi.Namespace + "/" + PVCForVMI(vmi))
	if err != nil {
		return err
	}

	if exists {
		pvc := obj.(*v1.PersistentVolumeClaim)
		updateVolumeStatus(vmi, pvc.Spec.AccessModes[0])
		return nil
	}

	storageClass, volumeMode, err := bs.DetermineStorageClassAndVolumeMode(vmi)
	if err != nil {
		return fmt.Errorf("failed to determine the backend storage class and volume mode: %w", err)
	}

	accessMode := bs.getAccessMode(*storageClass, *volumeMode)
	ownerReferences := vmi.OwnerReferences
	if len(vmi.OwnerReferences) == 0 {
		// If the VMI has no owner, then it did not originate from a VM.
		// In that case, we tie the PVC to the VMI, rendering it quite useless since it won't actually persist.
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
			AccessModes: []v1.PersistentVolumeAccessMode{accessMode},
			Resources: v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{v1.ResourceStorage: resource.MustParse(PVCSize)},
			},
			StorageClassName: storageClass,
			VolumeMode:       volumeMode,
		},
	}

	updateVolumeStatus(vmi, accessMode)

	pvc, err = bs.client.CoreV1().PersistentVolumeClaims(vmi.Namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}

	return err
}

// IsPVCReady returns true if either:
// - No PVC is needed for the VMI since it doesn't use backend storage
// - The backend storage PVC is bound
// - The backend storage PVC is pending uses a WaitForFirstConsumer storage class
func (bs *BackendStorage) IsPVCReady(vmi *corev1.VirtualMachineInstance) (bool, error) {
	if !IsBackendStorageNeededForVMI(&vmi.Spec) {
		return true, nil
	}

	pvc, err := bs.client.CoreV1().PersistentVolumeClaims(vmi.Namespace).Get(context.Background(), PVCForVMI(vmi), metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	switch pvc.Status.Phase {
	case v1.ClaimBound:
		return true, nil
	case v1.ClaimLost:
		return false, fmt.Errorf("backend storage PVC lost")
	case v1.ClaimPending:
		if pvc.Spec.StorageClassName == nil {
			return false, fmt.Errorf("no storage class name")
		}
		obj, exists, err := bs.scStore.GetByKey(*pvc.Spec.StorageClassName)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, fmt.Errorf("storage class %s not found", *pvc.Spec.StorageClassName)
		}
		sc := obj.(*storagev1.StorageClass)
		if sc.VolumeBindingMode != nil && *sc.VolumeBindingMode == storagev1.VolumeBindingWaitForFirstConsumer {
			return true, nil
		}
	}

	return false, nil
}

func PodVolumeName(vmiName string) string {
	return vmiName + "-state"
}

func (bs *BackendStorage) DetermineStorageClassAndVolumeMode(vmi *corev1.VirtualMachineInstance) (*string, *v1.PersistentVolumeMode, error) {
	storageClass, err := bs.getStorageClass()
	if err != nil {
		return nil, nil, err
	}

	volumeMode := bs.clusterConfig.GetVMStateVolumeMode()
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
				pvc, exists, _, err := types.IsPVCBlockFromIndexer(bs.pvcIndexer, vmi.Namespace, pvcName)
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
