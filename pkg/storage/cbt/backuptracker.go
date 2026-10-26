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

package cbt

import (
	"errors"
	"fmt"

	backupv1 "kubevirt.io/api/backup/v1beta1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
)

func isTrackerDeleting(tracker *backupv1.VirtualMachineBackupTracker) bool {
	return tracker != nil && tracker.DeletionTimestamp != nil
}

func (ctrl *VMBackupController) runTrackerWorker() {
	for ctrl.ExecuteTracker() {
	}
}

func (ctrl *VMBackupController) ExecuteTracker() bool {
	key, quit := ctrl.trackerQueue.Get()
	if quit {
		return false
	}
	defer ctrl.trackerQueue.Done(key)

	err := ctrl.executeTracker(key)
	if err != nil {
		if errors.Is(err, errActiveBackups) {
			log.Log.V(4).Infof("reenqueuing VirtualMachineBackupTracker %v: %v", key, err)
		} else {
			log.Log.Reason(err).Infof("reenqueuing VirtualMachineBackupTracker %v", key)
		}
		ctrl.trackerQueue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineBackupTracker %v", key)
		ctrl.trackerQueue.Forget(key)
	}
	return true
}

func (ctrl *VMBackupController) executeTracker(key string) error {
	logger := log.Log.With("VirtualMachineBackupTracker", key)
	logger.V(3).Infof("Processing tracker %s", key)

	storeObj, exists, err := ctrl.backupTrackerInformer.GetStore().GetByKey(key)
	if err != nil {
		logger.Errorf("Error getting tracker from store: %v", err)
		return err
	}
	if !exists {
		logger.V(3).Infof("Tracker %s no longer exists in store", key)
		return nil
	}

	tracker, ok := storeObj.(*backupv1.VirtualMachineBackupTracker)
	if !ok {
		logger.Errorf("Unexpected resource type: %T", storeObj)
		return fmt.Errorf("unexpected resource %+v", storeObj)
	}

	if isTrackerDeleting(tracker) && controller.HasFinalizer(tracker, backupv1.VirtualMachineBackupTrackerFinalizer) {
		return ctrl.handleTrackerDeletion(tracker)
	}

	return nil
}

func (ctrl *VMBackupController) handleTrackerDeletion(tracker *backupv1.VirtualMachineBackupTracker) error {
	logger := log.Log.With("VirtualMachineBackupTracker", tracker.Name)
	logger.Infof("Handling deletion for tracker %s/%s", tracker.Namespace, tracker.Name)

	hasActiveBackups, err := ctrl.trackerHasActiveBackups(tracker)
	if err != nil {
		return fmt.Errorf("failed to check active backups for tracker %s/%s: %w", tracker.Namespace, tracker.Name, err)
	}
	if hasActiveBackups {
		return fmt.Errorf("tracker %s/%s has active backups, cannot remove finalizer: %w", tracker.Namespace, tracker.Name, errActiveBackups)
	}

	return ctrl.removeTrackerFinalizer(tracker)
}

func (ctrl *VMBackupController) trackerHasActiveBackups(tracker *backupv1.VirtualMachineBackupTracker) (bool, error) {
	trackerKey := fmt.Sprintf("%s/%s", tracker.Namespace, tracker.Name)
	backupKeys, err := ctrl.backupInformer.GetIndexer().IndexKeys("backupTracker", trackerKey)
	if err != nil {
		return false, err
	}

	for _, key := range backupKeys {
		obj, exists, err := ctrl.backupInformer.GetStore().GetByKey(key)
		if err != nil {
			return false, fmt.Errorf("failed to get backup %s from store: %w", key, err)
		}
		if !exists {
			continue
		}
		backup, ok := obj.(*backupv1.VirtualMachineBackup)
		if !ok {
			continue
		}
		if !IsBackupTerminal(backup) {
			return true, nil
		}
	}

	return false, nil
}
