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
	"fmt"

	"github.com/openshift/library-go/pkg/build/naming"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"

	v1 "kubevirt.io/api/core/v1"

	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/storage/utilityvolume"
)

const (
	backupTargetPVCSuffix = "backup-target-pvc"
)

func backupTargetVolumeName(backupName string) string {
	return naming.GetName(backupName, backupTargetPVCSuffix, validation.DNS1035LabelMaxLength)
}

var (
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
	return utilityvolume.Attached(vmi, volumeName)
}

func (ctrl *VMBackupController) backupTargetPVCDetached(vmi *v1.VirtualMachineInstance, volumeName string) bool {
	return utilityvolume.Detached(vmi, volumeName)
}

func (ctrl *VMBackupController) attachBackupTargetPVC(vmi *v1.VirtualMachineInstance, pvcName string, volumeName string) error {
	return utilityvolume.Attach(ctrl.client, vmi, volumeName, pvcName, v1.Backup)
}

func (ctrl *VMBackupController) detachBackupTargetPVC(vmi *v1.VirtualMachineInstance, volumeName string) error {
	return utilityvolume.Detach(ctrl.client, vmi, volumeName)
}
