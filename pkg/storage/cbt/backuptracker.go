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
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

func trackerNeedsCheckpointRedefinition(tracker *backupv1.VirtualMachineBackupTracker) bool {
	return tracker != nil &&
		tracker.Status != nil &&
		tracker.Status.CheckpointRedefinitionRequired != nil &&
		*tracker.Status.CheckpointRedefinitionRequired &&
		tracker.Status.LatestCheckpoint != nil &&
		tracker.Status.LatestCheckpoint.Name != ""
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
		log.Log.Reason(err).Infof("reenqueuing VirtualMachineBackupTracker %v for redefinition", key)
		ctrl.trackerQueue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineBackupTracker redefinition %v", key)
		ctrl.trackerQueue.Forget(key)
	}
	return true
}

func (ctrl *VMBackupController) executeTracker(key string) error {
	logger := log.Log.With("VirtualMachineBackupTracker", key)
	logger.V(3).Infof("Processing tracker checkpoint redefinition %s", key)

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

	if !trackerNeedsCheckpointRedefinition(tracker) {
		return nil
	}

	return ctrl.handleCheckpointRedefinition(tracker)
}

func (ctrl *VMBackupController) handleCheckpointRedefinition(tracker *backupv1.VirtualMachineBackupTracker) error {
	logger := log.Log.With("VirtualMachineBackupTracker", tracker.Name)
	logger.Infof("Handling checkpoint redefinition for tracker %s/%s", tracker.Namespace, tracker.Name)

	vmiName := tracker.Spec.Source.Name
	vmi, exists, err := ctrl.getVMI(tracker.Namespace, vmiName)
	if err != nil {
		return fmt.Errorf("failed to get VMI %s/%s: %w", tracker.Namespace, vmiName, err)
	}
	if !exists || vmi == nil {
		return fmt.Errorf("VMI %s/%s not found", tracker.Namespace, vmiName)
	}

	checkpoint := tracker.Status.LatestCheckpoint
	logger.Infof("Calling RedefineCheckpoint for VMI %s with checkpoint %s", vmiName, checkpoint.Name)

	err = ctrl.client.VirtualMachineInstance(tracker.Namespace).RedefineCheckpoint(context.Background(), vmiName, checkpoint)
	if err != nil {
		return ctrl.handleRedefinitionError(tracker, err)
	}

	logger.Infof("Checkpoint redefinition successful for tracker %s/%s", tracker.Namespace, tracker.Name)
	return ctrl.clearRedefinitionFlag(tracker)
}

func (ctrl *VMBackupController) handleRedefinitionError(tracker *backupv1.VirtualMachineBackupTracker, err error) error {
	logger := log.Log.With("VirtualMachineBackupTracker", tracker.Name)

	if isCheckpointInvalidError(err) {
		logger.Warningf("Checkpoint invalid, clearing latestcheckpoint: %v", err)
		ctrl.recorder.Eventf(tracker, corev1.EventTypeWarning, "CheckpointRedefinitionFailed",
			"Failed to redefine checkpoint %s: %v. Checkpoint cleared, next backup will be full.",
			tracker.Status.LatestCheckpoint.Name, err)
		return ctrl.clearCheckpointAndFlag(tracker)
	}

	logger.Errorf("Checkpoint redefinition failed: %v", err)
	return err
}

func isCheckpointInvalidError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "422") && strings.Contains(errStr, "Unprocessable Entity")
}

func (ctrl *VMBackupController) clearRedefinitionFlag(tracker *backupv1.VirtualMachineBackupTracker) error {
	return ctrl.patchTrackerStatus(tracker, patch.WithRemove("/status/checkpointRedefinitionRequired"))
}

func (ctrl *VMBackupController) clearCheckpointAndFlag(tracker *backupv1.VirtualMachineBackupTracker) error {
	return ctrl.patchTrackerStatus(tracker,
		patch.WithRemove("/status/checkpointRedefinitionRequired"),
		patch.WithRemove("/status/latestCheckpoint"),
	)
}

func (ctrl *VMBackupController) patchTrackerStatus(tracker *backupv1.VirtualMachineBackupTracker, opts ...patch.PatchOption) error {
	patchBytes, err := patch.New(opts...).GeneratePayload()
	if err != nil {
		return err
	}
	_, err = ctrl.client.VirtualMachineBackupTracker(tracker.Namespace).Patch(
		context.Background(),
		tracker.Name,
		types.JSONPatchType,
		patchBytes,
		metav1.PatchOptions{},
		"status",
	)
	return err
}
