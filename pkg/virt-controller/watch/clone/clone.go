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
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
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
	pvcBound        bool

	isCloneFailing bool
	failEvent      Event
	failReason     string
}

// vmCloneInfo stores the current vmclone information
type vmCloneInfo struct {
	vmClone      *clonev1alpha1.VirtualMachineClone
	sourceType   cloneSourceType
	snapshot     *snapshotv1.VirtualMachineSnapshot
	snapshotName string
	sourceVm     *k6tv1.VirtualMachine
	restore      *snapshotv1.VirtualMachineRestore
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

	if vmClone.Status.Phase == clonev1alpha1.Succeeded {
		_, vmExists, err := ctrl.vmInformer.GetStore().GetByKey(fmt.Sprintf("%s/%s", vmClone.Namespace, *vmClone.Status.TargetName))
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

func (ctrl *VMCloneController) sync(vmClone *clonev1alpha1.VirtualMachineClone) (syncInfoType, error) {
	cloneInfo, err := ctrl.retrieveCloneInfo(vmClone)
	if err != nil {
		return syncInfoType{}, err
	}

	if ctrl.getTargetType(cloneInfo.vmClone) == targetTypeVM {
		return ctrl.syncTargetVM(cloneInfo), nil
	}
	return syncInfoType{err: fmt.Errorf("target type is unknown: %s", ctrl.getTargetType(cloneInfo.vmClone))}, nil
}

// retrieveCloneInfo initializes all the snapshot and restore information that can be populated from the vm clone resource
func (ctrl *VMCloneController) retrieveCloneInfo(vmClone *clonev1alpha1.VirtualMachineClone) (*vmCloneInfo, error) {
	sourceInfo := vmClone.Spec.Source
	cloneInfo := vmCloneInfo{
		vmClone:    vmClone,
		sourceType: cloneSourceType(sourceInfo.Kind),
	}

	switch cloneSourceType(sourceInfo.Kind) {
	case sourceTypeVM:
		sourceVMObj, err := ctrl.getSource(vmClone, sourceInfo.Name, vmClone.Namespace, string(sourceTypeVM), ctrl.vmInformer.GetStore())
		if err != nil {
			return nil, err
		}

		sourceVM := sourceVMObj.(*k6tv1.VirtualMachine)
		cloneInfo.sourceVm = sourceVM

	case sourceTypeSnapshot:
		sourceSnapshotObj, err := ctrl.getSource(vmClone, sourceInfo.Name, vmClone.Namespace, string(sourceTypeSnapshot), ctrl.snapshotInformer.GetStore())
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
	case clonev1alpha1.PhaseUnset, clonev1alpha1.SnapshotInProgress:

		if vmCloneInfo.sourceType == sourceTypeVM {
			if vmClone.Status.SnapshotName == nil {
				_, syncInfo = ctrl.createSnapshotFromVm(vmClone, vmCloneInfo.sourceVm, syncInfo)
				return syncInfo
			}
		}

		vmCloneInfo.snapshot, syncInfo = ctrl.verifySnapshotReady(vmClone, vmCloneInfo.snapshotName, vmCloneInfo.vmClone.Namespace, syncInfo)
		if syncInfo.isFailingOrError() || !syncInfo.snapshotReady {
			return syncInfo
		}

		fallthrough

	case clonev1alpha1.RestoreInProgress:
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

			syncInfo = ctrl.createRestoreFromVm(vmClone, vm, vmCloneInfo.snapshot, syncInfo)
			return syncInfo
		}

		syncInfo = ctrl.verifyRestoreReady(vmClone, vmCloneInfo.vmClone.Namespace, syncInfo)
		if syncInfo.isFailingOrError() || !syncInfo.restoreReady {
			return syncInfo
		}

		fallthrough

	case clonev1alpha1.CreatingTargetVM:

		syncInfo = ctrl.verifyVmReady(vmClone, syncInfo)
		if syncInfo.isFailingOrError() {
			return syncInfo
		}

		fallthrough

	case clonev1alpha1.Succeeded:

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
			assignPhase(clonev1alpha1.Succeeded)

		}
	}
	if isInPhase(vmClone, clonev1alpha1.Succeeded) {
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
		err := ctrl.cloneStatusUpdater.UpdateStatus(vmClone)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ctrl *VMCloneController) createSnapshotFromVm(vmClone *clonev1alpha1.VirtualMachineClone, vm *k6tv1.VirtualMachine, syncInfo syncInfoType) (*snapshotv1.VirtualMachineSnapshot, syncInfoType) {
	snapshot := generateSnapshot(vmClone, vm)
	log.Log.Object(vmClone).Infof("creating snapshot %s for clone %s", snapshot.Name, vmClone.Name)

	createdSnapshot, err := ctrl.client.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, v1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			syncInfo.setError(fmt.Errorf("failed creating snapshot %s for clone %s: %v", snapshot.Name, vmClone.Name, err))
			return snapshot, syncInfo
		}
		syncInfo.snapshotName = snapshot.Name
		return snapshot, syncInfo
	}

	snapshot = createdSnapshot
	ctrl.logAndRecord(vmClone, SnapshotCreated, fmt.Sprintf("created snapshot %s for clone %s", snapshot.Name, vmClone.Name))
	syncInfo.snapshotName = snapshot.Name

	log.Log.Object(vmClone).V(defaultVerbosityLevel).Infof("snapshot %s was just created, reenqueuing to let snapshot time to finish", snapshot.Name)
	return snapshot, syncInfo
}

func (ctrl *VMCloneController) verifySnapshotReady(vmClone *clonev1alpha1.VirtualMachineClone, name, namespace string, syncInfo syncInfoType) (*snapshotv1.VirtualMachineSnapshot, syncInfoType) {
	obj, exists, err := ctrl.snapshotInformer.GetStore().GetByKey(getKey(name, namespace))
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

	ctrl.logAndRecord(vmClone, SnapshotReady, fmt.Sprintf("snapshot %s for clone %s is ready to use", snapshot.Name, vmClone.Name))
	syncInfo.snapshotReady = true

	return snapshot, syncInfo
}

// This method assumes the snapshot exists. If it doesn't - syncInfo is updated accordingly.
func (ctrl *VMCloneController) getSnapshot(snapshotName string, sourceNamespace string, syncInfo syncInfoType) (*snapshotv1.VirtualMachineSnapshot, syncInfoType) {
	obj, exists, err := ctrl.snapshotInformer.GetStore().GetByKey(getKey(snapshotName, sourceNamespace))
	if !exists {
		// At this point the snapshot is already created. If it doesn't exist it means that it's deleted for some
		// reason and the clone should fail
		syncInfo.isCloneFailing = true
		syncInfo.failEvent = SnapshotDeleted
		syncInfo.failReason = fmt.Sprintf("snapshot %s does not exist anymore", snapshotName)
		return nil, syncInfo
	}
	if err != nil {
		syncInfo.setError(fmt.Errorf("error getting snapshot %s from cache: %v", snapshotName, err))
		return nil, syncInfo
	}
	snapshot := obj.(*snapshotv1.VirtualMachineSnapshot)

	return snapshot, syncInfo
}

func (ctrl *VMCloneController) createRestoreFromVm(vmClone *clonev1alpha1.VirtualMachineClone, vm *k6tv1.VirtualMachine, snapshot *snapshotv1.VirtualMachineSnapshot, syncInfo syncInfoType) syncInfoType {
	patches := generatePatches(vm, &vmClone.Spec, snapshot)
	restore := generateRestore(vmClone.Spec.Target, vm.Name, vmClone.Namespace, vmClone.Name, snapshot.Name, vmClone.UID, patches)
	log.Log.Object(vmClone).Infof("creating restore %s for clone %s", restore.Name, vmClone.Name)
	createdRestore, err := ctrl.client.VirtualMachineRestore(restore.Namespace).Create(context.Background(), restore, v1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
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

func (ctrl *VMCloneController) verifyRestoreReady(vmClone *clonev1alpha1.VirtualMachineClone, sourceNamespace string, syncInfo syncInfoType) syncInfoType {
	obj, exists, err := ctrl.restoreInformer.GetStore().GetByKey(getKey(*vmClone.Status.RestoreName, sourceNamespace))
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

func (ctrl *VMCloneController) verifyVmReady(vmClone *clonev1alpha1.VirtualMachineClone, syncInfo syncInfoType) syncInfoType {
	targetVMInfo := vmClone.Spec.Target

	_, exists, err := ctrl.vmInformer.GetStore().GetByKey(getKey(targetVMInfo.Name, vmClone.Namespace))
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

func (ctrl *VMCloneController) verifyPVCBound(vmClone *clonev1alpha1.VirtualMachineClone, syncInfo syncInfoType) syncInfoType {
	obj, exists, err := ctrl.restoreInformer.GetStore().GetByKey(getKey(*vmClone.Status.RestoreName, vmClone.Namespace))
	if !exists {
		syncInfo.setError(fmt.Errorf("restore %s is not created yet for clone %s", *vmClone.Status.RestoreName, vmClone.Name))
		return syncInfo
	} else if err != nil {
		syncInfo.setError(fmt.Errorf("error getting restore %s from cache for clone %s: %v", *vmClone.Status.SnapshotName, vmClone.Name, err))
		return syncInfo
	}

	restore := obj.(*snapshotv1.VirtualMachineRestore)
	for _, volumeRestore := range restore.Status.Restores {
		obj, exists, err = ctrl.pvcInformer.GetStore().GetByKey(getKey(volumeRestore.PersistentVolumeClaimName, vmClone.Namespace))
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

func (ctrl *VMCloneController) cleanupSnapshot(vmClone *clonev1alpha1.VirtualMachineClone, syncInfo syncInfoType) syncInfoType {
	err := ctrl.client.VirtualMachineSnapshot(vmClone.Namespace).Delete(context.Background(), *vmClone.Status.SnapshotName, v1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		syncInfo.setError(fmt.Errorf("cannot clean up snapshot %s for clone %s", *vmClone.Status.SnapshotName, vmClone.Name))
		return syncInfo
	}

	return syncInfo
}

func (ctrl *VMCloneController) cleanupRestore(vmClone *clonev1alpha1.VirtualMachineClone, syncInfo syncInfoType) syncInfoType {
	err := ctrl.client.VirtualMachineRestore(vmClone.Namespace).Delete(context.Background(), *vmClone.Status.RestoreName, v1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		syncInfo.setError(fmt.Errorf("cannot clean up restore %s for clone %s", *vmClone.Status.RestoreName, vmClone.Name))
		return syncInfo
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

func (ctrl *VMCloneController) getVmFromSnapshot(snapshot *snapshotv1.VirtualMachineSnapshot) (*k6tv1.VirtualMachine, error) {
	contentName := virtsnapshot.GetVMSnapshotContentName(snapshot)
	contentKey := getKey(contentName, snapshot.Namespace)

	contentObj, exists, err := ctrl.snapshotContentInformer.GetStore().GetByKey(contentKey)
	if !exists {
		return nil, fmt.Errorf("snapshot content %s in namespace %s does not exist", contentName, snapshot.Namespace)
	} else if err != nil {
		return nil, err
	}

	content := contentObj.(*snapshotv1.VirtualMachineSnapshotContent)
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
