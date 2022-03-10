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

	"k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"
	k6tv1 "kubevirt.io/api/core/v1"
	snapshotv1alpha1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/log"
)

type cloneSourceType string

const (
	sourceTypeVM cloneSourceType = "VirtualMachine"
)

type cloneTargetType string

const (
	targetTypeVM cloneTargetType = "VirtualMachine"
	defaultType  cloneTargetType = targetTypeVM
)

type syncInfo struct {
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

	reenqueueInfo reenqueueInfo
}
type reenqueueInfo struct {
	reenqueueReason string
}

func (ctrl *VMCloneController) execute(key string) (error, reenqueueInfo) {
	var syncInfo syncInfo
	doNotReenqueue := reenqueueInfo{}
	logger := log.Log

	obj, cloneExists, err := ctrl.vmCloneInformer.GetStore().GetByKey(key)
	if err != nil {
		return err, doNotReenqueue
	}

	var vmClone *clonev1alpha1.VirtualMachineClone
	if cloneExists {
		vmClone = obj.(*clonev1alpha1.VirtualMachineClone)
		logger = logger.Object(vmClone)
	} else {
		return nil, doNotReenqueue
	}

	sourceInfo := vmClone.Spec.Source
	switch cloneSourceType(sourceInfo.Kind) {
	case sourceTypeVM:
		vmKey := getKey(sourceInfo.Name, vmClone.Namespace)
		obj, vmExists, err := ctrl.vmInformer.GetStore().GetByKey(vmKey)
		if err != nil {
			return fmt.Errorf("error getting VM %s in namespace %s from cache: %v", sourceInfo.Name, vmClone.Namespace, err), doNotReenqueue
		}
		if !vmExists {
			return fmt.Errorf("VM %s in namespace %s does not exist", sourceInfo.Name, vmClone.Namespace), doNotReenqueue
		}
		sourceVM := obj.(*k6tv1.VirtualMachine)

		syncInfo = ctrl.syncSourceVM(key, sourceVM, vmClone)
		if syncInfo.err != nil {
			return syncInfo.err, syncInfo.reenqueueInfo
		}
	default:
		return fmt.Errorf("clone %s is defined with an unknown source type %s", vmClone.Name, sourceInfo.Kind), doNotReenqueue
	}

	err = ctrl.updateStatus(vmClone)
	if err != nil {
		return err, syncInfo.reenqueueInfo
	}

	return nil, syncInfo.reenqueueInfo
}

func (ctrl *VMCloneController) syncSourceVM(key string, source *k6tv1.VirtualMachine, vmClone *clonev1alpha1.VirtualMachineClone) syncInfo {
	var targetType cloneTargetType
	if vmClone.Spec.Target != nil {
		targetType = cloneTargetType(vmClone.Spec.Target.Kind)
	} else {
		targetType = defaultType
	}

	switch targetType {
	case targetTypeVM:
		return ctrl.syncSourceVMTargetVM(key, source, vmClone)

	default:
		return syncInfo{err: fmt.Errorf("target type is unknown: %s", targetType)}
	}
}

func (ctrl *VMCloneController) syncSourceVMTargetVM(key string, source *k6tv1.VirtualMachine, vmClone *clonev1alpha1.VirtualMachineClone) syncInfo {
	syncInfo := syncInfo{}
	logger := log.Log.Object(vmClone)

	var snapshot *snapshotv1alpha1.VirtualMachineSnapshot
	targetVMInfo := vmClone.Spec.Target

	switch vmClone.Status.Phase {
	case clonev1alpha1.PhaseUnset, clonev1alpha1.SnapshotInProgress:

		// Create snapshot
		if vmClone.Status.SnapshotName == nil {
			snapshot = generateSnapshot(vmClone, source)
			logger.Infof("creating snapshot %s for clone %s", snapshot.Name, vmClone.Name)

			snapshot, syncInfo.err = ctrl.client.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, v1.CreateOptions{})
			if syncInfo.err != nil {
				return addErrorToSyncInfo(syncInfo, fmt.Errorf("failed creating snapshot %s for clone %s: %v", snapshot.Name, vmClone.Name, syncInfo.err))
			}

			ctrl.logAndRecord(vmClone, SnapshotCreated, fmt.Sprintf("created snapshot %s for clone %s", snapshot.Name, vmClone.Name))
			syncInfo.snapshotName = snapshot.Name

			return addReenqueueToSyncInfo(syncInfo, fmt.Sprintf("snapshot %s was just created, reenqueuing to let snapshot time to finish", snapshot.Name))
		}

		// Make sure snapshot is ready for use
		obj, exists, err := ctrl.snapshotInformer.GetStore().GetByKey(getKey(*vmClone.Status.SnapshotName, source.Namespace))
		if !exists {
			return addErrorToSyncInfo(syncInfo, fmt.Errorf("snapshot %s is not created yet for clone %s", *vmClone.Status.SnapshotName, vmClone.Name))
		} else if err != nil {
			return addErrorToSyncInfo(syncInfo, fmt.Errorf("error getting snapshot %s from cache for clone %s: %v", *vmClone.Status.SnapshotName, vmClone.Name, err))
		}
		snapshot = obj.(*snapshotv1alpha1.VirtualMachineSnapshot)
		logger.Infof("found snapshot %s for clone %s", snapshot.Name, vmClone.Name)

		if snapshot.Status.ReadyToUse == nil || *snapshot.Status.ReadyToUse == false {
			return addReenqueueToSyncInfo(syncInfo, fmt.Sprintf("snapshot %s for clone %s is not ready to use yet", snapshot.Name, vmClone.Name))
		}

		ctrl.logAndRecord(vmClone, SnapshotReady, fmt.Sprintf("snapshot %s for clone %s is ready to use", snapshot.Name, vmClone.Name))
		syncInfo.snapshotReady = true

		fallthrough

	case clonev1alpha1.RestoreInProgress:

		if snapshot == nil {
			obj, exists, err := ctrl.snapshotInformer.GetStore().GetByKey(getKey(*vmClone.Status.SnapshotName, source.Namespace))
			if !exists {
				// At this point the snapshot is already created. If it doesn't exist it means that it's deleted for some
				// reason and the clone should fail
				syncInfo.isCloneFailing = true
				syncInfo.failEvent = SnapshotDeleted
				syncInfo.failReason = fmt.Sprintf("snapshot %s does not exist anymore", *vmClone.Status.SnapshotName)
				return syncInfo
			}
			if err != nil {
				return addErrorToSyncInfo(syncInfo, fmt.Errorf("error getting snapshot %s from cache for clone %s: %v", *vmClone.Status.SnapshotName, vmClone.Name, err))
			}
			snapshot = obj.(*snapshotv1alpha1.VirtualMachineSnapshot)
		}

		// Create restore
		if vmClone.Status.RestoreName == nil {
			restore := generateRestore(targetVMInfo, source.Name, vmClone.Namespace, vmClone.Name, snapshot.Name, vmClone.UID)
			logger.Infof("creating restore %s for clone %s", restore.Name, vmClone.Name)

			restore, syncInfo.err = ctrl.client.VirtualMachineRestore(restore.Namespace).Create(context.Background(), restore, v1.CreateOptions{})
			if syncInfo.err != nil {
				return addErrorToSyncInfo(syncInfo, fmt.Errorf("failed creating restore %s for clone %s: %v", restore.Name, vmClone.Name, syncInfo.err))
			}

			ctrl.logAndRecord(vmClone, RestoreCreated, fmt.Sprintf("created restore %s for clone %s", restore.Name, vmClone.Name))
			syncInfo.restoreName = restore.Name

			return addReenqueueToSyncInfo(syncInfo, fmt.Sprintf("restore %s was just created, reenqueuing to let snapshot time to finish", restore.Name))
		}

		// Make sure restore is ready for use
		obj, exists, err := ctrl.restoreInformer.GetStore().GetByKey(getKey(*vmClone.Status.RestoreName, source.Namespace))
		if !exists {
			return addErrorToSyncInfo(syncInfo, fmt.Errorf("restore %s is not created yet for clone %s", *vmClone.Status.SnapshotName, vmClone.Name))
		} else if err != nil {
			return addErrorToSyncInfo(syncInfo, fmt.Errorf("error getting snapshot %s from cache for clone %s: %v", *vmClone.Status.SnapshotName, vmClone.Name, err))
		}

		restore := obj.(*snapshotv1alpha1.VirtualMachineRestore)
		logger.Infof("found target restore %s for clone %s", restore.Name, vmClone.Name)

		if restore.Status == nil || restore.Status.Complete == nil || *restore.Status.Complete == false {
			return addReenqueueToSyncInfo(syncInfo, fmt.Sprintf("restore %s for clone %s is not ready to use yet", restore.Name, vmClone.Name))
		}

		ctrl.logAndRecord(vmClone, RestoreReady, fmt.Sprintf("restore %s for clone %s is ready to use", restore.Name, vmClone.Name))
		syncInfo.restoreReady = true
		syncInfo.targetVMName = restore.Spec.Target.Name

		fallthrough

	case clonev1alpha1.CreatingTargetVM:

		// Make sure target VM is created and ready
		_, exists, err := ctrl.vmInformer.GetStore().GetByKey(getKey(targetVMInfo.Name, vmClone.Namespace))
		if !exists {
			return addErrorToSyncInfo(syncInfo, fmt.Errorf("target VM %s is not created yet for clone %s", targetVMInfo.Name, vmClone.Name))
		} else if err != nil {
			return addErrorToSyncInfo(syncInfo, fmt.Errorf("error getting VM %s from cache for clone %s: %v", *vmClone.Status.SnapshotName, targetVMInfo.Name, err))
		}

		ctrl.logAndRecord(vmClone, TargetVMCreated, fmt.Sprintf("created target VM %s for clone %s", targetVMInfo.Name, vmClone.Name))
		syncInfo.targetVMCreated = true

		// Clean up snapshot & restore
		err = ctrl.client.VirtualMachineSnapshot(vmClone.Namespace).Delete(context.Background(), *vmClone.Status.SnapshotName, v1.DeleteOptions{})
		if !errors.IsNotFound(err) && err != nil {
			return addErrorToSyncInfo(syncInfo, fmt.Errorf("cannot clean up snapshot %s for clone %s", *vmClone.Status.SnapshotName, vmClone.Name))
		}

		err = ctrl.client.VirtualMachineRestore(vmClone.Namespace).Delete(context.Background(), *vmClone.Status.RestoreName, v1.DeleteOptions{})
		if !errors.IsNotFound(err) && err != nil {
			return addErrorToSyncInfo(syncInfo, fmt.Errorf("cannot clean up restore %s for clone %s", *vmClone.Status.RestoreName, vmClone.Name))
		}

	default:
		log.Log.Object(vmClone).Infof("clone %s is in phase %s - nothing to do", vmClone.Name, string(vmClone.Status.Phase))
	}

	return syncInfo
}

func (ctrl *VMCloneController) updateStatus(vmClone *clonev1alpha1.VirtualMachineClone) error {
	return nil
}

func (ctrl *VMCloneController) logAndRecord(vmClone *clonev1alpha1.VirtualMachineClone, event Event, msg string) {
	ctrl.recorder.Eventf(vmClone, corev1.EventTypeNormal, string(event), msg)
	log.Log.Object(vmClone).Infof(msg)
}

func addErrorToSyncInfo(info syncInfo, err error) syncInfo {
	info.err = err
	return info
}

func addReenqueueToSyncInfo(info syncInfo, reason string) syncInfo {
	info.reenqueueInfo = reenqueueInfo{reenqueueReason: reason}
	return info
}

func (r reenqueueInfo) toReenqueue() bool {
	return r.reenqueueReason != ""
}
