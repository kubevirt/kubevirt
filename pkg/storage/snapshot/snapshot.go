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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package snapshot

import (
	"context"
	"fmt"
	"strings"
	"time"

	vsv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubevirtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
)

const (
	vmSnapshotFinalizer = "snapshot.kubevirt.io/vmsnapshot-protection"

	vmSnapshotContentFinalizer = "snapshot.kubevirt.io/vmsnapshotcontent-protection"

	snapshotSourceNameLabel = "snapshot.kubevirt.io/source-vm-name"

	snapshotSourceNamespaceLabel = "snapshot.kubevirt.io/source-vm-namespace"

	defaultVolumeSnapshotClassAnnotation = "snapshot.storage.kubernetes.io/is-default-class"

	vmSnapshotContentCreateEvent = "SuccessfulVirtualMachineSnapshotContentCreate"

	volumeSnapshotCreateEvent = "SuccessfulVolumeSnapshotCreate"

	volumeSnapshotMissingEvent = "VolumeSnapshotMissing"

	vmSnapshotDeadlineExceededError = "snapshot deadline exceeded"

	snapshotRetryInterval = 5 * time.Second

	contentDeletionInterval = 5 * time.Second
)

func VmSnapshotReady(vmSnapshot *snapshotv1.VirtualMachineSnapshot) bool {
	return vmSnapshot.Status != nil && vmSnapshot.Status.ReadyToUse != nil && *vmSnapshot.Status.ReadyToUse
}

func vmSnapshotContentCreated(vmSnapshotContent *snapshotv1.VirtualMachineSnapshotContent) bool {
	return vmSnapshotContent.Status != nil && vmSnapshotContent.Status.CreationTime != nil
}

func vmSnapshotContentReady(vmSnapshotContent *snapshotv1.VirtualMachineSnapshotContent) bool {
	return vmSnapshotContent.Status != nil && vmSnapshotContent.Status.ReadyToUse != nil && *vmSnapshotContent.Status.ReadyToUse
}

func vmSnapshotError(vmSnapshot *snapshotv1.VirtualMachineSnapshot) *snapshotv1.Error {
	if vmSnapshot != nil && vmSnapshot.Status != nil && vmSnapshot.Status.Error != nil {
		return vmSnapshot.Status.Error
	}
	return nil
}

func vmSnapshotFailed(vmSnapshot *snapshotv1.VirtualMachineSnapshot) bool {
	return vmSnapshot.Status != nil && vmSnapshot.Status.Phase == snapshotv1.Failed
}

func vmSnapshotSucceeded(vmSnapshot *snapshotv1.VirtualMachineSnapshot) bool {
	return vmSnapshot.Status != nil && vmSnapshot.Status.Phase == snapshotv1.Succeeded
}

func vmSnapshotProgressing(vmSnapshot *snapshotv1.VirtualMachineSnapshot) bool {
	return vmSnapshotError(vmSnapshot) == nil && !VmSnapshotReady(vmSnapshot) &&
		!vmSnapshotFailed(vmSnapshot) && !vmSnapshotSucceeded(vmSnapshot)
}

func deleteContentPolicy(vmSnapshot *snapshotv1.VirtualMachineSnapshot) bool {
	return vmSnapshot.Spec.DeletionPolicy == nil ||
		*vmSnapshot.Spec.DeletionPolicy == snapshotv1.VirtualMachineSnapshotContentDelete
}

func shouldDeleteContent(vmSnapshot *snapshotv1.VirtualMachineSnapshot, content *snapshotv1.VirtualMachineSnapshotContent) bool {
	return deleteContentPolicy(vmSnapshot) || !vmSnapshotContentReady(content)
}

func vmSnapshotContentDeleting(content *snapshotv1.VirtualMachineSnapshotContent) bool {
	return content != nil && content.DeletionTimestamp != nil
}

func vmSnapshotDeleting(vmSnapshot *snapshotv1.VirtualMachineSnapshot) bool {
	return vmSnapshot != nil && vmSnapshot.DeletionTimestamp != nil
}

func vmSnapshotTerminating(vmSnapshot *snapshotv1.VirtualMachineSnapshot) bool {
	return vmSnapshotDeleting(vmSnapshot) || vmSnapshotDeadlineExceeded(vmSnapshot)
}

func contentDeletedIfNeeded(vmSnapshot *snapshotv1.VirtualMachineSnapshot, content *snapshotv1.VirtualMachineSnapshotContent) bool {
	return content == nil || !shouldDeleteContent(vmSnapshot, content)
}

// can unlock source either if the snapshot was completed or if snapshot deleted/exceeded deadline and the content is deleted if it should be
func canUnlockSource(vmSnapshot *snapshotv1.VirtualMachineSnapshot, content *snapshotv1.VirtualMachineSnapshotContent) bool {
	return !vmSnapshotProgressing(vmSnapshot) ||
		(vmSnapshotTerminating(vmSnapshot) && contentDeletedIfNeeded(vmSnapshot, content))
}

func vmSnapshotDeadlineExceeded(vmSnapshot *snapshotv1.VirtualMachineSnapshot) bool {
	if vmSnapshotFailed(vmSnapshot) {
		return true
	}
	if vmSnapshot.Status == nil || vmSnapshot.Status.Phase != snapshotv1.InProgress {
		return false
	}
	return timeUntilDeadline(vmSnapshot) < 0
}

func GetVMSnapshotContentName(vmSnapshot *snapshotv1.VirtualMachineSnapshot) string {
	if vmSnapshot.Status != nil && vmSnapshot.Status.VirtualMachineSnapshotContentName != nil {
		return *vmSnapshot.Status.VirtualMachineSnapshotContentName
	}

	return fmt.Sprintf("%s-%s", "vmsnapshot-content", vmSnapshot.UID)
}

func translateError(e *vsv1.VolumeSnapshotError) *snapshotv1.Error {
	if e == nil {
		return nil
	}

	return &snapshotv1.Error{
		Message: e.Message,
		Time:    e.Time,
	}
}

func (ctrl *VMSnapshotController) updateVMSnapshot(vmSnapshot *snapshotv1.VirtualMachineSnapshot) (time.Duration, error) {
	log.Log.V(3).Infof("Updating VirtualMachineSnapshot %s/%s", vmSnapshot.Namespace, vmSnapshot.Name)
	var retry time.Duration

	source, err := ctrl.getSnapshotSource(vmSnapshot)
	if err != nil {
		return 0, err
	}

	content, err := ctrl.getContent(vmSnapshot)
	if err != nil {
		return 0, err
	}

	// Make sure status is initialized before doing anything
	if vmSnapshot.Status != nil {
		if source != nil {
			if vmSnapshotProgressing(vmSnapshot) && !vmSnapshotTerminating(vmSnapshot) {
				// attempt to lock source
				// if fails will attempt again when source is updated
				if !source.Locked() {
					locked, err := source.Lock()
					if err != nil {
						return 0, err
					}

					log.Log.V(3).Infof("Attempt to lock source returned: %t", locked)

					retry = snapshotRetryInterval
				} else {
					// create content if does not exist
					if content == nil {
						if err := ctrl.createContent(vmSnapshot); err != nil {
							return 0, err
						}
					}
				}
			} else if canUnlockSource(vmSnapshot, content) {
				if _, err := source.Unlock(); err != nil {
					return 0, err
				}
			}
		}
	}

	if vmSnapshotTerminating(vmSnapshot) && content != nil {
		// Delete content if that's the policy or if the snapshot
		// is marked to be deleted and the content is not ready yet
		// - no point of keeping an unready content
		if shouldDeleteContent(vmSnapshot, content) {
			log.Log.V(2).Infof("Deleting vmsnapshotcontent %s/%s", content.Namespace, content.Name)

			err = ctrl.Client.VirtualMachineSnapshotContent(vmSnapshot.Namespace).Delete(context.Background(), content.Name, metav1.DeleteOptions{})
			if err != nil && !errors.IsNotFound(err) {
				return 0, err
			}
		} else {
			log.Log.V(2).Infof("NOT deleting vmsnapshotcontent %s/%s", content.Namespace, content.Name)
		}
	}

	if err = ctrl.updateSnapshotStatus(vmSnapshot, source); err != nil {
		return 0, err
	}

	if retry == 0 {
		return timeUntilDeadline(vmSnapshot), nil
	}

	return retry, nil
}

func (ctrl *VMSnapshotController) unfreezeSource(vmSnapshot *snapshotv1.VirtualMachineSnapshot) error {
	if vmSnapshot == nil {
		return nil
	}
	source, err := ctrl.getSnapshotSource(vmSnapshot)
	if err != nil {
		return err
	}

	if source != nil {
		if err := source.Unfreeze(); err != nil {
			return err
		}
	}
	return nil
}

func (ctrl *VMSnapshotController) removeContentFinalizer(content *snapshotv1.VirtualMachineSnapshotContent) error {
	if controller.HasFinalizer(content, vmSnapshotContentFinalizer) {
		cpy := content.DeepCopy()
		controller.RemoveFinalizer(cpy, vmSnapshotContentFinalizer)

		_, err := ctrl.Client.VirtualMachineSnapshotContent(cpy.Namespace).Update(context.Background(), cpy, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (ctrl *VMSnapshotController) updateVMSnapshotContent(content *snapshotv1.VirtualMachineSnapshotContent) (time.Duration, error) {
	log.Log.V(3).Infof("Updating VirtualMachineSnapshotContent %s/%s", content.Namespace, content.Name)

	var volumeSnapshotStatus []snapshotv1.VolumeSnapshotStatus
	var deletedSnapshots, skippedSnapshots []string
	var didFreeze bool

	vmSnapshot, err := ctrl.getVMSnapshot(content)
	if err != nil {
		return 0, err
	}

	if vmSnapshot == nil || vmSnapshotTerminating(vmSnapshot) {
		err = ctrl.unfreezeSource(vmSnapshot)
		if err != nil {
			return 0, err
		}
		err = ctrl.removeContentFinalizer(content)
		if err != nil {
			return 0, err
		}

		if vmSnapshot != nil && shouldDeleteContent(vmSnapshot, content) {
			return 0, nil
		}
	}

	if vmSnapshotContentDeleting(content) {
		log.Log.V(3).Infof("Content deleting %s/%s", content.Namespace, content.Name)
		return contentDeletionInterval, nil

	}

	currentlyCreated := vmSnapshotContentCreated(content)
	currentlyError := (content.Status != nil && content.Status.Error != nil) || vmSnapshotError(vmSnapshot) != nil

	for _, volumeBackup := range content.Spec.VolumeBackups {
		if volumeBackup.VolumeSnapshotName == nil {
			continue
		}

		vsName := *volumeBackup.VolumeSnapshotName

		volumeSnapshot, err := ctrl.GetVolumeSnapshot(content.Namespace, vsName)
		if err != nil {
			return 0, err
		}

		if volumeSnapshot == nil {
			// check if snapshot was deleted
			if currentlyCreated {
				log.Log.Warningf("VolumeSnapshot %s no longer exists", vsName)
				ctrl.Recorder.Eventf(
					content,
					corev1.EventTypeWarning,
					volumeSnapshotMissingEvent,
					"VolumeSnapshot %s no longer exists",
					vsName,
				)
				deletedSnapshots = append(deletedSnapshots, vsName)
				continue
			}

			if vmSnapshot == nil || vmSnapshotDeleting(vmSnapshot) {
				log.Log.V(3).Infof("Not creating snapshot %s because vm snapshot is deleted", vsName)
				skippedSnapshots = append(skippedSnapshots, vsName)
				continue
			}

			if currentlyError {
				log.Log.V(3).Infof("Not creating snapshot %s because in error state", vsName)
				skippedSnapshots = append(skippedSnapshots, vsName)
				continue
			}

			if !didFreeze {
				source, err := ctrl.getSnapshotSource(vmSnapshot)
				if err != nil {
					return 0, err
				}

				if source == nil {
					return 0, fmt.Errorf("unable to get snapshot source")
				}

				frozen, err := source.Frozen()
				if err != nil {
					return 0, err
				}

				if !frozen {
					err := source.Freeze()
					if err != nil {
						return 0, err
					}

					// assuming that VM is frozen once Freeze() returns
					// which should be the case
					// if Freeze() were async, we'd have to return
					// and only continue when source.Frozen() == true
				}

				didFreeze = true
			}

			volumeSnapshot, err = ctrl.createVolumeSnapshot(content, volumeBackup)
			if err != nil {
				return 0, err
			}
		}

		vss := snapshotv1.VolumeSnapshotStatus{
			VolumeSnapshotName: volumeSnapshot.Name,
		}

		if volumeSnapshot.Status != nil {
			vss.ReadyToUse = volumeSnapshot.Status.ReadyToUse
			vss.CreationTime = volumeSnapshot.Status.CreationTime
			vss.Error = translateError(volumeSnapshot.Status.Error)
		}

		volumeSnapshotStatus = append(volumeSnapshotStatus, vss)
	}

	created, ready := true, true
	errorMessage := ""
	contentCpy := content.DeepCopy()
	if contentCpy.Status == nil {
		contentCpy.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{}
	}
	contentCpy.Status.Error = nil

	if len(deletedSnapshots) > 0 {
		created, ready = false, false
		errorMessage = fmt.Sprintf("VolumeSnapshots (%s) missing", strings.Join(deletedSnapshots, ","))
	} else if len(skippedSnapshots) > 0 {
		created, ready = false, false
		if vmSnapshot == nil || vmSnapshotDeleting(vmSnapshot) {
			errorMessage = fmt.Sprintf("VolumeSnapshots (%s) skipped because vm snapshot is deleted", strings.Join(skippedSnapshots, ","))
		} else {
			errorMessage = fmt.Sprintf("VolumeSnapshots (%s) skipped because in error state", strings.Join(skippedSnapshots, ","))
		}
	} else {
		for _, vss := range volumeSnapshotStatus {
			if vss.CreationTime == nil {
				created = false
			}

			if vss.ReadyToUse == nil || !*vss.ReadyToUse {
				ready = false
			}
		}
	}

	if created && contentCpy.Status.CreationTime == nil {
		contentCpy.Status.CreationTime = currentTime()

		err = ctrl.unfreezeSource(vmSnapshot)
		if err != nil {
			return 0, err
		}
	}

	if errorMessage != "" {
		contentCpy.Status.Error = &snapshotv1.Error{
			Time:    currentTime(),
			Message: &errorMessage,
		}
	}

	contentCpy.Status.ReadyToUse = &ready
	contentCpy.Status.VolumeSnapshotStatus = volumeSnapshotStatus

	if !equality.Semantic.DeepEqual(content, contentCpy) {
		if _, err := ctrl.Client.VirtualMachineSnapshotContent(contentCpy.Namespace).Update(context.Background(), contentCpy, metav1.UpdateOptions{}); err != nil {
			return 0, err
		}
	}

	return 0, nil
}

func (ctrl *VMSnapshotController) createVolumeSnapshot(
	content *snapshotv1.VirtualMachineSnapshotContent,
	volumeBackup snapshotv1.VolumeBackup,
) (*vsv1.VolumeSnapshot, error) {
	log.Log.Infof("Attempting to create VolumeSnapshot %s", *volumeBackup.VolumeSnapshotName)

	sc := volumeBackup.PersistentVolumeClaim.Spec.StorageClassName
	if sc == nil {
		return nil, fmt.Errorf("%s/%s VolumeSnapshot requested but no storage class",
			content.Namespace, volumeBackup.PersistentVolumeClaim.Name)
	}

	volumeSnapshotClass, err := ctrl.getVolumeSnapshotClass(*sc)
	if err != nil {
		log.Log.Warningf("Couldn't find VolumeSnapshotClass for %s", *sc)
		return nil, err
	}

	t := true
	snapshot := &vsv1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name: *volumeBackup.VolumeSnapshotName,
			Labels: map[string]string{
				snapshotSourceNameLabel:      content.Spec.Source.VirtualMachine.Name,
				snapshotSourceNamespaceLabel: content.Spec.Source.VirtualMachine.Namespace,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         snapshotv1.SchemeGroupVersion.String(),
					Kind:               "VirtualMachineSnapshotContent",
					Name:               content.Name,
					UID:                content.UID,
					Controller:         &t,
					BlockOwnerDeletion: &t,
				},
			},
		},
		Spec: vsv1.VolumeSnapshotSpec{
			Source: vsv1.VolumeSnapshotSource{
				PersistentVolumeClaimName: &volumeBackup.PersistentVolumeClaim.Name,
			},
			VolumeSnapshotClassName: &volumeSnapshotClass,
		},
	}

	volumeSnapshot, err := ctrl.Client.KubernetesSnapshotClient().SnapshotV1().
		VolumeSnapshots(content.Namespace).
		Create(context.Background(), snapshot, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	ctrl.Recorder.Eventf(
		content,
		corev1.EventTypeNormal,
		volumeSnapshotCreateEvent,
		"Successfully created VolumeSnapshot %s",
		snapshot.Name,
	)

	return volumeSnapshot, nil
}

func (ctrl *VMSnapshotController) getSnapshotSource(vmSnapshot *snapshotv1.VirtualMachineSnapshot) (snapshotSource, error) {
	switch vmSnapshot.Spec.Source.Kind {
	case "VirtualMachine":
		vm, err := ctrl.getVM(vmSnapshot)
		if err != nil {
			return nil, err
		}

		if vm == nil {
			return nil, nil
		}

		return &vmSnapshotSource{
			vm:         vm,
			snapshot:   vmSnapshot,
			controller: ctrl,
		}, nil
	}

	return nil, fmt.Errorf("unknown source %+v", vmSnapshot.Spec.Source)
}

func (ctrl *VMSnapshotController) createContent(vmSnapshot *snapshotv1.VirtualMachineSnapshot) error {
	source, err := ctrl.getSnapshotSource(vmSnapshot)
	if err != nil {
		return err
	}

	var volumeBackups []snapshotv1.VolumeBackup
	pvcs, err := source.PersistentVolumeClaims()
	if err != nil {
		return err
	}
	for volumeName, pvcName := range pvcs {
		pvc, err := ctrl.getSnapshotPVC(vmSnapshot.Namespace, pvcName)
		if err != nil {
			return err
		}

		if pvc == nil {
			log.Log.Warningf("No snapshot PVC for %s/%s", vmSnapshot.Namespace, pvcName)
			continue
		}

		volumeSnapshotName := fmt.Sprintf("vmsnapshot-%s-volume-%s", vmSnapshot.UID, volumeName)
		vb := snapshotv1.VolumeBackup{
			VolumeName: volumeName,
			PersistentVolumeClaim: snapshotv1.PersistentVolumeClaim{
				ObjectMeta: *getSimplifiedMetaObject(pvc.ObjectMeta),
				Spec:       *pvc.Spec.DeepCopy(),
			},
			VolumeSnapshotName: &volumeSnapshotName,
		}

		volumeBackups = append(volumeBackups, vb)
	}

	sourceSpec, err := source.Spec()
	if err != nil {
		return err
	}
	content := &snapshotv1.VirtualMachineSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{
			Name:       GetVMSnapshotContentName(vmSnapshot),
			Namespace:  vmSnapshot.Namespace,
			Finalizers: []string{vmSnapshotContentFinalizer},
		},
		Spec: snapshotv1.VirtualMachineSnapshotContentSpec{
			VirtualMachineSnapshotName: &vmSnapshot.Name,
			Source:                     sourceSpec,
			VolumeBackups:              volumeBackups,
		},
	}

	_, err = ctrl.Client.VirtualMachineSnapshotContent(content.Namespace).Create(context.Background(), content, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	ctrl.Recorder.Eventf(
		vmSnapshot,
		corev1.EventTypeNormal,
		vmSnapshotContentCreateEvent,
		"Successfully created VirtualMachineSnapshotContent %s",
		content.Name,
	)

	return nil
}

func (ctrl *VMSnapshotController) getSnapshotPVC(namespace, volumeName string) (*corev1.PersistentVolumeClaim, error) {
	obj, exists, err := ctrl.PVCInformer.GetStore().GetByKey(cacheKeyFunc(namespace, volumeName))
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	pvc := obj.(*corev1.PersistentVolumeClaim).DeepCopy()

	if pvc.Spec.VolumeName == "" {
		log.Log.Warningf("Unbound PVC %s/%s", pvc.Namespace, pvc.Name)
		return nil, nil
	}

	if pvc.Spec.StorageClassName == nil {
		log.Log.Warningf("No storage class for PVC %s/%s", pvc.Namespace, pvc.Name)
		return nil, nil
	}

	volumeSnapshotClass, err := ctrl.getVolumeSnapshotClass(*pvc.Spec.StorageClassName)
	if err != nil {
		return nil, err
	}

	if volumeSnapshotClass != "" {
		return pvc, nil
	}

	return nil, nil
}

func (ctrl *VMSnapshotController) getVolumeSnapshotClass(storageClassName string) (string, error) {
	obj, exists, err := ctrl.StorageClassInformer.GetStore().GetByKey(storageClassName)
	if !exists || err != nil {
		return "", err
	}

	storageClass := obj.(*storagev1.StorageClass).DeepCopy()

	var matches []vsv1.VolumeSnapshotClass
	volumeSnapshotClasses := ctrl.getVolumeSnapshotClasses()
	for _, volumeSnapshotClass := range volumeSnapshotClasses {
		if volumeSnapshotClass.Driver == storageClass.Provisioner {
			matches = append(matches, volumeSnapshotClass)
		}
	}

	if len(matches) == 0 {
		log.Log.Warningf("No VolumeSnapshotClass for %s", storageClassName)
		return "", nil
	}

	if len(matches) == 1 {
		return matches[0].Name, nil
	}

	for _, volumeSnapshotClass := range matches {
		for annotation := range volumeSnapshotClass.Annotations {
			if annotation == defaultVolumeSnapshotClassAnnotation {
				return volumeSnapshotClass.Name, nil
			}
		}
	}

	return "", fmt.Errorf("%d matching VolumeSnapshotClasses for %s", len(matches), storageClassName)
}

func (ctrl *VMSnapshotController) updateSnapshotStatus(vmSnapshot *snapshotv1.VirtualMachineSnapshot, source snapshotSource) error {
	f := false
	vmSnapshotCpy := vmSnapshot.DeepCopy()
	if vmSnapshotCpy.Status == nil {
		vmSnapshotCpy.Status = &snapshotv1.VirtualMachineSnapshotStatus{
			ReadyToUse: &f,
		}
	}

	if source != nil {
		uid := source.UID()
		vmSnapshotCpy.Status.SourceUID = &uid
	}

	content, err := ctrl.getContent(vmSnapshot)
	if err != nil {
		return err
	}

	if vmSnapshotDeleting(vmSnapshotCpy) {
		// Enable the vmsnapshot to be deleted only in case it completed
		// or after waiting until the content is deleted if needed
		if !vmSnapshotProgressing(vmSnapshot) || contentDeletedIfNeeded(vmSnapshotCpy, content) {
			controller.RemoveFinalizer(vmSnapshotCpy, vmSnapshotFinalizer)
		}
	} else {
		// since no status subresource can update metadata and status
		controller.AddFinalizer(vmSnapshotCpy, vmSnapshotFinalizer)

		if content != nil && content.Status != nil {
			// content exists and is initialized
			vmSnapshotCpy.Status.VirtualMachineSnapshotContentName = &content.Name
			vmSnapshotCpy.Status.CreationTime = content.Status.CreationTime
			vmSnapshotCpy.Status.ReadyToUse = content.Status.ReadyToUse
			vmSnapshotCpy.Status.Error = content.Status.Error
		}
	}

	if vmSnapshotDeadlineExceeded(vmSnapshotCpy) {
		vmSnapshotCpy.Status.Phase = snapshotv1.Failed
		updateSnapshotCondition(vmSnapshotCpy, newProgressingCondition(corev1.ConditionFalse, vmSnapshotDeadlineExceededError))
		updateSnapshotCondition(vmSnapshotCpy, newFailureCondition(corev1.ConditionTrue, vmSnapshotDeadlineExceededError))
	} else if vmSnapshotProgressing(vmSnapshotCpy) {
		vmSnapshotCpy.Status.Phase = snapshotv1.InProgress
		if source != nil {
			if source.Locked() {
				updateSnapshotCondition(vmSnapshotCpy, newProgressingCondition(corev1.ConditionTrue, "Source locked and operation in progress"))
			} else {
				updateSnapshotCondition(vmSnapshotCpy, newProgressingCondition(corev1.ConditionFalse, "Source not locked"))
			}

			online, err := source.Online()
			if err != nil {
				return err
			}

			indications := []snapshotv1.Indication{}
			if online {
				indications = append(indications, snapshotv1.VMSnapshotOnlineSnapshotIndication)

				ga, err := source.GuestAgent()
				if err != nil {
					return err
				}

				if ga {
					indications = append(indications, snapshotv1.VMSnapshotGuestAgentIndication)

				} else {
					indications = append(indications, snapshotv1.VMSnapshotNoGuestAgentIndication)
				}
			}
			vmSnapshotCpy.Status.Indications = indications
		} else {
			updateSnapshotCondition(vmSnapshotCpy, newProgressingCondition(corev1.ConditionFalse, "Source does not exist"))
		}
		updateSnapshotCondition(vmSnapshotCpy, newReadyCondition(corev1.ConditionFalse, "Not ready"))
		if vmSnapshotDeleting(vmSnapshotCpy) {
			vmSnapshotCpy.Status.Phase = snapshotv1.Deleting
			updateSnapshotCondition(vmSnapshotCpy, newProgressingCondition(corev1.ConditionFalse, "VM snapshot is deleting"))
			updateSnapshotCondition(vmSnapshotCpy, newReadyCondition(corev1.ConditionFalse, "VM snapshot is deleting"))
		}
	} else if vmSnapshotError(vmSnapshotCpy) != nil {
		updateSnapshotCondition(vmSnapshotCpy, newProgressingCondition(corev1.ConditionFalse, "In error state"))
		updateSnapshotCondition(vmSnapshotCpy, newReadyCondition(corev1.ConditionFalse, "Error"))
	} else if VmSnapshotReady(vmSnapshotCpy) {
		vmSnapshotCpy.Status.Phase = snapshotv1.Succeeded
		updateSnapshotCondition(vmSnapshotCpy, newProgressingCondition(corev1.ConditionFalse, "Operation complete"))
		updateSnapshotCondition(vmSnapshotCpy, newReadyCondition(corev1.ConditionTrue, "Operation complete"))
		updateSnapshotSnapshotableVolumes(vmSnapshotCpy, content)
	} else {
		vmSnapshotCpy.Status.Phase = snapshotv1.Unknown
		updateSnapshotCondition(vmSnapshotCpy, newProgressingCondition(corev1.ConditionUnknown, "Unknown state"))
		updateSnapshotCondition(vmSnapshotCpy, newReadyCondition(corev1.ConditionUnknown, "Unknown state"))
	}

	if !equality.Semantic.DeepEqual(vmSnapshot, vmSnapshotCpy) {
		if _, err := ctrl.Client.VirtualMachineSnapshot(vmSnapshotCpy.Namespace).Update(context.Background(), vmSnapshotCpy, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func updateSnapshotSnapshotableVolumes(snapshot *snapshotv1.VirtualMachineSnapshot, content *snapshotv1.VirtualMachineSnapshotContent) {
	if content == nil {
		return
	}
	vm := content.Spec.Source.VirtualMachine
	if vm == nil || vm.Spec.Template == nil {
		return
	}
	volumes := vm.Spec.Template.Spec.Volumes

	volumeBackups := make(map[string]bool)
	for _, volumeBackup := range content.Spec.VolumeBackups {
		volumeBackups[volumeBackup.VolumeName] = true
	}

	var excludedVolumes []string
	var includedVolumes []string
	for _, volume := range volumes {
		if _, ok := volumeBackups[volume.Name]; ok {
			includedVolumes = append(includedVolumes, volume.Name)
		} else {
			excludedVolumes = append(excludedVolumes, volume.Name)
		}
	}
	snapshot.Status.SnapshotVolumes = &snapshotv1.SnapshotVolumesLists{
		IncludedVolumes: includedVolumes,
		ExcludedVolumes: excludedVolumes,
	}
}

func (ctrl *VMSnapshotController) updateVolumeSnapshotStatuses(vm *kubevirtv1.VirtualMachine) error {
	log.Log.V(3).Infof("Update volume snapshot status for VM [%s/%s]", vm.Namespace, vm.Name)

	vmCopy := vm.DeepCopy()
	var statuses []kubevirtv1.VolumeSnapshotStatus
	for i, volume := range vmCopy.Spec.Template.Spec.Volumes {
		log.Log.V(3).Infof("Update volume snapshot status for volume [%s]", volume.Name)
		status := ctrl.getVolumeSnapshotStatus(vmCopy, &vmCopy.Spec.Template.Spec.Volumes[i])
		statuses = append(statuses, status)
	}

	vmCopy.Status.VolumeSnapshotStatuses = statuses
	if equality.Semantic.DeepEqual(vmCopy.Status.VolumeSnapshotStatuses, vm.Status.VolumeSnapshotStatuses) {
		return nil
	}
	return ctrl.vmStatusUpdater.UpdateStatus(vmCopy)
}

func (ctrl *VMSnapshotController) getVolumeSnapshotStatus(vm *kubevirtv1.VirtualMachine, volume *kubevirtv1.Volume) kubevirtv1.VolumeSnapshotStatus {
	if volume == nil {
		return kubevirtv1.VolumeSnapshotStatus{
			Name:    volume.Name,
			Enabled: false,
			Reason:  fmt.Sprintf("Volume is nil [%s]", volume.Name),
		}
	}

	snapshottable := ctrl.isVolumeSnapshottable(volume)
	if !snapshottable {
		return kubevirtv1.VolumeSnapshotStatus{
			Name:    volume.Name,
			Enabled: false,
			Reason:  fmt.Sprintf("Snapshot is not supported for this volumeSource type [%s]", volume.Name),
		}
	}

	sc, err := ctrl.getVolumeStorageClass(vm.Namespace, volume)
	if err != nil {
		return kubevirtv1.VolumeSnapshotStatus{Name: volume.Name, Enabled: false, Reason: err.Error()}
	}

	snap, err := ctrl.getVolumeSnapshotClass(sc)
	if err != nil {
		return kubevirtv1.VolumeSnapshotStatus{Name: volume.Name, Enabled: false, Reason: err.Error()}
	}

	if snap == "" {
		return kubevirtv1.VolumeSnapshotStatus{
			Name:    volume.Name,
			Enabled: false,
			Reason:  fmt.Sprintf("No VolumeSnapshotClass: Volume snapshots are not configured for this StorageClass [%s] [%s]", sc, volume.Name),
		}
	}

	return kubevirtv1.VolumeSnapshotStatus{Name: volume.Name, Enabled: true}
}

func (ctrl *VMSnapshotController) isVolumeSnapshottable(volume *kubevirtv1.Volume) bool {
	return volume.VolumeSource.PersistentVolumeClaim != nil ||
		volume.VolumeSource.DataVolume != nil ||
		volume.VolumeSource.MemoryDump != nil
}

func (ctrl *VMSnapshotController) getStorageClassNameForPVC(pvcKey string) (string, error) {
	obj, exists, err := ctrl.PVCInformer.GetStore().GetByKey(pvcKey)
	if err != nil {
		return "", err
	}

	if !exists {
		log.Log.V(3).Infof("PVC not in cache [%s]", pvcKey)
		return "", fmt.Errorf("PVC not found")
	}
	pvc := obj.(*corev1.PersistentVolumeClaim)
	if pvc.Spec.StorageClassName != nil {
		return *pvc.Spec.StorageClassName, nil
	}
	return "", nil
}

func (ctrl *VMSnapshotController) getVolumeStorageClass(namespace string, volume *kubevirtv1.Volume) (string, error) {
	// TODO Add Ephemeral (add "|| volume.VolumeSource.Ephemeral != nil" to the `if` below)
	if volume.VolumeSource.PersistentVolumeClaim != nil {
		pvcKey := cacheKeyFunc(namespace, volume.VolumeSource.PersistentVolumeClaim.ClaimName)
		storageClassName, err := ctrl.getStorageClassNameForPVC(pvcKey)
		if err != nil {
			return "", err
		}
		return storageClassName, nil
	}

	if volume.VolumeSource.MemoryDump != nil {
		pvcKey := cacheKeyFunc(namespace, volume.VolumeSource.MemoryDump.ClaimName)
		storageClassName, err := ctrl.getStorageClassNameForPVC(pvcKey)
		if err != nil {
			return "", err
		}
		return storageClassName, nil
	}

	if volume.VolumeSource.DataVolume != nil {
		storageClassName, err := ctrl.getStorageClassNameForDV(namespace, volume.VolumeSource.DataVolume.Name)
		if err != nil {
			return "", err
		}
		return storageClassName, nil
	}

	return "", fmt.Errorf("volume type has no StorageClass defined")
}

func (ctrl *VMSnapshotController) getStorageClassNameForDV(namespace string, dvName string) (string, error) {
	// First, look up DV's StorageClass
	key := cacheKeyFunc(namespace, dvName)

	obj, exists, err := ctrl.DVInformer.GetStore().GetByKey(key)
	if err != nil {
		return "", err
	}

	if !exists {
		log.Log.V(3).Infof("DV is not in cache [%s]", key)
		return ctrl.getStorageClassNameForPVC(key)
	}

	dv := obj.(*cdiv1.DataVolume)
	if dv.Spec.PVC != nil && dv.Spec.PVC.StorageClassName != nil && *dv.Spec.PVC.StorageClassName != "" {
		return *dv.Spec.PVC.StorageClassName, nil
	}

	// Second, see if DV is owned by a VM, and if so, if the DVTemplate has a StorageClass
	for _, or := range dv.OwnerReferences {
		if or.Kind == "VirtualMachine" {

			vmKey := cacheKeyFunc(namespace, or.Name)
			storeObj, exists, err := ctrl.VMInformer.GetStore().GetByKey(vmKey)
			if err != nil || !exists {
				continue
			}

			vm, ok := storeObj.(*kubevirtv1.VirtualMachine)
			if !ok {
				continue
			}

			for _, dvTemplate := range vm.Spec.DataVolumeTemplates {
				if dvTemplate.Name == dvName && dvTemplate.Spec.PVC != nil && dvTemplate.Spec.PVC.StorageClassName != nil {
					return *dvTemplate.Spec.PVC.StorageClassName, nil
				}
			}
		}
	}

	// Third, if everything else fails, wait for PVC to read its StorageClass
	// NOTE: this will give possibly incorrect `false` value for the status until the
	// PVC is ready.
	return ctrl.getStorageClassNameForPVC(key)
}

func (ctrl *VMSnapshotController) getVM(vmSnapshot *snapshotv1.VirtualMachineSnapshot) (*kubevirtv1.VirtualMachine, error) {
	vmName := vmSnapshot.Spec.Source.Name

	obj, exists, err := ctrl.VMInformer.GetStore().GetByKey(cacheKeyFunc(vmSnapshot.Namespace, vmName))
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	return obj.(*kubevirtv1.VirtualMachine).DeepCopy(), nil
}

func (ctrl *VMSnapshotController) getContent(vmSnapshot *snapshotv1.VirtualMachineSnapshot) (*snapshotv1.VirtualMachineSnapshotContent, error) {
	contentName := GetVMSnapshotContentName(vmSnapshot)
	obj, exists, err := ctrl.VMSnapshotContentInformer.GetStore().GetByKey(cacheKeyFunc(vmSnapshot.Namespace, contentName))
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	return obj.(*snapshotv1.VirtualMachineSnapshotContent).DeepCopy(), nil
}

func (ctrl *VMSnapshotController) getVMSnapshot(vmSnapshotContent *snapshotv1.VirtualMachineSnapshotContent) (*snapshotv1.VirtualMachineSnapshot, error) {
	vmSnapshotName := vmSnapshotContent.Spec.VirtualMachineSnapshotName
	if vmSnapshotName == nil {
		return nil, fmt.Errorf("VirtualMachineSnapshotName is not initialized in vm snapshot content")
	}

	obj, exists, err := ctrl.VMSnapshotInformer.GetStore().GetByKey(cacheKeyFunc(vmSnapshotContent.Namespace, *vmSnapshotName))
	if err != nil || !exists {
		return nil, err
	}

	return obj.(*snapshotv1.VirtualMachineSnapshot).DeepCopy(), nil
}

func (ctrl *VMSnapshotController) getVMI(vm *kubevirtv1.VirtualMachine) (*kubevirtv1.VirtualMachineInstance, bool, error) {
	key, err := controller.KeyFunc(vm)
	if err != nil {
		return nil, false, err
	}

	obj, exists, err := ctrl.VMIInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		return nil, exists, err
	}

	return obj.(*kubevirtv1.VirtualMachineInstance).DeepCopy(), true, nil
}

func (ctrl *VMSnapshotController) checkVMIRunning(vm *kubevirtv1.VirtualMachine) (bool, error) {
	_, exists, err := ctrl.getVMI(vm)
	return exists, err
}

func checkVMRunning(vm *kubevirtv1.VirtualMachine) (bool, error) {
	rs, err := vm.RunStrategy()
	if err != nil {
		return false, err
	}

	return rs == kubevirtv1.RunStrategyAlways || rs == kubevirtv1.RunStrategyRerunOnFailure, nil
}

func updateSnapshotCondition(ss *snapshotv1.VirtualMachineSnapshot, c snapshotv1.Condition) {
	ss.Status.Conditions = updateCondition(ss.Status.Conditions, c, false)
}
