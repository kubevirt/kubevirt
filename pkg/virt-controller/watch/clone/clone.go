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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package clone

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/cache"

	virtsnapshot "kubevirt.io/kubevirt/pkg/storage/snapshot"

	"k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/utils/pointer"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"
	k6tv1 "kubevirt.io/api/core/v1"
	snapshotv1alpha1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/log"
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

	isCloneFailing bool
	failEvent      Event
	failReason     string

	// This flag is true when we need to reenqueue and return syncInfo from sync() for a reason not specified above.
	needToReenqueue bool
	logger          *log.FilteredLogger
}

func (ctrl *VMCloneController) execute(key string) error {
	logger := log.Log

	obj, cloneExists, err := ctrl.vmCloneInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	}

	var vmClone *clonev1alpha1.VirtualMachineClone
	if cloneExists {
		vmClone = obj.(*clonev1alpha1.VirtualMachineClone)
		logger = logger.Object(vmClone)
	} else {
		return nil
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

func (ctrl *VMCloneController) sync(vmClone *clonev1alpha1.VirtualMachineClone) (syncInfoType, error) {
	var syncInfo syncInfoType
	sourceInfo := vmClone.Spec.Source

	switch cloneSourceType(sourceInfo.Kind) {
	case sourceTypeVM:
		sourceVMObj, err := ctrl.getSource(vmClone, sourceInfo.Name, vmClone.Namespace, string(sourceTypeVM), ctrl.vmInformer.GetStore())
		if err != nil {
			return syncInfo, err
		}

		sourceVM := sourceVMObj.(*k6tv1.VirtualMachine)

		syncInfo = ctrl.syncSourceVM(sourceVM, vmClone)
		return syncInfo, nil

	case sourceTypeSnapshot:
		sourceSnapshotObj, err := ctrl.getSource(vmClone, sourceInfo.Name, vmClone.Namespace, string(sourceTypeSnapshot), ctrl.snapshotInformer.GetStore())
		if err != nil {
			return syncInfo, err
		}

		sourceSnapshot := sourceSnapshotObj.(*snapshotv1alpha1.VirtualMachineSnapshot)

		syncInfo = ctrl.syncSourceSnapshot(sourceSnapshot, vmClone)
		return syncInfo, nil

	default:
		return syncInfo, fmt.Errorf("clone %s is defined with an unknown source type %s", vmClone.Name, sourceInfo.Kind)
	}
}

func (ctrl *VMCloneController) syncSourceVM(source *k6tv1.VirtualMachine, vmClone *clonev1alpha1.VirtualMachineClone) syncInfoType {
	targetType := ctrl.getTargetType(vmClone)

	switch targetType {
	case targetTypeVM:
		return ctrl.syncSourceVMTargetVM(source, vmClone)

	default:
		return syncInfoType{err: fmt.Errorf("target type is unknown: %s", targetType)}
	}
}

func (ctrl *VMCloneController) syncSourceSnapshot(source *snapshotv1alpha1.VirtualMachineSnapshot, vmClone *clonev1alpha1.VirtualMachineClone) syncInfoType {
	targetType := ctrl.getTargetType(vmClone)

	switch targetType {
	case targetTypeVM:
		return ctrl.syncSourceSnapshotTargetVM(source, vmClone)

	default:
		return syncInfoType{err: fmt.Errorf("target type is unknown: %s", targetType)}
	}
}

func (ctrl *VMCloneController) syncSourceVMTargetVM(source *k6tv1.VirtualMachine, vmClone *clonev1alpha1.VirtualMachineClone) syncInfoType {
	syncInfo := syncInfoType{logger: log.Log.Object(vmClone)}

	var snapshot *snapshotv1alpha1.VirtualMachineSnapshot

	switch vmClone.Status.Phase {
	case clonev1alpha1.PhaseUnset, clonev1alpha1.SnapshotInProgress:

		if vmClone.Status.SnapshotName == nil {
			_, syncInfo = ctrl.createSnapshotFromVm(vmClone, source, syncInfo)
			return syncInfo
		}

		snapshot, syncInfo = ctrl.verifySnapshotReady(vmClone, *vmClone.Status.SnapshotName, source.Namespace, syncInfo)
		if syncInfo.toReenqueue() || !syncInfo.snapshotReady {
			return syncInfo
		}

		fallthrough

	case clonev1alpha1.RestoreInProgress:

		if snapshot == nil {
			snapshot, syncInfo = ctrl.getSnapshot(vmClone, source.Namespace, syncInfo)
			if syncInfo.toReenqueue() {
				return syncInfo
			}
		}

		if vmClone.Status.RestoreName == nil {
			syncInfo = ctrl.createRestoreFromVm(vmClone, source, snapshot.Name, syncInfo)
			return syncInfo
		}

		syncInfo = ctrl.verifyRestoreReady(vmClone, source.Namespace, syncInfo)
		if syncInfo.toReenqueue() {
			return syncInfo
		}

		fallthrough

	case clonev1alpha1.CreatingTargetVM:

		syncInfo = ctrl.verifyVmReady(vmClone, syncInfo)
		if syncInfo.toReenqueue() {
			return syncInfo
		}

		syncInfo = ctrl.cleanupSnapshot(vmClone, syncInfo)
		if syncInfo.toReenqueue() {
			return syncInfo
		}

		syncInfo = ctrl.cleanupRestore(vmClone, syncInfo)
		if syncInfo.toReenqueue() {
			return syncInfo
		}

	default:
		log.Log.Object(vmClone).Infof("clone %s is in phase %s - nothing to do", vmClone.Name, string(vmClone.Status.Phase))
	}

	return syncInfo
}

func (ctrl *VMCloneController) syncSourceSnapshotTargetVM(source *snapshotv1alpha1.VirtualMachineSnapshot, vmClone *clonev1alpha1.VirtualMachineClone) syncInfoType {
	syncInfo := syncInfoType{logger: log.Log.Object(vmClone)}

	switch vmClone.Status.Phase {
	case clonev1alpha1.PhaseUnset, clonev1alpha1.SnapshotInProgress:

		syncInfo.snapshotName = source.Name
		source, syncInfo = ctrl.verifySnapshotReady(vmClone, source.Name, source.Namespace, syncInfo)
		if syncInfo.toReenqueue() || !syncInfo.snapshotReady {
			return syncInfo
		}

		fallthrough

	case clonev1alpha1.RestoreInProgress:

		if vmClone.Status.RestoreName == nil {
			vm, err := ctrl.getVmFromSnapshot(source)
			if err != nil {
				return addErrorToSyncInfo(syncInfo, fmt.Errorf("cannot get VM manifest from snapshot: %v", err))
			}

			syncInfo = ctrl.createRestoreFromVm(vmClone, vm, source.Name, syncInfo)
			return syncInfo
		}

		syncInfo = ctrl.verifyRestoreReady(vmClone, source.Namespace, syncInfo)
		if syncInfo.toReenqueue() {
			return syncInfo
		}

		fallthrough

	case clonev1alpha1.CreatingTargetVM:

		syncInfo = ctrl.verifyVmReady(vmClone, syncInfo)
		if syncInfo.toReenqueue() {
			return syncInfo
		}

		syncInfo = ctrl.cleanupRestore(vmClone, syncInfo)
		if syncInfo.toReenqueue() {
			return syncInfo
		}

	default:
		log.Log.Object(vmClone).Infof("clone %s is in phase %s - nothing to do", vmClone.Name, string(vmClone.Status.Phase))
	}

	return syncInfo
}

func (ctrl *VMCloneController) updateStatus(origClone *clonev1alpha1.VirtualMachineClone, syncInfo syncInfoType) error {
	vmClone := origClone.DeepCopy()

	var phaseChanged bool
	assignPhase := func(phase clonev1alpha1.VirtualMachineClonePhase) {
		vmClone.Status.Phase = phase
		phaseChanged = true
	}

	if syncInfo.isCloneFailing {
		ctrl.logAndRecord(vmClone, syncInfo.failEvent, syncInfo.failReason)
		assignPhase(clonev1alpha1.Failed)
		updateCloneConditions(vmClone,
			newProgressingCondition(corev1.ConditionFalse, "Failed"),
			newReadyCondition(corev1.ConditionFalse, "Failed"),
		)
	}

	updateCloneConditions(vmClone,
		newProgressingCondition(corev1.ConditionTrue, "Still processing"),
		newReadyCondition(corev1.ConditionFalse, "Still processing"),
	)

	if isInPhase(vmClone, clonev1alpha1.PhaseUnset) {
		assignPhase(clonev1alpha1.SnapshotInProgress)
	}
	if isInPhase(vmClone, clonev1alpha1.SnapshotInProgress) {
		if snapshotName := syncInfo.snapshotName; snapshotName != "" {
			vmClone.Status.SnapshotName = pointer.String(snapshotName)
		}

		if syncInfo.snapshotReady {
			assignPhase(clonev1alpha1.RestoreInProgress)
		}
	}
	if isInPhase(vmClone, clonev1alpha1.RestoreInProgress) {
		if restoreName := syncInfo.restoreName; restoreName != "" {
			vmClone.Status.RestoreName = pointer.String(restoreName)
		}

		if syncInfo.restoreReady {
			assignPhase(clonev1alpha1.CreatingTargetVM)
		}
	}
	if isInPhase(vmClone, clonev1alpha1.CreatingTargetVM) {
		if targetVMName := syncInfo.targetVMName; targetVMName != "" {
			vmClone.Status.TargetName = pointer.String(targetVMName)
		}

		if syncInfo.targetVMCreated {
			vmClone.Status.SnapshotName = nil
			vmClone.Status.RestoreName = nil
			assignPhase(clonev1alpha1.Succeeded)

		}
	}
	if isInPhase(vmClone, clonev1alpha1.Succeeded) {
		updateCloneConditions(vmClone,
			newProgressingCondition(corev1.ConditionFalse, "Ready"),
			newReadyCondition(corev1.ConditionTrue, "Ready"),
		)
	}

	if !equality.Semantic.DeepEqual(vmClone.Status, origClone.Status) {
		if phaseChanged {
			log.Log.Object(vmClone).Infof("Changing phase to %s", vmClone.Status.Phase)
		}
		err := ctrl.cloneStatusUpdater.UpdateStatus(vmClone)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ctrl *VMCloneController) createSnapshotFromVm(vmClone *clonev1alpha1.VirtualMachineClone, vm *k6tv1.VirtualMachine, syncInfo syncInfoType) (*snapshotv1alpha1.VirtualMachineSnapshot, syncInfoType) {
	snapshot := generateSnapshot(vmClone, vm)
	syncInfo.logger.Infof("creating snapshot %s for clone %s", snapshot.Name, vmClone.Name)

	createdSnapshot, err := ctrl.client.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, v1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return snapshot, addErrorToSyncInfo(syncInfo, fmt.Errorf("failed creating snapshot %s for clone %s: %v", snapshot.Name, vmClone.Name, err))
		}
		syncInfo.snapshotName = snapshot.Name
		return snapshot, syncInfo
	}

	snapshot = createdSnapshot
	ctrl.logAndRecord(vmClone, SnapshotCreated, fmt.Sprintf("created snapshot %s for clone %s", snapshot.Name, vmClone.Name))
	syncInfo.snapshotName = snapshot.Name

	syncInfo.logger.V(defaultVerbosityLevel).Infof("snapshot %s was just created, reenqueuing to let snapshot time to finish", snapshot.Name)
	return snapshot, syncInfo
}

func (ctrl *VMCloneController) verifySnapshotReady(vmClone *clonev1alpha1.VirtualMachineClone, name, namespace string, syncInfo syncInfoType) (*snapshotv1alpha1.VirtualMachineSnapshot, syncInfoType) {
	obj, exists, err := ctrl.snapshotInformer.GetStore().GetByKey(getKey(name, namespace))
	if err != nil {
		return nil, addErrorToSyncInfo(syncInfo, fmt.Errorf("error getting snapshot %s from cache for clone %s: %v", name, vmClone.Name, err))
	} else if !exists {
		return nil, addErrorToSyncInfo(syncInfo, fmt.Errorf("snapshot %s is not created yet for clone %s", name, vmClone.Name))
	}
	snapshot := obj.(*snapshotv1alpha1.VirtualMachineSnapshot)
	syncInfo.logger.Infof("found snapshot %s for clone %s", snapshot.Name, vmClone.Name)

	if !virtsnapshot.VmSnapshotReady(snapshot) {
		syncInfo.logger.V(defaultVerbosityLevel).Infof("snapshot %s for clone %s is not ready to use yet", snapshot.Name, vmClone.Name)
		return snapshot, syncInfo
	}

	ctrl.logAndRecord(vmClone, SnapshotReady, fmt.Sprintf("snapshot %s for clone %s is ready to use", snapshot.Name, vmClone.Name))
	syncInfo.snapshotReady = true

	return snapshot, syncInfo
}

// This method assumes the snapshot exists. If it doesn't - syncInfo is updated accordingly.
func (ctrl *VMCloneController) getSnapshot(vmClone *clonev1alpha1.VirtualMachineClone, sourceNamespace string, syncInfo syncInfoType) (*snapshotv1alpha1.VirtualMachineSnapshot, syncInfoType) {
	obj, exists, err := ctrl.snapshotInformer.GetStore().GetByKey(getKey(*vmClone.Status.SnapshotName, sourceNamespace))
	if !exists {
		// At this point the snapshot is already created. If it doesn't exist it means that it's deleted for some
		// reason and the clone should fail
		syncInfo.isCloneFailing = true
		syncInfo.failEvent = SnapshotDeleted
		syncInfo.failReason = fmt.Sprintf("snapshot %s does not exist anymore", *vmClone.Status.SnapshotName)
		return nil, syncInfo
	}
	if err != nil {
		return nil, addErrorToSyncInfo(syncInfo, fmt.Errorf("error getting snapshot %s from cache for clone %s: %v", *vmClone.Status.SnapshotName, vmClone.Name, err))
	}
	snapshot := obj.(*snapshotv1alpha1.VirtualMachineSnapshot)

	return snapshot, syncInfo
}

func (ctrl *VMCloneController) createRestoreFromVm(vmClone *clonev1alpha1.VirtualMachineClone, vm *k6tv1.VirtualMachine, snapshotName string, syncInfo syncInfoType) syncInfoType {
	patches := generatePatches(vm, &vmClone.Spec)
	restore := generateRestore(vmClone.Spec.Target, vm.Name, vmClone.Namespace, vmClone.Name, snapshotName, vmClone.UID, patches)
	syncInfo.logger.Infof("creating restore %s for clone %s", restore.Name, vmClone.Name)
	createdRestore, err := ctrl.client.VirtualMachineRestore(restore.Namespace).Create(context.Background(), restore, v1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return addErrorToSyncInfo(syncInfo, fmt.Errorf("failed creating restore %s for clone %s: %v", restore.Name, vmClone.Name, err))
		}
		syncInfo.restoreName = restore.Name
		return syncInfo

	}
	restore = createdRestore
	ctrl.logAndRecord(vmClone, RestoreCreated, fmt.Sprintf("created restore %s for clone %s", restore.Name, vmClone.Name))
	syncInfo.restoreName = restore.Name

	syncInfo.logger.V(defaultVerbosityLevel).Infof("restore %s was just created, reenqueuing to let snapshot time to finish", restore.Name)
	return syncInfo
}

func (ctrl *VMCloneController) verifyRestoreReady(vmClone *clonev1alpha1.VirtualMachineClone, sourceNamespace string, syncInfo syncInfoType) syncInfoType {
	obj, exists, err := ctrl.restoreInformer.GetStore().GetByKey(getKey(*vmClone.Status.RestoreName, sourceNamespace))
	if !exists {
		return addErrorToSyncInfo(syncInfo, fmt.Errorf("restore %s is not created yet for clone %s", *vmClone.Status.SnapshotName, vmClone.Name))
	} else if err != nil {
		return addErrorToSyncInfo(syncInfo, fmt.Errorf("error getting snapshot %s from cache for clone %s: %v", *vmClone.Status.SnapshotName, vmClone.Name, err))
	}

	restore := obj.(*snapshotv1alpha1.VirtualMachineRestore)
	syncInfo.logger.Infof("found target restore %s for clone %s", restore.Name, vmClone.Name)

	if virtsnapshot.VmRestoreProgressing(restore) {
		syncInfo.logger.V(defaultVerbosityLevel).Infof("restore %s for clone %s is not ready to use yet", restore.Name, vmClone.Name)
		syncInfo.needToReenqueue = true
		return syncInfo
	}

	ctrl.logAndRecord(vmClone, RestoreReady, fmt.Sprintf("restore %s for clone %s is ready to use", restore.Name, vmClone.Name))
	syncInfo.restoreReady = true
	syncInfo.targetVMName = restore.Spec.Target.Name

	return syncInfo
}

func (ctrl *VMCloneController) verifyVmReady(vmClone *clonev1alpha1.VirtualMachineClone, syncInfo syncInfoType) syncInfoType {
	targetVMInfo := vmClone.Spec.Target

	_, exists, err := ctrl.vmInformer.GetStore().GetByKey(getKey(targetVMInfo.Name, vmClone.Namespace))
	if !exists {
		return addErrorToSyncInfo(syncInfo, fmt.Errorf("target VM %s is not created yet for clone %s", targetVMInfo.Name, vmClone.Name))
	} else if err != nil {
		return addErrorToSyncInfo(syncInfo, fmt.Errorf("error getting VM %s from cache for clone %s: %v", *vmClone.Status.SnapshotName, targetVMInfo.Name, err))
	}

	ctrl.logAndRecord(vmClone, TargetVMCreated, fmt.Sprintf("created target VM %s for clone %s", targetVMInfo.Name, vmClone.Name))
	syncInfo.targetVMCreated = true

	return syncInfo
}

func (ctrl *VMCloneController) cleanupSnapshot(vmClone *clonev1alpha1.VirtualMachineClone, syncInfo syncInfoType) syncInfoType {
	err := ctrl.client.VirtualMachineSnapshot(vmClone.Namespace).Delete(context.Background(), *vmClone.Status.SnapshotName, v1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return addErrorToSyncInfo(syncInfo, fmt.Errorf("cannot clean up snapshot %s for clone %s", *vmClone.Status.SnapshotName, vmClone.Name))
	}

	return syncInfo
}

func (ctrl *VMCloneController) cleanupRestore(vmClone *clonev1alpha1.VirtualMachineClone, syncInfo syncInfoType) syncInfoType {
	err := ctrl.client.VirtualMachineRestore(vmClone.Namespace).Delete(context.Background(), *vmClone.Status.RestoreName, v1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return addErrorToSyncInfo(syncInfo, fmt.Errorf("cannot clean up restore %s for clone %s", *vmClone.Status.RestoreName, vmClone.Name))
	}

	return syncInfo
}

func (ctrl *VMCloneController) logAndRecord(vmClone *clonev1alpha1.VirtualMachineClone, event Event, msg string) {
	ctrl.recorder.Eventf(vmClone, corev1.EventTypeNormal, string(event), msg)
	log.Log.Object(vmClone).Infof(msg)
}

func (ctrl *VMCloneController) getTargetType(vmClone *clonev1alpha1.VirtualMachineClone) cloneTargetType {
	if vmClone.Spec.Target != nil {
		return cloneTargetType(vmClone.Spec.Target.Kind)
	} else {
		return defaultType
	}
}

func (ctrl *VMCloneController) getSource(vmClone *clonev1alpha1.VirtualMachineClone, name, namespace, sourceKind string, store cache.Store) (interface{}, error) {
	key := getKey(name, namespace)
	obj, exists, err := store.GetByKey(key)
	if err != nil {
		return nil, fmt.Errorf("error getting %s %s in namespace %s from cache: %v", sourceKind, name, namespace, err)
	}
	if !exists {
		err = ctrl.updateStatus(vmClone, syncInfoType{
			isCloneFailing: true,
			failEvent:      SourceDoesNotExist,
			failReason:     fmt.Sprintf("%s %s does not exist in namespace %s", sourceKind, name, namespace),
		})

		if err != nil {
			log.Log.Errorf("updating status when source %s does not exist failed: %v", sourceKind, err)
		}

		return nil, fmt.Errorf("%s %s in namespace %s does not exist", sourceKind, name, namespace)
	}

	return obj, nil
}

func (ctrl *VMCloneController) getVmFromSnapshot(snapshot *snapshotv1alpha1.VirtualMachineSnapshot) (*k6tv1.VirtualMachine, error) {
	contentName := virtsnapshot.GetVMSnapshotContentName(snapshot)
	contentKey := getKey(contentName, snapshot.Namespace)

	contentObj, exists, err := ctrl.snapshotContentInformer.GetStore().GetByKey(contentKey)
	if !exists {
		return nil, fmt.Errorf("snapshot content %s in namespace %s does not exist", contentName, snapshot.Namespace)
	} else if err != nil {
		return nil, err
	}

	content := contentObj.(*snapshotv1alpha1.VirtualMachineSnapshotContent)
	contentVmSpec := content.Spec.Source.VirtualMachine

	vm := &k6tv1.VirtualMachine{
		ObjectMeta: contentVmSpec.ObjectMeta,
		Spec:       contentVmSpec.Spec,
		Status:     contentVmSpec.Status,
	}

	return vm, nil
}

func addErrorToSyncInfo(info syncInfoType, err error) syncInfoType {
	info.err = err
	return info
}

func (s *syncInfoType) toReenqueue() bool {
	return s.err != nil || s.isCloneFailing || s.needToReenqueue
}
