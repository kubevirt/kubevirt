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

	k8sv1 "k8s.io/api/core/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	container_disk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/controller"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
)

func needsHandleVolumeHotplug(hotplugVolumes []*v1.Volume, hotplugAttachmentPods []*k8sv1.Pod) bool {
	if len(hotplugAttachmentPods) > 1 {
		return true
	}
	// Determine if the ready volumes have changed compared to the current pod
	if len(hotplugAttachmentPods) == 1 && podVolumesMatchesReadyVolumes(hotplugAttachmentPods[0], hotplugVolumes) {
		return false
	}
	return len(hotplugVolumes) > 0 || len(hotplugAttachmentPods) > 0
}

func getActiveAndOldAttachmentPodsForVolumes(readyHotplugVolumes []*v1.Volume, hotplugAttachmentPods []*k8sv1.Pod) (*k8sv1.Pod, []*k8sv1.Pod) {
	return getActiveAndOldAttachmentPods(hotplugAttachmentPods, func(attachmentPod *k8sv1.Pod) bool {
		return podVolumesMatchesReadyVolumes(attachmentPod, readyHotplugVolumes)
	})
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

func (c *Controller) getReadyHotplugVolumes(hotplugVolumes []*v1.Volume, vmi *v1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod, dataVolumes []*cdiv1.DataVolume) ([]*v1.Volume, common.SyncError) {
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
			return nil, common.NewSyncError(fmt.Errorf("Error determining volume status %v", err), controller.PVCNotReadyReason)
		}
		if wffc {
			// Volume in WaitForFirstConsumer, it has not been populated by CDI yet. create a dummy pod
			logger.V(1).Infof("Volume %s/%s is in WaitForFistConsumer, triggering population", vmi.Namespace, volume.Name)
			syncError := c.triggerHotplugPopulation(volume, vmi, virtLauncherPod)
			if syncError != nil {
				return nil, syncError
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
	return readyHotplugVolumes, nil
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
