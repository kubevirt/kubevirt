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
	"fmt"
	"math"

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
// the same name gets a new volume
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
