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
	"sort"
	"strings"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
)

// addPVC handles the addition of a PVC, enqueuing affected VMIs.
func (c *Controller) addPVC(obj interface{}) {
	pvc := obj.(*k8sv1.PersistentVolumeClaim)
	if pvc.DeletionTimestamp != nil {
		return
	}
	persistentStateFor, exists := pvc.Labels[backendstorage.PVCPrefix]
	if exists {
		vmiKey := controller.NamespacedKey(pvc.Namespace, persistentStateFor)
		c.pvcExpectations.CreationObserved(vmiKey)
		c.Queue.Add(vmiKey)
		return // The PVC is a backend-storage PVC, won't be listed by `c.listVMIsMatchingDV()`
	}
	vmis, err := c.listVMIsMatchingDV(pvc.Namespace, pvc.Name)
	if err != nil {
		return
	}
	for _, vmi := range vmis {
		log.Log.V(4).Object(pvc).Infof("PVC created for vmi %s", vmi.Name)
		c.enqueueVirtualMachine(vmi)
	}
}

// updatePVC handles updates to a PVC, enqueuing affected VMIs if capacity changes.
func (c *Controller) updatePVC(old, cur interface{}) {
	curPVC := cur.(*k8sv1.PersistentVolumeClaim)
	oldPVC := old.(*k8sv1.PersistentVolumeClaim)
	if curPVC.ResourceVersion == oldPVC.ResourceVersion {
		// Periodic resync will send update events for all known PVCs.
		// Two different versions of the same PVC will always
		// have different RVs.
		return
	}
	if curPVC.DeletionTimestamp != nil {
		return
	}
	if equality.Semantic.DeepEqual(curPVC.Status.Capacity, oldPVC.Status.Capacity) {
		// We only do something when the capacity changes
		return
	}
	vmis, err := c.listVMIsMatchingDV(curPVC.Namespace, curPVC.Name)
	if err != nil {
		log.Log.Object(curPVC).Errorf("Error encountered getting VMIs for DataVolume: %v", err)
		return
	}
	for _, vmi := range vmis {
		log.Log.V(4).Object(curPVC).Infof("PVC updated for vmi %s", vmi.Name)
		c.enqueueVirtualMachine(vmi)
	}
}

// listVMIsMatchingDV finds all VMIs referencing a given DataVolume or PVC name.
func (c *Controller) listVMIsMatchingDV(namespace, dvName string) ([]*virtv1.VirtualMachineInstance, error) {
	// TODO - refactor if/when dv/pvc do not have the same name
	vmis := []*virtv1.VirtualMachineInstance{}
	for _, indexName := range []string{"dv", "pvc"} {
		objs, err := c.vmiIndexer.ByIndex(indexName, namespace+"/"+dvName)
		if err != nil {
			return nil, err
		}
		for _, obj := range objs {
			vmi := obj.(*virtv1.VirtualMachineInstance)
			vmis = append(vmis, vmi.DeepCopy())
		}
	}
	return vmis, nil
}

// handleBackendStorage manages backend storage PVC creation for the VMI.
func (c *Controller) handleBackendStorage(vmi *virtv1.VirtualMachineInstance) (string, common.SyncError) {
	key, err := controller.KeyFunc(vmi)
	if err != nil {
		return "", common.NewSyncError(err, controller.FailedBackendStorageCreateReason)
	}
	if !backendstorage.IsBackendStorageNeededForVMI(&vmi.Spec) {
		return "", nil
	}
	pvc := backendstorage.PVCForVMI(c.pvcIndexer, vmi)
	if pvc == nil {
		c.pvcExpectations.ExpectCreations(key, 1)
		if pvc, err = c.backendStorage.CreatePVCForVMI(vmi); err != nil {
			c.pvcExpectations.CreationObserved(key)
			return "", common.NewSyncError(err, controller.FailedBackendStorageCreateReason)
		}
	}
	return pvc.Name, nil
}

// updateVolumeStatus updates the VMI's VolumeStatus based on pod and volume state.
func (c *Controller) updateVolumeStatus(vmi *virtv1.VirtualMachineInstance, virtlauncherPod *k8sv1.Pod) error {
	oldStatus := vmi.Status.DeepCopy().VolumeStatus
	oldStatusMap := make(map[string]virtv1.VolumeStatus)
	for _, status := range oldStatus {
		oldStatusMap[status.Name] = status
	}

	hotplugVolumes := controller.GetHotplugVolumes(vmi, virtlauncherPod)
	hotplugVolumesMap := make(map[string]*virtv1.Volume)
	for _, volume := range hotplugVolumes {
		hotplugVolumesMap[volume.Name] = volume
	}

	attachmentPods, err := controller.AttachmentPods(virtlauncherPod, c.podIndexer)
	if err != nil {
		return err
	}

	attachmentPod, _ := getActiveAndOldAttachmentPods(hotplugVolumes, attachmentPods)

	newStatus := make([]virtv1.VolumeStatus, 0)

	backendStoragePVC := backendstorage.PVCForVMI(c.pvcIndexer, vmi)
	if backendStoragePVC != nil {
		if backendStorage, ok := oldStatusMap[backendStoragePVC.Name]; ok {
			newStatus = append(newStatus, backendStorage)
		}
	}

	for i, volume := range vmi.Spec.Volumes {
		status := virtv1.VolumeStatus{}
		if existingStatus, ok := oldStatusMap[volume.Name]; ok {
			status = existingStatus
		} else {
			status.Name = volume.Name
		}
		// Remove from map so I can detect existing volumes that have been removed from spec.
		delete(oldStatusMap, volume.Name)

		//if hotplugVolume, ok := hotplugVolumesMap[volume.Name]; ok {
		if _, ok := hotplugVolumesMap[volume.Name]; ok {
			if status.HotplugVolume == nil {
				status.HotplugVolume = &virtv1.HotplugVolumeStatus{}
			}
			if volume.MemoryDump != nil && status.MemoryDumpVolume == nil {
				status.MemoryDumpVolume = &virtv1.DomainMemoryDumpInfo{
					ClaimName: volume.Name,
				}
			}
			if attachmentPod == nil {
				if !c.volumeReady(status.Phase) {
					status.HotplugVolume.AttachPodUID = ""
					// Volume is not hotplugged in VM and Pod is gone, or hasn't been created yet, check for the PVC associated with the volume to set phase and message
					phase, reason, message := c.getVolumePhaseMessageReason(&vmi.Spec.Volumes[i], vmi.Namespace)
					status.Phase = phase
					status.Message = message
					status.Reason = reason
				}
			} else {
				status.HotplugVolume.AttachPodName = attachmentPod.Name
				if len(attachmentPod.Status.ContainerStatuses) == 1 && attachmentPod.Status.ContainerStatuses[0].Ready {
					status.HotplugVolume.AttachPodUID = attachmentPod.UID
				} else {
					// Remove UID of old pod if a new one is available, but not yet ready
					status.HotplugVolume.AttachPodUID = ""
				}
				if canMoveToAttachedPhase(status.Phase) {
					status.Phase = virtv1.HotplugVolumeAttachedToNode
					status.Message = fmt.Sprintf("Created hotplug attachment pod %s, for volume %s", attachmentPod.Name, volume.Name)
					status.Reason = controller.SuccessfulCreatePodReason
					c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, status.Reason, status.Message)
				}
			}
		}

		if volume.VolumeSource.PersistentVolumeClaim != nil || volume.VolumeSource.DataVolume != nil || volume.VolumeSource.MemoryDump != nil {
			pvcName := storagetypes.PVCNameFromVirtVolume(&volume)
			pvcInterface, pvcExists, _ := c.pvcIndexer.GetByKey(fmt.Sprintf("%s/%s", vmi.Namespace, pvcName))
			if pvcExists {
				pvc := pvcInterface.(*k8sv1.PersistentVolumeClaim)
				status.PersistentVolumeClaimInfo = &virtv1.PersistentVolumeClaimInfo{
					ClaimName:    pvc.Name,
					AccessModes:  pvc.Spec.AccessModes,
					VolumeMode:   pvc.Spec.VolumeMode,
					Capacity:     pvc.Status.Capacity,
					Requests:     pvc.Spec.Resources.Requests,
					Preallocated: storagetypes.IsPreallocated(pvc.ObjectMeta.Annotations),
				}
				filesystemOverhead, err := c.getFilesystemOverhead(pvc)
				if err != nil {
					log.Log.Reason(err).Errorf("Failed to get filesystem overhead for PVC %s/%s", vmi.Namespace, pvcName)
					return err
				}
				status.PersistentVolumeClaimInfo.FilesystemOverhead = &filesystemOverhead
			}
		}

		newStatus = append(newStatus, status)
	}

	// We have updated the status of current volumes, but if a volume was removed, we want to keep that status, until there is no
	// associated pod, then remove it. Any statuses left in the map are statuses without a matching volume in the spec.
	for volumeName, status := range oldStatusMap {
		attachmentPod := findAttachmentPodByVolumeName(volumeName, attachmentPods)
		if attachmentPod != nil {
			status.HotplugVolume.AttachPodName = attachmentPod.Name
			status.HotplugVolume.AttachPodUID = attachmentPod.UID
			status.Phase = virtv1.HotplugVolumeDetaching
			if attachmentPod.DeletionTimestamp != nil {
				status.Message = fmt.Sprintf("Deleted hotplug attachment pod %s, for volume %s", attachmentPod.Name, volumeName)
				status.Reason = controller.SuccessfulDeletePodReason
				c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, status.Reason, status.Message)
			}
			// If the pod exists, we keep the status.
			newStatus = append(newStatus, status)
		}
	}

	sort.SliceStable(newStatus, func(i, j int) bool {
		return strings.Compare(newStatus[i].Name, newStatus[j].Name) == -1
	})
	vmi.Status.VolumeStatus = newStatus
	return nil
}

// volumeReady checks if a volume is in a ready state.
func (c *Controller) volumeReady(phase virtv1.VolumePhase) bool {
	return phase == virtv1.VolumeReady
}

// getVolumePhaseMessageReason determines the phase, reason, and message for a volume.
func (c *Controller) getVolumePhaseMessageReason(volume *virtv1.Volume, namespace string) (virtv1.VolumePhase, string, string) {
	claimName := storagetypes.PVCNameFromVirtVolume(volume)
	pvcInterface, pvcExists, _ := c.pvcIndexer.GetByKey(fmt.Sprintf("%s/%s", namespace, claimName))
	if !pvcExists {
		return virtv1.VolumePending, controller.FailedPvcNotFoundReason, fmt.Sprintf("PVC %s not found", claimName)
	}
	pvc := pvcInterface.(*k8sv1.PersistentVolumeClaim)
	if pvc.Status.Phase == k8sv1.ClaimPending {
		return virtv1.VolumePending, controller.PVCNotReadyReason, "PVC is in phase ClaimPending"
	} else if pvc.Status.Phase == k8sv1.ClaimBound {
		return virtv1.VolumeBound, controller.PVCNotReadyReason, "PVC is in phase Bound"
	}
	return virtv1.VolumePending, controller.PVCNotReadyReason, "PVC is in phase Lost"
}

// getFilesystemOverhead retrieves the filesystem overhead for a PVC.
func (c *Controller) getFilesystemOverhead(pvc *k8sv1.PersistentVolumeClaim) (virtv1.Percent, error) {
	cdiInstances := len(c.cdiStore.List())
	if cdiInstances != 1 {
		if cdiInstances > 1 {
			log.Log.V(3).Object(pvc).Reason(storagetypes.ErrMultipleCdiInstances).Infof(storagetypes.FSOverheadMsg)
		} else {
			log.Log.V(3).Object(pvc).Reason(storagetypes.ErrFailedToFindCdi).Infof(storagetypes.FSOverheadMsg)
		}
		return storagetypes.DefaultFSOverhead, nil
	}
	cdiConfigInterface, cdiConfigExists, err := c.cdiConfigStore.GetByKey(storagetypes.ConfigName)
	if !cdiConfigExists || err != nil {
		return "0", fmt.Errorf("Failed to find CDIConfig but CDI exists: %w", err)
	}
	cdiConfig, ok := cdiConfigInterface.(*cdiv1.CDIConfig)
	if !ok {
		return "0", fmt.Errorf("Failed to convert CDIConfig object %v to type CDIConfig", cdiConfigInterface)
	}
	return storagetypes.GetFilesystemOverhead(pvc.Spec.VolumeMode, pvc.Spec.StorageClassName, cdiConfig)
}

func (c *Controller) syncVolumesUpdate(vmi *virtv1.VirtualMachineInstance) {
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()
	condition := virtv1.VirtualMachineInstanceCondition{
		Type:               virtv1.VirtualMachineInstanceVolumesChange,
		LastTransitionTime: v1.Now(),
		Status:             k8sv1.ConditionTrue,
		Message:            "migrate volumes",
	}
	vmiConditions.UpdateCondition(vmi, &condition)
}

func (c *Controller) requireVolumesUpdate(vmi *virtv1.VirtualMachineInstance) bool {
	if len(vmi.Status.MigratedVolumes) < 1 {
		return false
	}
	if controller.NewVirtualMachineInstanceConditionManager().HasCondition(vmi, virtv1.VirtualMachineInstanceVolumesChange) {
		return false
	}
	migVolsMap := make(map[string]string)
	for _, v := range vmi.Status.MigratedVolumes {
		migVolsMap[v.SourcePVCInfo.ClaimName] = v.DestinationPVCInfo.ClaimName
	}
	for _, v := range vmi.Spec.Volumes {
		claim := storagetypes.PVCNameFromVirtVolume(&v)
		if claim == "" {
			continue
		}
		if _, ok := migVolsMap[claim]; !ok {
			return true
		}
	}

	return false
}
