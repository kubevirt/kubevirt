package vmi

import (
	"fmt"
	"sort"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
)

func needsHandleVolumeOrResourceClaimHotplug(hotplugVolumes []*v1.Volume, hotplugResourceClaims []*v1.ResourceClaim, hotplugAttachmentPods []*k8sv1.Pod) bool {
	return needsHandleVolumeHotplug(hotplugVolumes, hotplugAttachmentPods) || needsHandleResourceClaimHotplug(hotplugResourceClaims, hotplugAttachmentPods)
}

func (c *Controller) handleHotplugs(hotplugVolumes []*v1.Volume, hotplugResourceClaims []*v1.ResourceClaim, hotplugAttachmentPods []*k8sv1.Pod, vmi *v1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod, dataVolumes []*cdiv1.DataVolume) common.SyncError {
	logger := log.Log.Object(vmi)

	readyHotplugVolumes, err := c.getReadyHotplugVolumes(hotplugVolumes, vmi, virtLauncherPod, dataVolumes)
	if err != nil {
		return err
	}
	readyResourceClaims, err := c.getReadyHotplugResourceClaims(hotplugResourceClaims, vmi, virtLauncherPod)
	if err != nil {
		return err
	}

	currentPod, oldPods := getActiveAndOldAttachmentPods(hotplugAttachmentPods, func(attachmentPod *k8sv1.Pod) bool {
		return podVolumesMatchesReadyVolumes(attachmentPod, readyHotplugVolumes) && podResourceClaimsMatchesReadyResourceClaims(attachmentPod, readyResourceClaims)
	})

	if currentPod == nil && !hasPendingPods(oldPods) && (len(readyHotplugVolumes) > 0 || len(readyResourceClaims) > 0) {
		// The threshold defines how long we should delay requeueing based on the number
		// of ready hotplug volumes and resource claims.
		//
		// The delay scales linearly:
		//   total = len(readyHotplugVolumes) + len(readyResourceClaims)
		//   a threshold = (total / 10) seconds
		//
		// Note: integer division is used, so the delay increases only for each
		// group of 10 objects:
		//   0–9   -> 0s
		//   10–19 -> 1s
		//   20–29 -> 2s
		//
		// The rate limit is applied only if the oldest pod was created less than
		// `threshold` ago. In that case, reconciliation is requeued for the
		// remaining time within that window.
		//
		// This provides a simple rate limiting mechanism to avoid excessive
		// reconcile retries when many attachments are being processed
		threshold := time.Duration((len(readyHotplugVolumes)+len(readyResourceClaims))/10) * time.Second
		if rateLimited, waitTime := c.requeueAfter(oldPods, threshold); rateLimited {
			key, err := controller.KeyFunc(vmi)
			if err != nil {
				logger.Object(vmi).Reason(err).Error("failed to extract key from virtualmachine.")
				return common.NewSyncError(fmt.Errorf("failed to extract key from virtualmachine. %v", err), controller.FailedHotplugSyncReason)
			}
			c.Queue.AddAfter(key, waitTime)
		} else {
			if newPod, err := c.createAttachmentPod(vmi, virtLauncherPod, readyHotplugVolumes, readyResourceClaims); err != nil {
				return err
			} else {
				currentPod = newPod
			}
		}
	}
	if err := c.cleanupAttachmentPods(currentPod, oldPods, vmi, len(readyHotplugVolumes), len(readyResourceClaims)); err != nil {
		return err
	}

	return nil
}

// cleanupAttachmentPods deletes all old attachment pods when the following is true
// 1. There is a currentPod that is running. (not nil and phase.Status == Running)
// 2. There are no readyVolumes (numReadyVolumes == 0) and no readyDevices (numReadyDevices == 0)
// 3. The newest oldPod is not running and not marked for deletion.
// If any of those are true, it will not delete the newest oldPod, since that one is the latest
// pod that is closest to the desired state.
func (c *Controller) cleanupAttachmentPods(currentPod *k8sv1.Pod, oldPods []*k8sv1.Pod, vmi *v1.VirtualMachineInstance, numReadyVolumes, numReadyDevices int) common.SyncError {
	foundRunning := false

	volumeStatusMap := make(map[string]v1.VolumeStatus)
	for _, vs := range vmi.Status.VolumeStatus {
		if vs.HotplugVolume != nil {
			volumeStatusMap[vs.Name] = vs
		}
	}

	for _, vmiVolume := range vmi.Spec.Volumes {
		if storagetypes.IsHotplugVolume(&vmiVolume) {
			delete(volumeStatusMap, vmiVolume.Name)
		}
	}

	deviceStatusMap := make(map[string]v1.DeviceStatusInfo)
	if vmi.Status.DeviceStatus != nil {
		for _, ds := range vmi.Status.DeviceStatus.HostDeviceStatuses {
			if ds.Hotplug != nil {
				deviceStatusMap[ds.Name] = ds
			}
		}
	}

	for _, vmiRc := range vmi.Spec.ResourceClaims {
		if vmiRc.Hotpluggable {
			delete(deviceStatusMap, vmiRc.Name)
		}
	}

	currentPodIsNotRunning := currentPod == nil || currentPod.Status.Phase != k8sv1.PodRunning
	for _, attachmentPod := range oldPods {
		if !foundRunning &&
			attachmentPod.Status.Phase == k8sv1.PodRunning && attachmentPod.DeletionTimestamp == nil &&
			(numReadyVolumes > 0 || numReadyDevices > 0) &&
			currentPodIsNotRunning {
			foundRunning = true
			continue
		}
		volumesNotReadyForDelete := 0

		for _, podVolume := range attachmentPod.Spec.Volumes {
			volumeStatus, ok := volumeStatusMap[podVolume.Name]
			if ok && !volumeReadyForPodDelete(volumeStatus.Phase) {
				volumesNotReadyForDelete++
			}
		}

		if volumesNotReadyForDelete > 0 {
			log.Log.Object(vmi).V(3).Infof("Not deleting attachment pod %s, because there are still %d volumes to be unmounted", attachmentPod.Name, volumesNotReadyForDelete)
			continue
		}

		devicesNotReadyForDelete := 0

		for _, podResourceClaim := range attachmentPod.Spec.ResourceClaims {
			deviceStatus, ok := deviceStatusMap[podResourceClaim.Name]
			if ok && !deviceReadyForPodDelete(deviceStatus.Phase) {
				devicesNotReadyForDelete++
			}
		}

		if devicesNotReadyForDelete > 0 {
			log.Log.Object(vmi).V(3).Infof("Not deleting attachment pod %s, because there are still %d devices to be detached", attachmentPod.Name, devicesNotReadyForDelete)
			continue
		}

		if err := c.deleteAttachmentPod(vmi, attachmentPod); err != nil {
			return common.NewSyncError(fmt.Errorf("Error deleting attachment pod %v", err), controller.FailedDeletePodReason)
		}

		log.Log.Object(vmi).V(3).Infof("Deleted attachment pod %s", attachmentPod.Name)
	}
	return nil
}

func (c *Controller) createAttachmentPod(vmi *v1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod, volumes []*v1.Volume, resourceClaims []*v1.ResourceClaim) (*k8sv1.Pod, common.SyncError) {
	attachmentPodTemplate, err := c.createAttachmentPodTemplate(vmi, virtLauncherPod, volumes, resourceClaims)
	if err != nil {
		return nil, common.NewSyncError(fmt.Errorf("Error rendering attachment pod template %v", err), controller.FailedCreatePodReason)
	}
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

func (c *Controller) createAttachmentPodTemplate(vmi *v1.VirtualMachineInstance, virtlauncherPod *k8sv1.Pod, volumes []*v1.Volume, resourceClaims []*v1.ResourceClaim) (*k8sv1.Pod, error) {
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

	volumeNamesPVCMap, err := storagetypes.VirtVolumesToPVCMap(newVolumes, c.pvcIndexer, virtlauncherPod.Namespace)
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

	if len(volumeNamesPVCMap) > 0 || hasContainerDisk || len(resourceClaims) > 0 {
		return c.templateService.RenderHotplugAttachmentPodTemplate(volumes, resourceClaims, virtlauncherPod, vmi, volumeNamesPVCMap)
	}
	return nil, err
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

func getActiveAndOldAttachmentPods(hotplugAttachmentPods []*k8sv1.Pod, match func(attachmentPod *k8sv1.Pod) bool) (*k8sv1.Pod, []*k8sv1.Pod) {
	sort.Slice(hotplugAttachmentPods, func(i, j int) bool {
		return hotplugAttachmentPods[i].CreationTimestamp.Time.Before(hotplugAttachmentPods[j].CreationTimestamp.Time)
	})

	var currentPod *k8sv1.Pod
	oldPods := make([]*k8sv1.Pod, 0)
	for _, attachmentPod := range hotplugAttachmentPods {
		if !match(attachmentPod) {
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
