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
	"errors"
	"fmt"
	"math"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubevirtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/log"

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
	return vmi != nil && scratch.Annotations[overlayOwnerVMIUIDAnnotation] == string(vmi.UID)
}

// errCaptureLost means the capture cannot complete and the volume can go
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

	condition := overlayCondition(vmi)
	switch {
	case condition == nil:
		if len(content.Status.VolumeSnapshotStatus) > 0 {
			// A second set of overlays would span two points in time
			return false, fmt.Errorf("the overlays of %s were merged before every volume was captured: %w", content.Name, errCaptureLost)
		}

		return false, ctrl.requestOverlays(vmSnapshot, content, vmi)

	case condition.Reason == kubevirtv1.VirtualMachineInstanceReasonOverlaysReady:
		return true, nil

	case condition.Reason == kubevirtv1.VirtualMachineInstanceReasonOverlaySnapshotFailed:
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

	// The overlays may hold the only copy of what the guest wrote, so wait for a
	// launcher that can confirm otherwise
	if !overlaysOwnedByRunningVMI(scratch, vmi) {
		return 0, false, nil
	}

	condition := overlayCondition(vmi)
	switch {
	case condition == nil || condition.Status == corev1.ConditionFalse:
		// The launcher only clears its record once every disk is back on its base
		// image, so either the commit landed or the overlays never went on
		done, err := ctrl.removeScratchVolume(content)
		if err != nil || done {
			return 0, false, err
		}
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

// cleanupOverlays merges the overlays of a snapshot that is going away and
// reports when there is nothing left. Commits whatever state the VolumeSnapshots
// are in, the guest still has to get back on its base images.
func (ctrl *VMSnapshotController) cleanupOverlays(content *snapshotv1.VirtualMachineSnapshotContent) (bool, error) {
	scratch, err := ctrl.getScratchPVC(content)
	if err != nil {
		return false, err
	}

	if scratch == nil {
		return true, nil
	}

	vmi, err := ctrl.getSourceVMI(content)
	if err != nil {
		return false, err
	}

	// Only discard overlays a launcher has confirmed were merged
	if !overlaysOwnedByRunningVMI(scratch, vmi) {
		return false, nil
	}

	condition := overlayCondition(vmi)
	switch {
	case condition == nil || condition.Status == corev1.ConditionFalse:
		return ctrl.removeScratchVolume(content)

	case condition.Reason == kubevirtv1.VirtualMachineInstanceReasonOverlayCommitting:
		return false, nil

	default:
		return false, ctrl.requestCommit(content, vmi)
	}
}

// removeScratchVolume detaches the volume from the VMI, then deletes the PVC.
// A PVC a pod still mounts stays alive on kubernetes.io/pvc-protection.
func (ctrl *VMSnapshotController) removeScratchVolume(content *snapshotv1.VirtualMachineSnapshotContent) (bool, error) {
	vmi, err := ctrl.getSourceVMI(content)
	if err != nil {
		return false, err
	}

	if !scratchVolumeDetached(vmi, content) {
		return false, ctrl.detachScratchVolume(vmi, content)
	}

	scratch, err := ctrl.getScratchPVC(content)
	if err != nil || scratch == nil {
		return true, err
	}

	return true, ctrl.deleteScratchPVC(scratch)
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

	if !scratchVolumeAttached(vmi, content) {
		return ctrl.attachScratchVolume(vmi, content)
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
	return overlayScratchPVCPrefix + string(content.UID)
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
				overlayOwnerVMIUIDAnnotation: string(vmi.UID),
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

func (ctrl *VMSnapshotController) attachScratchVolume(vmi *kubevirtv1.VirtualMachineInstance, content *snapshotv1.VirtualMachineSnapshotContent) error {
	name := scratchPVCName(content)
	return utilityvolume.Attach(ctrl.Client, vmi, name, name, kubevirtv1.SnapshotOverlay)
}

// detachScratchVolume only requests the detach, scratchVolumeDetached reports
// when it has happened
func (ctrl *VMSnapshotController) detachScratchVolume(vmi *kubevirtv1.VirtualMachineInstance, content *snapshotv1.VirtualMachineSnapshotContent) error {
	return utilityvolume.Detach(ctrl.Client, vmi, scratchPVCName(content))
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
	if err != nil || !exists {
		return nil, err
	}

	return obj.(*corev1.PersistentVolumeClaim).DeepCopy(), nil
}

// deleteScratchPVC drops the finalizer before deleting, so a volume already
// marked for deletion is not left held by a finalizer nothing will remove
func (ctrl *VMSnapshotController) deleteScratchPVC(pvc *corev1.PersistentVolumeClaim) error {
	if controller.HasFinalizer(pvc, overlayScratchFinalizer) {
		updated := pvc.DeepCopy()
		controller.RemoveFinalizer(updated, overlayScratchFinalizer)
		var err error
		updated, err = ctrl.Client.CoreV1().PersistentVolumeClaims(updated.Namespace).Update(context.Background(), updated, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to release the overlay scratch volume %s: %w", pvc.Name, err)
		}
		pvc = updated
	}

	if pvc.DeletionTimestamp != nil {
		return nil
	}

	err := ctrl.Client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(context.Background(), pvc.Name, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete the overlay scratch volume %s: %w", pvc.Name, err)
	}
	return nil
}
