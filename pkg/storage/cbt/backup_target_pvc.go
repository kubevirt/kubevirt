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

	"github.com/openshift/library-go/pkg/build/naming"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/pointer"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

const (
	backupTargetPVCSuffix = "backup-target-pvc"
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
	pvcNotFoundMsg              = "PVC %s/%s doesnt exist"

	backupTargetPVCNameNilMsg = "backup target PVC name is nil"
)

func (ctrl *VMBackupController) verifyBackupTargetPVC(pvcName *string, namespace string) (string, error) {
	if pvcName == nil {
		return "", fmt.Errorf("%s", backupTargetPVCNameNilMsg)
	}
	objKey := types.NamespacedName{Namespace: namespace, Name: *pvcName}.String()
	obj, exists, err := ctrl.pvcStore.GetByKey(objKey)
	if err != nil {
		return "", fmt.Errorf("error getting PVC from store: %w", err)
	}

	if !exists {
		return fmt.Sprintf(pvcNotFoundMsg, namespace, *pvcName), nil
	}
	pvc := obj.(*corev1.PersistentVolumeClaim)
	if storagetypes.IsPVCBlock(pvc.Spec.VolumeMode) {
		return "", fmt.Errorf(backupTargetPVCBlockModeMsg, namespace, *pvcName)
	}

	return "", nil
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
