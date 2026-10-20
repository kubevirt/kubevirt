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

package snapshot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"

	kubevirtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/storage/snapshotoverlay"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/storage/utilityvolume"
)

const (
	// Higher than most guests sustain. An overlay only occupies what is written
	// into it, so over-sizing costs quota and under-sizing fails the snapshot
	overlayWriteRateBytesPerSecond = 125 * storagetypes.MiB

	overlayWriteBurstFactor = 2

	// qcow2 metadata overhead on a fully dirtied overlay
	overlayMetadataPercent = 10
)

func snapshotMode(vmSnapshot *snapshotv1.VirtualMachineSnapshot) snapshotv1.SnapshotMode {
	if vmSnapshot != nil && vmSnapshot.Spec.SnapshotMode != nil {
		return *vmSnapshot.Spec.SnapshotMode
	}

	return snapshotv1.SnapshotModeDirect
}

// supportedSnapshotMode reports whether this version knows the mode. The CRD has
// no enum, so a newer mode reaches an older control plane as a plain string
func supportedSnapshotMode(mode snapshotv1.SnapshotMode) bool {
	return mode == snapshotv1.SnapshotModeDirect || mode == snapshotv1.SnapshotModeExternal
}

// externalMode reads the mode off the content, not the VMSnapshot, which the
// content outlives while it is being cleaned up
func externalMode(content *snapshotv1.VirtualMachineSnapshotContent) bool {
	return content.Status != nil &&
		content.Status.SnapshotMode != nil &&
		*content.Status.SnapshotMode == snapshotv1.SnapshotModeExternal
}

func (ctrl *VMSnapshotController) overlaysRestored(content *snapshotv1.VirtualMachineSnapshotContent, vm *kubevirtv1.VirtualMachine) (bool, error) {
	if !snapshotoverlay.HasUncommittedOverlays(vm) || vm.Annotations[lastRestoreAnnotation] == "" ||
		vm.Annotations[lastRestoreAnnotation] == content.Spec.Source.VirtualMachine.Annotations[lastRestoreAnnotation] {
		return false, nil
	}
	restores, err := ctrl.Client.VirtualMachineRestore(vm.Namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return false, err
	}
	var restore *snapshotv1.VirtualMachineRestore
	for i := range restores.Items {
		if hasLastRestoreAnnotation(&restores.Items[i], vm) {
			restore = &restores.Items[i]
			break
		}
	}
	if restore == nil || !vmRestoreCompleted(restore) || !metav1.IsControlledBy(restore, vm) {
		return false, nil
	}
	restoredPVCs := sets.New[string]()
	for _, volume := range restore.Status.Restores {
		restoredPVCs.Insert(volume.PersistentVolumeClaimName)
	}
	for _, claim := range storagetypes.GetPVCsFromVolumes(vm.Spec.Template.Spec.Volumes) {
		for _, backup := range content.Spec.VolumeBackups {
			if claim == backup.PersistentVolumeClaim.Name && !restoredPVCs.Has(claim) {
				return false, nil
			}
		}
	}
	return true, nil
}

func (ctrl *VMSnapshotController) overlayCleanupPending(vm *kubevirtv1.VirtualMachine) (bool, error) {
	if vm.Status.SnapshotInProgress != nil {
		return true, nil
	}
	for _, obj := range ctrl.VMSnapshotContentInformer.GetStore().List() {
		content := obj.(*snapshotv1.VirtualMachineSnapshotContent)
		source := content.Spec.Source.VirtualMachine
		if content.Namespace != vm.Namespace || source == nil || source.UID != vm.UID || !externalMode(content) {
			continue
		}
		scratch, err := ctrl.getScratchPVC(content)
		if err != nil || scratch != nil {
			return true, err
		}
	}
	return false, nil
}

func overlayCondition(vmi *kubevirtv1.VirtualMachineInstance) *kubevirtv1.VirtualMachineInstanceCondition {
	return controller.NewVirtualMachineInstanceConditionManager().
		GetCondition(vmi, kubevirtv1.VirtualMachineInstanceOverlaySnapshotActive)
}

// overlaysOwnedByRunningVMI reports whether the running VMI is the one that
// created the overlays. Checked before anything else the VMI reports.
func overlaysOwnedByRunningVMI(scratch *corev1.PersistentVolumeClaim, vmi *kubevirtv1.VirtualMachineInstance) bool {
	return vmi != nil && vmi.UID != "" && scratch.Annotations[storagetypes.OverlayOwnerVMIUIDAnnotation] == string(vmi.UID)
}

// sourceOnOverlays reports whether the source VM's VMI is already running on
// overlays, which a second snapshot has to wait out.
func (ctrl *VMSnapshotController) sourceOnOverlays(vmSnapshot *snapshotv1.VirtualMachineSnapshot) bool {
	if vmSnapshot.Spec.Source.Kind != "VirtualMachine" {
		return false
	}

	obj, exists, err := ctrl.VMIInformer.GetStore().GetByKey(
		cacheKeyFunc(vmSnapshot.Namespace, vmSnapshot.Spec.Source.Name))
	if err != nil || !exists {
		return false
	}

	condition := overlayCondition(obj.(*kubevirtv1.VirtualMachineInstance))
	return condition != nil && condition.Status == corev1.ConditionTrue
}

// errCaptureLost means capture cannot complete. Cleanup still needs a verified report.
var errCaptureLost = errors.New("no capture can follow")

// createDiskOverlays moves the guest onto qcow2 overlays, one step per
// reconcile. Returns true once every disk is on an overlay.
func (ctrl *VMSnapshotController) createDiskOverlays(
	vmSnapshot *snapshotv1.VirtualMachineSnapshot,
	content *snapshotv1.VirtualMachineSnapshotContent,
) (bool, error) {
	vmi, err := ctrl.getSourceVMI(content)
	if err != nil {
		return false, err
	}

	if vmi != nil && vmi.IsFinal() {
		return false, fmt.Errorf("the source of %s stopped: %w", content.Name, errCaptureLost)
	}

	// Checked here to avoid provisioning a volume that will go unused. The
	// launcher checks too, since the agent can drop before the call lands
	if vmi == nil || !controller.NewVirtualMachineInstanceConditionManager().
		HasConditionWithStatus(vmi, kubevirtv1.VirtualMachineInstanceAgentConnected, corev1.ConditionTrue) {
		return false, fmt.Errorf("external mode needs a running guest agent on the source of %s", content.Name)
	}

	scratch, err := ctrl.getScratchPVC(content)
	if err != nil {
		return false, err
	}
	if content.Status.CreationTime != nil ||
		(scratch != nil && (!controller.HasFinalizer(scratch, overlayScratchFinalizer) || scratch.DeletionTimestamp != nil)) {
		return false, fmt.Errorf("the cleanup of %s already began: %w", content.Name, errCaptureLost)
	}

	condition := overlayCondition(vmi)
	if condition == nil {
		if len(content.Status.VolumeSnapshotStatus) > 0 {
			// A second set of overlays would span two points in time
			return false, fmt.Errorf("the overlays of %s were merged before every volume was captured: %w", content.Name, errCaptureLost)
		}

		return false, ctrl.requestOverlays(vmSnapshot, content, vmi)
	}

	switch condition.Status {
	case corev1.ConditionTrue:
		if !overlayReportMatches(scratch, vmi) {
			return false, nil
		}
		switch condition.Reason {
		case kubevirtv1.VirtualMachineInstanceReasonOverlaysReady:
			return true, nil
		case kubevirtv1.VirtualMachineInstanceReasonOverlayCommitting,
			kubevirtv1.VirtualMachineInstanceReasonOverlayCommitFailed:
			return false, fmt.Errorf("the overlays of %s are being merged before capture completed: %w", content.Name, errCaptureLost)
		}
	case corev1.ConditionFalse:
		switch condition.Reason {
		case kubevirtv1.VirtualMachineInstanceReasonOverlaySnapshotFailed,
			kubevirtv1.VirtualMachineInstanceReasonOverlayCommitted:
			if !overlayReportMatches(scratch, vmi) {
				return false, ctrl.requestOverlays(vmSnapshot, content, vmi)
			}
			// The launcher put the disks back on their base images.
			return false, fmt.Errorf("the overlays of %s could not be taken: %s: %w", content.Name, condition.Message, errCaptureLost)
		}
	}
	return false, nil
}

func (ctrl *VMSnapshotController) failOverlaySnapshot(vmSnapshot *snapshotv1.VirtualMachineSnapshot, snapshotError *snapshotv1.Error) error {
	updated := vmSnapshot.DeepCopy()
	if updated.Status == nil {
		updated.Status = &snapshotv1.VirtualMachineSnapshotStatus{}
	}
	updated.Status.Phase = snapshotv1.Failed
	updated.Status.ReadyToUse = pointer.P(false)
	updated.Status.Error = snapshotError
	updateSnapshotCondition(updated, newFailureCondition(corev1.ConditionTrue, vmSnapshotOverlaysGoneEarlyError))
	updateSnapshotCondition(updated, newProgressingCondition(corev1.ConditionFalse, "Operation failed"))
	updateSnapshotCondition(updated, newReadyCondition(corev1.ConditionFalse, "Not ready"))
	_, err := ctrl.Client.VirtualMachineSnapshot(updated.Namespace).UpdateStatus(context.Background(), updated, metav1.UpdateOptions{})
	return err
}

// commitOverlays merges the overlays back into the base images and removes the
// scratch volume. Returns true when the capture is complete: every VolumeSnapshot
// readyToUse while the guest is still on its overlays. The caller turns that
// into the content's creationTime.
func (ctrl *VMSnapshotController) commitOverlays(
	content *snapshotv1.VirtualMachineSnapshotContent,
	ready bool,
) (requeue time.Duration, captured bool, err error) {
	scratch, err := ctrl.getScratchPVC(content)
	if err != nil {
		return 0, false, err
	}

	if scratch == nil {
		// Either there never were overlays, or they are already merged
		return 0, false, nil
	}

	vmi, err := ctrl.getSourceVMI(content)
	if err != nil {
		return 0, false, err
	}

	if !controller.HasFinalizer(scratch, overlayScratchFinalizer) || overlayCleanupVerified(scratch, vmi) {
		done, err := ctrl.removeScratchVolume(content)
		if done || err != nil {
			return 0, false, err
		}
		return snapshotRetryInterval, false, nil
	}

	// The overlays may hold the only copy of what the guest wrote, so wait for a
	// launcher that can confirm otherwise
	if !overlayReportMatches(scratch, vmi) {
		return snapshotRetryInterval, false, nil
	}

	condition := overlayCondition(vmi)

	switch {
	case condition == nil || condition.Status != corev1.ConditionTrue:
		return snapshotRetryInterval, false, nil

	case condition.Reason == kubevirtv1.VirtualMachineInstanceReasonOverlayCommitFailed:
		// Not a capture even with the overlays still on. The commit runs disk by
		// disk, so a partial failure already moved some base images
		ctrl.Recorder.Eventf(
			content,
			corev1.EventTypeWarning,
			overlayCommitFailedEvent,
			"Retrying the commit of the overlays of %s: %s",
			content.Name,
			condition.Message,
		)
		return snapshotRetryInterval, false, ctrl.requestCommit(content, vmi)

	case condition.Reason == kubevirtv1.VirtualMachineInstanceReasonOverlaysReady:
		if !ready {
			// A VolumeSnapshot still reading a base image needs it to stay still
			return snapshotRetryInterval, false, nil
		}

		if content.Status.CreationTime == nil {
			// Record the capture before the commit starts writing base images, since
			// a lost status update would leave nothing to say it was good
			return snapshotRetryInterval, true, nil
		}

		return snapshotRetryInterval, true, ctrl.requestCommit(content, vmi)

	default:
		// Preparing or Committing: the launcher is mid-transaction
		return snapshotRetryInterval, false, nil
	}
}

// overlaysGoneEarly reports overlays that went away before the capture
// completed. No second set can follow, so the failure deadline is pointless.
func (ctrl *VMSnapshotController) overlaysGoneEarly(content *snapshotv1.VirtualMachineSnapshotContent) (bool, error) {
	if content == nil || !externalMode(content) ||
		content.Status.CreationTime != nil ||
		len(content.Status.VolumeSnapshotStatus) == 0 {
		// Not External, already captured, or the overlays never went on
		return false, nil
	}

	vmi, err := ctrl.getSourceVMI(content)
	if err != nil {
		return false, err
	}

	condition := overlayCondition(vmi)

	// OverlaysReady is the only state still protecting the base images.
	// Committing is writing them, CommitFailed has written some.
	return condition == nil ||
		condition.Status != corev1.ConditionTrue ||
		condition.Reason != kubevirtv1.VirtualMachineInstanceReasonOverlaysReady, nil
}

// overlayReportMatches binds a launcher report to this exact scratch transaction.
func overlayReportMatches(scratch *corev1.PersistentVolumeClaim, vmi *kubevirtv1.VirtualMachineInstance) bool {
	return scratch != nil && overlaysOwnedByRunningVMI(scratch, vmi) && scratch.UID != "" && vmi.Status.SnapshotOverlayPVCUID == scratch.UID
}

func overlaysCommitted(vm *kubevirtv1.VirtualMachine, scratch *corev1.PersistentVolumeClaim) bool {
	return scratch.UID != "" && vm.Annotations[storagetypes.OverlaysCommittedAnnotation] == string(scratch.UID)
}

func overlayCleanupVerified(scratch *corev1.PersistentVolumeClaim, vmi *kubevirtv1.VirtualMachineInstance) bool {
	if !overlayReportMatches(scratch, vmi) {
		return false
	}
	condition := overlayCondition(vmi)
	return condition != nil && condition.Status == corev1.ConditionFalse &&
		(condition.Reason == kubevirtv1.VirtualMachineInstanceReasonOverlayCommitted ||
			condition.Reason == kubevirtv1.VirtualMachineInstanceReasonOverlaySnapshotFailed)
}

// overlaysRequested reports whether the launcher may have been asked for the
// overlays. The reference goes on the VMI that owns the scratch volume right before the request.
func overlaysRequested(scratch *corev1.PersistentVolumeClaim, vmi *kubevirtv1.VirtualMachineInstance, content *snapshotv1.VirtualMachineSnapshotContent) (bool, error) {
	if !overlaysOwnedByRunningVMI(scratch, vmi) {
		return true, nil
	}
	reference, err := storagetypes.GetOverlaySnapshotReference(vmi)
	if err != nil {
		return true, err
	}
	return reference != nil && reference.ContentUID == content.UID, nil
}

// cleanupOverlays commits an abandoned transaction, then completes its durable cleanup.
func (ctrl *VMSnapshotController) cleanupOverlays(content *snapshotv1.VirtualMachineSnapshotContent) (bool, error) {
	if !externalMode(content) {
		return true, nil
	}
	scratch, err := ctrl.getScratchPVC(content)
	if err != nil {
		return false, err
	}
	vmi, err := ctrl.getSourceVMI(content)
	if err != nil {
		return false, err
	}
	// A final VMI reports what its launcher verified last, and no launcher is left to commit
	if scratch == nil || !controller.HasFinalizer(scratch, overlayScratchFinalizer) ||
		!overlayReportMatches(scratch, vmi) || overlayCleanupVerified(scratch, vmi) || vmi.IsFinal() {
		return ctrl.removeScratchVolume(content)
	}
	condition := overlayCondition(vmi)
	if condition == nil || condition.Status != corev1.ConditionTrue || condition.Reason == kubevirtv1.VirtualMachineInstanceReasonOverlayCommitting {
		return false, nil
	}
	return false, ctrl.requestCommit(content, vmi)
}

// removeScratchVolume records cleanup permission before requesting detach. Once
// protection is released, later reconciliations can finish even without a launcher.
func (ctrl *VMSnapshotController) removeScratchVolume(content *snapshotv1.VirtualMachineSnapshotContent) (bool, error) {
	scratch, err := ctrl.getScratchPVC(content)
	if err != nil {
		return false, err
	}
	vmi, err := ctrl.getSourceVMI(content)
	if err != nil {
		return false, err
	}
	// A final VMI is never updated again, its launcher has stopped
	launcherGone := vmi == nil || vmi.IsFinal()
	if scratch != nil && controller.HasFinalizer(scratch, overlayScratchFinalizer) {
		requested, err := overlaysRequested(scratch, vmi, content)
		if err != nil {
			return false, err
		}
		verified := !requested || overlayCleanupVerified(scratch, vmi)
		if !verified {
			if !launcherGone {
				return false, nil
			}
			discard, err := ctrl.canDiscardOverlays(content, scratch)
			if err != nil || !discard {
				return false, err
			}
		}
		if err := ctrl.releaseScratchPVC(scratch); err != nil {
			return false, err
		}
		if !verified {
			ctrl.Recorder.Eventf(content, corev1.EventTypeWarning, "SnapshotOverlayDataDiscarded",
				"Discarding scratch PVC %s without a verified commit for VM %s. Retained base PVCs may be missing writes.",
				scratch.Name, content.Spec.Source.VirtualMachine.Name)
		}
		return false, nil
	}
	reference, err := storagetypes.GetOverlaySnapshotReference(vmi)
	if err != nil {
		return false, err
	}
	referenced := reference != nil && reference.ContentUID == content.UID
	if !launcherGone && (!scratchVolumeDetached(vmi, content) || referenced) {
		return false, ctrl.detachScratchVolume(vmi, content)
	}
	if scratch == nil {
		return true, nil
	}
	// Delete acceptance does not mean that kubelet has stopped using the PVC.
	return false, ctrl.deleteScratchPVC(scratch)
}

// canDiscardOverlays reports whether VM deletion, a successful restore or a manual commit
// authorizes discarding unused overlays.
func (ctrl *VMSnapshotController) canDiscardOverlays(content *snapshotv1.VirtualMachineSnapshotContent, scratch *corev1.PersistentVolumeClaim) (bool, error) {
	source := content.Spec.Source.VirtualMachine
	if source == nil || source.UID == "" || !externalMode(content) {
		return false, nil
	}
	obj, exists, err := ctrl.VMInformer.GetStore().GetByKey(cacheKeyFunc(content.Namespace, source.Name))
	if err != nil || !exists {
		return false, err
	}
	vm := obj.(*kubevirtv1.VirtualMachine)
	if vm.UID != source.UID {
		return false, nil
	}
	if vm.DeletionTimestamp == nil && !overlaysCommitted(vm, scratch) {
		restored, err := ctrl.overlaysRestored(content, vm)
		if err != nil || !restored {
			return false, err
		}
	}
	// Confirm destructive cleanup against the API rather than a VMI cache miss.
	vmi, err := ctrl.Client.VirtualMachineInstance(content.Namespace).Get(context.Background(), source.Name, metav1.GetOptions{})
	if err == nil && !vmi.IsFinal() {
		return false, nil
	}
	if err != nil && !k8serrors.IsNotFound(err) {
		return false, err
	}
	pods, err := ctrl.Client.CoreV1().Pods(content.Namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return false, err
	}
	for _, pod := range pods.Items {
		if controller.PodIsDown(&pod) {
			continue
		}
		if owner := scratch.Annotations[storagetypes.OverlayOwnerVMIUIDAnnotation]; owner != "" && pod.Labels[kubevirtv1.CreatedByLabel] == owner {
			return false, nil
		}
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == scratch.Name {
				return false, nil
			}
		}
	}
	return true, nil
}

func (ctrl *VMSnapshotController) cleanupDeletingSource(vmSnapshot *snapshotv1.VirtualMachineSnapshot, content *snapshotv1.VirtualMachineSnapshotContent, source snapshotSource) (time.Duration, error) {
	if content != nil && (content.Spec.Source.VirtualMachine == nil || content.Spec.Source.VirtualMachine.UID != source.UID()) {
		return 0, fmt.Errorf("snapshot content does not belong to the deleting VM")
	}
	if !vmSnapshotFailed(vmSnapshot) && (content == nil || !vmSnapshotContentCreated(content)) {
		return snapshotRetryInterval, ctrl.failOverlaySnapshot(vmSnapshot, &snapshotv1.Error{
			Time:    currentTime(),
			Message: pointer.P("Source VM was deleted before snapshot capture completed"),
		})
	}
	if content != nil {
		if !vmSnapshotContentCreated(content) && vmSnapshotContentReady(content) {
			updated := content.DeepCopy()
			updated.Status.ReadyToUse = pointer.P(false)
			updated.Status.Error = vmSnapshot.Status.Error
			return snapshotRetryInterval, ctrl.updateVmSnapshotContentStatus(content, updated)
		}
		done, err := ctrl.removeScratchVolume(content)
		if err != nil || !done {
			return snapshotRetryInterval, err
		}
	}
	if _, err := ctrl.updateSnapshotStatus(vmSnapshot, source); err != nil {
		return 0, err
	}
	_, err := source.Unlock()
	return 0, err
}

func (ctrl *VMSnapshotController) settleInterruptedSource(vmSnapshot *snapshotv1.VirtualMachineSnapshot, content *snapshotv1.VirtualMachineSnapshotContent, source snapshotSource) (time.Duration, error) {
	if content == nil || !externalMode(content) {
		return snapshotRetryInterval, nil
	}
	if content.Spec.Source.VirtualMachine == nil || content.Spec.Source.VirtualMachine.UID != source.UID() {
		return 0, fmt.Errorf("snapshot content does not belong to the interrupted VM")
	}
	if !vmSnapshotContentCreated(content) && !vmSnapshotFailed(vmSnapshot) {
		return snapshotRetryInterval, ctrl.failOverlaySnapshot(vmSnapshot, &snapshotv1.Error{
			Time:    currentTime(),
			Message: pointer.P("Source VM stopped before snapshot capture completed"),
		})
	}
	if _, err := source.Unlock(); err != nil {
		return 0, err
	}
	if _, err := ctrl.updateSnapshotStatus(vmSnapshot, source); err != nil {
		return 0, err
	}
	vm := source.VirtualMachine()
	// canDiscardOverlays only releases the scratch PVC with the committed UID
	committed := vm.Annotations[storagetypes.OverlaysCommittedAnnotation]
	if committed == "" {
		restored, err := ctrl.overlaysRestored(content, vm)
		if err != nil || !restored {
			return snapshotRetryInterval, err
		}
	}
	done, err := ctrl.removeScratchVolume(content)
	if err != nil || !done {
		return snapshotRetryInterval, err
	}
	pending, err := ctrl.overlayCleanupPending(vm)
	if err != nil || pending {
		return snapshotRetryInterval, err
	}
	vm = vm.DeepCopy()
	controller.NewVirtualMachineConditionManager().RemoveCondition(vm, kubevirtv1.VirtualMachineManualRecoveryRequired)
	if _, err := ctrl.Client.VirtualMachine(vm.Namespace).UpdateStatus(context.Background(), vm, metav1.UpdateOptions{}); err != nil {
		return snapshotRetryInterval, err
	}
	if committed == "" {
		return snapshotRetryInterval, nil
	}
	path := "/metadata/annotations/" + patch.EscapeJSONPointer(storagetypes.OverlaysCommittedAnnotation)
	payload, err := patch.New(patch.WithTest(path, committed), patch.WithRemove(path)).GeneratePayload()
	if err != nil {
		return 0, err
	}
	_, err = ctrl.Client.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
	return snapshotRetryInterval, err
}

// requestOverlays creates the scratch PVC, hotplugs it, then calls
// ExternalSnapshot. One step per call, each waiting for the last on the VMI.
func (ctrl *VMSnapshotController) requestOverlays(
	vmSnapshot *snapshotv1.VirtualMachineSnapshot,
	content *snapshotv1.VirtualMachineSnapshotContent,
	vmi *kubevirtv1.VirtualMachineInstance,
) error {
	scratch, err := ctrl.getScratchPVC(content)
	if err != nil {
		return err
	}

	if scratch == nil {
		_, err := ctrl.createScratchPVC(vmSnapshot, content, vmi)
		return err
	}

	if !controller.HasFinalizer(scratch, overlayScratchFinalizer) || scratch.DeletionTimestamp != nil || !overlaysOwnedByRunningVMI(scratch, vmi) {
		return fmt.Errorf("the scratch volume of %s cannot start a new capture: %w", content.Name, errCaptureLost)
	}
	if scratch.UID == "" {
		return fmt.Errorf("the scratch volume of %s has no UID", content.Name)
	}
	found, err := utilityvolume.Find(vmi, scratchPVCName(content), scratchPVCName(content), kubevirtv1.SnapshotOverlay)
	if err != nil {
		return err
	}
	if !found {
		return ctrl.hotplugScratchVolume(vmi, content)
	}
	if !scratchVolumeAttached(vmi, content) {
		return nil
	}
	reference, err := storagetypes.GetOverlaySnapshotReference(vmi)
	if err != nil {
		return err
	}
	expected := &storagetypes.SnapshotOverlayReference{ContentUID: content.UID, ScratchPVCUID: scratch.UID}
	// Written right before the request, so cleanup can tell a launcher that was never asked
	if reference == nil || *reference != *expected {
		return ctrl.initializeOverlayReference(vmi, expected)
	}

	options := &kubevirtv1.SnapshotOverlayOptions{VolumeName: scratchPVCName(content)}
	for _, volumeBackup := range content.Spec.VolumeBackups {
		options.Volumes = append(options.Volumes, volumeBackup.VolumeName)
	}
	return ctrl.Client.VirtualMachineInstance(vmi.Namespace).ExternalSnapshot(context.Background(), vmi.Name, options)
}

// requestCommit calls CommitSnapshot, a no-op if a commit is already running
func (ctrl *VMSnapshotController) requestCommit(
	content *snapshotv1.VirtualMachineSnapshotContent,
	vmi *kubevirtv1.VirtualMachineInstance,
) error {
	return ctrl.Client.VirtualMachineInstance(vmi.Namespace).CommitSnapshot(
		context.Background(),
		vmi.Name,
		&kubevirtv1.SnapshotOverlayOptions{VolumeName: scratchPVCName(content)},
	)
}

func (ctrl *VMSnapshotController) getSourceVMI(content *snapshotv1.VirtualMachineSnapshotContent) (*kubevirtv1.VirtualMachineInstance, error) {
	vm := content.Spec.Source.VirtualMachine
	if vm == nil {
		return nil, nil
	}

	obj, exists, err := ctrl.VMIInformer.GetStore().GetByKey(cacheKeyFunc(content.Namespace, vm.Name))
	if err != nil || !exists {
		return nil, err
	}

	return obj.(*kubevirtv1.VirtualMachineInstance).DeepCopy(), nil
}

// scratchPVCName is derived from the content UID, so a snapshot recreated under
// the same name gets a new volume. It doubles as the name the PVC is hotplugged
// under.
func scratchPVCName(content *snapshotv1.VirtualMachineSnapshotContent) string {
	return storagetypes.ScratchPVCName(content.UID)
}

// calculateScratchSize returns min(failureDeadline x 125Mi/s x 2, disks x 1.1).
// The first bound is what the guest could write before the snapshot gives up,
// the second is what the overlays could hold.
func calculateScratchSize(vmSnapshot *snapshotv1.VirtualMachineSnapshot, content *snapshotv1.VirtualMachineSnapshotContent) (resource.Quantity, error) {
	if vmSnapshot.Spec.OverlayScratchSize != nil {
		return *vmSnapshot.Spec.OverlayScratchSize, nil
	}

	capacityBound := int64(0)
	for _, volumeBackup := range content.Spec.VolumeBackups {
		capacityBound += volumeBackup.PersistentVolumeClaim.Spec.Resources.Requests.Storage().Value()
	}
	if capacityBound == 0 {
		return resource.Quantity{}, fmt.Errorf("cannot size the overlay scratch volume, %s has no volumes to snapshot", content.Name)
	}
	capacityBound += capacityBound / 100 * overlayMetadataPercent

	// A deadline of zero is switched off, so there is no window to bound by
	writeBound := int64(math.MaxInt64)
	if deadline := getFailureDeadline(vmSnapshot); deadline > 0 {
		writeBound = int64(deadline.Seconds()) * overlayWriteRateBytesPerSecond * overlayWriteBurstFactor
	}

	size := min(writeBound, capacityBound)

	// Round up to a whole GiB, since storage is provisioned in whole units
	size = (size + storagetypes.GiB - 1) / storagetypes.GiB * storagetypes.GiB

	return *resource.NewQuantity(size, resource.BinarySI), nil
}

// scratchStorageClass returns the storage class of the VM's first disk. It ranges
// over the VM's volumes because the volume backups come from a map and have no order
func scratchStorageClass(content *snapshotv1.VirtualMachineSnapshotContent) *string {
	vm := content.Spec.Source.VirtualMachine
	if vm == nil || vm.Spec.Template == nil {
		return nil
	}

	for _, volume := range vm.Spec.Template.Spec.Volumes {
		for _, volumeBackup := range content.Spec.VolumeBackups {
			if volumeBackup.VolumeName == volume.Name {
				return volumeBackup.PersistentVolumeClaim.Spec.StorageClassName
			}
		}
	}
	return nil
}

func (ctrl *VMSnapshotController) createScratchPVC(
	vmSnapshot *snapshotv1.VirtualMachineSnapshot,
	content *snapshotv1.VirtualMachineSnapshotContent,
	vmi *kubevirtv1.VirtualMachineInstance,
) (*corev1.PersistentVolumeClaim, error) {
	size, err := calculateScratchSize(vmSnapshot, content)
	if err != nil {
		return nil, err
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scratchPVCName(content),
			Namespace: content.Namespace,
			Annotations: map[string]string{
				// Read on the crash path, to tell a launcher that finished the commit
				// from one that died mid-commit
				storagetypes.OverlayOwnerVMIUIDAnnotation: string(vmi.UID),
			},
			// Held until commit or disposal after VM deletion. The owner reference only
			// collects once the content is gone, so it is a backstop
			Finalizers: []string{overlayScratchFinalizer},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         snapshotv1.SchemeGroupVersion.String(),
				Kind:               "VirtualMachineSnapshotContent",
				Name:               content.Name,
				UID:                content.UID,
				Controller:         pointer.P(true),
				BlockOwnerDeletion: pointer.P(true),
			}},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: scratchStorageClass(content),
			// The overlays are files in a directory, whatever the disks they shadow
			VolumeMode: pointer.P(corev1.PersistentVolumeFilesystem),
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: size},
			},
		},
	}

	created, err := ctrl.Client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create the overlay scratch volume %s: %w", pvc.Name, err)
		}
		return ctrl.getScratchPVC(content)
	}

	log.Log.Object(vmSnapshot).Infof("Created the overlay scratch volume %s (%s) for VMI %s", created.Name, size.String(), vmi.UID)
	ctrl.Recorder.Eventf(
		vmSnapshot,
		corev1.EventTypeNormal,
		scratchPVCCreateEvent,
		"Successfully created overlay scratch volume %s",
		created.Name,
	)

	return created, nil
}

func (ctrl *VMSnapshotController) hotplugScratchVolume(vmi *kubevirtv1.VirtualMachineInstance, content *snapshotv1.VirtualMachineSnapshotContent) error {
	payload, err := utilityvolume.AddPatch(vmi, scratchPVCName(content), scratchPVCName(content), kubevirtv1.SnapshotOverlay)
	if err != nil {
		return err
	}
	_, err = ctrl.Client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
	return err
}

func (ctrl *VMSnapshotController) initializeOverlayReference(vmi *kubevirtv1.VirtualMachineInstance, reference *storagetypes.SnapshotOverlayReference) error {
	encoded, err := json.Marshal(reference)
	if err != nil {
		return err
	}
	annotations := make(map[string]string, len(vmi.Annotations)+1)
	maps.Copy(annotations, vmi.Annotations)
	annotations[storagetypes.OverlaySnapshotReferenceAnnotation] = string(encoded)
	return ctrl.patchVMI(vmi, patch.WithTest("/metadata/annotations", vmi.Annotations), patch.WithAdd("/metadata/annotations", annotations))
}

func (ctrl *VMSnapshotController) patchVMI(vmi *kubevirtv1.VirtualMachineInstance, options ...patch.PatchOption) error {
	patchSet := patch.New(patch.WithTest("/metadata/uid", vmi.UID))
	if vmi.ResourceVersion != "" {
		patchSet.AddOption(patch.WithTest("/metadata/resourceVersion", vmi.ResourceVersion))
	}
	patchSet.AddOption(options...)
	payload, err := patchSet.GeneratePayload()
	if err != nil {
		return err
	}
	_, err = ctrl.Client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
	return err
}

// detachScratchVolume only requests the detach. scratchVolumeDetached reports
// when it has happened
func (ctrl *VMSnapshotController) detachScratchVolume(vmi *kubevirtv1.VirtualMachineInstance, content *snapshotv1.VirtualMachineSnapshotContent) error {
	var options []patch.PatchOption
	volumes := make([]kubevirtv1.UtilityVolume, 0, len(vmi.Spec.UtilityVolumes))
	for _, volume := range vmi.Spec.UtilityVolumes {
		if volume.Name != scratchPVCName(content) {
			volumes = append(volumes, volume)
		}
	}
	if len(volumes) != len(vmi.Spec.UtilityVolumes) {
		options = append(options, patch.WithTest("/spec/utilityVolumes", vmi.Spec.UtilityVolumes))
		if len(volumes) == 0 {
			options = append(options, patch.WithRemove("/spec/utilityVolumes"))
		} else {
			options = append(options, patch.WithReplace("/spec/utilityVolumes", volumes))
		}
	}
	annotations := maps.Clone(vmi.Annotations)
	reference, err := storagetypes.GetOverlaySnapshotReference(vmi)
	if err != nil {
		return err
	}
	if reference != nil && reference.ContentUID == content.UID {
		delete(annotations, storagetypes.OverlaySnapshotReferenceAnnotation)
	}
	if len(annotations) != len(vmi.Annotations) {
		options = append(options, patch.WithTest("/metadata/annotations", vmi.Annotations), patch.WithReplace("/metadata/annotations", annotations))
	}
	if len(options) == 0 {
		return nil
	}
	return ctrl.patchVMI(vmi, options...)
}

func scratchVolumeAttached(vmi *kubevirtv1.VirtualMachineInstance, content *snapshotv1.VirtualMachineSnapshotContent) bool {
	return utilityvolume.Attached(vmi, scratchPVCName(content))
}

func scratchVolumeDetached(vmi *kubevirtv1.VirtualMachineInstance, content *snapshotv1.VirtualMachineSnapshotContent) bool {
	return utilityvolume.Detached(vmi, scratchPVCName(content))
}

func (ctrl *VMSnapshotController) getScratchPVC(content *snapshotv1.VirtualMachineSnapshotContent) (*corev1.PersistentVolumeClaim, error) {
	key := cacheKeyFunc(content.Namespace, scratchPVCName(content))
	obj, exists, err := ctrl.PVCInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil, err
	}
	var pvc *corev1.PersistentVolumeClaim
	if exists {
		pvc = obj.(*corev1.PersistentVolumeClaim).DeepCopy()
	} else {
		pvc, err = ctrl.Client.CoreV1().PersistentVolumeClaims(content.Namespace).Get(context.Background(), scratchPVCName(content), metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
	}
	owner := metav1.GetControllerOf(pvc)
	if owner == nil || owner.Kind != "VirtualMachineSnapshotContent" || owner.UID != content.UID || owner.Name != content.Name {
		return nil, fmt.Errorf("overlay scratch PVC %s owner does not match content %s (%s)", pvc.Name, content.Name, content.UID)
	}
	return pvc, nil
}

// releaseScratchPVC persists permission to clean up after commit, restore or VM deletion.
// Identity checks prevent releasing a recreated or changed PVC.
func (ctrl *VMSnapshotController) releaseScratchPVC(pvc *corev1.PersistentVolumeClaim) error {
	updated := pvc.DeepCopy()
	controller.RemoveFinalizer(updated, overlayScratchFinalizer)
	patchSet := patch.New(
		patch.WithTest("/metadata/uid", pvc.UID),
		patch.WithTest("/metadata/resourceVersion", pvc.ResourceVersion),
		patch.WithTest("/metadata/finalizers", pvc.Finalizers),
		patch.WithReplace("/metadata/finalizers", updated.Finalizers),
	)
	payload, err := patchSet.GeneratePayload()
	if err != nil {
		return err
	}
	_, err = ctrl.Client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Patch(context.Background(), pvc.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to release the overlay scratch volume %s: %w", pvc.Name, err)
	}
	return nil
}

// deleteScratchPVC refuses while the finalizer is held, so only a persisted release authorizes the delete.
func (ctrl *VMSnapshotController) deleteScratchPVC(pvc *corev1.PersistentVolumeClaim) error {
	if controller.HasFinalizer(pvc, overlayScratchFinalizer) {
		return fmt.Errorf("overlay scratch volume %s is still protected", pvc.Name)
	}
	if pvc.DeletionTimestamp != nil {
		return nil
	}
	err := ctrl.Client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(context.Background(), pvc.Name, metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: pointer.P(pvc.UID), ResourceVersion: pointer.P(pvc.ResourceVersion)}})
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete the overlay scratch volume %s: %w", pvc.Name, err)
	}
	return nil
}
