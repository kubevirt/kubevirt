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

package clone

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	clone "kubevirt.io/api/clone/v1beta1"
	k6tv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/pointer"
	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
	virtsnapshot "kubevirt.io/kubevirt/pkg/storage/snapshot"
)

type cloneSourceType string

const (
	sourceTypeVM       cloneSourceType = "VirtualMachine"
	sourceTypeSnapshot cloneSourceType = "VirtualMachineSnapshot"
)

type cloneTargetType string

const (
	targetTypeVM cloneTargetType = "VirtualMachine"
	defaultType  cloneTargetType = targetTypeVM
)

type syncInfoType struct {
	err             error
	snapshotName    string
	snapshotReady   bool
	restoreName     string
	restoreReady    bool
	targetVMName    string
	targetVMCreated bool
	pvcBound        bool

	event          Event
	reason         string
	isCloneFailing bool
	isClonePending bool
}

// vmCloneInfo stores the current vmclone information
type vmCloneInfo struct {
	vmClone      *clone.VirtualMachineClone
	sourceType   cloneSourceType
	snapshot     *snapshotv1.VirtualMachineSnapshot
	snapshotName string
	sourceVm     *k6tv1.VirtualMachine
}

func (ctrl *VMCloneController) execute(key string) error {
	logger := log.Log

	obj, cloneExists, err := ctrl.vmCloneIndexer.GetByKey(key)
	if err != nil {
		return err
	}

	var vmClone *clone.VirtualMachineClone
	if cloneExists {
		vmClone = obj.(*clone.VirtualMachineClone)
		logger = logger.Object(vmClone)
	} else {
		return nil
	}

	if vmClone.Status.Phase == clone.Succeeded {
		_, vmExists, err := ctrl.vmStore.GetByKey(fmt.Sprintf("%s/%s", vmClone.Namespace, *vmClone.Status.TargetName))
		if err != nil {
			return err
		}

		if !vmExists {
			if vmClone.DeletionTimestamp == nil {
				logger.V(3).Infof("Deleting vm clone for deleted vm %s/%s", vmClone.Namespace, *vmClone.Status.TargetName)
				return ctrl.client.VirtualMachineClone(vmClone.Namespace).Delete(context.Background(), vmClone.Name, v1.DeleteOptions{})
			}
			// nothing to process for a vm clone that's being deleted
			return nil
		}
	}

	syncInfo, err := ctrl.sync(vmClone)
	if err != nil {
		return fmt.Errorf("sync error: %v", err)
	}

	err = ctrl.updateStatus(vmClone, syncInfo)
	if err != nil {
		return fmt.Errorf("error updating status: %v", err)
	}

	if syncErr := syncInfo.err; syncErr != nil {
		return fmt.Errorf("sync error: %v", syncErr)
	}

	return nil
}

func (ctrl *VMCloneController) sync(vmClone *clone.VirtualMachineClone) (syncInfoType, error) {
	cloneInfo, err := ctrl.retrieveCloneInfo(vmClone)
	if err != nil {
		switch errors.Unwrap(err) {
		case ErrSourceDoesntExist:
			// If source does not exist we will wait for source
			// to be created and then vmclone will get reconciled again.
			return syncInfoType{
				isClonePending: true,
				event:          SourceDoesNotExist,
				reason:         err.Error(),
			}, nil
		case ErrSourceWithBackendStorage:
			return syncInfoType{
				isCloneFailing: true,
				event:          SourceWithBackendStorageInvalid,
				reason:         err.Error(),
			}, nil
		default:
			return syncInfoType{}, err
		}
	}

	if ctrl.getTargetType(cloneInfo.vmClone) == targetTypeVM {
		return ctrl.syncTargetVM(cloneInfo), nil
	}
	return syncInfoType{err: fmt.Errorf("target type is unknown: %s", ctrl.getTargetType(cloneInfo.vmClone))}, nil
}

// retrieveCloneInfo initializes all the snapshot and restore information that can be populated from the vm clone resource
func (ctrl *VMCloneController) retrieveCloneInfo(vmClone *clone.VirtualMachineClone) (*vmCloneInfo, error) {
	sourceInfo := vmClone.Spec.Source
	cloneInfo := vmCloneInfo{
		vmClone:    vmClone,
		sourceType: cloneSourceType(sourceInfo.Kind),
	}

	switch cloneSourceType(sourceInfo.Kind) {
	case sourceTypeVM:
		sourceVMObj, err := ctrl.getSource(vmClone, sourceInfo.Name, vmClone.Namespace, string(sourceTypeVM), ctrl.vmStore)
		if err != nil {
			return nil, err
		}

		sourceVM := sourceVMObj.(*k6tv1.VirtualMachine)
		if backendstorage.IsBackendStorageNeeded(sourceVM) {
			return nil, fmt.Errorf("%w: VM %s/%s", ErrSourceWithBackendStorage, vmClone.Namespace, sourceInfo.Name)
		}
		cloneInfo.sourceVm = sourceVM

	case sourceTypeSnapshot:
		sourceSnapshotObj, err := ctrl.getSource(vmClone, sourceInfo.Name, vmClone.Namespace, string(sourceTypeSnapshot), ctrl.snapshotStore)
		if err != nil {
			return nil, err
		}

		sourceSnapshot := sourceSnapshotObj.(*snapshotv1.VirtualMachineSnapshot)
		cloneInfo.snapshot = sourceSnapshot
		cloneInfo.snapshotName = sourceSnapshot.Name

	default:
		return nil, fmt.Errorf("clone %s is defined with an unknown source type %s", vmClone.Name, sourceInfo.Kind)
	}

	if cloneInfo.snapshotName == "" && vmClone.Status.SnapshotName != nil {
		cloneInfo.snapshotName = *vmClone.Status.SnapshotName
	}

	return &cloneInfo, nil
}

func (ctrl *VMCloneController) syncTargetVM(vmCloneInfo *vmCloneInfo) syncInfoType {
	vmClone := vmCloneInfo.vmClone
	syncInfo := syncInfoType{}

	switch vmClone.Status.Phase {
	case clone.PhaseUnset, clone.SnapshotInProgress:

		if vmCloneInfo.sourceType == sourceTypeVM {
			if vmClone.Status.SnapshotName == nil {
				syncInfo = ctrl.createSnapshotFromVm(vmClone, vmCloneInfo.sourceVm, syncInfo)
				return syncInfo
			}
		}

		vmCloneInfo.snapshot, syncInfo = ctrl.verifySnapshotReady(vmClone, vmCloneInfo.snapshotName, vmCloneInfo.vmClone.Namespace, syncInfo)
		if syncInfo.isFailingOrError() || !syncInfo.snapshotReady {
			return syncInfo
		}

		fallthrough

	case clone.RestoreInProgress:
		// Here we have to know the snapshot name
		if vmCloneInfo.snapshot == nil {
			vmCloneInfo.snapshot, syncInfo = ctrl.getSnapshot(vmCloneInfo.snapshotName, vmCloneInfo.vmClone.Namespace, syncInfo)
			if syncInfo.isFailingOrError() {
				return syncInfo
			}
		}

		if vmClone.Status.RestoreName == nil {
			vm, err := ctrl.getVmFromSnapshot(vmCloneInfo.snapshot)
			if err != nil {
				syncInfo.setError(fmt.Errorf("cannot get VM manifest from snapshot: %v", err))
				return syncInfo
			}

			syncInfo = ctrl.createRestoreFromVm(vmClone, vm, vmCloneInfo.snapshotName, syncInfo)
			return syncInfo
		}

		syncInfo = ctrl.verifyRestoreReady(vmClone, vmCloneInfo.vmClone.Namespace, syncInfo)
		if syncInfo.isFailingOrError() || !syncInfo.restoreReady {
			return syncInfo
		}

		fallthrough

	case clone.CreatingTargetVM:

		syncInfo = ctrl.verifyVmReady(vmClone, syncInfo)
		if syncInfo.isFailingOrError() {
			return syncInfo
		}

		fallthrough

	case clone.Succeeded:

		if vmClone.Status.RestoreName != nil {
			syncInfo = ctrl.verifyPVCBound(vmClone, syncInfo)
			if syncInfo.isFailingOrError() || !syncInfo.pvcBound {
				return syncInfo
			}

			syncInfo = ctrl.cleanupRestore(vmClone, syncInfo)
			if syncInfo.isFailingOrError() {
				return syncInfo
			}

			if vmCloneInfo.sourceType == sourceTypeVM {
				syncInfo = ctrl.cleanupSnapshot(vmClone, syncInfo)
				if syncInfo.isFailingOrError() {
					return syncInfo
				}
			}
		}

	default:
		log.Log.Object(vmClone).Infof("clone %s is in phase %s - nothing to do", vmClone.Name, string(vmClone.Status.Phase))
	}

	return syncInfo
}

func (ctrl *VMCloneController) updateStatus(origClone *clone.VirtualMachineClone, syncInfo syncInfoType) error {
	vmClone := origClone.DeepCopy()

	var phaseChanged bool
	assignPhase := func(phase clone.VirtualMachineClonePhase) {
		vmClone.Status.Phase = phase
		phaseChanged = true
	}

	switch {
	case syncInfo.isClonePending:
		ctrl.logAndRecord(vmClone, syncInfo.event, syncInfo.reason)
		updateCloneConditions(vmClone,
			newProgressingCondition(corev1.ConditionFalse, "Pending"),
			newReadyCondition(corev1.ConditionFalse, syncInfo.reason),
		)
	case syncInfo.isCloneFailing:
		ctrl.logAndRecord(vmClone, syncInfo.event, syncInfo.reason)
		assignPhase(clone.Failed)
		updateCloneConditions(vmClone,
			newProgressingCondition(corev1.ConditionFalse, "Failed"),
			newReadyCondition(corev1.ConditionFalse, syncInfo.reason),
		)
	default:
		updateCloneConditions(vmClone,
			newProgressingCondition(corev1.ConditionTrue, "Still processing"),
			newReadyCondition(corev1.ConditionFalse, "Still processing"),
		)
	}

	if isInPhase(vmClone, clone.PhaseUnset) && !syncInfo.isClonePending {
		assignPhase(clone.SnapshotInProgress)
	}
	if isInPhase(vmClone, clone.SnapshotInProgress) {
		if snapshotName := syncInfo.snapshotName; snapshotName != "" {
			vmClone.Status.SnapshotName = pointer.P(snapshotName)
		}

		if syncInfo.snapshotReady {
			assignPhase(clone.RestoreInProgress)
		}
	}
	if isInPhase(vmClone, clone.RestoreInProgress) {
		if restoreName := syncInfo.restoreName; restoreName != "" {
			vmClone.Status.RestoreName = pointer.P(restoreName)
		}

		if syncInfo.restoreReady {
			assignPhase(clone.CreatingTargetVM)
		}
	}
	if isInPhase(vmClone, clone.CreatingTargetVM) {
		if targetVMName := syncInfo.targetVMName; targetVMName != "" {
			vmClone.Status.TargetName = pointer.P(targetVMName)
		}

		if syncInfo.targetVMCreated {
			assignPhase(clone.Succeeded)

		}
	}
	if isInPhase(vmClone, clone.Succeeded) {
		updateCloneConditions(vmClone,
			newProgressingCondition(corev1.ConditionFalse, "Ready"),
			newReadyCondition(corev1.ConditionTrue, "Ready"),
		)
	}

	if syncInfo.pvcBound {
		vmClone.Status.SnapshotName = nil
		vmClone.Status.RestoreName = nil
	}

	if !equality.Semantic.DeepEqual(vmClone.Status, origClone.Status) {
		if phaseChanged {
			log.Log.Object(vmClone).Infof("Changing phase to %s", vmClone.Status.Phase)
		}
		_, err := ctrl.client.VirtualMachineClone(vmClone.Namespace).UpdateStatus(context.Background(), vmClone, v1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func validateVolumeSnapshotStatus(vm *k6tv1.VirtualMachine) error {
	var vssErr error

	for _, v := range vm.Spec.Template.Spec.Volumes {
		if v.PersistentVolumeClaim != nil || v.DataVolume != nil {
			found := false
			for _, vss := range vm.Status.VolumeSnapshotStatuses {
				if v.Name == vss.Name {
					if !vss.Enabled {
						vssErr = errors.Join(vssErr, fmt.Errorf(ErrVolumeNotSnapshotable, v.Name))
					}
					found = true
					break
				}
			}
			if !found {
				vssErr = errors.Join(vssErr, fmt.Errorf(ErrVolumeSnapshotSupportUnknown, v.Name))
			}
		}
	}

	return vssErr
}

func (ctrl *VMCloneController) createSnapshotFromVm(vmClone *clone.VirtualMachineClone, vm *k6tv1.VirtualMachine, syncInfo syncInfoType) syncInfoType {
	err := validateVolumeSnapshotStatus(vm)
	if err != nil {
		return syncInfoType{
			isClonePending: true,
			event:          VMVolumeSnapshotsInvalid,
			reason:         err.Error(),
		}
	}

	snapshot := generateSnapshot(vmClone, vm)
	log.Log.Object(vmClone).Infof("creating snapshot %s for clone %s", snapshot.Name, vmClone.Name)

	createdSnapshot, err := ctrl.client.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, v1.CreateOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			syncInfo.setError(fmt.Errorf("failed creating snapshot %s for clone %s: %v", snapshot.Name, vmClone.Name, err))
			return syncInfo
		}
		syncInfo.snapshotName = snapshot.Name
		return syncInfo
	}

	snapshot = createdSnapshot
	ctrl.logAndRecord(vmClone, SnapshotCreated, fmt.Sprintf("created snapshot %s for clone %s", snapshot.Name, vmClone.Name))
	syncInfo.snapshotName = snapshot.Name

	log.Log.Object(vmClone).V(defaultVerbosityLevel).Infof("snapshot %s was just created, reenqueuing to let snapshot time to finish", snapshot.Name)
	return syncInfo
}

func (ctrl *VMCloneController) verifySnapshotReady(vmClone *clone.VirtualMachineClone, name, namespace string, syncInfo syncInfoType) (*snapshotv1.VirtualMachineSnapshot, syncInfoType) {
	obj, exists, err := ctrl.snapshotStore.GetByKey(getKey(name, namespace))
	if err != nil {
		syncInfo.setError(fmt.Errorf("error getting snapshot %s from cache for clone %s: %v", name, vmClone.Name, err))
		return nil, syncInfo
	} else if !exists {
		syncInfo.setError(fmt.Errorf("snapshot %s is not created yet for clone %s", name, vmClone.Name))
		return nil, syncInfo
	}
	snapshot := obj.(*snapshotv1.VirtualMachineSnapshot)
	log.Log.Object(vmClone).Infof("found snapshot %s for clone %s", snapshot.Name, vmClone.Name)

	if !virtsnapshot.VmSnapshotReady(snapshot) {
		log.Log.Object(vmClone).V(defaultVerbosityLevel).Infof("snapshot %s for clone %s is not ready to use yet", snapshot.Name, vmClone.Name)
		return snapshot, syncInfo
	}

	if err := ctrl.verifySnapshotContent(snapshot); err != nil {
		// At this point the snapshot is already succeded and ready.
		// If there is an issue with the snapshot content something is not right
		// and the clone should fail
		syncInfo.isCloneFailing = true
		syncInfo.event = SnapshotContentInvalid
		syncInfo.reason = err.Error()
		return nil, syncInfo
	}

	ctrl.logAndRecord(vmClone, SnapshotReady, fmt.Sprintf("snapshot %s for clone %s is ready to use", snapshot.Name, vmClone.Name))
	syncInfo.snapshotReady = true

	return snapshot, syncInfo
}

func (ctrl *VMCloneController) getSnapshotContent(snapshot *snapshotv1.VirtualMachineSnapshot) (*snapshotv1.VirtualMachineSnapshotContent, error) {
	contentName := virtsnapshot.GetVMSnapshotContentName(snapshot)
	contentKey := getKey(contentName, snapshot.Namespace)

	contentObj, exists, err := ctrl.snapshotContentStore.GetByKey(contentKey)
	if !exists {
		return nil, fmt.Errorf("snapshot content %s in namespace %s does not exist", contentName, snapshot.Namespace)
	} else if err != nil {
		return nil, err
	}

	return contentObj.(*snapshotv1.VirtualMachineSnapshotContent), nil
}

func (ctrl *VMCloneController) verifySnapshotContent(snapshot *snapshotv1.VirtualMachineSnapshot) error {
	content, err := ctrl.getSnapshotContent(snapshot)
	if err != nil {
		return err
	}

	if content.Spec.VirtualMachineSnapshotName == nil {
		return fmt.Errorf("cannot get snapshot name from content %s", content.Name)
	}

	snapshotName := *content.Spec.VirtualMachineSnapshotName
	vm := content.Spec.Source.VirtualMachine

	if vm.Spec.Template == nil {
		return nil
	}

	if backendstorage.IsBackendStorageNeeded(vm) {
		return fmt.Errorf("%w: snapshot %s/%s", ErrSourceWithBackendStorage, snapshot.Namespace, snapshot.Name)
	}

	var volumesNotBackedUpErr error
	for _, volume := range vm.Spec.Template.Spec.Volumes {
		if volume.PersistentVolumeClaim == nil && volume.DataVolume == nil {
			continue
		}

		foundBackup := false
		for _, volumeBackup := range content.Spec.VolumeBackups {
			if volume.Name == volumeBackup.VolumeName {
				foundBackup = true
				break
			}
		}

		if !foundBackup {
			volumesNotBackedUpErr = errors.Join(volumesNotBackedUpErr, fmt.Errorf(ErrVolumeNotBackedUp, volume.Name, snapshotName))
		}
	}

	return volumesNotBackedUpErr
}

// This method assumes the snapshot exists. If it doesn't - syncInfo is updated accordingly.
func (ctrl *VMCloneController) getSnapshot(snapshotName string, sourceNamespace string, syncInfo syncInfoType) (*snapshotv1.VirtualMachineSnapshot, syncInfoType) {
	obj, exists, err := ctrl.snapshotStore.GetByKey(getKey(snapshotName, sourceNamespace))
	if !exists {
		// At this point the snapshot is already created. If it doesn't exist it means that it's deleted for some
		// reason and the clone should fail
		syncInfo.isCloneFailing = true
		syncInfo.event = SnapshotDeleted
		syncInfo.reason = fmt.Sprintf("snapshot %s does not exist anymore", snapshotName)
		return nil, syncInfo
	}
	if err != nil {
		syncInfo.setError(fmt.Errorf("error getting snapshot %s from cache: %v", snapshotName, err))
		return nil, syncInfo
	}
	snapshot := obj.(*snapshotv1.VirtualMachineSnapshot)

	return snapshot, syncInfo
}

func (ctrl *VMCloneController) createRestoreFromVm(vmClone *clone.VirtualMachineClone, vm *k6tv1.VirtualMachine, snapshotName string, syncInfo syncInfoType) syncInfoType {
	patches, err := generatePatches(vm, &vmClone.Spec)
	if err != nil {
		retErr := fmt.Errorf("error generating patches for clone %s: %v", vmClone.Name, err)
		ctrl.recorder.Event(vmClone, corev1.EventTypeWarning, string(RestoreCreationFailed), retErr.Error())
		syncInfo.setError(retErr)
		return syncInfo
	}
	restore := generateRestore(vmClone.Spec.Target, vm.Name, vmClone.Namespace, vmClone.Name, snapshotName, vmClone.UID, patches)
	log.Log.Object(vmClone).Infof("creating restore %s for clone %s", restore.Name, vmClone.Name)
	createdRestore, err := ctrl.client.VirtualMachineRestore(restore.Namespace).Create(context.Background(), restore, v1.CreateOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			retErr := fmt.Errorf("failed creating restore %s for clone %s: %v", restore.Name, vmClone.Name, err)
			ctrl.recorder.Event(vmClone, corev1.EventTypeWarning, string(RestoreCreationFailed), retErr.Error())
			syncInfo.setError(retErr)
			return syncInfo
		}
		syncInfo.restoreName = restore.Name
		return syncInfo
	}
	restore = createdRestore
	ctrl.logAndRecord(vmClone, RestoreCreated, fmt.Sprintf("created restore %s for clone %s", restore.Name, vmClone.Name))
	syncInfo.restoreName = restore.Name

	log.Log.Object(vmClone).V(defaultVerbosityLevel).Infof("restore %s was just created, reenqueuing to let snapshot time to finish", restore.Name)
	return syncInfo
}

func (ctrl *VMCloneController) verifyRestoreReady(vmClone *clone.VirtualMachineClone, sourceNamespace string, syncInfo syncInfoType) syncInfoType {
	obj, exists, err := ctrl.restoreStore.GetByKey(getKey(*vmClone.Status.RestoreName, sourceNamespace))
	if !exists {
		syncInfo.setError(fmt.Errorf("restore %s is not created yet for clone %s", *vmClone.Status.RestoreName, vmClone.Name))
		return syncInfo
	} else if err != nil {
		syncInfo.setError(fmt.Errorf("error getting restore %s from cache for clone %s: %v", *vmClone.Status.RestoreName, vmClone.Name, err))
		return syncInfo
	}

	restore := obj.(*snapshotv1.VirtualMachineRestore)
	log.Log.Object(vmClone).Infof("found target restore %s for clone %s", restore.Name, vmClone.Name)

	if virtsnapshot.VmRestoreProgressing(restore) {
		log.Log.Object(vmClone).V(defaultVerbosityLevel).Infof("restore %s for clone %s is not ready to use yet", restore.Name, vmClone.Name)
		return syncInfo
	}

	ctrl.logAndRecord(vmClone, RestoreReady, fmt.Sprintf("restore %s for clone %s is ready to use", restore.Name, vmClone.Name))
	syncInfo.restoreReady = true
	syncInfo.targetVMName = restore.Spec.Target.Name

	return syncInfo
}

func (ctrl *VMCloneController) verifyVmReady(vmClone *clone.VirtualMachineClone, syncInfo syncInfoType) syncInfoType {
	targetVMInfo := vmClone.Spec.Target

	_, exists, err := ctrl.vmStore.GetByKey(getKey(targetVMInfo.Name, vmClone.Namespace))
	if !exists {
		syncInfo.setError(fmt.Errorf("target VM %s is not created yet for clone %s", targetVMInfo.Name, vmClone.Name))
		return syncInfo
	} else if err != nil {
		syncInfo.setError(fmt.Errorf("error getting VM %s from cache for clone %s: %v", targetVMInfo.Name, vmClone.Name, err))
		return syncInfo
	}

	ctrl.logAndRecord(vmClone, TargetVMCreated, fmt.Sprintf("created target VM %s for clone %s", targetVMInfo.Name, vmClone.Name))
	syncInfo.targetVMCreated = true

	return syncInfo
}

func (ctrl *VMCloneController) verifyPVCBound(vmClone *clone.VirtualMachineClone, syncInfo syncInfoType) syncInfoType {
	obj, exists, err := ctrl.restoreStore.GetByKey(getKey(*vmClone.Status.RestoreName, vmClone.Namespace))
	if !exists {
		syncInfo.setError(fmt.Errorf("restore %s is not created yet for clone %s", *vmClone.Status.RestoreName, vmClone.Name))
		return syncInfo
	} else if err != nil {
		syncInfo.setError(fmt.Errorf("error getting restore %s from cache for clone %s: %v", *vmClone.Status.SnapshotName, vmClone.Name, err))
		return syncInfo
	}

	restore := obj.(*snapshotv1.VirtualMachineRestore)
	for _, volumeRestore := range restore.Status.Restores {
		obj, exists, err = ctrl.pvcStore.GetByKey(getKey(volumeRestore.PersistentVolumeClaimName, vmClone.Namespace))
		if !exists {
			syncInfo.setError(fmt.Errorf("PVC %s is not created yet for clone %s", volumeRestore.PersistentVolumeClaimName, vmClone.Name))
			return syncInfo
		} else if err != nil {
			syncInfo.setError(fmt.Errorf("error getting PVC %s from cache for clone %s: %v", volumeRestore.PersistentVolumeClaimName, vmClone.Name, err))
			return syncInfo
		}

		pvc := obj.(*corev1.PersistentVolumeClaim)
		if pvc.Status.Phase != corev1.ClaimBound {
			log.Log.Object(vmClone).V(defaultVerbosityLevel).Infof("pvc %s for clone %s is not bound yet", pvc.Name, vmClone.Name)
			return syncInfo
		}
	}

	ctrl.logAndRecord(vmClone, PVCBound, fmt.Sprintf("all PVC for clone %s are bound", vmClone.Name))
	syncInfo.pvcBound = true

	return syncInfo

}

func (ctrl *VMCloneController) cleanupSnapshot(vmClone *clone.VirtualMachineClone, syncInfo syncInfoType) syncInfoType {
	err := ctrl.client.VirtualMachineSnapshot(vmClone.Namespace).Delete(context.Background(), *vmClone.Status.SnapshotName, v1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		syncInfo.setError(fmt.Errorf("cannot clean up snapshot %s for clone %s", *vmClone.Status.SnapshotName, vmClone.Name))
		return syncInfo
	}

	return syncInfo
}

func (ctrl *VMCloneController) cleanupRestore(vmClone *clone.VirtualMachineClone, syncInfo syncInfoType) syncInfoType {
	err := ctrl.client.VirtualMachineRestore(vmClone.Namespace).Delete(context.Background(), *vmClone.Status.RestoreName, v1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		syncInfo.setError(fmt.Errorf("cannot clean up restore %s for clone %s", *vmClone.Status.RestoreName, vmClone.Name))
		return syncInfo
	}

	return syncInfo
}

func (ctrl *VMCloneController) logAndRecord(vmClone *clone.VirtualMachineClone, event Event, msg string) {
	ctrl.recorder.Eventf(vmClone, corev1.EventTypeNormal, string(event), msg)
	log.Log.Object(vmClone).Infof(msg)
}

func (ctrl *VMCloneController) getTargetType(vmClone *clone.VirtualMachineClone) cloneTargetType {
	if vmClone.Spec.Target != nil {
		return cloneTargetType(vmClone.Spec.Target.Kind)
	} else {
		return defaultType
	}
}

func (ctrl *VMCloneController) getSource(vmClone *clone.VirtualMachineClone, name, namespace, sourceKind string, store cache.Store) (interface{}, error) {
	key := getKey(name, namespace)
	obj, exists, err := store.GetByKey(key)
	if err != nil {
		return nil, fmt.Errorf("error getting %s %s in namespace %s from cache: %v", sourceKind, name, namespace, err)
	}
	if !exists {
		return nil, fmt.Errorf("%w: %s %s/%s", ErrSourceDoesntExist, sourceKind, namespace, name)
	}

	return obj, nil
}

func (ctrl *VMCloneController) getVmFromSnapshot(snapshot *snapshotv1.VirtualMachineSnapshot) (*k6tv1.VirtualMachine, error) {
	content, err := ctrl.getSnapshotContent(snapshot)
	if err != nil {
		return nil, err
	}

	contentVmSpec := content.Spec.Source.VirtualMachine

	vm := &k6tv1.VirtualMachine{
		ObjectMeta: contentVmSpec.ObjectMeta,
		Spec:       contentVmSpec.Spec,
		Status:     contentVmSpec.Status,
	}

	return vm, nil
}

func (s *syncInfoType) setError(err error) {
	s.err = err
}

func (s *syncInfoType) isFailingOrError() bool {
	return s.err != nil || s.isCloneFailing
}
