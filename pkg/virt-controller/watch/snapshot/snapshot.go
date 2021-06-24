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
	"reflect"
	"strings"
	"time"

	vsv1beta1 "github.com/kubernetes-csi/external-snapshotter/v2/pkg/apis/volumesnapshot/v1beta1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"

	kubevirtv1 "kubevirt.io/client-go/api/v1"
	snapshotv1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	"kubevirt.io/kubevirt/pkg/controller"
)

const (
	sourceFinalizer = "snapshot.kubevirt.io/snapshot-source-protection"

	vmSnapshotFinalizer = "snapshot.kubevirt.io/vmsnapshot-protection"

	vmSnapshotContentFinalizer = "snapshot.kubevirt.io/vmsnapshotcontent-protection"

	defaultVolumeSnapshotClassAnnotation = "snapshot.storage.kubernetes.io/is-default-class"

	vmSnapshotContentCreateEvent = "SuccessfulVirtualMachineSnapshotContentCreate"

	volumeSnapshotCreateEvent = "SuccessfulVolumeSnapshotCreate"

	volumeSnapshotMissingEvent = "VolumeSnapshotMissing"

	snapshotRetryInterval = 5 * time.Second
)

type snapshotSource interface {
	UID() types.UID
	Locked() bool
	Lock() (bool, error)
	Unlock() (bool, error)
	Spec() snapshotv1.SourceSpec
	PersistentVolumeClaims() map[string]string
}

type vmSnapshotSource struct {
	vm         *kubevirtv1.VirtualMachine
	snapshot   *snapshotv1.VirtualMachineSnapshot
	controller *VMSnapshotController
}

func vmSnapshotReady(vmSnapshot *snapshotv1.VirtualMachineSnapshot) bool {
	return vmSnapshot.Status != nil && vmSnapshot.Status.ReadyToUse != nil && *vmSnapshot.Status.ReadyToUse
}

func vmSnapshotContentReady(vmSnapshotContent *snapshotv1.VirtualMachineSnapshotContent) bool {
	return vmSnapshotContent.Status != nil && vmSnapshotContent.Status.ReadyToUse != nil && *vmSnapshotContent.Status.ReadyToUse
}

func vmSnapshotError(vmSnapshot *snapshotv1.VirtualMachineSnapshot) *snapshotv1.Error {
	if vmSnapshot.Status != nil && vmSnapshot.Status.Error != nil {
		return vmSnapshot.Status.Error
	}
	return nil
}

func vmSnapshotProgressing(vmSnapshot *snapshotv1.VirtualMachineSnapshot) bool {
	return vmSnapshotError(vmSnapshot) == nil && !vmSnapshotReady(vmSnapshot)
}

func getVMSnapshotContentName(vmSnapshot *snapshotv1.VirtualMachineSnapshot) string {
	if vmSnapshot.Status != nil && vmSnapshot.Status.VirtualMachineSnapshotContentName != nil {
		return *vmSnapshot.Status.VirtualMachineSnapshotContentName
	}

	return fmt.Sprintf("%s-%s", "vmsnapshot-content", vmSnapshot.UID)
}

func translateError(e *vsv1beta1.VolumeSnapshotError) *snapshotv1.Error {
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

	// Make sure status is initialized
	if vmSnapshot.Status == nil {
		return 0, ctrl.updateSnapshotStatus(vmSnapshot, nil)
	}

	source, err := ctrl.getSnapshotSource(vmSnapshot)
	if err != nil {
		return 0, err
	}

	// unlock the source if done/error
	if !vmSnapshotProgressing(vmSnapshot) && source != nil {
		if updated, err := source.Unlock(); updated || err != nil {
			return 0, err
		}
	}

	// check deleted
	if vmSnapshot.DeletionTimestamp != nil {
		return 0, ctrl.cleanupVMSnapshot(vmSnapshot)
	}

	if source != nil && vmSnapshotProgressing(vmSnapshot) {
		// attempt to lock source
		// if fails will attempt again when source is updated
		if !source.Locked() {
			locked, err := source.Lock()
			if err != nil {
				return 0, err
			}

			log.Log.V(3).Infof("Attempt to lock source returned: %t", locked)

			return snapshotRetryInterval, nil
		}

		// add source finalizer and maybe other stuff
		updated, err := ctrl.initVMSnapshot(vmSnapshot)
		if updated || err != nil {
			return 0, err
		}

		content, err := ctrl.getContent(vmSnapshot)
		if err != nil {
			return 0, err
		}

		// create content if does not exist
		if content == nil {
			return 0, ctrl.createContent(vmSnapshot)
		}
	}

	if err = ctrl.updateSnapshotStatus(vmSnapshot, source); err != nil {
		return 0, err
	}

	return 0, nil
}

func (ctrl *VMSnapshotController) updateVMSnapshotContent(content *snapshotv1.VirtualMachineSnapshotContent) (time.Duration, error) {
	log.Log.V(3).Infof("Updating VirtualMachineSnapshotContent %s/%s", content.Namespace, content.Name)

	var volumeSnapshotStatus []snapshotv1.VolumeSnapshotStatus
	var deletedSnapshots, skippedSnapshots []string

	currentlyReady := vmSnapshotContentReady(content)
	currentlyError := content.Status != nil && content.Status.Error != nil
	checkedGuestAgentParticipation := false

	for _, volumeBackup := range content.Spec.VolumeBackups {
		if volumeBackup.VolumeSnapshotName == nil {
			continue
		}

		vsName := *volumeBackup.VolumeSnapshotName

		volumeSnapshot, err := ctrl.getVolumeSnapshot(content.Namespace, vsName)
		if err != nil {
			return 0, err
		}

		if volumeSnapshot == nil {
			// check if snapshot was deleted
			if currentlyReady {
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

			if currentlyError {
				log.Log.V(3).Infof("Not creating snapshot %s because in error state", vsName)
				skippedSnapshots = append(skippedSnapshots, vsName)
				continue
			}

			if !checkedGuestAgentParticipation {
				vmSnapshot, err := ctrl.getVMSnapshot(content)
				if err != nil {
					return 0, err
				}
				err = ctrl.freezeGuestFSIfNeeded(vmSnapshot)
				if err != nil {
					return 0, err
				}
				checkedGuestAgentParticipation = true
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

	ready := true
	errorMessage := ""
	contentCpy := content.DeepCopy()
	if contentCpy.Status == nil {
		contentCpy.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{}
	}
	contentCpy.Status.Error = nil

	if len(deletedSnapshots) > 0 {
		ready = false
		errorMessage = fmt.Sprintf("VolumeSnapshots (%s) missing", strings.Join(deletedSnapshots, ","))
	} else if len(skippedSnapshots) > 0 {
		ready = false
		errorMessage = fmt.Sprintf("VolumeSnapshots (%s) skipped because in error state", strings.Join(skippedSnapshots, ","))
	} else {
		for _, vss := range volumeSnapshotStatus {
			if vss.ReadyToUse == nil || !*vss.ReadyToUse {
				ready = false
				break
			}
		}
	}

	if ready && (contentCpy.Status.ReadyToUse == nil || !*contentCpy.Status.ReadyToUse) {
		contentCpy.Status.CreationTime = currentTime()
	}

	if errorMessage != "" {
		contentCpy.Status.Error = &snapshotv1.Error{
			Time:    currentTime(),
			Message: &errorMessage,
		}
	}

	contentCpy.Status.ReadyToUse = &ready
	contentCpy.Status.VolumeSnapshotStatus = volumeSnapshotStatus

	if !reflect.DeepEqual(content, contentCpy) {
		if _, err := ctrl.Client.VirtualMachineSnapshotContent(contentCpy.Namespace).Update(context.Background(), contentCpy, metav1.UpdateOptions{}); err != nil {
			return 0, err
		}
	}

	return 0, nil
}

func (ctrl *VMSnapshotController) createVolumeSnapshot(
	content *snapshotv1.VirtualMachineSnapshotContent,
	volumeBackup snapshotv1.VolumeBackup,
) (*vsv1beta1.VolumeSnapshot, error) {
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
	snapshot := &vsv1beta1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name: *volumeBackup.VolumeSnapshotName,
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
		Spec: vsv1beta1.VolumeSnapshotSpec{
			Source: vsv1beta1.VolumeSnapshotSource{
				PersistentVolumeClaimName: &volumeBackup.PersistentVolumeClaim.Name,
			},
			VolumeSnapshotClassName: &volumeSnapshotClass,
		},
	}

	volumeSnapshot, err := ctrl.Client.KubernetesSnapshotClient().SnapshotV1beta1().
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

func (ctrl *VMSnapshotController) initVMSnapshot(vmSnapshot *snapshotv1.VirtualMachineSnapshot) (bool, error) {
	if controller.HasFinalizer(vmSnapshot, vmSnapshotFinalizer) {
		return false, nil
	}

	vmSnapshotCpy := vmSnapshot.DeepCopy()
	controller.AddFinalizer(vmSnapshotCpy, vmSnapshotFinalizer)

	onlineSnapshot, err := ctrl.checkOnlineSnapshotting(vmSnapshotCpy)
	if err != nil {
		return false, err
	}

	if onlineSnapshot {
		log.Log.V(3).Infof("Updating snapshot indications that the vm is online during the snapshot")
		vmSnapshotCpy.Status.Indications = append(vmSnapshotCpy.Status.Indications, snapshotv1.VMSnapshotOnlineSnapshotIndication)
		vmSnapshotCpy.Status.Indications = append(vmSnapshotCpy.Status.Indications, snapshotv1.VMSnapshotNoGuestAgentIndication)
	}

	if _, err := ctrl.Client.VirtualMachineSnapshot(vmSnapshot.Namespace).Update(context.Background(), vmSnapshotCpy, metav1.UpdateOptions{}); err != nil {
		return false, err
	}

	return true, nil
}

func (ctrl *VMSnapshotController) cleanupVMSnapshot(vmSnapshot *snapshotv1.VirtualMachineSnapshot) error {
	// TODO check restore in progress

	if vmSnapshotProgressing(vmSnapshot) {
		// will put the snapshot in error state
		return ctrl.updateSnapshotStatus(vmSnapshot, nil)
	}

	content, err := ctrl.getContent(vmSnapshot)
	if err != nil {
		return err
	}

	if content != nil {
		if controller.HasFinalizer(content, vmSnapshotContentFinalizer) {
			cpy := content.DeepCopy()
			controller.RemoveFinalizer(cpy, vmSnapshotContentFinalizer)

			_, err := ctrl.Client.VirtualMachineSnapshotContent(cpy.Namespace).Update(context.Background(), cpy, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}

		if vmSnapshot.Spec.DeletionPolicy == nil ||
			*vmSnapshot.Spec.DeletionPolicy == snapshotv1.VirtualMachineSnapshotContentDelete {
			log.Log.V(2).Infof("Deleting vmsnapshotcontent %s/%s", content.Namespace, content.Name)

			err = ctrl.Client.VirtualMachineSnapshotContent(vmSnapshot.Namespace).Delete(context.Background(), content.Name, metav1.DeleteOptions{})
			if err != nil && !errors.IsNotFound(err) {
				return err
			}
		} else {
			log.Log.V(2).Infof("NOT deleting vmsnapshotcontent %s/%s", content.Namespace, content.Name)
		}
	}

	if controller.HasFinalizer(vmSnapshot, vmSnapshotFinalizer) {
		vmSnapshotCpy := vmSnapshot.DeepCopy()
		controller.RemoveFinalizer(vmSnapshotCpy, vmSnapshotFinalizer)

		_, err := ctrl.Client.VirtualMachineSnapshot(vmSnapshotCpy.Namespace).Update(context.Background(), vmSnapshotCpy, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (ctrl *VMSnapshotController) createContent(vmSnapshot *snapshotv1.VirtualMachineSnapshot) error {
	source, err := ctrl.getSnapshotSource(vmSnapshot)
	if err != nil {
		return err
	}

	var volumeBackups []snapshotv1.VolumeBackup
	for volumeName, pvcName := range source.PersistentVolumeClaims() {
		pvc, err := ctrl.getSnapshotPVC(vmSnapshot.Namespace, pvcName)
		if err != nil {
			return err
		}

		if pvc == nil {
			log.Log.Warningf("No VolumeSnapshotClass for %s/%s", vmSnapshot.Namespace, pvcName)
			continue
		}

		volumeSnapshotName := fmt.Sprintf("vmsnapshot-%s-volume-%s", vmSnapshot.UID, volumeName)
		vb := snapshotv1.VolumeBackup{
			VolumeName: volumeName,
			PersistentVolumeClaim: snapshotv1.PersistentVolumeClaim{
				ObjectMeta: *pvc.ObjectMeta.DeepCopy(),
				Spec:       *pvc.Spec.DeepCopy(),
			},
			VolumeSnapshotName: &volumeSnapshotName,
		}

		volumeBackups = append(volumeBackups, vb)
	}

	content := &snapshotv1.VirtualMachineSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{
			Name:       getVMSnapshotContentName(vmSnapshot),
			Namespace:  vmSnapshot.Namespace,
			Finalizers: []string{vmSnapshotContentFinalizer},
		},
		Spec: snapshotv1.VirtualMachineSnapshotContentSpec{
			VirtualMachineSnapshotName: &vmSnapshot.Name,
			Source:                     source.Spec(),
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

	var matches []vsv1beta1.VolumeSnapshotClass
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

func (ctrl *VMSnapshotController) updateVMSnapshotError(vmSnapshot *snapshotv1.VirtualMachineSnapshot, reason string) error {
	vmSnapshot.Status.Error = newError(reason)
	updateSnapshotCondition(vmSnapshot, newProgressingCondition(corev1.ConditionFalse, reason))
	updateSnapshotCondition(vmSnapshot, newReadyCondition(corev1.ConditionFalse, reason))
	if _, err := ctrl.Client.VirtualMachineSnapshot(vmSnapshot.Namespace).Update(context.Background(), vmSnapshot, metav1.UpdateOptions{}); err != nil {
		log.Log.Errorf("Failed to update error: '%s' on the vm snapshot %s", reason, vmSnapshot.Name)
		return err
	}
	return nil
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

	if vmSnapshotCpy.DeletionTimestamp != nil {
		// go into error state
		if vmSnapshotProgressing(vmSnapshotCpy) {
			reason := "Snapshot cancelled"
			vmSnapshotCpy.Status.Error = newError(reason)
			updateSnapshotCondition(vmSnapshotCpy, newProgressingCondition(corev1.ConditionFalse, reason))
			updateSnapshotCondition(vmSnapshotCpy, newReadyCondition(corev1.ConditionFalse, reason))
		}
	} else {
		content, err := ctrl.getContent(vmSnapshot)
		if err != nil {
			return err
		}

		if content != nil && content.Status != nil {
			// content exists and is initialized
			vmSnapshotCpy.Status.VirtualMachineSnapshotContentName = &content.Name
			vmSnapshotCpy.Status.CreationTime = content.Status.CreationTime
			vmSnapshotCpy.Status.ReadyToUse = content.Status.ReadyToUse
			vmSnapshotCpy.Status.Error = content.Status.Error

			if vmSnapshotReady(vmSnapshotCpy) {
				err := ctrl.unfreezeGuestFSIfNeeded(vmSnapshotCpy)
				if err != nil {
					reason := "Failed unfreezing guest FS"
					ctrl.updateVMSnapshotError(vmSnapshotCpy, reason)
					return err
				}
			}

		}
	}

	if vmSnapshotProgressing(vmSnapshotCpy) {
		source, err := ctrl.getSnapshotSource(vmSnapshot)
		if err != nil {
			return err
		}

		if source != nil {
			if source.Locked() {
				updateSnapshotCondition(vmSnapshotCpy, newProgressingCondition(corev1.ConditionTrue, "Source locked and operation in progress"))
			} else {
				updateSnapshotCondition(vmSnapshotCpy, newProgressingCondition(corev1.ConditionFalse, "Source not locked"))
			}
		} else {
			updateSnapshotCondition(vmSnapshotCpy, newProgressingCondition(corev1.ConditionFalse, "Source does not exist"))
		}
		updateSnapshotCondition(vmSnapshotCpy, newReadyCondition(corev1.ConditionFalse, "Not ready"))
	} else if vmSnapshotError(vmSnapshotCpy) != nil {
		updateSnapshotCondition(vmSnapshotCpy, newProgressingCondition(corev1.ConditionFalse, "In error state"))
		updateSnapshotCondition(vmSnapshotCpy, newReadyCondition(corev1.ConditionFalse, "Error"))
	} else if vmSnapshotReady(vmSnapshotCpy) {
		updateSnapshotCondition(vmSnapshotCpy, newProgressingCondition(corev1.ConditionFalse, "Operation complete"))
		updateSnapshotCondition(vmSnapshotCpy, newReadyCondition(corev1.ConditionTrue, "Operation complete"))
	} else {
		updateSnapshotCondition(vmSnapshotCpy, newProgressingCondition(corev1.ConditionUnknown, "Unknown state"))
		updateSnapshotCondition(vmSnapshotCpy, newReadyCondition(corev1.ConditionUnknown, "Unknown state"))
	}

	if !reflect.DeepEqual(vmSnapshot, vmSnapshotCpy) {
		if _, err := ctrl.Client.VirtualMachineSnapshot(vmSnapshotCpy.Namespace).Update(context.Background(), vmSnapshotCpy, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func vmsnapshotHasIndication(vmSnapshot *snapshotv1.VirtualMachineSnapshot, indication snapshotv1.Indication) bool {
	for _, currIndication := range vmSnapshot.Status.Indications {
		if indication == currIndication {
			return true
		}
	}
	return false
}

func (ctrl *VMSnapshotController) checkGuestAgentParticipation(vmSnapshot *snapshotv1.VirtualMachineSnapshot) (bool, error) {
	condManager := controller.NewVirtualMachineInstanceConditionManager()
	vm, err := ctrl.getVM(vmSnapshot)
	if err != nil {
		return false, err
	}
	vmi, exists, err := ctrl.getVMI(vm)
	if err != nil || !exists {
		return false, err
	}

	return condManager.HasCondition(vmi, kubevirtv1.VirtualMachineInstanceAgentConnected), nil
}

func (ctrl *VMSnapshotController) replaceGuestAgentIndication(vmSnapshot *snapshotv1.VirtualMachineSnapshot) error {
	for i, indication := range vmSnapshot.Status.Indications {
		if indication == snapshotv1.VMSnapshotNoGuestAgentIndication {
			vmSnapshot.Status.Indications[i] = snapshotv1.VMSnapshotGuestAgentIndication
			break
		}
	}

	if _, err := ctrl.Client.VirtualMachineSnapshot(vmSnapshot.Namespace).Update(context.Background(), vmSnapshot, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

func (ctrl *VMSnapshotController) freezeGuestFSIfNeeded(vmSnapshot *snapshotv1.VirtualMachineSnapshot) error {
	if !vmsnapshotHasIndication(vmSnapshot, snapshotv1.VMSnapshotOnlineSnapshotIndication) ||
		findCondition(vmSnapshot.Status.Conditions, newFreezingCondition(corev1.ConditionTrue, vmSnapshotSourceFrozen)) {
		return nil
	}

	GAParticipation, err := ctrl.checkGuestAgentParticipation(vmSnapshot)
	if err != nil || !GAParticipation {
		return err
	}

	vmName := vmSnapshot.Spec.Source.Name
	log.Log.V(3).Infof("Freezing vm %s file system before taking the snapshot", vmName)

	err = ctrl.Client.VirtualMachineInstance(vmSnapshot.Namespace).Freeze(vmName)
	if err != nil {
		reason := "Failed freezing guest FS"
		ctrl.updateVMSnapshotError(vmSnapshot, reason)
		return err
	}

	return ctrl.replaceGuestAgentIndication(vmSnapshot)
}

func (ctrl *VMSnapshotController) unfreezeGuestFSIfNeeded(vmSnapshot *snapshotv1.VirtualMachineSnapshot) error {
	if !vmsnapshotHasIndication(vmSnapshot, snapshotv1.VMSnapshotGuestAgentIndication) {
		return nil
	}

	vmName := vmSnapshot.Spec.Source.Name
	log.Log.V(3).Infof("Unfreezing vm %s file system after taking the snapshot", vmName)

	err := ctrl.Client.VirtualMachineInstance(vmSnapshot.Namespace).Unfreeze(vmName)
	if err != nil {
		return err
	}

	return nil
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

func (ctrl *VMSnapshotController) getVolumeStorageClass(namespace string, volume *kubevirtv1.Volume) (string, error) {
	// TODO Add Ephemeral (add "|| volume.VolumeSource.Ephemeral != nil" to the `if` below)
	if volume.VolumeSource.PersistentVolumeClaim != nil {
		pvcKey := cacheKeyFunc(namespace, volume.VolumeSource.PersistentVolumeClaim.ClaimName)
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
	if volume.VolumeSource.DataVolume != nil {
		storageClassName, err := ctrl.getStorageClassNameForDV(namespace, volume.VolumeSource.DataVolume.Name)
		if err != nil {
			return "", err
		}
		return storageClassName, nil
	}
	return "", fmt.Errorf("Volume type has no StorageClass defined")
}

func (ctrl *VMSnapshotController) getStorageClassNameForDV(namespace string, dvName string) (string, error) {
	// First, look up DV's StorageClass
	key := cacheKeyFunc(namespace, dvName)

	obj, exists, err := ctrl.DVInformer.GetStore().GetByKey(key)
	if err != nil {
		return "", err
	}

	if !exists {
		log.Log.V(3).Infof("DV not in cache [%s]", key)
		return "", fmt.Errorf("DV '%s' not found", key)
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
	pvcKey := cacheKeyFunc(namespace, dvName)
	// TODO Change when PVC rename PR is implemented

	obj, exists, err = ctrl.PVCInformer.GetStore().GetByKey(pvcKey)
	if err != nil {
		return "", err
	}

	if !exists {
		log.Log.V(3).Infof("PVC not in cache [%s]", pvcKey)
		return "", fmt.Errorf("PVC for the DataVolume `%s` not found", dvName)
	}

	pvc := obj.(*corev1.PersistentVolumeClaim)
	if pvc.Spec.StorageClassName != nil {
		return *pvc.Spec.StorageClassName, nil
	}

	log.Log.V(3).Info("PVC has no StorageClassName")
	return "", nil
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
	contentName := getVMSnapshotContentName(vmSnapshot)
	obj, exists, err := ctrl.VMSnapshotContentInformer.GetStore().GetByKey(cacheKeyFunc(vmSnapshot.Namespace, contentName))
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	return obj.(*snapshotv1.VirtualMachineSnapshotContent).DeepCopy(), nil
}

func checkVMRunning(vm *kubevirtv1.VirtualMachine) (bool, error) {
	rs, err := vm.RunStrategy()
	if err != nil {
		return false, err
	}

	return rs != kubevirtv1.RunStrategyHalted, nil
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

func (ctrl *VMSnapshotController) checkOnlineSnapshotting(vmSnapshot *snapshotv1.VirtualMachineSnapshot) (bool, error) {
	vm, err := ctrl.getVM(vmSnapshot)
	if err != nil {
		return false, err
	}

	vmRunning, err := checkVMRunning(vm)
	if err != nil {
		return false, err
	}

	exists, err := ctrl.checkVMIRunning(vm)
	if err != nil {
		return false, err
	}

	return (vmRunning || exists), nil
}

func (s *vmSnapshotSource) UID() types.UID {
	return s.vm.UID
}

func (s *vmSnapshotSource) Locked() bool {
	return s.vm.Status.SnapshotInProgress != nil &&
		*s.vm.Status.SnapshotInProgress == s.snapshot.Name &&
		controller.HasFinalizer(s.vm, sourceFinalizer)
}

func (s *vmSnapshotSource) Lock() (bool, error) {
	if s.Locked() {
		return true, nil
	}

	vmRunning, err := checkVMRunning(s.vm)
	if err != nil {
		return false, err
	}

	exists, err := s.controller.checkVMIRunning(s.vm)
	if err != nil {
		return false, err
	}

	if !vmRunning && !exists {
		pvcNames := s.pvcNames()
		pods, err := podsUsingPVCs(s.controller.PodInformer, s.vm.Namespace, pvcNames)
		if err != nil {
			return false, err
		}

		if len(pods) > 0 {
			log.Log.V(3).Infof("Vm is offline but %d pods using PVCs %+v", len(pods), pvcNames)
			return false, nil
		}
	}

	if s.vm.Status.SnapshotInProgress != nil && *s.vm.Status.SnapshotInProgress != s.snapshot.Name {
		log.Log.V(3).Infof("Snapshot %s in progress", *s.vm.Status.SnapshotInProgress)
		return false, nil
	}

	log.Log.Infof("Adding VM snapshot finalizer to %s", s.vm.Name)

	vmCopy := s.vm.DeepCopy()

	if vmCopy.Status.SnapshotInProgress == nil {
		vmCopy.Status.SnapshotInProgress = &s.snapshot.Name
		// unfortunately, status updater does not return the updated resource
		// but the controller is watching VMs so will get notified
		return false, s.controller.vmStatusUpdater.UpdateStatus(vmCopy)
	}

	if !controller.HasFinalizer(vmCopy, sourceFinalizer) {
		controller.AddFinalizer(vmCopy, sourceFinalizer)
		_, err = s.controller.Client.VirtualMachine(vmCopy.Namespace).Update(vmCopy)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func (s *vmSnapshotSource) Unlock() (bool, error) {
	if s.vm.Status.SnapshotInProgress == nil || *s.vm.Status.SnapshotInProgress != s.snapshot.Name {
		return false, nil
	}

	var err error
	vmCopy := s.vm.DeepCopy()

	if controller.HasFinalizer(vmCopy, sourceFinalizer) {
		controller.RemoveFinalizer(vmCopy, sourceFinalizer)
		vmCopy, err = s.controller.Client.VirtualMachine(vmCopy.Namespace).Update(vmCopy)
		if err != nil {
			return false, err
		}
	}

	vmCopy.Status.SnapshotInProgress = nil
	err = s.controller.vmStatusUpdater.UpdateStatus(vmCopy)
	if err != nil {
		return true, err
	}

	return true, nil
}

func (s *vmSnapshotSource) Spec() snapshotv1.SourceSpec {
	vmCpy := s.vm.DeepCopy()
	vmCpy.Status = kubevirtv1.VirtualMachineStatus{}
	return snapshotv1.SourceSpec{
		VirtualMachine: vmCpy,
	}
}

func (s *vmSnapshotSource) PersistentVolumeClaims() map[string]string {
	return getPVCsFromVolumes(s.vm.Spec.Template.Spec.Volumes)
}

func (s *vmSnapshotSource) pvcNames() sets.String {
	pvcs := s.PersistentVolumeClaims()
	ss := sets.NewString()
	for _, pvc := range pvcs {
		ss.Insert(pvc)
	}
	return ss
}

func getPVCsFromVolumes(volumes []kubevirtv1.Volume) map[string]string {
	pvcs := map[string]string{}

	for _, volume := range volumes {
		var pvcName string

		if volume.PersistentVolumeClaim != nil {
			pvcName = volume.PersistentVolumeClaim.ClaimName
		} else if volume.DataVolume != nil {
			// TODO Change when PVC Renaming is merged.
			pvcName = volume.DataVolume.Name
		} else {
			continue
		}

		pvcs[volume.Name] = pvcName
	}

	return pvcs
}

func updateSnapshotCondition(ss *snapshotv1.VirtualMachineSnapshot, c snapshotv1.Condition) {
	ss.Status.Conditions = updateCondition(ss.Status.Conditions, c, false)
}
