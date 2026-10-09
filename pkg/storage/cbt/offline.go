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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	offlineVMNotFoundMsg       = "vm %s does not exist, cannot do offline backup"
	offlineNoVolumesMsg        = "vm %s has no volumes eligible for backup"
	offlineStalePodsMsg        = "vm %s still has non-terminal virt-launcher pods, waiting before starting offline backup"
	offlineConcurrentMsg       = "another offline backup %s is already in progress for vm %s"
	offlineVMDeletedMsg        = "vm %s was deleted during the offline backup"
	offlinePushFailedMsg       = "offline push exporter pod failed"
	offlineEmptyIncrementalMsg = "offline incremental backup has no changed blocks since the base checkpoint; start the VM before taking another incremental"

	exportPodFailedReason        = "PodFailed"
	exportEmptyIncrementalReason = "EmptyIncremental"
)

func isOfflineBackup(backup *backupv1.VirtualMachineBackup) bool {
	return backup.Status != nil && backup.Status.Offline != nil && *backup.Status.Offline
}

// Once status.offline is set it stays offline thereafter, even if a VMI later appears.
func (ctrl *VMBackupController) shouldReconcileOffline(backup *backupv1.VirtualMachineBackup, vmiExists bool, backupStatus *v1.VirtualMachineInstanceBackupStatus) bool {
	if isOfflineBackup(backup) {
		return true
	}
	if !ctrl.clusterConfig.OfflineIncrementalBackupEnabled() {
		return false
	}
	return !vmiExists && backupStatus == nil && backup.Status.Type == ""
}

func (ctrl *VMBackupController) reconcileOffline(backup *backupv1.VirtualMachineBackup, backupTracker *backupv1.VirtualMachineBackupTracker, sourceName string, backupDeleting bool) error {
	if backupDeleting {
		return ctrl.reconcileOfflineDeleting(backup, backupTracker)
	}

	if !isOfflineBackup(backup) {
		return ctrl.startOfflineBackup(backup, backupTracker, sourceName)
	}

	if exists, err := ctrl.sourceVMExists(backup, sourceName); err != nil {
		return err
	} else if !exists {
		return ctrl.failVMDeleted(backup, sourceName)
	}

	if isPushMode(backup) {
		return ctrl.reconcileOfflinePush(backup, backupTracker)
	}
	return ctrl.reconcileOfflinePull(backup, backupTracker)
}

func (ctrl *VMBackupController) failVMDeleted(backup *backupv1.VirtualMachineBackup, sourceName string) error {
	if err := ctrl.cleanupBackupExport(backup); err != nil {
		return err
	}
	ctrl.setFailed(backup, backupv1.ReasonVMDeleted, fmt.Sprintf(offlineVMDeletedMsg, sourceName))
	return nil
}

func (ctrl *VMBackupController) startOfflineBackup(backup *backupv1.VirtualMachineBackup, backupTracker *backupv1.VirtualMachineBackupTracker, sourceName string) error {
	reason, err := ctrl.offlinePreflight(backup, sourceName)
	if err != nil {
		return err
	}
	if reason != "" {
		setInitializing(backup, reason)
		return nil
	}

	vm, err := ctrl.getVM(backup.Namespace, sourceName)
	if err != nil {
		return err
	}

	volumes := offlineEligibleVolumes(vm)
	if len(volumes) == 0 {
		ctrl.setFailed(backup, backupv1.ReasonFailed, fmt.Sprintf(offlineNoVolumesMsg, sourceName))
		return nil
	}

	if backup.Spec.Mode == nil {
		backup.Spec.Mode = pointer.P(backupv1.PullMode)
	}
	if isPushMode(backup) {
		pvcReason, err := ctrl.verifyBackupTargetPVC(backup.Spec.PvcName, backup.Namespace)
		if err != nil {
			return err
		}
		if pvcReason != "" {
			setInitializing(backup, pvcReason)
			return nil
		}
	}

	if err := ctrl.addBackupFinalizer(backup); err != nil {
		return err
	}

	backup.Status.Type = backupv1.Full
	if isIncrementalBackup(backup, backupTracker) {
		backup.Status.Type = backupv1.Incremental
	}
	backup.Status.CheckpointName = pointer.P(offlineCheckpointName(backup))
	backup.Status.IncludedVolumes = volumes
	backup.Status.Offline = pointer.P(true)
	setProgressing(backup)

	log.Log.Object(backup).Infof("Started offline %s backup for VM %s", backup.Status.Type, sourceName)
	return nil
}

func (ctrl *VMBackupController) offlinePreflight(backup *backupv1.VirtualMachineBackup, sourceName string) (string, error) {
	exists, err := ctrl.sourceVMExists(backup, sourceName)
	if err != nil {
		return "", err
	}
	if !exists {
		return fmt.Sprintf(offlineVMNotFoundMsg, sourceName), nil
	}

	stale, err := ctrl.hasNonTerminalLauncherPods(backup.Namespace, sourceName)
	if err != nil {
		return "", err
	}
	if stale {
		return fmt.Sprintf(offlineStalePodsMsg, sourceName), nil
	}

	if other := ctrl.otherOfflineBackupInProgress(backup, sourceName); other != "" {
		return fmt.Sprintf(offlineConcurrentMsg, other, sourceName), nil
	}

	return "", nil
}

func (ctrl *VMBackupController) hasNonTerminalLauncherPods(namespace, vmName string) (bool, error) {
	objs, err := ctrl.podIndexer.ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return false, err
	}
	for _, obj := range objs {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			continue
		}
		if pod.Labels[v1.AppLabel] != "virt-launcher" {
			continue
		}
		if pod.Annotations[v1.DomainAnnotation] != vmName {
			continue
		}
		if !controller.PodIsDown(pod) {
			return true, nil
		}
	}
	return false, nil
}

func (ctrl *VMBackupController) otherOfflineBackupInProgress(backup *backupv1.VirtualMachineBackup, sourceName string) string {
	for _, obj := range ctrl.backupInformer.GetStore().List() {
		other, ok := obj.(*backupv1.VirtualMachineBackup)
		if !ok || other.Namespace != backup.Namespace || other.Name == backup.Name {
			continue
		}
		if !isOfflineBackup(other) || IsBackupTerminal(other) || isBackupDeleting(other) {
			continue
		}
		if getSourceNameFromSpec(other) == sourceName {
			return other.Name
		}
	}
	return ""
}

func (ctrl *VMBackupController) reconcileOfflinePull(backup *backupv1.VirtualMachineBackup, backupTracker *backupv1.VirtualMachineBackupTracker) error {
	if isPullBackupTTLExpired(backup) {
		if err := ctrl.cleanupBackupExport(backup); err != nil {
			return err
		}
		ctrl.setFailed(backup, backupv1.ReasonFailed, backupTTLExpiredMsg)
		return nil
	}

	vmExport, err := ctrl.getOrCreateBackupExport(backup)
	if err != nil {
		return err
	}
	if vmExport == nil {
		return nil
	}

	if done, err := ctrl.failIfEmptyIncremental(backup, vmExport); done || err != nil {
		return err
	}

	return ctrl.populateExportLinks(backup, vmExport)
}

func (ctrl *VMBackupController) reconcileOfflinePush(backup *backupv1.VirtualMachineBackup, backupTracker *backupv1.VirtualMachineBackupTracker) error {
	vmExport, err := ctrl.getOrCreateBackupExport(backup)
	if err != nil {
		return err
	}
	if vmExport == nil || vmExport.Status == nil {
		return nil
	}

	if done, err := ctrl.failIfEmptyIncremental(backup, vmExport); done || err != nil {
		return err
	}

	// The push exporter pod isn't visible in this controller's pod informer, so
	// key off the export status instead.
	if isBackupExportPushFailed(vmExport) {
		ctrl.setFailed(backup, backupv1.ReasonFailed, offlinePushFailedMsg)
		return ctrl.cleanupBackupExport(backup)
	}
	if vmExport.Status.Phase == exportv1.Terminated {
		return ctrl.completeOfflineBackup(backup, backupTracker)
	}
	return nil
}

func isBackupExportPushFailed(vmExport *exportv1.VirtualMachineExport) bool {
	for _, cond := range vmExport.Status.Conditions {
		if cond.Type == exportv1.ConditionReady {
			return cond.Reason == exportPodFailedReason
		}
	}
	return false
}

func isBackupExportEmptyIncremental(vmExport *exportv1.VirtualMachineExport) bool {
	for _, cond := range vmExport.Status.Conditions {
		if cond.Type == exportv1.ConditionReady {
			return cond.Reason == exportEmptyIncrementalReason
		}
	}
	return false
}

func (ctrl *VMBackupController) failIfEmptyIncremental(backup *backupv1.VirtualMachineBackup, vmExport *exportv1.VirtualMachineExport) (bool, error) {
	if vmExport.Status == nil || !isBackupExportEmptyIncremental(vmExport) {
		return false, nil
	}
	if err := ctrl.cleanupBackupExport(backup); err != nil {
		return false, err
	}
	ctrl.setFailed(backup, backupv1.ReasonEmptyIncremental, offlineEmptyIncrementalMsg)
	return true, nil
}

func (ctrl *VMBackupController) reconcileOfflineDeleting(backup *backupv1.VirtualMachineBackup, backupTracker *backupv1.VirtualMachineBackupTracker) error {
	// Only advance if the export became ready and served; a backup deleted while
	// still preparing never wrote its checkpoint bitmap, so advancing would
	// corrupt the chain.
	if isPullMode(backup) && !IsBackupTerminal(backup) && isBackupExportReady(backup) {
		if err := ctrl.advanceOfflineTracker(backup, backupTracker); err != nil {
			return err
		}
	}
	if err := ctrl.cleanupBackupExport(backup); err != nil {
		return err
	}
	return ctrl.removeBackupFinalizer(backup)
}

func (ctrl *VMBackupController) completeOfflineBackup(backup *backupv1.VirtualMachineBackup, backupTracker *backupv1.VirtualMachineBackupTracker) error {
	if err := ctrl.advanceOfflineTracker(backup, backupTracker); err != nil {
		return err
	}
	if err := ctrl.cleanupBackupExport(backup); err != nil {
		return err
	}
	ctrl.setQuiescedCondition(backup, string(backupv1.QuiesceSkipped))
	setComplete(backup)
	ctrl.recorder.Eventf(backup, corev1.EventTypeNormal, backupCompletedEvent, backupCompleted)
	return nil
}

func offlineEligibleVolumes(vm *v1.VirtualMachine) []backupv1.BackupVolumeInfo {
	if vm.Spec.Template == nil {
		return nil
	}
	var volumes []backupv1.BackupVolumeInfo
	for i := range vm.Spec.Template.Spec.Volumes {
		volume := vm.Spec.Template.Spec.Volumes[i]
		if IsCBTEligibleVolume(&volume) {
			volumes = append(volumes, backupv1.BackupVolumeInfo{VolumeName: volume.Name})
		}
	}
	return volumes
}

func offlineCheckpointName(backup *backupv1.VirtualMachineBackup) string {
	return fmt.Sprintf("backup-%s", backup.Name)
}

func getSourceNameFromSpec(backup *backupv1.VirtualMachineBackup) string {
	return backup.Spec.Source.Name
}

func (ctrl *VMBackupController) getVM(namespace, name string) (*v1.VirtualMachine, error) {
	objKey := types.NamespacedName{Namespace: namespace, Name: name}.String()
	obj, exists, err := ctrl.vmStore.GetByKey(objKey)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("VM %s/%s not found", namespace, name)
	}
	return obj.(*v1.VirtualMachine), nil
}

func (ctrl *VMBackupController) advanceOfflineTracker(backup *backupv1.VirtualMachineBackup, backupTracker *backupv1.VirtualMachineBackupTracker) error {
	if backupTracker == nil || backup.Status.CheckpointName == nil {
		return nil
	}
	checkpoint := backupv1.BackupCheckpoint{
		Name:         *backup.Status.CheckpointName,
		CreationTime: &backup.CreationTimestamp,
		Volumes:      backup.Status.IncludedVolumes,
	}
	return ctrl.patchTrackerCheckpoint(backup.Namespace, backupTracker, checkpoint)
}
