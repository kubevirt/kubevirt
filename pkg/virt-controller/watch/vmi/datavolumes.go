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
 * Copyright The KubeVirt Authors
 *
 */

package vmi

import (
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
)

func (c *Controller) areDataVolumesReady(vmi *v1.VirtualMachineInstance, dataVolumes []*cdiv1.DataVolume) (bool, bool, common.SyncError) {

	ready := true
	wffc := false

	for _, volume := range vmi.Spec.Volumes {
		// Check both DVs and PVCs
		if volume.VolumeSource.DataVolume != nil || volume.VolumeSource.PersistentVolumeClaim != nil {
			volumeReady, volumeWffc, err := storagetypes.VolumeReadyToAttachToNode(vmi.Namespace, volume, dataVolumes, c.dataVolumeIndexer, c.pvcIndexer)
			if err != nil {
				if _, ok := err.(storagetypes.PvcNotFoundError); ok {
					// due to the eventually consistent nature of controllers, CDI or users may need some time to actually crate the PVC.
					// We wait for them to appear.
					c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, controller.FailedPvcNotFoundReason, "PVC %s/%s does not exist, waiting for it to appear", vmi.Namespace, storagetypes.PVCNameFromVirtVolume(&volume))
					return false, false, &informalSyncError{err: fmt.Errorf("PVC %s/%s does not exist, waiting for it to appear", vmi.Namespace, storagetypes.PVCNameFromVirtVolume(&volume)), reason: controller.FailedPvcNotFoundReason}
				} else {
					c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedPvcNotFoundReason, "Error determining if volume is ready: %v", err)
					return false, false, common.NewSyncError(fmt.Errorf("Error determining if volume is ready %v", err), controller.FailedDataVolumeImportReason)
				}
			}
			wffc = wffc || volumeWffc
			// Ready only becomes false if WFFC is also false.
			ready = ready && (volumeReady || volumeWffc)
		}
	}

	return ready, wffc, nil
}

func aggregateDataVolumesConditions(vmiCopy *v1.VirtualMachineInstance, dvs []*cdiv1.DataVolume) {
	if len(dvs) == 0 {
		return
	}

	dvsReadyCondition := v1.VirtualMachineInstanceCondition{
		Status:  k8sv1.ConditionTrue,
		Type:    v1.VirtualMachineInstanceDataVolumesReady,
		Reason:  v1.VirtualMachineInstanceReasonAllDVsReady,
		Message: "All of the VMI's DVs are bound and not running",
	}

	for _, dv := range dvs {
		cStatus := statusOfReadyCondition(dv.Status.Conditions)
		if cStatus != k8sv1.ConditionTrue {
			dvsReadyCondition.Reason = v1.VirtualMachineInstanceReasonNotAllDVsReady
			if cStatus == k8sv1.ConditionFalse {
				dvsReadyCondition.Status = cStatus
			} else if dvsReadyCondition.Status == k8sv1.ConditionTrue {
				dvsReadyCondition.Status = cStatus
			}
		}
	}

	if dvsReadyCondition.Status != k8sv1.ConditionTrue {
		dvsReadyCondition.Message = "Not all of the VMI's DVs are ready"
	}

	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()
	vmiConditions.UpdateCondition(vmiCopy, &dvsReadyCondition)
}

func statusOfReadyCondition(conditions []cdiv1.DataVolumeCondition) k8sv1.ConditionStatus {
	for _, condition := range conditions {
		if condition.Type == cdiv1.DataVolumeReady {
			return condition.Status
		}
	}
	return k8sv1.ConditionUnknown
}
