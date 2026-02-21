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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/storage/types"
)

const (
	backupTargetPVCPrefix = "backup-target-pvc"
)

func backupTargetVolumeName(backupName string) string {
	return fmt.Sprintf("%s-%s", backupName, backupTargetPVCPrefix)
}

var (
	failedTargetPVCAttach       = "failed to attach target backup pvc: %s"
	failedTargetPVCDetach       = "failed to detach target backup pvc: %s"
	attachTargetPVCMsg          = "attaching backup target pvc %s to vmi %s"
	attachInProgressMsg         = "backup target PVC %s is being attached to VMI %s"
	detachTargetPVCMsg          = "detaching backup target pvc from vmi %s"
	backupTargetPVCBlockModeMsg = "backup target PVC must be a filesystem PVC, provided pvc %s/%s is block"
	pvcNotFoundMsg              = "PVC %s/%s doesnt exist"

	backupTargetPVCNameNilMsg = "backup target PVC name is nil"
)

func (ctrl *VMBackupController) verifyBackupTargetPVC(pvcName *string, namespace string) *SyncInfo {
	if pvcName == nil {
		log.Log.Error(backupTargetPVCNameNilMsg)
		return syncInfoError(fmt.Errorf("%s", backupTargetPVCNameNilMsg))
	}
	objKey := cacheKeyFunc(namespace, *pvcName)
	obj, exists, err := ctrl.pvcStore.GetByKey(objKey)
	if err != nil {
		err = fmt.Errorf("error getting PVC from store: %w", err)
		log.Log.Error(err.Error())
		return syncInfoError(err)
	}

	if !exists {
		return &SyncInfo{
			event:  backupInitializingEvent,
			reason: fmt.Sprintf(pvcNotFoundMsg, namespace, *pvcName),
		}
	}
	pvc := obj.(*corev1.PersistentVolumeClaim)
	if types.IsPVCBlock(pvc.Spec.VolumeMode) {
		return syncInfoError(fmt.Errorf(backupTargetPVCBlockModeMsg, namespace, *pvcName))
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

func (ctrl *VMBackupController) attachBackupTargetPVC(vmi *v1.VirtualMachineInstance, pvcName string, volumeName string) *SyncInfo {
	// Check if we already patched the VMI with the utilityVolume
	for _, vol := range vmi.Spec.UtilityVolumes {
		if vol.Name == volumeName {
			return &SyncInfo{
				event:  backupInitializingEvent,
				reason: fmt.Sprintf(attachInProgressMsg, pvcName, vmi.Name),
			}
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
		err = fmt.Errorf("failed to generate attach backup target PVC patch: %w", err)
		log.Log.Error(err.Error())
		return syncInfoError(err)
	}

	_, err = ctrl.client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, k8stypes.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		failedPatchErr := fmt.Errorf(failedTargetPVCAttach, err)
		log.Log.Object(vmi).Errorf("%s", failedPatchErr.Error())
		return syncInfoError(failedPatchErr)
	}

	pvcAttachMsg := fmt.Sprintf(attachTargetPVCMsg, pvcName, vmi.Name)
	log.Log.Object(vmi).Infof("%s", pvcAttachMsg)

	return &SyncInfo{
		event:  backupInitializingEvent,
		reason: pvcAttachMsg,
	}
}

func (ctrl *VMBackupController) detachBackupTargetPVC(vmi *v1.VirtualMachineInstance, volumeName string, event string) *SyncInfo {
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
		failedPatchErr := fmt.Errorf(failedTargetPVCDetach, err)
		log.Log.Object(vmi).Errorf("Failed to generate patch: %s", failedPatchErr.Error())
		return syncInfoError(failedPatchErr)
	}

	_, err = ctrl.client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, k8stypes.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		failedPatchErr := fmt.Errorf(failedTargetPVCDetach, err)
		log.Log.Object(vmi).Errorf("Failed to patch VMI: %s", failedPatchErr.Error())
		return syncInfoError(failedPatchErr)
	}

	pvcDetachMsg := fmt.Sprintf(detachTargetPVCMsg, vmi.Name)
	log.Log.Object(vmi).Infof("%s", pvcDetachMsg)

	return &SyncInfo{
		event:  event,
		reason: pvcDetachMsg,
	}
}
