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
	"fmt"
	"maps"
	"math"
	"slices"

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

func (ctrl *VMSnapshotController) hotplugScratchVolume(vmi *kubevirtv1.VirtualMachineInstance, content *snapshotv1.VirtualMachineSnapshotContent) error {
	volumes := append(slices.Clone(vmi.Spec.UtilityVolumes), kubevirtv1.UtilityVolume{
		Name: scratchPVCName(content),
		PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
			ClaimName: scratchPVCName(content),
		},
		Type: pointer.P(kubevirtv1.SnapshotOverlay),
	})
	return ctrl.patchVMI(vmi, patch.WithTest("/spec/utilityVolumes", vmi.Spec.UtilityVolumes), patch.WithAdd("/spec/utilityVolumes", volumes))
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

// scratchUtilityVolumeFound rejects another source under the scratch volume's name
func scratchUtilityVolumeFound(vmi *kubevirtv1.VirtualMachineInstance, content *snapshotv1.VirtualMachineSnapshotContent) (bool, error) {
	for _, volume := range vmi.Spec.UtilityVolumes {
		if volume.Name != scratchPVCName(content) {
			continue
		}
		if volume.ClaimName != scratchPVCName(content) || volume.Type == nil || *volume.Type != kubevirtv1.SnapshotOverlay {
			return false, fmt.Errorf("scratch utility volume %s has a different source", volume.Name)
		}
		return true, nil
	}
	return false, nil
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

// releaseScratchPVC persists the launcher's verified cleanup permission.
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
