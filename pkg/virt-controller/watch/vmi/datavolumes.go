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
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/client-go/tools/cache"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
)

// addDataVolume handles the addition of a DataVolume, enqueuing affected VMIs.
func (c *Controller) addDataVolume(obj interface{}) {
	dataVolume := obj.(*cdiv1.DataVolume)
	if dataVolume.DeletionTimestamp != nil {
		c.deleteDataVolume(dataVolume)
		return
	}
	vmis, err := c.listVMIsMatchingDV(dataVolume.Namespace, dataVolume.Name)
	if err != nil {
		return
	}
	for _, vmi := range vmis {
		log.Log.V(4).Object(dataVolume).Infof("DataVolume created for vmi %s", vmi.Name)
		c.enqueueVirtualMachine(vmi)
	}
}

// updateDataVolume handles updates to a DataVolume, enqueuing affected VMIs.
func (c *Controller) updateDataVolume(old, cur interface{}) {
	curDataVolume := cur.(*cdiv1.DataVolume)
	oldDataVolume := old.(*cdiv1.DataVolume)
	if curDataVolume.ResourceVersion == oldDataVolume.ResourceVersion {
		// Periodic resync will send update events for all known DataVolumes.
		// Two different versions of the same dataVolume will always
		// have different RVs.
		return
	}
	if curDataVolume.DeletionTimestamp != nil {
		labelChanged := !equality.Semantic.DeepEqual(curDataVolume.Labels, oldDataVolume.Labels)
		// having a DataVOlume marked for deletion is enough
		// to count as a deletion expectation
		c.deleteDataVolume(curDataVolume)
		if labelChanged {
			// we don't need to check the oldDataVolume.DeletionTimestamp
			// because DeletionTimestamp cannot be unset.
			c.deleteDataVolume(oldDataVolume)
		}
		return
	}
	vmis, err := c.listVMIsMatchingDV(curDataVolume.Namespace, curDataVolume.Name)
	if err != nil {
		log.Log.Object(curDataVolume).Errorf("Error encountered during datavolume update: %v", err)
		return
	}
	for _, vmi := range vmis {
		log.Log.V(4).Object(curDataVolume).Infof("DataVolume updated for vmi %s", vmi.Name)
		c.enqueueVirtualMachine(vmi)
	}
}

// deleteDataVolume handles the deletion of a DataVolume, enqueuing affected VMIs.
func (c *Controller) deleteDataVolume(obj interface{}) {
	dataVolume, ok := obj.(*cdiv1.DataVolume)
	// When a delete is dropped, the relist will notice a dataVolume in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the dataVolume
	// changed labels the new vmi will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf(tombstoneGetObjectErrFmt, obj)).Error(deleteNotifFailed)
			return
		}
		dataVolume, ok = tombstone.Obj.(*cdiv1.DataVolume)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a DataVolume %#v", obj)).Error(deleteNotifFailed)
			return
		}
	}
	vmis, err := c.listVMIsMatchingDV(dataVolume.Namespace, dataVolume.Name)
	if err != nil {
		return
	}
	for _, vmi := range vmis {
		log.Log.V(4).Object(dataVolume).Infof("DataVolume deleted for vmi %s", vmi.Name)
		c.enqueueVirtualMachine(vmi)
	}
}

func (c *Controller) areDataVolumesReady(vmi *v1.VirtualMachineInstance, dataVolumes []*cdiv1.DataVolume) (bool, bool, common.SyncError) {

	ready := true
	wffc := false

	for _, volume := range vmi.Spec.Volumes {
		// Check both DVs and PVCs
		if (volume.VolumeSource.DataVolume != nil && !volume.VolumeSource.DataVolume.Hotpluggable) ||
			(volume.VolumeSource.PersistentVolumeClaim != nil && !volume.VolumeSource.PersistentVolumeClaim.Hotpluggable) {
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
