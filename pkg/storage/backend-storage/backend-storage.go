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
	"strings"

	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"

	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/tpm"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

const (
	PVCPrefix = "persistent-state-for"
	PVCSize   = "10Mi"

	// LabelApplyStorageProfile is a label used by the CDI mutating webhook
	// to modify the PVC according to the storage profile.
	LabelApplyStorageProfile = "cdi.kubevirt.io/applyStorageProfile"
)

func basePVC(vmi *corev1.VirtualMachineInstance) string {
	return PVCPrefix + "-" + vmi.Name
}

func PVCForVMI(pvcStore cache.Store, vmi *corev1.VirtualMachineInstance) *v1.PersistentVolumeClaim {
	var legacyPVC *v1.PersistentVolumeClaim

	objs := pvcStore.List()
	for _, obj := range objs {
		pvc := obj.(*v1.PersistentVolumeClaim)
		if pvc.Namespace != vmi.Namespace {
			continue
		}
		if pvc.DeletionTimestamp != nil {
			continue
		}
		vmName, found := pvc.Labels[PVCPrefix]
		if found && vmName == vmi.Name {
			return pvc
		}
		if pvc.Name == basePVC(vmi) {
			legacyPVC = pvc
		}
	}

	return legacyPVC
}

func pvcForMigrationTargetFromStore(pvcStore cache.Store, migration *corev1.VirtualMachineInstanceMigration) *v1.PersistentVolumeClaim {
	objs := pvcStore.List()
	for _, obj := range objs {
		pvc := obj.(*v1.PersistentVolumeClaim)
		if pvc.Namespace != migration.Namespace {
			continue
		}
		migrationName, found := pvc.Labels[corev1.MigrationNameLabel]
		if found && migrationName == migration.Name {
			return pvc
		}
	}

	return nil

}

func PVCForMigrationTarget(pvcStore cache.Store, migration *corev1.VirtualMachineInstanceMigration) *v1.PersistentVolumeClaim {
	if migration.Status.MigrationState != nil && migration.Status.MigrationState.TargetPersistentStatePVCName != "" {
		obj, exists, err := pvcStore.GetByKey(migration.Namespace + "/" + migration.Status.MigrationState.TargetPersistentStatePVCName)
		if err != nil || !exists {
			return nil
		}
		return obj.(*v1.PersistentVolumeClaim)
	}

	return pvcForMigrationTargetFromStore(pvcStore, migration)
}

func (bs *BackendStorage) labelLegacyPVC(pvc *v1.PersistentVolumeClaim, name string) {
	labelPatch := patch.New()
	if len(pvc.Labels) == 0 {
		labelPatch.AddOption(patch.WithAdd("/metadata/labels", map[string]string{PVCPrefix: name}))
	} else {
		labelPatch.AddOption(patch.WithReplace("/metadata/labels/"+PVCPrefix, name))
	}
	labelPatchPayload, err := labelPatch.GeneratePayload()
	if err == nil {
		_, err = bs.client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Patch(context.Background(), pvc.Name, types.JSONPatchType, labelPatchPayload, metav1.PatchOptions{})
		if err != nil {
			log.Log.Reason(err).Warningf("failed to label legacy PVC %s/%s", pvc.Namespace, pvc.Name)
		}
	}
}

func CurrentPVCName(vmi *corev1.VirtualMachineInstance) string {
	for _, volume := range vmi.Status.VolumeStatus {
		if strings.Contains(volume.Name, basePVC(vmi)) {
			return volume.PersistentVolumeClaimInfo.ClaimName
		}
	}

	return ""
}

func HasPersistentEFI(vmiSpec *corev1.VirtualMachineInstanceSpec) bool {
	return vmiSpec.Domain.Firmware != nil &&
		vmiSpec.Domain.Firmware.Bootloader != nil &&
		vmiSpec.Domain.Firmware.Bootloader.EFI != nil &&
		vmiSpec.Domain.Firmware.Bootloader.EFI.Persistent != nil &&
		*vmiSpec.Domain.Firmware.Bootloader.EFI.Persistent
}

func IsBackendStorageNeededForVMI(vmiSpec *corev1.VirtualMachineInstanceSpec) bool {
	return tpm.HasPersistentDevice(vmiSpec) || HasPersistentEFI(vmiSpec)
}

func IsBackendStorageNeededForVM(vm *corev1.VirtualMachine) bool {
	if vm.Spec.Template == nil {
		return false
	}
	return tpm.HasPersistentDevice(&vm.Spec.Template.Spec) || HasPersistentEFI(&vm.Spec.Template.Spec)
}

func MigrationHandoff(client kubecli.KubevirtClient, pvcStore cache.Store, migration *corev1.VirtualMachineInstanceMigration) error {
	if migration == nil || migration.Status.MigrationState == nil ||
		migration.Status.MigrationState.SourcePersistentStatePVCName == "" ||
		migration.Status.MigrationState.TargetPersistentStatePVCName == "" {
		return fmt.Errorf("missing source and/or target PVC name(s)")
	}

	sourcePVC := migration.Status.MigrationState.SourcePersistentStatePVCName
	targetPVC := migration.Status.MigrationState.TargetPersistentStatePVCName

	if sourcePVC == targetPVC {
		// RWX backend-storage, nothing to do
		return nil
	}

	// Let's label the target first, then remove the source.
	// The target might already be labelled if this function was already called for this migration
	target := pvcForMigrationTargetFromStore(pvcStore, migration)
	if target == nil {
		return fmt.Errorf("target PVC not found for migration %s/%s", migration.Namespace, migration.Name)
	}
	labels := target.Labels
	if labels == nil {
		labels = make(map[string]string)
	}

	existing, ok := labels[PVCPrefix]
	if ok && existing != migration.Spec.VMIName {
		return fmt.Errorf("target PVC for %s is already labelled for another VMI: %s", migration.Spec.VMIName, existing)
	}

	if _, migrationLabelExists := target.Labels[corev1.MigrationNameLabel]; migrationLabelExists {
		labelPatchPayload, err := patch.New(
			patch.WithReplace("/metadata/labels/"+PVCPrefix, migration.Spec.VMIName),
			patch.WithTest("/metadata/labels/"+patch.EscapeJSONPointer(corev1.MigrationNameLabel), migration.Name),
			patch.WithRemove("/metadata/labels/"+patch.EscapeJSONPointer(corev1.MigrationNameLabel)),
		).GeneratePayload()

		if err != nil {
			return fmt.Errorf("failed to generate PVC patch: %v", err)
		}
		_, err = client.CoreV1().PersistentVolumeClaims(migration.Namespace).Patch(context.Background(), targetPVC, types.JSONPatchType, labelPatchPayload, metav1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("failed to patch PVC: %v", err)
		}
	}

	err := client.CoreV1().PersistentVolumeClaims(migration.Namespace).Delete(context.Background(), sourcePVC, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete PVC: %v", err)
	}

	return nil
}

func MigrationAbort(client kubecli.KubevirtClient, migration *corev1.VirtualMachineInstanceMigration) error {
	if migration == nil || migration.Status.MigrationState == nil ||
		migration.Status.MigrationState.TargetPersistentStatePVCName == "" {
		return nil
	}

	sourcePVC := migration.Status.MigrationState.SourcePersistentStatePVCName
	targetPVC := migration.Status.MigrationState.TargetPersistentStatePVCName

	if sourcePVC == targetPVC {
		// RWX backend-storage, nothing to delete
		return nil
	}

	err := client.CoreV1().PersistentVolumeClaims(migration.Namespace).Delete(context.Background(), targetPVC, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete PVC: %v", err)
	}

	return nil
}

type BackendStorage struct {
	client        kubecli.KubevirtClient
	clusterConfig *virtconfig.ClusterConfig
	scStore       cache.Store
	spStore       cache.Store
	pvcStore      cache.Store
}

func NewBackendStorage(client kubecli.KubevirtClient, clusterConfig *virtconfig.ClusterConfig, scStore cache.Store, spStore cache.Store, pvcStore cache.Store) *BackendStorage {
	return &BackendStorage{
		client:        client,
		clusterConfig: clusterConfig,
		scStore:       scStore,
		spStore:       spStore,
		pvcStore:      pvcStore,
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

func (bs *BackendStorage) UpdateVolumeStatus(vmi *corev1.VirtualMachineInstance, pvc *v1.PersistentVolumeClaim) {
	if vmi.Status.VolumeStatus == nil {
		vmi.Status.VolumeStatus = []corev1.VolumeStatus{}
	}
	for i := range vmi.Status.VolumeStatus {
		if vmi.Status.VolumeStatus[i].Name == pvc.Name {
			if vmi.Status.VolumeStatus[i].PersistentVolumeClaimInfo == nil {
				vmi.Status.VolumeStatus[i].PersistentVolumeClaimInfo = &corev1.PersistentVolumeClaimInfo{}
			}
			vmi.Status.VolumeStatus[i].PersistentVolumeClaimInfo.ClaimName = pvc.Name
			vmi.Status.VolumeStatus[i].PersistentVolumeClaimInfo.AccessModes = pvc.Spec.AccessModes
			return
		}
	}
	vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, corev1.VolumeStatus{
		Name: pvc.Name,
		PersistentVolumeClaimInfo: &corev1.PersistentVolumeClaimInfo{
			ClaimName:   pvc.Name,
			AccessModes: pvc.Spec.AccessModes,
		},
	})
}

func (bs *BackendStorage) createPVC(vmi *corev1.VirtualMachineInstance, labels map[string]string) (*v1.PersistentVolumeClaim, error) {
	storageClass, err := bs.getStorageClass()
	if err != nil {
		return nil, err
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

	// Adding this label to allow the PVC to be processed by the CDI WebhookPvcRendering mutating webhook,
	// which must be enabled in the CDI CR via feature gate.
	// This mutating webhook processes the PVC based on its associated StorageProfile.
	// For example, a profile can define a minimum supported volume size via the annotation:
	// cdi.kubevirt.io/minimumSupportedPvcSize: 4Gi
	// This helps avoid issues with provisioners that reject the hardcoded 10Mi PVC size used here.
	labels[LabelApplyStorageProfile] = "true"

	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName:    basePVC(vmi) + "-",
			OwnerReferences: ownerReferences,
			Labels:          labels,
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
	if err != nil {
		return nil, err
	}

	return pvc, nil
}

func (bs *BackendStorage) CreatePVCForVMI(vmi *corev1.VirtualMachineInstance) (*v1.PersistentVolumeClaim, error) {
	pvc := PVCForVMI(bs.pvcStore, vmi)
	if pvc == nil {
		return bs.createPVC(vmi, map[string]string{PVCPrefix: vmi.Name})
	}

	if _, exists := pvc.Labels[PVCPrefix]; !exists {
		bs.labelLegacyPVC(pvc, vmi.Name)
	}

	return pvc, nil
}

func (bs *BackendStorage) CreatePVCForMigrationTarget(vmi *corev1.VirtualMachineInstance, migrationName string) (*v1.PersistentVolumeClaim, error) {
	pvc := PVCForVMI(bs.pvcStore, vmi)

	if len(pvc.Status.AccessModes) > 0 && pvc.Status.AccessModes[0] == v1.ReadWriteMany {
		// The source PVC is RWX, so it can be used for the target too
		return pvc, nil
	}

	return bs.createPVC(vmi, map[string]string{corev1.MigrationNameLabel: migrationName})
}

// IsPVCReady returns true if either:
// - No PVC is needed for the VMI since it doesn't use backend storage
// - The backend storage PVC is bound
// - The backend storage PVC is pending uses a WaitForFirstConsumer storage class
func (bs *BackendStorage) IsPVCReady(vmi *corev1.VirtualMachineInstance, pvcName string) (bool, error) {
	if !IsBackendStorageNeededForVMI(&vmi.Spec) {
		return true, nil
	}

	obj, exists, err := bs.pvcStore.GetByKey(vmi.Namespace + "/" + pvcName)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, fmt.Errorf("pvc %s not found in namespace %s", pvcName, vmi.Namespace)
	}
	pvc := obj.(*v1.PersistentVolumeClaim)

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
