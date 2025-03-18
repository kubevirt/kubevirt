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

	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"k8s.io/client-go/tools/cache"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	"k8s.io/apimachinery/pkg/api/errors"

	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

const (
	PVCPrefix = "persistent-state-for-"
	PVCSize   = "10Mi"
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
	return HasPersistentTPMDevice(&vm.Spec.Template.Spec) || HasPersistentEFI(&vm.Spec.Template.Spec)
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

	k8sDefault := ""
	kvDefault := ""
	for _, obj := range bs.scStore.List() {
		sc := obj.(*storagev1.StorageClass)
		if sc.Annotations["storageclass.kubevirt.io/is-default-virt-class"] == "true" {
			kvDefault = sc.Name
		}
		if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
			k8sDefault = sc.Name
		}
	}

	if kvDefault != "" {
		return kvDefault, nil
	} else if k8sDefault != "" {
		return k8sDefault, nil
	} else {
		return "", fmt.Errorf("no default storage class found")
	}
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
	storageProfile := obj.(*cdiv1.StorageProfile)

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

func (bs *BackendStorage) UpdateVolumeStatus(vmi *corev1.VirtualMachineInstance) {
	if !IsBackendStorageNeededForVMI(&vmi.Spec) {
		return
	}
	name := PVCForVMI(vmi)
	obj, exists, err := bs.pvcIndexer.GetByKey(vmi.Namespace + "/" + name)
	if err != nil || !exists {
		return
	}
	if vmi.Status.VolumeStatus == nil {
		vmi.Status.VolumeStatus = []corev1.VolumeStatus{}
	}
	pvc := obj.(*v1.PersistentVolumeClaim)
	for i := range vmi.Status.VolumeStatus {
		if vmi.Status.VolumeStatus[i].Name == name {
			if vmi.Status.VolumeStatus[i].PersistentVolumeClaimInfo == nil {
				vmi.Status.VolumeStatus[i].PersistentVolumeClaimInfo = &corev1.PersistentVolumeClaimInfo{}
			}
			vmi.Status.VolumeStatus[i].PersistentVolumeClaimInfo.ClaimName = name
			vmi.Status.VolumeStatus[i].PersistentVolumeClaimInfo.AccessModes = []v1.PersistentVolumeAccessMode{pvc.Spec.AccessModes[0]}
			return
		}
	}
	vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, corev1.VolumeStatus{
		Name: name,
		PersistentVolumeClaimInfo: &corev1.PersistentVolumeClaimInfo{
			ClaimName:   name,
			AccessModes: []v1.PersistentVolumeAccessMode{pvc.Spec.AccessModes[0]},
		},
	})
}

func (bs *BackendStorage) CreateIfNeeded(vmi *corev1.VirtualMachineInstance) error {
	if !IsBackendStorageNeededForVMI(&vmi.Spec) {
		return nil
	}

	_, exists, err := bs.pvcIndexer.GetByKey(vmi.Namespace + "/" + PVCForVMI(vmi))
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	storageClass, err := bs.getStorageClass()
	if err != nil {
		return err
	}
	mode := v1.PersistentVolumeFilesystem
	accessMode := bs.getAccessMode(storageClass, mode)
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
			StorageClassName: &storageClass,
			VolumeMode:       &mode,
		},
	}

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
