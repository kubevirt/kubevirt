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
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"

	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/trace"

	virtv1 "kubevirt.io/api/core/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"k8s.io/apimachinery/pkg/api/equality"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/util/migrations"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/descheduler"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
)

func (c *Controller) sync(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod, dataVolumes []*cdiv1.DataVolume) (common.SyncError, *k8sv1.Pod) {
	key := controller.VirtualMachineInstanceKey(vmi)
	defer virtControllerVMIWorkQueueTracer.StepTrace(key, "sync", trace.Field{Key: "VMI Name", Value: vmi.Name})

	if vmi.DeletionTimestamp != nil {
		err := c.deleteAllMatchingPods(vmi)
		if err != nil {
			return common.NewSyncError(fmt.Errorf("failed to delete pod: %v", err), controller.FailedDeletePodReason), pod
		}
		return nil, pod
	}

	if vmi.IsFinal() {
		err := c.deleteAllAttachmentPods(vmi)
		if err != nil {
			return common.NewSyncError(fmt.Errorf("failed to delete attachment pods: %v", err), controller.FailedHotplugSyncReason), pod
		}
		return nil, pod
	}

	if err := c.deleteOrphanedAttachmentPods(vmi); err != nil {
		log.Log.Reason(err).Errorf("failed to delete orphaned attachment pods %s: %v", key, err)
		// do not return; just log the error
	}

	dataVolumesReady, isWaitForFirstConsumer, syncErr := c.areDataVolumesReady(vmi, dataVolumes)
	if syncErr != nil {
		return syncErr, pod
	}

	if !controller.PodExists(pod) {
		// If we came ever that far to detect that we already created a pod, we don't create it again
		if !vmi.IsUnprocessed() {
			return nil, pod
		}
		// let's check if we already have topology hints or if we are still waiting for them
		if vmi.Status.TopologyHints == nil && c.topologyHinter.IsTscFrequencyRequired(vmi) {
			log.Log.V(3).Object(vmi).Infof("Delaying pod creation until topology hints are set")
			return nil, pod
		}
		// ensure that all dataVolumes associated with the VMI are ready before creating the pod
		if !dataVolumesReady {
			log.Log.V(3).Object(vmi).Infof("Delaying pod creation while DataVolume populates or while we wait for PVCs to appear.")
			return nil, pod
		}
		// ensure the VMI doesn't have an unfinished migration before creating the pod
		activeMigration, err := migrations.ActiveMigrationExistsForVMI(c.migrationIndexer, vmi)
		if err != nil {
			return common.NewSyncError(err, controller.FailedCreatePodReason), pod
		}
		if activeMigration {
			log.Log.V(3).Object(vmi).Infof("Delaying pod creation because an active migration exists for the VMI.")
			// We still need to return an error to ensure the VMI gets re-enqueued
			return common.NewSyncError(fmt.Errorf("active migration exists"), controller.FailedCreatePodReason), pod
		}

		backendStoragePVCName, syncErr := c.handleBackendStorage(vmi)
		if syncErr != nil {
			return syncErr, pod
		}
		// If a backend-storage PVC was just created but not yet seen by the informer, give it time
		if !c.pvcExpectations.SatisfiedExpectations(key) {
			return nil, pod
		}
		backendStorageReady, err := c.backendStorage.IsPVCReady(vmi, backendStoragePVCName)
		if err != nil {
			return common.NewSyncError(err, controller.FailedBackendStorageProbeReason), pod
		}
		if !backendStorageReady {
			log.Log.V(2).Object(vmi).Infof("Delaying pod creation while backend storage populates.")
			return common.NewSyncError(fmt.Errorf("PVC pending"), controller.BackendStorageNotReadyReason), pod
		}

		var templatePod *k8sv1.Pod
		if isWaitForFirstConsumer {
			log.Log.V(3).Object(vmi).Infof("Scheduling temporary pod for WaitForFirstConsumer DV")
			templatePod, err = c.templateService.RenderLaunchManifestNoVm(vmi)
		} else {
			templatePod, err = c.templateService.RenderLaunchManifest(vmi)
		}
		if _, ok := err.(storagetypes.PvcNotFoundError); ok {
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedPvcNotFoundReason, services.FailedToRenderLaunchManifestErrFormat, err)
			return &informalSyncError{fmt.Errorf(services.FailedToRenderLaunchManifestErrFormat, err), controller.FailedPvcNotFoundReason}, pod
		} else if err != nil {
			return common.NewSyncError(fmt.Errorf(services.FailedToRenderLaunchManifestErrFormat, err), controller.FailedCreatePodReason), pod
		}

		var validateErrors []error
		for _, cause := range c.validateNetworkSpec(k8sfield.NewPath("spec"), &vmi.Spec, c.clusterConfig) {
			validateErrors = append(validateErrors, errors.New(cause.String()))
		}
		if validateErr := errors.Join(validateErrors...); validateErrors != nil {
			return common.NewSyncError(fmt.Errorf("failed create validation: %v", validateErr), "FailedCreateValidation"), pod
		}

		vmiKey := controller.VirtualMachineInstanceKey(vmi)
		pod, err := c.createPod(vmiKey, vmi.Namespace, templatePod)
		if k8serrors.IsForbidden(err) && strings.Contains(err.Error(), "violates PodSecurity") {
			psaErr := fmt.Errorf("failed to create pod for vmi %s/%s, it needs a privileged namespace to run: %w", vmi.GetNamespace(), vmi.GetName(), err)
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedCreatePodReason, services.FailedToRenderLaunchManifestErrFormat, psaErr)
			return common.NewSyncError(psaErr, controller.FailedCreatePodReason), nil
		}
		if err != nil {
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedCreatePodReason, "Error creating pod: %v", err)
			return common.NewSyncError(fmt.Errorf("failed to create virtual machine pod: %v", err), controller.FailedCreatePodReason), nil
		}
		c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, controller.SuccessfulCreatePodReason, "Created virtual machine pod %s", pod.Name)
		return nil, pod
	}

	if !isWaitForFirstConsumer {
		err := c.cleanupWaitForFirstConsumerTemporaryPods(vmi, pod)
		if err != nil {
			return common.NewSyncError(fmt.Errorf("failed to clean up temporary pods: %v", err), controller.FailedHotplugSyncReason), pod
		}
	}

	if !isTempPod(pod) && controller.IsPodReady(pod) {
		newAnnotations := map[string]string{descheduler.EvictOnlyAnnotation: ""}
		maps.Copy(newAnnotations, c.netAnnotationsGenerator.GenerateFromActivePod(vmi, pod))
		patchedPod, err := c.syncPodAnnotations(pod, newAnnotations)
		if err != nil {
			return common.NewSyncError(err, controller.FailedPodPatchReason), pod
		}
		pod = patchedPod

		hotplugVolumes := controller.GetHotplugVolumes(vmi, pod)
		hotplugAttachmentPods, err := controller.AttachmentPods(pod, c.podIndexer)
		if err != nil {
			return common.NewSyncError(fmt.Errorf("failed to get attachment pods: %v", err), controller.FailedHotplugSyncReason), pod
		}

		if pod.DeletionTimestamp == nil && needsHandleHotplug(hotplugVolumes, hotplugAttachmentPods) {
			var hotplugSyncErr common.SyncError
			hotplugSyncErr = c.handleHotplugVolumes(hotplugVolumes, hotplugAttachmentPods, vmi, pod, dataVolumes)
			if hotplugSyncErr != nil {
				if hotplugSyncErr.Reason() == controller.MissingAttachmentPodReason {
					// We are missing an essential hotplug pod. Delete all pods associated with the VMI.
					if err := c.deleteAllMatchingPods(vmi); err != nil {
						log.Log.Warningf("failed to deleted VMI %s pods: %v", vmi.GetUID(), err)
					}
				} else {
					return hotplugSyncErr, pod
				}
			}
		}
	}
	return nil, pod
}

// updateStatus handles the VMI's lifecycle status updates.
func (c *Controller) updateStatus(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod, dataVolumes []*cdiv1.DataVolume, syncErr common.SyncError) error {
	key := controller.VirtualMachineInstanceKey(vmi)
	defer virtControllerVMIWorkQueueTracer.StepTrace(key, "updateStatus", trace.Field{Key: "VMI Name", Value: vmi.Name})

	hasFailedDataVolume := storagetypes.HasFailedDataVolumes(dataVolumes)

	// there is no reason to check for waitForFirstConsumer is there are failed DV's
	hasWffcDataVolume := !hasFailedDataVolume && storagetypes.HasWFFCDataVolumes(dataVolumes)
	conditionManager := controller.NewVirtualMachineInstanceConditionManager()
	podConditionManager := controller.NewPodConditionManager()

	vmiCopy := vmi.DeepCopy()
	vmiPodExists := controller.PodExists(pod) && !isTempPod(pod)
	tempPodExists := controller.PodExists(pod) && isTempPod(pod)

	vmiCopy, err := c.setActivePods(vmiCopy)
	if err != nil {
		return fmt.Errorf("Error detecting vmi pods: %v", err)
	}

	c.syncReadyConditionFromPod(vmiCopy, pod)
	if vmiPodExists {
		var foundImage string
		for _, container := range pod.Spec.Containers {
			if container.Name == "compute" {
				foundImage = container.Image
				break
			}
		}
		vmiCopy = c.setLauncherContainerInfo(vmiCopy, foundImage)

		if err := c.syncPausedConditionToPod(vmiCopy, pod); err != nil {
			return fmt.Errorf("error syncing paused condition to pod: %v", err)
		}

		if err := c.syncDynamicLabelsToPod(vmiCopy, pod); err != nil {
			return fmt.Errorf("error syncing labels to pod: %v", err)
		}
	}

	aggregateDataVolumesConditions(vmiCopy, dataVolumes)

	if pvc := backendstorage.PVCForVMI(c.pvcIndexer, vmi); pvc != nil {
		c.backendStorage.UpdateVolumeStatus(vmiCopy, pvc)
	}

	switch {
	case vmi.IsUnprocessed():
		if vmiPodExists {
			vmiCopy.Status.Phase = virtv1.Scheduling
		} else if vmi.DeletionTimestamp != nil || hasFailedDataVolume {
			vmiCopy.Status.Phase = virtv1.Failed
		} else {
			vmiCopy.Status.Phase = virtv1.Pending
			if vmi.Status.TopologyHints == nil {
				if topologyHints, tscRequirement, err := c.topologyHinter.TopologyHintsForVMI(vmi); err != nil && tscRequirement == topology.RequiredForBoot {
					c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedGatherhingClusterTopologyHints, err.Error())
					return common.NewSyncError(err, controller.FailedGatherhingClusterTopologyHints)
				} else if topologyHints != nil {
					vmiCopy.Status.TopologyHints = topologyHints
				}
			}
			if hasWffcDataVolume {
				condition := virtv1.VirtualMachineInstanceCondition{
					Type:   virtv1.VirtualMachineInstanceProvisioning,
					Status: k8sv1.ConditionTrue,
				}
				if !conditionManager.HasCondition(vmiCopy, condition.Type) {
					vmiCopy.Status.Conditions = append(vmiCopy.Status.Conditions, condition)
				}
				if tempPodExists {
					// Add PodScheduled False condition to the VM
					if podConditionManager.HasConditionWithStatus(pod, k8sv1.PodScheduled, k8sv1.ConditionFalse) {
						conditionManager.AddPodCondition(vmiCopy, podConditionManager.GetCondition(pod, k8sv1.PodScheduled))
					} else if conditionManager.HasCondition(vmiCopy, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled)) {
						// Remove PodScheduling condition from the VM
						conditionManager.RemoveCondition(vmiCopy, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled))
					}
					if controller.IsPodFailedOrGoingDown(pod) {
						vmiCopy.Status.Phase = virtv1.Failed
					}
				}
			}
			if syncErr != nil && syncErr.Reason() == controller.FailedPvcNotFoundReason {
				condition := virtv1.VirtualMachineInstanceCondition{
					Type:    virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled),
					Reason:  k8sv1.PodReasonUnschedulable,
					Message: syncErr.Error(),
					Status:  k8sv1.ConditionFalse,
				}
				if conditionManager.HasCondition(vmiCopy, condition.Type) {
					conditionManager.RemoveCondition(vmiCopy, condition.Type)
				}
				vmiCopy.Status.Conditions = append(vmiCopy.Status.Conditions, condition)
			}
		}
	case vmi.IsScheduling():
		// Remove InstanceProvisioning condition from the VM
		if conditionManager.HasCondition(vmiCopy, virtv1.VirtualMachineInstanceProvisioning) {
			conditionManager.RemoveCondition(vmiCopy, virtv1.VirtualMachineInstanceProvisioning)
		}
		if vmiPodExists {
			// ensure that the QOS class on the VMI matches to Pods QOS class
			if pod.Status.QOSClass == "" {
				vmiCopy.Status.QOSClass = nil
			} else {
				vmiCopy.Status.QOSClass = &pod.Status.QOSClass
			}

			// Add PodScheduled False condition to the VM
			if podConditionManager.HasConditionWithStatus(pod, k8sv1.PodScheduled, k8sv1.ConditionFalse) {
				conditionManager.AddPodCondition(vmiCopy, podConditionManager.GetCondition(pod, k8sv1.PodScheduled))
			} else if conditionManager.HasCondition(vmiCopy, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled)) {
				// Remove PodScheduling condition from the VM
				conditionManager.RemoveCondition(vmiCopy, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled))
			}

			if imageErr := checkForContainerImageError(pod); imageErr != nil {
				// only overwrite syncErr if imageErr != nil
				syncErr = imageErr
			}

			if controller.IsPodReady(pod) && vmi.DeletionTimestamp == nil {
				// fail vmi creation if CPU pinning has been requested but the Pod QOS is not Guaranteed
				podQosClass := pod.Status.QOSClass
				if podQosClass != k8sv1.PodQOSGuaranteed && vmi.IsCPUDedicated() {
					c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedGuaranteePodResourcesReason, "failed to guarantee pod resources")
					syncErr = common.NewSyncError(fmt.Errorf("failed to guarantee pod resources"), controller.FailedGuaranteePodResourcesReason)
					break
				}

				// Storage
				// Initialize the volume status field with information
				// about the PVCs that the VMI is consuming. This prevents
				// virt-handler from needing to make API calls to GET the pvc
				// during reconcile
				if err := c.updateVolumeStatus(vmiCopy, pod); err != nil {
					return err
				}

				// Network
				if err := c.updateNetworkStatus(vmiCopy, pod); err != nil {
					log.Log.Errorf("failed to update the interface status: %v", err)
				}

				// vmi is still owned by the controller but pod is already ready,
				// so let's hand over the vmi too
				vmiCopy.Status.Phase = virtv1.Scheduled
				if vmiCopy.Labels == nil {
					vmiCopy.Labels = map[string]string{}
				}
				vmiCopy.ObjectMeta.Labels[virtv1.NodeNameLabel] = pod.Spec.NodeName
				vmiCopy.Status.NodeName = pod.Spec.NodeName

				// Set the VMI migration transport now before the VMI can be migrated
				// This status field is needed to support the migration of legacy virt-launchers
				// to newer ones. In an absence of this field on the vmi, the target launcher
				// will set up a TCP proxy, as expected by a legacy virt-launcher.
				if shouldSetMigrationTransport(pod) {
					vmiCopy.Status.MigrationTransport = virtv1.MigrationTransportUnix
				}

				// Allocate the CID if VSOCK is enabled.
				if util.IsAutoAttachVSOCK(vmiCopy) {
					if err := c.cidsMap.Allocate(vmiCopy); err != nil {
						return err
					}
				}
			} else if controller.IsPodDownOrGoingDown(pod) {
				vmiCopy.Status.Phase = virtv1.Failed
			}
		} else {
			// someone other than the controller deleted the pod unexpectedly
			vmiCopy.Status.Phase = virtv1.Failed
		}
	case vmi.IsFinal():
		allDeleted, err := c.allPodsDeleted(vmi)
		if err != nil {
			return err
		}

		if allDeleted {
			log.Log.V(3).Object(vmi).Infof("All pods have been deleted, removing finalizer")
			controller.RemoveFinalizer(vmiCopy, virtv1.VirtualMachineInstanceFinalizer)
			if vmiCopy.Labels != nil {
				delete(vmiCopy.Labels, virtv1.OutdatedLauncherImageLabel)
			}
			vmiCopy.Status.LauncherContainerImageVersion = ""
		}

		if !c.hasOwnerVM(vmi) && len(vmiCopy.Finalizers) > 0 {
			// if there's no owner VM around still, then remove the VM controller's finalizer if it exists
			controller.RemoveFinalizer(vmiCopy, virtv1.VirtualMachineControllerFinalizer)
		}

	case vmi.IsRunning():
		if !vmiPodExists {
			vmiCopy.Status.Phase = virtv1.Failed
			break
		}

		// Storage
		if err := c.updateVolumeStatus(vmiCopy, pod); err != nil {
			return err
		}

		// Network
		if err := c.updateNetworkStatus(vmiCopy, pod); err != nil {
			log.Log.Errorf("failed to update the interface status: %v", err)
		}

		if c.requireCPUHotplug(vmiCopy) {
			syncHotplugCondition(vmiCopy, virtv1.VirtualMachineInstanceVCPUChange)
		}

		if c.requireMemoryHotplug(vmiCopy) {
			c.syncMemoryHotplug(vmiCopy)
		}

		if c.requireVolumesUpdate(vmiCopy) {
			c.syncVolumesUpdate(vmiCopy)
		}

	case vmi.IsScheduled():
		if !vmiPodExists {
			vmiCopy.Status.Phase = virtv1.Failed
			break
		}

		if err := c.updateVolumeStatus(vmiCopy, pod); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown vmi phase %v", vmi.Status.Phase)
	}

	// VMI is owned by virt-handler, so patch instead of update
	if vmi.IsRunning() || vmi.IsScheduled() {
		patchSet := prepareVMIPatch(vmi, vmiCopy)
		if patchSet.IsEmpty() {
			return nil
		}
		patchBytes, err := patchSet.GeneratePayload()
		if err != nil {
			return fmt.Errorf("error preparing VMI patch: %v", err)
		}

		_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, v1.PatchOptions{})
		// We could not retry if the "test" fails but we have no sane way to detect that right now: https://github.com/kubernetes/kubernetes/issues/68202 for details
		// So just retry like with any other errors
		if err != nil {
			return fmt.Errorf("patching of vmi conditions and activePods failed: %v", err)
		}

		return nil
	}

	reason := ""
	if syncErr != nil {
		reason = syncErr.Reason()
	}
	conditionManager.CheckFailure(vmiCopy, syncErr, reason)
	controller.SetVMIPhaseTransitionTimestamp(&vmi.Status, &vmiCopy.Status)

	// If we detect a change on the vmi we update the vmi
	vmiChanged := !equality.Semantic.DeepEqual(vmi.Status, vmiCopy.Status) || !equality.Semantic.DeepEqual(vmi.Finalizers, vmiCopy.Finalizers) || !equality.Semantic.DeepEqual(vmi.Annotations, vmiCopy.Annotations) || !equality.Semantic.DeepEqual(vmi.Labels, vmiCopy.Labels)
	if vmiChanged {
		key := controller.VirtualMachineInstanceKey(vmi)
		c.vmiExpectations.SetExpectations(key, 1, 0)
		_, err := c.clientset.VirtualMachineInstance(vmi.Namespace).Update(context.Background(), vmiCopy, v1.UpdateOptions{})
		if err != nil {
			c.vmiExpectations.LowerExpectations(key, 1, 0)
			return err
		}
	}

	return nil
}

// prepareVMIPatch generates a patch set for updating the VMI status.
func prepareVMIPatch(oldVMI, newVMI *virtv1.VirtualMachineInstance) *patch.PatchSet {
	patchSet := patch.New()

	// TODO(vladikr): Move to storage
	if !equality.Semantic.DeepEqual(newVMI.Status.VolumeStatus, oldVMI.Status.VolumeStatus) {
		// VolumeStatus changed which means either removed or added volumes.
		if oldVMI.Status.VolumeStatus == nil {
			patchSet.AddOption(patch.WithAdd("/status/volumeStatus", newVMI.Status.VolumeStatus))
		} else {
			patchSet.AddOption(
				patch.WithTest("/status/volumeStatus", oldVMI.Status.VolumeStatus),
				patch.WithReplace("/status/volumeStatus", newVMI.Status.VolumeStatus),
			)
		}
		log.Log.V(3).Object(oldVMI).Infof("Patching Volume Status")
	}
	// We don't own the object anymore, so patch instead of update
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()
	if !vmiConditions.ConditionsEqual(oldVMI, newVMI) {
		patchSet.AddOption(
			patch.WithTest("/status/conditions", oldVMI.Status.Conditions),
			patch.WithReplace("/status/conditions", newVMI.Status.Conditions),
		)
		log.Log.V(3).Object(oldVMI).Infof("Patching VMI conditions")
	}

	if !equality.Semantic.DeepEqual(newVMI.Status.ActivePods, oldVMI.Status.ActivePods) {
		patchSet.AddOption(
			patch.WithTest("/status/activePods", oldVMI.Status.ActivePods),
			patch.WithReplace("/status/activePods", newVMI.Status.ActivePods),
		)
		log.Log.V(3).Object(oldVMI).Infof("Patching VMI activePods")
	}

	if newVMI.Status.Phase != oldVMI.Status.Phase {
		patchSet.AddOption(
			patch.WithTest("/status/phase", oldVMI.Status.Phase),
			patch.WithReplace("/status/phase", newVMI.Status.Phase),
		)
		log.Log.V(3).Object(oldVMI).Infof("Patching VMI phase")
	}

	if newVMI.Status.LauncherContainerImageVersion != oldVMI.Status.LauncherContainerImageVersion {
		if oldVMI.Status.LauncherContainerImageVersion == "" {
			patchSet.AddOption(patch.WithAdd("/status/launcherContainerImageVersion", newVMI.Status.LauncherContainerImageVersion))
		} else {
			patchSet.AddOption(
				patch.WithTest("/status/launcherContainerImageVersion", oldVMI.Status.LauncherContainerImageVersion),
				patch.WithReplace("/status/launcherContainerImageVersion", newVMI.Status.LauncherContainerImageVersion),
			)
		}
	}

	if !equality.Semantic.DeepEqual(oldVMI.Labels, newVMI.Labels) {
		if oldVMI.Labels == nil {
			patchSet.AddOption(patch.WithAdd("/metadata/labels", newVMI.Labels))
		} else {
			patchSet.AddOption(
				patch.WithTest("/metadata/labels", oldVMI.Labels),
				patch.WithReplace("/metadata/labels", newVMI.Labels),
			)
		}
	}

	// TODO(vladikr): Move to networking
	if !equality.Semantic.DeepEqual(newVMI.Status.Interfaces, oldVMI.Status.Interfaces) {
		patchSet.AddOption(
			patch.WithTest("/status/interfaces", oldVMI.Status.Interfaces),
			patch.WithAdd("/status/interfaces", newVMI.Status.Interfaces),
		)
		log.Log.V(3).Object(oldVMI).Infof("Patching Interface Status")
	}

	return patchSet
}

// These "dynamic" labels are Pod labels which may diverge from the VMI over time that we want to keep in sync.
func (c *Controller) syncDynamicLabelsToPod(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) error {
	patchSet := patch.New()
	dynamicLabels := []string{
		virtv1.NodeNameLabel,
		virtv1.OutdatedLauncherImageLabel,
	}
	podMeta := pod.ObjectMeta.DeepCopy()
	if podMeta.Labels == nil {
		podMeta.Labels = map[string]string{}
	}
	changed := false
	for _, key := range dynamicLabels {
		vmiVal, vmiLabelExists := vmi.Labels[key]
		podVal, podLabelExists := podMeta.Labels[key]
		if vmiLabelExists == podLabelExists && vmiVal == podVal {
			continue
		}
		changed = true
		if !vmiLabelExists {
			delete(podMeta.Labels, key)
		} else {
			podMeta.Labels[key] = vmiVal
		}
	}
	if !changed {
		return nil
	}
	if pod.ObjectMeta.Labels == nil {
		patchSet.AddOption(patch.WithAdd("/metadata/labels", podMeta.Labels))
	} else {
		patchSet.AddOption(
			patch.WithTest("/metadata/labels", pod.ObjectMeta.Labels),
			patch.WithReplace("/metadata/labels", podMeta.Labels),
		)
	}
	if patchSet.IsEmpty() {
		return nil
	}
	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return err
	}
	if _, err := c.clientset.CoreV1().Pods(pod.Namespace).Patch(context.Background(), pod.Name, types.JSONPatchType, patchBytes, v1.PatchOptions{}); err != nil {
		log.Log.Object(pod).Errorf("failed to sync dynamic pod labels during sync: %v", err)
		return err
	}
	return nil
}

func (c *Controller) syncPodAnnotations(pod *k8sv1.Pod, newAnnotations map[string]string) (*k8sv1.Pod, error) {
	patchSet := patch.New()
	for key, newValue := range newAnnotations {
		if podAnnotationValue, keyExist := pod.Annotations[key]; !keyExist || podAnnotationValue != newValue {
			patchSet.AddOption(
				patch.WithAdd(fmt.Sprintf("/metadata/annotations/%s", patch.EscapeJSONPointer(key)), newValue),
			)
		}
	}
	if patchSet.IsEmpty() {
		return pod, nil
	}
	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return pod, fmt.Errorf("failed to generate patch payload: %w", err)
	}
	patchedPod, err := c.clientset.CoreV1().Pods(pod.Namespace).Patch(context.Background(), pod.Name, types.JSONPatchType, patchBytes, v1.PatchOptions{})
	if err != nil {
		log.Log.Object(pod).Errorf("failed to sync pod annotations during sync: %v", err)
		return nil, err
	}
	return patchedPod, nil
}

func (c *Controller) setLauncherContainerInfo(vmi *virtv1.VirtualMachineInstance, curPodImage string) *virtv1.VirtualMachineInstance {
	if curPodImage != "" && curPodImage != c.templateService.GetLauncherImage() {
		if vmi.Labels == nil {
			vmi.Labels = map[string]string{}
		}
		vmi.Labels[virtv1.OutdatedLauncherImageLabel] = ""
	} else {
		if vmi.Labels != nil {
			delete(vmi.Labels, virtv1.OutdatedLauncherImageLabel)
		}
	}
	vmi.Status.LauncherContainerImageVersion = curPodImage
	return vmi
}

func (c *Controller) hasOwnerVM(vmi *virtv1.VirtualMachineInstance) bool {
	controllerRef := v1.GetControllerOf(vmi)
	if controllerRef == nil || controllerRef.Kind != virtv1.VirtualMachineGroupVersionKind.Kind {
		return false
	}
	obj, exists, _ := c.vmStore.GetByKey(controller.NamespacedKey(vmi.Namespace, controllerRef.Name))
	if !exists {
		return false
	}
	ownerVM := obj.(*virtv1.VirtualMachine)
	return controllerRef.UID == ownerVM.UID
}

func (c *Controller) syncReadyConditionFromPod(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) {
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()
	podConditions := controller.NewPodConditionManager()
	now := v1.Now()
	if pod == nil || isTempPod(pod) {
		vmiConditions.UpdateCondition(vmi, &virtv1.VirtualMachineInstanceCondition{
			Type:               virtv1.VirtualMachineInstanceReady,
			Status:             k8sv1.ConditionFalse,
			Reason:             virtv1.PodNotExistsReason,
			Message:            "virt-launcher pod has not yet been scheduled",
			LastProbeTime:      now,
			LastTransitionTime: now,
		})
	} else if controller.IsPodDownOrGoingDown(pod) {
		vmiConditions.UpdateCondition(vmi, &virtv1.VirtualMachineInstanceCondition{
			Type:               virtv1.VirtualMachineInstanceReady,
			Status:             k8sv1.ConditionFalse,
			Reason:             virtv1.PodTerminatingReason,
			Message:            "virt-launcher pod is terminating",
			LastProbeTime:      now,
			LastTransitionTime: now,
		})
	} else if !vmi.IsRunning() {
		vmiConditions.UpdateCondition(vmi, &virtv1.VirtualMachineInstanceCondition{
			Type:               virtv1.VirtualMachineInstanceReady,
			Status:             k8sv1.ConditionFalse,
			Reason:             virtv1.GuestNotRunningReason,
			Message:            "Guest VM is not reported as running",
			LastProbeTime:      now,
			LastTransitionTime: now,
		})
	} else if podReadyCond := podConditions.GetCondition(pod, k8sv1.PodReady); podReadyCond != nil {
		vmiConditions.UpdateCondition(vmi, &virtv1.VirtualMachineInstanceCondition{
			Type:               virtv1.VirtualMachineInstanceReady,
			Status:             podReadyCond.Status,
			Reason:             podReadyCond.Reason,
			Message:            podReadyCond.Message,
			LastProbeTime:      podReadyCond.LastProbeTime,
			LastTransitionTime: podReadyCond.LastTransitionTime,
		})
	} else {
		vmiConditions.UpdateCondition(vmi, &virtv1.VirtualMachineInstanceCondition{
			Type:               virtv1.VirtualMachineInstanceReady,
			Status:             k8sv1.ConditionFalse,
			Reason:             virtv1.PodConditionMissingReason,
			Message:            "virt-launcher pod is missing the Ready condition",
			LastProbeTime:      now,
			LastTransitionTime: now,
		})
	}
}

func (c *Controller) syncPausedConditionToPod(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) error {
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()
	podConditions := controller.NewPodConditionManager()
	podCopy := pod.DeepCopy()
	now := v1.Now()
	if vmiConditions.HasConditionWithStatus(vmi, virtv1.VirtualMachineInstancePaused, k8sv1.ConditionTrue) {
		if podConditions.HasConditionWithStatus(pod, virtv1.VirtualMachineUnpaused, k8sv1.ConditionTrue) {
			podConditions.UpdateCondition(podCopy, &k8sv1.PodCondition{
				Type:               virtv1.VirtualMachineUnpaused,
				Status:             k8sv1.ConditionFalse,
				Reason:             "Paused",
				Message:            "the virtual machine is paused",
				LastProbeTime:      now,
				LastTransitionTime: now,
			})
		}
	} else {
		if !podConditions.HasConditionWithStatus(pod, virtv1.VirtualMachineUnpaused, k8sv1.ConditionTrue) {
			podConditions.UpdateCondition(podCopy, &k8sv1.PodCondition{
				Type:               virtv1.VirtualMachineUnpaused,
				Status:             k8sv1.ConditionTrue,
				Reason:             "NotPaused",
				Message:            "the virtual machine is not paused",
				LastProbeTime:      now,
				LastTransitionTime: now,
			})
		}
	}
	patchSet := preparePodPatch(pod, podCopy)
	if patchSet.IsEmpty() {
		return nil
	}
	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return fmt.Errorf("error preparing pod patch: %v", err)
	}
	log.Log.V(3).Object(pod).Infof("Patching pod conditions")
	_, err = c.clientset.CoreV1().Pods(pod.Namespace).Patch(context.TODO(), pod.Name, types.JSONPatchType, patchBytes, v1.PatchOptions{}, "status")
	// We could not retry if the "test" fails but we have no sane way to detect that right now:
	// https://github.com/kubernetes/kubernetes/issues/68202 for details
	// So just retry like with any other errors
	if err != nil {
		log.Log.Object(pod).Errorf("Patching of pod conditions failed: %v", err)
		return fmt.Errorf("patching of pod conditions failed: %v", err)
	}
	return nil
}

// checkForContainerImageError checks if an error has occured while handling the image of any of the pod's containers
// (including init containers), and returns a syncErr with the details of the error, or nil otherwise.
func checkForContainerImageError(pod *k8sv1.Pod) common.SyncError {
	containerStatuses := append(append([]k8sv1.ContainerStatus{}, pod.Status.InitContainerStatuses...), pod.Status.ContainerStatuses...)
	for _, containerStatus := range containerStatuses {
		if containerStatus.State.Waiting == nil {
			continue
		}
		reason := containerStatus.State.Waiting.Reason
		if reason == controller.ErrImagePullReason || reason == controller.ImagePullBackOffReason {
			return common.NewSyncError(fmt.Errorf(containerStatus.State.Waiting.Message), reason)
		}
	}
	return nil
}

func (c *Controller) deleteAllMatchingPods(vmi *virtv1.VirtualMachineInstance) error {
	pods, err := c.listPodsFromNamespace(vmi.Namespace)
	if err != nil {
		return err
	}
	vmiKey := controller.VirtualMachineInstanceKey(vmi)
	for _, pod := range pods {
		if pod.DeletionTimestamp != nil && !isPodFinal(pod) || !controller.IsControlledBy(pod, vmi) {
			continue
		}
		if err = c.deletePod(vmiKey, pod, v1.DeleteOptions{}); err != nil {
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedDeletePodReason, "Failed to delete virtual machine pod %s", pod.Name)
			return err
		}
		c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, controller.SuccessfulDeletePodReason, "Deleted virtual machine pod %s", pod.Name)
	}
	return nil
}

func isPodFinal(pod *k8sv1.Pod) bool {
	return pod.Status.Phase == k8sv1.PodSucceeded || pod.Status.Phase == k8sv1.PodFailed
}

func (c *Controller) listPodsFromNamespace(namespace string) ([]*k8sv1.Pod, error) {
	objs, err := c.podIndexer.ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	pods := []*k8sv1.Pod{}
	for _, obj := range objs {
		pod := obj.(*k8sv1.Pod)
		pods = append(pods, pod)
	}
	return pods, nil
}

func (c *Controller) setActivePods(vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachineInstance, error) {
	pods, err := c.listPodsFromNamespace(vmi.Namespace)
	if err != nil {
		return nil, err
	}
	activePods := make(map[types.UID]string)
	count := 0
	for _, pod := range pods {
		if !controller.IsControlledBy(pod, vmi) {
			continue
		}
		count++
		activePods[pod.UID] = pod.Spec.NodeName
	}
	if count == 0 && vmi.Status.ActivePods == nil {
		return vmi, nil
	}
	vmi.Status.ActivePods = activePods
	return vmi, nil
}

func (c *Controller) allPodsDeleted(vmi *virtv1.VirtualMachineInstance) (bool, error) {
	pods, err := c.listPodsFromNamespace(vmi.Namespace)
	if err != nil {
		return false, err
	}
	for _, pod := range pods {
		if controller.IsControlledBy(pod, vmi) {
			return false, nil
		}
	}
	return true, nil
}

func (c *Controller) deletePod(vmiKey string, pod *k8sv1.Pod, options v1.DeleteOptions) error {
	c.podExpectations.ExpectDeletions(vmiKey, []string{controller.PodKey(pod)})
	err := c.clientset.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, options)
	if err != nil {
		c.podExpectations.DeletionObserved(vmiKey, controller.PodKey(pod))
	}
	return err
}

func (c *Controller) createPod(key, namespace string, pod *k8sv1.Pod) (*k8sv1.Pod, error) {
	c.podExpectations.ExpectCreations(key, 1)
	pod, err := c.clientset.CoreV1().Pods(namespace).Create(context.Background(), pod, v1.CreateOptions{})
	if err != nil {
		c.podExpectations.CreationObserved(key)
	}
	return pod, err
}

func isTempPod(pod *k8sv1.Pod) bool {
	_, ok := pod.Annotations[virtv1.EphemeralProvisioningObject]
	return ok
}

func shouldSetMigrationTransport(pod *k8sv1.Pod) bool {
	_, ok := pod.Annotations[virtv1.MigrationTransportUnixAnnotation]
	return ok
}

func (c *Controller) cleanupWaitForFirstConsumerTemporaryPods(vmi *virtv1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod) error {
	triggerPods, err := c.waitForFirstConsumerTemporaryPods(vmi, virtLauncherPod)
	if err != nil {
		return err
	}
	return c.deleteRunningOrFinishedWFFCPods(vmi, triggerPods...)
}

func (c *Controller) deleteRunningOrFinishedWFFCPods(vmi *virtv1.VirtualMachineInstance, pods ...*k8sv1.Pod) error {
	for _, pod := range pods {
		err := c.deleteRunningFinishedOrFailedPod(vmi, pod)
		if err != nil && !k8serrors.IsNotFound(err) {
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedDeletePodReason, "Failed to delete WaitForFirstConsumer temporary pod %s", pod.Name)
			return err
		}
		c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, controller.SuccessfulDeletePodReason, "Deleted WaitForFirstConsumer temporary pod %s", pod.Name)
	}
	return nil
}

func (c *Controller) deleteRunningFinishedOrFailedPod(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) error {
	zero := int64(0)
	if pod.Status.Phase == k8sv1.PodRunning || pod.Status.Phase == k8sv1.PodSucceeded || pod.Status.Phase == k8sv1.PodFailed {
		vmiKey := controller.VirtualMachineInstanceKey(vmi)
		return c.deletePod(vmiKey, pod, v1.DeleteOptions{GracePeriodSeconds: &zero})
	}
	return nil
}

func (c *Controller) waitForFirstConsumerTemporaryPods(vmi *virtv1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod) ([]*k8sv1.Pod, error) {
	var temporaryPods []*k8sv1.Pod
	// Get all pods from the namespace
	pods, err := c.listPodsFromNamespace(vmi.Namespace)
	if err != nil {
		return temporaryPods, err
	}
	for _, pod := range pods {
		// Cleanup candidates are temporary pods that are either controlled by the VMI or the virt launcher pod
		if !isTempPod(pod) {
			continue
		}
		if controller.IsControlledBy(pod, vmi) {
			temporaryPods = append(temporaryPods, pod)
		}
		if ownerRef := controller.GetControllerOf(pod); ownerRef != nil && ownerRef.UID == virtLauncherPod.UID {
			temporaryPods = append(temporaryPods, pod)
		}
	}
	return temporaryPods, nil
}

func (c *Controller) requireCPUHotplug(vmi *virtv1.VirtualMachineInstance) bool {
	if vmi.Status.CurrentCPUTopology == nil || vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.MaxSockets == 0 {
		return false
	}
	cpuTopoLogyFromStatus := &virtv1.CPU{
		Cores:   vmi.Status.CurrentCPUTopology.Cores,
		Sockets: vmi.Status.CurrentCPUTopology.Sockets,
		Threads: vmi.Status.CurrentCPUTopology.Threads,
	}
	return hardware.GetNumberOfVCPUs(vmi.Spec.Domain.CPU) != hardware.GetNumberOfVCPUs(cpuTopoLogyFromStatus)
}

func (c *Controller) requireMemoryHotplug(vmi *virtv1.VirtualMachineInstance) bool {
	if vmi.Status.Memory == nil || vmi.Spec.Domain.Memory == nil || vmi.Spec.Domain.Memory.Guest == nil || vmi.Spec.Domain.Memory.MaxGuest == nil {
		return false
	}
	return vmi.Spec.Domain.Memory.Guest.Value() != vmi.Status.Memory.GuestRequested.Value()
}

func (c *Controller) syncMemoryHotplug(vmi *virtv1.VirtualMachineInstance) {
	syncHotplugCondition(vmi, virtv1.VirtualMachineInstanceMemoryChange)
	// store additionalGuestMemoryOverheadRatio
	overheadRatio := c.clusterConfig.GetConfig().AdditionalGuestMemoryOverheadRatio
	if overheadRatio != nil {
		if vmi.Labels == nil {
			vmi.Labels = map[string]string{}
		}
		vmi.Labels[virtv1.MemoryHotplugOverheadRatioLabel] = *overheadRatio
	}
}

func preparePodPatch(oldPod, newPod *k8sv1.Pod) *patch.PatchSet {
	podConditions := controller.NewPodConditionManager()
	if podConditions.ConditionsEqual(oldPod, newPod) {
		return patch.New()
	}
	return patch.New(
		patch.WithTest("/status/conditions", oldPod.Status.Conditions),
		patch.WithReplace("/status/conditions", newPod.Status.Conditions),
	)
}
