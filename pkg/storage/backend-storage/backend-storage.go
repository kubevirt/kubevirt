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
	"errors"
	"fmt"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	corev1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	virtwait "kubevirt.io/kubevirt/pkg/apimachinery/wait"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/migrations"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	PVCPrefix = "persistent-state-for"
	PVCSize   = "10Mi"
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
		key := controller.NamespacedKey(migration.Namespace, migration.Status.MigrationState.TargetPersistentStatePVCName)
		obj, exists, err := pvcStore.GetByKey(key)
		if err != nil || !exists {
			return nil
		}
		return obj.(*v1.PersistentVolumeClaim)
	}

	return pvcForMigrationTargetFromStore(pvcStore, migration)
}

func RecoverFromBrokenMigration(client kubecli.KubevirtClient, migrationIndexer cache.Indexer, pvcStore cache.Store, vmi *corev1.VirtualMachineInstance, launcherImage string) (*v1.PersistentVolumeClaim, error, bool) {
	var pvc *v1.PersistentVolumeClaim

	migration, err := migrations.InterruptedMigrationForVMI(migrationIndexer, vmi)
	if err != nil || migration == nil {
		return nil, err, false
	}
	if migration.Status.MigrationState == nil ||
		migration.Status.MigrationState.TargetPersistentStatePVCName == migration.Status.MigrationState.SourcePersistentStatePVCName {
		// The migration either didn't actually start, or the backend storage is RWX. Either way we're good, we can delete it.
		err := client.VirtualMachineInstanceMigration(migration.Namespace).Delete(context.Background(), migration.Name, metav1.DeleteOptions{})
		return nil, err, false
	}

	// An interrupted migration exists. Creating a job to check if the source PVC contains /meta/migrated,
	// which would indicate that the libvirt migration finished.
	// A JobComplete condition indicates the file is present, the migration was successful and the target PVC prevails
	// A JobFailed condition indicated the file is absent, the migration didn't finish and the source PVC prevails
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "backend-storage-recover-",
		},
		Spec: batchv1.JobSpec{
			ActiveDeadlineSeconds:   pointer.P(int64(30)),
			BackoffLimit:            pointer.P(int32(1)),
			TTLSecondsAfterFinished: pointer.P(int32(30)),
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "backend-storage-recover-",
				},
				Spec: v1.PodSpec{
					RestartPolicy: v1.RestartPolicyNever,
					SecurityContext: &v1.PodSecurityContext{
						RunAsNonRoot: pointer.P(true),
						RunAsUser:    pointer.P(int64(util.NonRootUID)),
						RunAsGroup:   pointer.P(int64(util.NonRootUID)),
						FSGroup:      pointer.P(int64(util.NonRootUID)),
						SeccompProfile: &v1.SeccompProfile{
							Type: v1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []v1.Container{{
						Name: "container",
						SecurityContext: &v1.SecurityContext{
							AllowPrivilegeEscalation: pointer.P(false),
							Capabilities:             &v1.Capabilities{Drop: []v1.Capability{"ALL"}},
						},
						Image:   launcherImage,
						Command: []string{"ls"},
						Args:    []string{"/meta/migrated"},
						VolumeMounts: []v1.VolumeMount{{
							Name:      "backend-storage",
							MountPath: "/meta",
							SubPath:   "meta",
						}},
					}},
					Volumes: []v1.Volume{{
						Name: "backend-storage",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
								ClaimName: migration.Status.MigrationState.SourcePersistentStatePVCName,
							},
						},
					}},
				},
			},
		},
	}

	job, err = client.BatchV1().Jobs(vmi.Namespace).Create(context.Background(), job, metav1.CreateOptions{})
	if err != nil {
		return nil, err, false
	}

	// Give the job 35 seconds to complete
	success := false
	err = virtwait.PollImmediately(time.Second, 35*time.Second, func(ctx context.Context) (done bool, err error) {
		job, err := client.BatchV1().Jobs(job.Namespace).Get(ctx, job.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		for _, c := range job.Status.Conditions {
			switch c.Type {
			case batchv1.JobComplete:
				if c.Status == v1.ConditionTrue {
					pvc, err = MigrationHandoff(client, pvcStore, migration)
					success = true
					return true, err
				}
			case batchv1.JobFailed:
				if c.Status == v1.ConditionTrue {
					pvc, err = MigrationAbort(client, migration)
					return true, err
				}
			case batchv1.JobSuspended, batchv1.JobFailureTarget, batchv1.JobSuccessCriteriaMet:
				break
			}
		}
		return false, nil
	})
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return nil, err, false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return nil, fmt.Errorf("migration recovery job failed to finish"), false
	}

	// The handoff/abort was successful, we don't want to fail if the job deletion fails
	_ = client.BatchV1().Jobs(vmi.Namespace).Delete(context.Background(), job.Name, metav1.DeleteOptions{
		PropagationPolicy: pointer.P(metav1.DeletePropagationBackground),
	})

	err = client.VirtualMachineInstanceMigration(vmi.Namespace).Delete(context.Background(), migration.Name, metav1.DeleteOptions{})

	return pvc, err, success
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
		if strings.HasPrefix(volume.Name, basePVC(vmi)) {
			return volume.Name
		}
	}

	return ""
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

func MigrationHandoff(client kubecli.KubevirtClient, pvcStore cache.Store, migration *corev1.VirtualMachineInstanceMigration) (*v1.PersistentVolumeClaim, error) {
	if migration == nil || migration.Status.MigrationState == nil ||
		migration.Status.MigrationState.SourcePersistentStatePVCName == "" ||
		migration.Status.MigrationState.TargetPersistentStatePVCName == "" {
		return nil, fmt.Errorf("missing source and/or target PVC name(s)")
	}

	sourcePVC := migration.Status.MigrationState.SourcePersistentStatePVCName
	targetPVC := migration.Status.MigrationState.TargetPersistentStatePVCName

	if sourcePVC == targetPVC {
		// RWX backend-storage, nothing to do
		return nil, nil
	}

	// Let's label the target first, then remove the source.
	// The target might already be labelled if this function was already called for this migration
	target := pvcForMigrationTargetFromStore(pvcStore, migration)
	if target == nil {
		return nil, fmt.Errorf("target PVC not found for migration %s/%s", migration.Namespace, migration.Name)
	}
	labels := target.Labels
	if labels == nil {
		labels = make(map[string]string)
	}

	existing, ok := labels[PVCPrefix]
	if ok && existing != migration.Spec.VMIName {
		return nil, fmt.Errorf("target PVC for %s is already labelled for another VMI: %s", migration.Spec.VMIName, existing)
	}

	if _, migrationLabelExists := target.Labels[corev1.MigrationNameLabel]; migrationLabelExists {
		labelPatchPayload, err := patch.New(
			patch.WithReplace("/metadata/labels/"+PVCPrefix, migration.Spec.VMIName),
			patch.WithTest("/metadata/labels/"+patch.EscapeJSONPointer(corev1.MigrationNameLabel), migration.Name),
			patch.WithRemove("/metadata/labels/"+patch.EscapeJSONPointer(corev1.MigrationNameLabel)),
		).GeneratePayload()

		if err != nil {
			return nil, fmt.Errorf("failed to generate PVC patch: %v", err)
		}
		target, err = client.CoreV1().PersistentVolumeClaims(migration.Namespace).Patch(context.Background(), targetPVC, types.JSONPatchType, labelPatchPayload, metav1.PatchOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to patch PVC: %v", err)
		}
	}

	err := client.CoreV1().PersistentVolumeClaims(migration.Namespace).Delete(context.Background(), sourcePVC, metav1.DeleteOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to delete PVC: %v", err)
	}

	return target, nil
}

func MigrationAbort(client kubecli.KubevirtClient, migration *corev1.VirtualMachineInstanceMigration) (*v1.PersistentVolumeClaim, error) {
	if migration == nil || migration.Status.MigrationState == nil ||
		migration.Status.MigrationState.TargetPersistentStatePVCName == "" {
		return nil, nil
	}

	sourcePVC := migration.Status.MigrationState.SourcePersistentStatePVCName
	targetPVC := migration.Status.MigrationState.TargetPersistentStatePVCName

	if sourcePVC == targetPVC {
		// RWX backend-storage, nothing to delete
		return nil, nil
	}

	err := client.CoreV1().PersistentVolumeClaims(migration.Namespace).Delete(context.Background(), targetPVC, metav1.DeleteOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to delete PVC: %v", err)
	}

	return client.CoreV1().PersistentVolumeClaims(migration.Namespace).Get(context.Background(), sourcePVC, metav1.GetOptions{})
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

	obj, exists, err := bs.pvcStore.GetByKey(controller.NamespacedKey(vmi.Namespace, pvcName))
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
