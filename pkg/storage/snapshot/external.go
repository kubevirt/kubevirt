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

	kubevirtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
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

	// Checked here to avoid provisioning a volume that will go unused. The
	// launcher checks too, the agent can drop before the call lands
	if vmi == nil || !controller.NewVirtualMachineInstanceConditionManager().
		HasConditionWithStatus(vmi, kubevirtv1.VirtualMachineInstanceAgentConnected, corev1.ConditionTrue) {
		return false, fmt.Errorf("external mode needs a running guest agent on the source of %s", content.Name)
	}

	scratch, err := ctrl.getScratchPVC(content)
	if err != nil {
		return false, err
	}
	if content.Status.CreationTime != nil || content.Status.Error != nil ||
		(scratch != nil && (!controller.HasFinalizer(scratch, overlayScratchFinalizer) || scratch.DeletionTimestamp != nil)) {
		return false, fmt.Errorf("the cleanup of %s already began: %w", content.Name, errCaptureLost)
	}

	condition := overlayCondition(vmi)
	if condition != nil && !overlayReportMatches(scratch, vmi) {
		if condition.Status == corev1.ConditionFalse && (condition.Reason == kubevirtv1.VirtualMachineInstanceReasonOverlayCommitted || condition.Reason == kubevirtv1.VirtualMachineInstanceReasonOverlaySnapshotFailed) {
			return false, ctrl.requestOverlays(vmSnapshot, content, vmi)
		}
		return false, nil
	}

	switch {
	case condition == nil:
		if len(content.Status.VolumeSnapshotStatus) > 0 {
			// A second set of overlays would span two points in time
			return false, fmt.Errorf("the overlays of %s were merged before every volume was captured: %w", content.Name, errCaptureLost)
		}

		return false, ctrl.requestOverlays(vmSnapshot, content, vmi)

	case condition.Status == corev1.ConditionTrue && condition.Reason == kubevirtv1.VirtualMachineInstanceReasonOverlaysReady:
		return true, nil

	case condition.Status == corev1.ConditionFalse && (condition.Reason == kubevirtv1.VirtualMachineInstanceReasonOverlaySnapshotFailed ||
		condition.Reason == kubevirtv1.VirtualMachineInstanceReasonOverlayCommitted):
		// The launcher put the disks back on their base images
		return false, fmt.Errorf("the overlays of %s could not be taken: %s: %w", content.Name, condition.Message, errCaptureLost)

	default:
		// Preparing, or an earlier transaction still committing
		return false, nil
	}
}

// commitOverlays merges the overlays back into the base images and removes the
// scratch volume. Returns true when the capture is complete: every VolumeSnapshot
// readyToUse while the guest is still on its overlays. The caller turns that
// into the content's creationTime.
func (ctrl *VMSnapshotController) commitOverlays(
	content *snapshotv1.VirtualMachineSnapshotContent,
	ready bool,
) (time.Duration, bool, error) {
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
			// Record the capture before the commit starts writing base images, a
			// lost status update would leave nothing to say it was good
			return snapshotRetryInterval, true, nil
		}

		return snapshotRetryInterval, true, ctrl.requestCommit(content, vmi)

	default:
		// Preparing or Committing, the launcher is mid-transaction
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

func overlayCleanupVerified(scratch *corev1.PersistentVolumeClaim, vmi *kubevirtv1.VirtualMachineInstance) bool {
	if !overlayReportMatches(scratch, vmi) {
		return false
	}
	condition := overlayCondition(vmi)
	return condition != nil && condition.Status == corev1.ConditionFalse &&
		(condition.Reason == kubevirtv1.VirtualMachineInstanceReasonOverlayCommitted ||
			condition.Reason == kubevirtv1.VirtualMachineInstanceReasonOverlaySnapshotFailed)
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
	if scratch == nil || !controller.HasFinalizer(scratch, overlayScratchFinalizer) {
		return ctrl.removeScratchVolume(content)
	}
	vmi, err := ctrl.getSourceVMI(content)
	if err != nil {
		return false, err
	}
	if !overlayReportMatches(scratch, vmi) {
		return false, nil
	}
	if overlayCleanupVerified(scratch, vmi) {
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
	if scratch != nil && controller.HasFinalizer(scratch, overlayScratchFinalizer) {
		if !overlayCleanupVerified(scratch, vmi) {
			return false, nil
		}
		return false, ctrl.releaseScratchPVC(scratch)
	}
	reference, err := storagetypes.GetOverlaySnapshotReference(vmi)
	if err != nil {
		return false, err
	}
	contextAttached := vmi != nil && (vmi.Annotations[kubevirtv1.PendingOverlayRecoveryAnnotation] == scratchPVCName(content) || (reference != nil && reference.ContentUID == content.UID))
	if !scratchVolumeDetached(vmi, content) || contextAttached {
		return false, ctrl.detachScratchVolume(vmi, content)
	}
	if scratch == nil {
		return true, nil
	}
	// Delete acceptance does not mean that kubelet has stopped using the PVC.
	return false, ctrl.deleteScratchPVC(scratch)
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
	found := false
	for _, volume := range vmi.Spec.UtilityVolumes {
		if volume.Name == scratchPVCName(content) && (volume.ClaimName != scratchPVCName(content) || volume.Type == nil || *volume.Type != kubevirtv1.SnapshotOverlay) {
			return fmt.Errorf("scratch utility volume %s has a different source", volume.Name)
		}
		found = found || volume.Name == scratchPVCName(content)
	}
	reference, err := storagetypes.GetOverlaySnapshotReference(vmi)
	if err != nil {
		return err
	}
	expected := &storagetypes.SnapshotOverlayReference{ContentUID: content.UID, ScratchPVCUID: scratch.UID}
	if scratch.UID == "" {
		return fmt.Errorf("the scratch volume of %s has no UID", content.Name)
	}
	if reference == nil || *reference != *expected || !found {
		return ctrl.initializeOverlayReference(vmi, content, expected)
	}
	if !scratchVolumeAttached(vmi, content) {
		return nil
	}

	return ctrl.Client.VirtualMachineInstance(vmi.Namespace).ExternalSnapshot(
		context.Background(),
		vmi.Name,
		&kubevirtv1.SnapshotOverlayOptions{VolumeName: scratchPVCName(content)},
	)
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

	// Round up to a whole GiB, storage is provisioned in whole units
	size = (size + storagetypes.GiB - 1) / storagetypes.GiB * storagetypes.GiB

	return *resource.NewQuantity(size, resource.BinarySI), nil
}

// scratchStorageClass returns the storage class of the VM's first disk. Ranges
// over the VM's volumes, the volume backups come from a map and have no order
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
			// Held until the overlays are committed. The owner reference below only
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

func (ctrl *VMSnapshotController) initializeOverlayReference(vmi *kubevirtv1.VirtualMachineInstance, content *snapshotv1.VirtualMachineSnapshotContent, reference *storagetypes.SnapshotOverlayReference) error {
	encoded, err := json.Marshal(reference)
	if err != nil {
		return err
	}
	annotations := make(map[string]string, len(vmi.Annotations)+1)
	maps.Copy(annotations, vmi.Annotations)
	annotations[storagetypes.OverlaySnapshotReferenceAnnotation] = string(encoded)
	patchSet := patch.New(patch.WithTest("/metadata/uid", vmi.UID), patch.WithTest("/metadata/annotations", vmi.Annotations), patch.WithAdd("/metadata/annotations", annotations))
	if vmi.ResourceVersion != "" {
		patchSet.AddOption(patch.WithTest("/metadata/resourceVersion", vmi.ResourceVersion))
	}
	volumes := append([]kubevirtv1.UtilityVolume(nil), vmi.Spec.UtilityVolumes...)
	found := false
	for _, volume := range volumes {
		if volume.Name == scratchPVCName(content) {
			if volume.ClaimName != scratchPVCName(content) || volume.Type == nil || *volume.Type != kubevirtv1.SnapshotOverlay {
				return fmt.Errorf("scratch utility volume %s has a different source", volume.Name)
			}
			found = true
		}
	}
	if !found {
		volumes = append(volumes, kubevirtv1.UtilityVolume{Name: scratchPVCName(content), PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{ClaimName: scratchPVCName(content)}, Type: pointer.P(kubevirtv1.SnapshotOverlay)})
		patchSet.AddOption(patch.WithTest("/spec/utilityVolumes", vmi.Spec.UtilityVolumes), patch.WithAdd("/spec/utilityVolumes", volumes))
	}
	payload, err := patchSet.GeneratePayload()
	if err != nil {
		return err
	}
	_, err = ctrl.Client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
	return err
}

// detachScratchVolume only requests the detach, scratchVolumeDetached reports
// when it has happened
func (ctrl *VMSnapshotController) detachScratchVolume(vmi *kubevirtv1.VirtualMachineInstance, content *snapshotv1.VirtualMachineSnapshotContent) error {
	patchSet := patch.New(patch.WithTest("/metadata/uid", vmi.UID))
	if vmi.ResourceVersion != "" {
		patchSet.AddOption(patch.WithTest("/metadata/resourceVersion", vmi.ResourceVersion))
	}
	volumes := make([]kubevirtv1.UtilityVolume, 0, len(vmi.Spec.UtilityVolumes))
	for _, volume := range vmi.Spec.UtilityVolumes {
		if volume.Name != scratchPVCName(content) {
			volumes = append(volumes, volume)
		}
	}
	changed := len(volumes) != len(vmi.Spec.UtilityVolumes)
	if changed {
		patchSet.AddOption(patch.WithTest("/spec/utilityVolumes", vmi.Spec.UtilityVolumes))
		if len(volumes) == 0 {
			patchSet.AddOption(patch.WithRemove("/spec/utilityVolumes"))
		} else {
			patchSet.AddOption(patch.WithReplace("/spec/utilityVolumes", volumes))
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
	if annotations[kubevirtv1.PendingOverlayRecoveryAnnotation] == scratchPVCName(content) {
		delete(annotations, kubevirtv1.PendingOverlayRecoveryAnnotation)
	}
	if len(annotations) != len(vmi.Annotations) {
		changed = true
		patchSet.AddOption(patch.WithTest("/metadata/annotations", vmi.Annotations), patch.WithReplace("/metadata/annotations", annotations))
	}
	if !changed {
		return nil
	}
	payload, err := patchSet.GeneratePayload()
	if err != nil {
		return err
	}
	_, err = ctrl.Client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
	return err
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

// releaseScratchPVC persists the launcher's verified cleanup permission. Tests
// prevent a stale reconciliation from authorizing a recreated or changed PVC.
func (ctrl *VMSnapshotController) releaseScratchPVC(pvc *corev1.PersistentVolumeClaim) error {
	updated := pvc.DeepCopy()
	controller.RemoveFinalizer(updated, overlayScratchFinalizer)
	patchSet := patch.New(patch.WithTest("/metadata/uid", pvc.UID), patch.WithTest("/metadata/resourceVersion", pvc.ResourceVersion), patch.WithTest("/metadata/finalizers", pvc.Finalizers), patch.WithReplace("/metadata/finalizers", updated.Finalizers))
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

// deleteScratchPVC only uses cleanup permission that was already persisted.
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
