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

package vmi

import (
	"errors"
	"fmt"
	"sort"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	container_disk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
)

func needsHandleHotplug(hotplugVolumes []*v1.Volume, hotplugAttachmentPods []*k8sv1.Pod) bool {
	if len(hotplugAttachmentPods) > 1 {
		return true
	}
	// Determine if the ready volumes have changed compared to the current pod
	if len(hotplugAttachmentPods) == 1 && podVolumesMatchesReadyVolumes(hotplugAttachmentPods[0], hotplugVolumes) {
		return false
	}
	return len(hotplugVolumes) > 0 || len(hotplugAttachmentPods) > 0
}

func getActiveAndOldAttachmentPods(readyHotplugVolumes []*v1.Volume, hotplugAttachmentPods []*k8sv1.Pod) (*k8sv1.Pod, []*k8sv1.Pod) {
	sort.Slice(hotplugAttachmentPods, func(i, j int) bool {
		return hotplugAttachmentPods[i].CreationTimestamp.Time.Before(hotplugAttachmentPods[j].CreationTimestamp.Time)
	})

	var currentPod *k8sv1.Pod
	oldPods := make([]*k8sv1.Pod, 0)
	for _, attachmentPod := range hotplugAttachmentPods {
		if !podVolumesMatchesReadyVolumes(attachmentPod, readyHotplugVolumes) {
			oldPods = append(oldPods, attachmentPod)
		} else {
			if currentPod != nil {
				oldPods = append(oldPods, currentPod)
			}
			currentPod = attachmentPod
		}
	}
	sort.Slice(oldPods, func(i, j int) bool {
		return oldPods[i].CreationTimestamp.Time.After(oldPods[j].CreationTimestamp.Time)
	})
	return currentPod, oldPods
}

// cleanupAttachmentPods deletes all old attachment pods when the following is true
// 1. There is a currentPod that is running. (not nil and phase.Status == Running)
// 2. There are no readyVolumes (numReadyVolumes == 0)
// 3. The newest oldPod is not running and not marked for deletion.
// If any of those are true, it will not delete the newest oldPod, since that one is the latest
// pod that is closest to the desired state.
func (c *Controller) cleanupAttachmentPods(currentPod *k8sv1.Pod, oldPods []*k8sv1.Pod, vmi *v1.VirtualMachineInstance, numReadyVolumes int) common.SyncError {
	foundRunning := false

	var statusMap = make(map[string]v1.VolumeStatus)
	for _, vs := range vmi.Status.VolumeStatus {
		if vs.HotplugVolume != nil {
			statusMap[vs.Name] = vs
		}
	}

	for _, vmiVolume := range vmi.Spec.Volumes {
		if storagetypes.IsHotplugVolume(&vmiVolume) {
			delete(statusMap, vmiVolume.Name)
		}
	}

	currentPodIsNotRunning := currentPod == nil || currentPod.Status.Phase != k8sv1.PodRunning
	for _, attachmentPod := range oldPods {
		if !foundRunning &&
			attachmentPod.Status.Phase == k8sv1.PodRunning && attachmentPod.DeletionTimestamp == nil &&
			numReadyVolumes > 0 &&
			currentPodIsNotRunning {
			foundRunning = true
			continue
		}
		volumesNotReadyForDelete := 0

		for _, podVolume := range attachmentPod.Spec.Volumes {
			volumeStatus, ok := statusMap[podVolume.Name]
			if ok && !volumeReadyForPodDelete(volumeStatus.Phase) {
				volumesNotReadyForDelete++
			}
		}

		if volumesNotReadyForDelete > 0 {
			log.Log.Object(vmi).V(3).Infof("Not deleting attachment pod %s, because there are still %d volumes to be unmounted", attachmentPod.Name, volumesNotReadyForDelete)
			continue
		}

		if err := c.deleteAttachmentPod(vmi, attachmentPod); err != nil {
			return common.NewSyncError(fmt.Errorf("Error deleting attachment pod %v", err), controller.FailedDeletePodReason)
		}

		log.Log.Object(vmi).V(3).Infof("Deleted attachment pod %s", attachmentPod.Name)
	}
	return nil
}

func volumeReadyForPodDelete(phase v1.VolumePhase) bool {
	switch phase {
	case v1.VolumeReady:
		return false
	case v1.HotplugVolumeMounted:
		return false
	}
	return true
}

func (c *Controller) handleHotplugVolumes(hotplugVolumes []*v1.Volume, hotplugAttachmentPods []*k8sv1.Pod, vmi *v1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod, dataVolumes []*cdiv1.DataVolume) common.SyncError {
	logger := log.Log.Object(vmi)

	readyHotplugVolumes := make([]*v1.Volume, 0)
	// Find all ready volumes
	for _, volume := range hotplugVolumes {
		if container_disk.IsHotplugContainerDisk(volume) {
			readyHotplugVolumes = append(readyHotplugVolumes, volume)
			continue
		}
		var err error
		ready, wffc, err := storagetypes.VolumeReadyToAttachToNode(vmi.Namespace, *volume, dataVolumes, c.dataVolumeIndexer, c.pvcIndexer)
		if err != nil {
			return common.NewSyncError(fmt.Errorf("Error determining volume status %v", err), controller.PVCNotReadyReason)
		}
		if wffc {
			// Volume in WaitForFirstConsumer, it has not been populated by CDI yet. create a dummy pod
			logger.V(1).Infof("Volume %s/%s is in WaitForFistConsumer, triggering population", vmi.Namespace, volume.Name)
			syncError := c.triggerHotplugPopulation(volume, vmi, virtLauncherPod)
			if syncError != nil {
				return syncError
			}
			continue
		}
		if !ready {
			// Volume not ready, skip until it is.
			logger.V(3).Infof("Skipping hotplugged volume: %s, not ready", volume.Name)
			continue
		}
		readyHotplugVolumes = append(readyHotplugVolumes, volume)
	}

	currentPod, oldPods := getActiveAndOldAttachmentPods(readyHotplugVolumes, hotplugAttachmentPods)
	if currentPod == nil && !hasPendingPods(oldPods) && len(readyHotplugVolumes) > 0 {
		if rateLimited, waitTime := c.requeueAfter(oldPods, time.Duration(len(readyHotplugVolumes)/-10)); rateLimited {
			key, err := controller.KeyFunc(vmi)
			if err != nil {
				logger.Object(vmi).Reason(err).Error("failed to extract key from virtualmachine.")
				return common.NewSyncError(fmt.Errorf("failed to extract key from virtualmachine. %v", err), controller.FailedHotplugSyncReason)
			}
			c.Queue.AddAfter(key, waitTime)
		} else {
			if newPod, err := c.createAttachmentPod(vmi, virtLauncherPod, readyHotplugVolumes); err != nil {
				return err
			} else {
				currentPod = newPod
			}
		}
	}
	if err := c.cleanupAttachmentPods(currentPod, oldPods, vmi, len(readyHotplugVolumes)); err != nil {
		return err
	}

	return nil
}

func (c *Controller) createAttachmentPod(vmi *v1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod, volumes []*v1.Volume) (*k8sv1.Pod, common.SyncError) {
	attachmentPodTemplate, _ := c.createAttachmentPodTemplate(vmi, virtLauncherPod, volumes)
	if attachmentPodTemplate == nil {
		return nil, nil
	}
	vmiKey := controller.VirtualMachineInstanceKey(vmi)
	pod, err := c.createPod(vmiKey, vmi.Namespace, attachmentPodTemplate)
	if err != nil {
		c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedCreatePodReason, "Error creating attachment pod: %v", err)
		return nil, common.NewSyncError(fmt.Errorf("Error creating attachment pod %v", err), controller.FailedCreatePodReason)
	}
	c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, controller.SuccessfulCreatePodReason, "Created attachment pod %s", pod.Name)
	return pod, nil
}

func (c *Controller) triggerHotplugPopulation(volume *v1.Volume, vmi *v1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod) common.SyncError {
	populateHotplugPodTemplate, err := c.createAttachmentPopulateTriggerPodTemplate(volume, virtLauncherPod, vmi)
	if err != nil {
		return common.NewSyncError(fmt.Errorf("Error creating trigger pod template %v", err), controller.FailedCreatePodReason)
	}
	if populateHotplugPodTemplate != nil { // nil means the PVC is not populated yet.
		vmiKey := controller.VirtualMachineInstanceKey(vmi)
		_, err = c.createPod(vmiKey, vmi.Namespace, populateHotplugPodTemplate)
		if err != nil {
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedCreatePodReason, "Error creating hotplug population trigger pod for volume %s: %v", volume.Name, err)
			return common.NewSyncError(fmt.Errorf("Error creating hotplug population trigger pod %v", err), controller.FailedCreatePodReason)
		}
		c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, controller.SuccessfulCreatePodReason, "Created hotplug trigger pod for volume %s", volume.Name)
	}
	return nil
}

func syncHotplugCondition(vmi *v1.VirtualMachineInstance, conditionType v1.VirtualMachineInstanceConditionType) {
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()
	condition := v1.VirtualMachineInstanceCondition{
		Type:   conditionType,
		Status: k8sv1.ConditionTrue,
	}
	if !vmiConditions.HasCondition(vmi, condition.Type) {
		vmiConditions.UpdateCondition(vmi, &condition)
		log.Log.Object(vmi).V(4).Infof("adding hotplug condition %s", conditionType)
	}
}

func canMoveToAttachedPhase(currentPhase v1.VolumePhase) bool {
	return currentPhase == "" || currentPhase == v1.VolumeBound || currentPhase == v1.VolumePending
}

func findAttachmentPodByVolumeName(volumeName string, attachmentPods []*k8sv1.Pod) *k8sv1.Pod {
	for _, pod := range attachmentPods {
		for _, podVolume := range pod.Spec.Volumes {
			if podVolume.Name == volumeName {
				return pod
			}
		}
	}
	return nil
}

func (c *Controller) createAttachmentPodTemplate(vmi *v1.VirtualMachineInstance, virtlauncherPod *k8sv1.Pod, volumes []*v1.Volume) (*k8sv1.Pod, error) {
	logger := log.Log.Object(vmi)

	var hasContainerDisk bool
	var newVolumes []*v1.Volume
	for _, volume := range volumes {
		if volume.VolumeSource.ContainerDisk != nil {
			hasContainerDisk = true
			continue
		}
		newVolumes = append(newVolumes, volume)
	}

	volumeNamesPVCMap, err := storagetypes.VirtVolumesToPVCMap(volumes, c.pvcIndexer, virtlauncherPod.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get PVC map: %v", err)
	}
	for volumeName, pvc := range volumeNamesPVCMap {
		//Verify the PVC is ready to be used.
		populated, err := cdiv1.IsSucceededOrPendingPopulation(pvc, func(name, namespace string) (*cdiv1.DataVolume, error) {
			dv, exists, _ := c.dataVolumeIndexer.GetByKey(fmt.Sprintf("%s/%s", namespace, name))
			if !exists {
				return nil, fmt.Errorf("unable to find datavolume %s/%s", namespace, name)
			}
			return dv.(*cdiv1.DataVolume), nil
		})
		if err != nil {
			return nil, err
		}
		if !populated {
			logger.Infof("Unable to hotplug, claim %s found, but not ready", pvc.Name)
			delete(volumeNamesPVCMap, volumeName)
		}
	}

	if len(volumeNamesPVCMap) > 0 || hasContainerDisk {
		return c.templateService.RenderHotplugAttachmentPodTemplate(volumes, virtlauncherPod, vmi, volumeNamesPVCMap)
	}
	return nil, err
}

func (c *Controller) createAttachmentPopulateTriggerPodTemplate(volume *v1.Volume, virtlauncherPod *k8sv1.Pod, vmi *v1.VirtualMachineInstance) (*k8sv1.Pod, error) {
	claimName := storagetypes.PVCNameFromVirtVolume(volume)
	if claimName == "" {
		return nil, errors.New("Unable to hotplug, claim not PVC or Datavolume")
	}

	pvc, exists, isBlock, err := storagetypes.IsPVCBlockFromStore(c.pvcIndexer, virtlauncherPod.Namespace, claimName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("Unable to trigger hotplug population, claim %s not found", claimName)
	}
	pod, err := c.templateService.RenderHotplugAttachmentTriggerPodTemplate(volume, virtlauncherPod, vmi, pvc.Name, isBlock, true)
	return pod, err
}

func (c *Controller) deleteAllAttachmentPods(vmi *v1.VirtualMachineInstance) error {
	virtlauncherPod, err := controller.CurrentVMIPod(vmi, c.podIndexer)
	if err != nil {
		return err
	}
	if virtlauncherPod != nil {
		attachmentPods, err := controller.AttachmentPods(virtlauncherPod, c.podIndexer)
		if err != nil {
			return err
		}
		for _, attachmentPod := range attachmentPods {
			err := c.deleteAttachmentPod(vmi, attachmentPod)
			if err != nil && !k8serrors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}

func (c *Controller) deleteOrphanedAttachmentPods(vmi *v1.VirtualMachineInstance) error {
	pods, err := c.listPodsFromNamespace(vmi.Namespace)
	if err != nil {
		return fmt.Errorf("failed to list pods from namespace %s: %v", vmi.Namespace, err)
	}

	for _, pod := range pods {
		if !controller.IsControlledBy(pod, vmi) {
			continue
		}

		if !controller.PodIsDown(pod) {
			continue
		}

		attachmentPods, err := controller.AttachmentPods(pod, c.podIndexer)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to get attachment pods %s: %v", controller.PodKey(pod), err)
			// do not return; continue the cleanup...
			continue
		}

		for _, attachmentPod := range attachmentPods {
			if err := c.deleteAttachmentPod(vmi, attachmentPod); err != nil {
				log.Log.Reason(err).Errorf("failed to delete attachment pod %s: %v", controller.PodKey(attachmentPod), err)
				// do not return; continue the cleanup...
			}
		}
	}

	return nil
}

func (c *Controller) deleteAttachmentPod(vmi *v1.VirtualMachineInstance, attachmentPod *k8sv1.Pod) error {
	if attachmentPod.DeletionTimestamp != nil {
		return nil
	}

	vmiKey := controller.VirtualMachineInstanceKey(vmi)

	err := c.deletePod(vmiKey, attachmentPod, metav1.DeleteOptions{
		GracePeriodSeconds: pointer.P(int64(0)),
	})
	if err != nil {
		c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedDeletePodReason, "Failed to delete attachment pod %s", attachmentPod.Name)
		return err
	}
	c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, controller.SuccessfulDeletePodReason, "Deleted attachment pod %s", attachmentPod.Name)
	return nil
}

func podVolumesMatchesReadyVolumes(attachmentPod *k8sv1.Pod, volumes []*v1.Volume) bool {
	const (
		// -2 for empty dir and token
		subVols = 2
		// -4 for hotplug with ContainerDisk. 3 empty dir + token
		subVolsWithContainerDisk = 4
	)
	containerDisksNames := make(map[string]struct{})
	for _, ctr := range attachmentPod.Spec.Containers {
		if name, ok := attachmentPod.GetAnnotations()[ctr.Name]; ok {
			containerDisksNames[name] = struct{}{}
		}
	}

	var sub = subVols
	if len(containerDisksNames) > 0 {
		sub = subVolsWithContainerDisk
	}

	countAttachmentVolumes := len(attachmentPod.Spec.Volumes) - sub + len(containerDisksNames)

	if countAttachmentVolumes != len(volumes) {
		return false
	}

	podVolumeMap := make(map[string]struct{})
	for _, volume := range attachmentPod.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil {
			podVolumeMap[volume.Name] = struct{}{}
		}
	}

	for _, volume := range volumes {
		if container_disk.IsHotplugContainerDisk(volume) {
			delete(containerDisksNames, volume.Name)
			continue
		}
		delete(podVolumeMap, volume.Name)
	}
	return len(podVolumeMap) == 0 && len(containerDisksNames) == 0
}

func hasPendingPods(pods []*k8sv1.Pod) bool {
	for _, pod := range pods {
		if pod.Status.Phase == k8sv1.PodRunning || pod.Status.Phase == k8sv1.PodSucceeded || pod.Status.Phase == k8sv1.PodFailed {
			continue
		}
		return true
	}
	return false
}

func (c *Controller) requeueAfter(oldPods []*k8sv1.Pod, threshold time.Duration) (bool, time.Duration) {
	if len(oldPods) > 0 && oldPods[0].CreationTimestamp.Time.After(time.Now().Add(-1*threshold)) {
		return true, threshold - time.Since(oldPods[0].CreationTimestamp.Time)
	}
	return false, 0
}
