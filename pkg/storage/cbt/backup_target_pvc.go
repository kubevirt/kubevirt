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
 * Copyright The KubeVirt Authors.
 *
 */

package cbt

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/openshift/library-go/pkg/build/naming"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/pointer"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

const (
	backupTargetPVCSuffix       = "backup-target-pvc"
	backupTargetAutoCreateLabel = "backup.kubevirt.io/auto-created"
	// 20% headroom covers filesystem journal and backup metadata overhead on the target PVC.
	backupTargetFSOverhead = 0.2
)

func backupTargetVolumeName(backupName string) string {
	return naming.GetName(backupName, backupTargetPVCSuffix, validation.DNS1035LabelMaxLength)
}

var (
	failedTargetPVCAttach       = "failed to attach target backup pvc: %s"
	failedTargetPVCDetach       = "failed to detach target backup pvc: %s"
	attachTargetPVCMsg          = "attaching backup target pvc %s to vmi %s"
	detachTargetPVCMsg          = "detaching backup target pvc from vmi %s"
	backupTargetPVCBlockModeMsg = "backup target PVC must be a filesystem PVC, provided pvc %s/%s is block"
	pvcNotFoundMsg              = "PVC %s/%s doesn't exist"
	creatingBackupTargetPVCMsg  = "creating backup target PVC %s/%s"
	backupTargetPVCNotOwnedMsg  = "PVC %s/%s already exists and is not owned by VirtualMachineBackup %s; provide a different name via spec.pvcName or remove the PVC"
)

// backupTargetPVCNotFoundError indicates the target PVC is not yet in the informer
// store. Callers may wait (Initializing) or requeue depending on context.
type backupTargetPVCNotFoundError struct {
	namespace string
	name      string
}

func (e *backupTargetPVCNotFoundError) Error() string {
	return fmt.Sprintf(pvcNotFoundMsg, e.namespace, e.name)
}

func isBackupTargetPVCNotFound(err error) bool {
	var notFound *backupTargetPVCNotFoundError
	return errors.As(err, &notFound)
}

// ensureBackupTargetPVC resolves the target PVC name, creates it when spec.pvcName is
// omitted, verifies it is usable, and records status.pvcName.
// Returns (pvcName, initializingReason, error). A non-empty reason means the caller
// should wait (set Initializing) without failing the backup.
func (ctrl *VMBackupController) ensureBackupTargetPVC(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) (string, string, error) {
	pvcName, autoCreate := resolveBackupTargetPVCName(backup)

	if autoCreate {
		if err := ctrl.createBackupTargetPVCIfNeeded(backup, vmi, pvcName); err != nil {
			return "", "", err
		}
	}

	if err := ctrl.verifyBackupTargetPVC(pointer.P(pvcName), backup.Namespace); err != nil {
		if isBackupTargetPVCNotFound(err) {
			if autoCreate {
				// Record the intended name so the next reconcile reuses it while the
				// informer catches up (requeue via error; there is no PVC→backup enqueue).
				backup.Status.PvcName = pointer.P(pvcName)
				return pvcName, "", fmt.Errorf("waiting for backup target PVC to be observed: %w", err)
			}
			return pvcName, err.Error(), nil
		}
		return "", "", err
	}

	backup.Status.PvcName = pointer.P(pvcName)
	return pvcName, "", nil
}

func resolveBackupTargetPVCName(backup *backupv1.VirtualMachineBackup) (string, bool) {
	if backup.Spec.PvcName != nil && *backup.Spec.PvcName != "" {
		return *backup.Spec.PvcName, false
	}
	if backup.Status != nil && backup.Status.PvcName != nil && *backup.Status.PvcName != "" {
		return *backup.Status.PvcName, true
	}
	return backupTargetVolumeName(backup.Name), true
}

func (ctrl *VMBackupController) createBackupTargetPVCIfNeeded(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance, pvcName string) error {
	objKey := types.NamespacedName{Namespace: backup.Namespace, Name: pvcName}.String()
	obj, exists, err := ctrl.pvcStore.GetByKey(objKey)
	if err != nil {
		return fmt.Errorf("error getting PVC from store: %w", err)
	}
	if exists {
		return validateReusableAutoBackupTargetPVC(backup, obj.(*corev1.PersistentVolumeClaim))
	}

	size, err := ctrl.calculateBackupTargetPVCSize(vmi)
	if err != nil {
		return err
	}
	storageClass, err := ctrl.storageClassForBackupTarget(vmi)
	if err != nil {
		return err
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: backup.Namespace,
			Labels: map[string]string{
				backupTargetAutoCreateLabel: "true",
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(backup, backupv1.SchemeGroupVersion.WithKind(
					backupv1.VirtualMachineBackupGroupVersionKind.Kind)),
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: *size,
				},
			},
			StorageClassName: &storageClass,
			VolumeMode:       pointer.P(corev1.PersistentVolumeFilesystem),
		},
	}

	log.Log.V(3).Object(backup).Infof(creatingBackupTargetPVCMsg, backup.Namespace, pvcName)
	_, err = ctrl.client.CoreV1().PersistentVolumeClaims(backup.Namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if err == nil {
		return nil
	}
	if !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create backup target PVC %s/%s: %w", backup.Namespace, pvcName, err)
	}
	return ctrl.validateExistingAutoBackupTargetPVC(backup, pvcName)
}

// validateExistingAutoBackupTargetPVC loads a PVC that already exists (store or API)
// and ensures it is safe to reuse for this auto-provisioned backup target.
func (ctrl *VMBackupController) validateExistingAutoBackupTargetPVC(backup *backupv1.VirtualMachineBackup, pvcName string) error {
	objKey := types.NamespacedName{Namespace: backup.Namespace, Name: pvcName}.String()
	obj, exists, err := ctrl.pvcStore.GetByKey(objKey)
	if err != nil {
		return fmt.Errorf("error getting PVC from store: %w", err)
	}
	if exists {
		return validateReusableAutoBackupTargetPVC(backup, obj.(*corev1.PersistentVolumeClaim))
	}

	pvc, err := ctrl.client.CoreV1().PersistentVolumeClaims(backup.Namespace).Get(context.Background(), pvcName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("PVC %s/%s reported as existing but could not be read: %w", backup.Namespace, pvcName, err)
	}
	return validateReusableAutoBackupTargetPVC(backup, pvc)
}

func validateReusableAutoBackupTargetPVC(backup *backupv1.VirtualMachineBackup, pvc *corev1.PersistentVolumeClaim) error {
	// VolumeMode is immutable after creation; this still rejects pre-existing block
	// PVCs that collide with the auto-generated name (or any unexpected reuse).
	if storagetypes.IsPVCBlock(pvc.Spec.VolumeMode) {
		return fmt.Errorf(backupTargetPVCBlockModeMsg, pvc.Namespace, pvc.Name)
	}
	if !metav1.IsControlledBy(pvc, backup) {
		return fmt.Errorf(backupTargetPVCNotOwnedMsg, pvc.Namespace, pvc.Name, backup.Name)
	}
	return nil
}

func (ctrl *VMBackupController) calculateBackupTargetPVCSize(vmi *v1.VirtualMachineInstance) (*resource.Quantity, error) {
	total := resource.NewQuantity(0, resource.BinarySI)
	found := false

	for i := range vmi.Spec.Volumes {
		volume := &vmi.Spec.Volumes[i]
		if !IsCBTEligibleVolume(volume) {
			continue
		}
		claimName := storagetypes.PVCNameFromVirtVolume(volume)
		if claimName == "" {
			continue
		}
		pvc, err := storagetypes.GetPersistentVolumeClaimFromCache(vmi.Namespace, claimName, ctrl.pvcStore)
		if err != nil {
			return nil, err
		}
		if pvc == nil {
			return nil, fmt.Errorf("source PVC %s/%s for volume %s not found", vmi.Namespace, claimName, volume.Name)
		}
		size := pvcStorageSize(pvc)
		if size == nil {
			return nil, fmt.Errorf("unable to determine size of source PVC %s/%s", vmi.Namespace, claimName)
		}
		total.Add(*size)
		found = true
	}

	if !found {
		return nil, fmt.Errorf("no CBT-eligible PVC volumes found to size backup target")
	}

	withOverhead := resource.NewQuantity(int64(math.Ceil(float64(total.Value())*(1+backupTargetFSOverhead))), resource.BinarySI)
	return withOverhead, nil
}

// pvcStorageSize returns the usable disk size using the same min(request, capacity)
// rule as CDI / GetDiskCapacity. That avoids treating hostpath/HPP capacity (often the
// whole filesystem) as the disk size. Falls back to the request alone when capacity
// is not yet reported.
func pvcStorageSize(pvc *corev1.PersistentVolumeClaim) *resource.Quantity {
	info := &v1.PersistentVolumeClaimInfo{
		Capacity: pvc.Status.Capacity,
		Requests: pvc.Spec.Resources.Requests,
	}
	if size := storagetypes.GetDiskCapacity(info); size != nil {
		return resource.NewQuantity(*size, resource.BinarySI)
	}
	if qty, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		copied := qty.DeepCopy()
		return &copied
	}
	return nil
}

// storageClassForBackupTarget picks the storage class of the boot-candidate CBT disk
// (lowest bootOrder among CBT disks, else first CBT-eligible volume), matching virtctl.
func (ctrl *VMBackupController) storageClassForBackupTarget(vmi *v1.VirtualMachineInstance) (string, error) {
	volumeName, err := bootCandidateCBTVolume(vmi)
	if err != nil {
		return "", err
	}
	claimName := ""
	for i := range vmi.Spec.Volumes {
		if vmi.Spec.Volumes[i].Name == volumeName {
			claimName = storagetypes.PVCNameFromVirtVolume(&vmi.Spec.Volumes[i])
			break
		}
	}
	if claimName == "" {
		return "", fmt.Errorf("boot candidate volume %s has no PVC/DataVolume; provide spec.pvcName", volumeName)
	}
	pvc, err := storagetypes.GetPersistentVolumeClaimFromCache(vmi.Namespace, claimName, ctrl.pvcStore)
	if err != nil {
		return "", err
	}
	if pvc == nil {
		return "", fmt.Errorf("source PVC %s/%s not found", vmi.Namespace, claimName)
	}
	return ctrl.storageClassFromPVC(pvc)
}

func bootCandidateCBTVolume(vmi *v1.VirtualMachineInstance) (string, error) {
	diskByName := map[string]*v1.Disk{}
	for i := range vmi.Spec.Domain.Devices.Disks {
		disk := &vmi.Spec.Domain.Devices.Disks[i]
		diskByName[disk.Name] = disk
	}

	var bestName string
	var bestOrder uint
	foundBootOrder := false

	for i := range vmi.Spec.Volumes {
		volume := &vmi.Spec.Volumes[i]
		if !IsCBTEligibleVolume(volume) {
			continue
		}
		disk, ok := diskByName[volume.Name]
		if !ok || disk.BootOrder == nil {
			continue
		}
		if !foundBootOrder || *disk.BootOrder < bestOrder {
			bestOrder = *disk.BootOrder
			bestName = volume.Name
			foundBootOrder = true
		}
	}
	if foundBootOrder {
		return bestName, nil
	}

	for i := range vmi.Spec.Volumes {
		volume := &vmi.Spec.Volumes[i]
		if IsCBTEligibleVolume(volume) {
			return volume.Name, nil
		}
	}
	return "", fmt.Errorf("no CBT-eligible volume found to determine storage class")
}

func (ctrl *VMBackupController) storageClassFromPVC(pvc *corev1.PersistentVolumeClaim) (string, error) {
	if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName != "" {
		return *pvc.Spec.StorageClassName, nil
	}
	if pvc.Spec.VolumeName == "" {
		return "", fmt.Errorf("PVC %s/%s has no storageClassName and is not bound; provide spec.pvcName", pvc.Namespace, pvc.Name)
	}
	pv, err := ctrl.client.CoreV1().PersistentVolumes().Get(context.Background(), pvc.Spec.VolumeName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get PV %s for PVC %s/%s: %w", pvc.Spec.VolumeName, pvc.Namespace, pvc.Name, err)
	}
	if pv.Spec.StorageClassName == "" {
		return "", fmt.Errorf("PV %s has no storageClassName; provide spec.pvcName", pv.Name)
	}
	return pv.Spec.StorageClassName, nil
}

func (ctrl *VMBackupController) verifyBackupTargetPVC(pvcName *string, namespace string) error {
	if pvcName == nil || *pvcName == "" {
		return fmt.Errorf("backup target PVC name is empty")
	}
	objKey := types.NamespacedName{Namespace: namespace, Name: *pvcName}.String()
	obj, exists, err := ctrl.pvcStore.GetByKey(objKey)
	if err != nil {
		return fmt.Errorf("error getting PVC from store: %w", err)
	}

	if !exists {
		return &backupTargetPVCNotFoundError{namespace: namespace, name: *pvcName}
	}
	pvc := obj.(*corev1.PersistentVolumeClaim)
	if storagetypes.IsPVCBlock(pvc.Spec.VolumeMode) {
		return fmt.Errorf(backupTargetPVCBlockModeMsg, namespace, *pvcName)
	}

	return nil
}

func (ctrl *VMBackupController) backupTargetPVCAttached(vmi *v1.VirtualMachineInstance, volumeName string) bool {
	if vmi == nil {
		return false
	}
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.Name == volumeName {
			return volumeStatus.HotplugVolume != nil && volumeStatus.Phase == v1.HotplugVolumeMounted
		}
	}
	return false
}

func (ctrl *VMBackupController) backupTargetPVCDetached(vmi *v1.VirtualMachineInstance, volumeName string) bool {
	if vmi == nil {
		return true
	}

	for _, vol := range vmi.Spec.UtilityVolumes {
		if vol.Name == volumeName {
			return false
		}
	}

	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.Name == volumeName {
			return false
		}
	}

	return true
}

func (ctrl *VMBackupController) attachBackupTargetPVC(vmi *v1.VirtualMachineInstance, pvcName string, volumeName string) error {
	for _, vol := range vmi.Spec.UtilityVolumes {
		if vol.Name == volumeName {
			return nil
		}
	}

	backupVolume := v1.UtilityVolume{
		Name: volumeName,
		PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
			ClaimName: pvcName,
		},
		Type: pointer.P(v1.Backup),
	}

	patchSet := patch.New(
		patch.WithTest("/spec/utilityVolumes", vmi.Spec.UtilityVolumes),
	)

	newUtilityVolumes := append(vmi.Spec.UtilityVolumes, backupVolume)
	if len(vmi.Spec.UtilityVolumes) > 0 {
		patchSet.AddOption(patch.WithReplace("/spec/utilityVolumes", newUtilityVolumes))
	} else {
		patchSet.AddOption(patch.WithAdd("/spec/utilityVolumes", newUtilityVolumes))
	}

	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return fmt.Errorf("failed to generate attach backup target PVC patch: %w", err)
	}

	_, err = ctrl.client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf(failedTargetPVCAttach, err)
	}

	log.Log.Object(vmi).Infof(attachTargetPVCMsg, pvcName, vmi.Name)
	return nil
}

func (ctrl *VMBackupController) detachBackupTargetPVC(vmi *v1.VirtualMachineInstance, volumeName string) error {
	if len(vmi.Spec.UtilityVolumes) == 0 {
		return nil
	}

	newUtilityVolumes := make([]v1.UtilityVolume, 0, len(vmi.Spec.UtilityVolumes))
	for _, vol := range vmi.Spec.UtilityVolumes {
		if vol.Name != volumeName {
			newUtilityVolumes = append(newUtilityVolumes, vol)
		}
	}

	patchSet := patch.New(
		patch.WithTest("/spec/utilityVolumes", vmi.Spec.UtilityVolumes),
	)
	if len(newUtilityVolumes) == 0 {
		patchSet.AddOption(patch.WithRemove("/spec/utilityVolumes"))
	} else {
		patchSet.AddOption(patch.WithReplace("/spec/utilityVolumes", newUtilityVolumes))
	}

	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return fmt.Errorf(failedTargetPVCDetach, err)
	}

	_, err = ctrl.client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf(failedTargetPVCDetach, err)
	}

	log.Log.Object(vmi).Infof(detachTargetPVCMsg, vmi.Name)
	return nil
}
