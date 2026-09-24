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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

const (
	offlineVMNotFoundMsg        = "vm %s does not exist, cannot do offline backup"
	offlineNoVolumesMsg         = "vm %s has no volumes eligible for backup"
	offlineStalePodsMsg         = "vm %s still has non-terminal virt-launcher pods, waiting before starting offline backup"
	offlineConcurrentMsg        = "another offline backup %s is already in progress for vm %s"
	offlineVMDeletedMsg         = "vm %s was deleted during the offline backup"
	offlineLauncherRaceMsg      = "vm %s started while the offline backup was in progress"
	offlineExporterPodFailedMsg = "offline backup exporter pod failed"
	offlineEmptyIncrementalMsg  = "offline incremental backup has no changed blocks since the base checkpoint; start the VM before taking another incremental"

	exportPodFailedReason        = "PodFailed"
	exportEmptyIncrementalReason = "EmptyIncremental"

	virtualMachineExportKind = "VirtualMachineExport"

	offlineCheckpointTimeFormat = "2006-01-02_15-04-05"
)

// Once status.offline is set it stays offline thereafter, even if a VMI later appears.
func (ctrl *VMBackupController) shouldReconcileOffline(backup *backupv1.VirtualMachineBackup, vmiExists bool, backupStatus *v1.VirtualMachineInstanceBackupStatus) bool {
	if backup.IsOffline() {
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

	if !backup.IsOffline() {
		return ctrl.startOfflineBackup(backup, backupTracker, sourceName)
	}

	if exists, err := ctrl.sourceVMExists(backup, sourceName); err != nil {
		return err
	} else if !exists {
		return ctrl.failVMDeleted(backup, sourceName)
	}

	// offlinePreflight only checked this once. A VMI can still start in the gap
	// before status.Offline reaches the VMI controller's informer.
	if stale, err := ctrl.hasNonTerminalLauncherPods(backup.Namespace, sourceName); err != nil {
		return err
	} else if stale {
		return ctrl.failLauncherPodRace(backup, sourceName)
	}

	if backup.IsPushMode() {
		return ctrl.reconcileOfflinePush(backup, backupTracker)
	}
	return ctrl.reconcileOfflinePull(backup, backupTracker)
}

func (ctrl *VMBackupController) cleanupBackupExportAndWait(backup *backupv1.VirtualMachineBackup) error {
	if err := ctrl.cleanupBackupExport(backup); err != nil {
		return err
	}
	return ctrl.waitForExporterPodGone(backup)
}

func (ctrl *VMBackupController) failVMDeleted(backup *backupv1.VirtualMachineBackup, sourceName string) error {
	if err := ctrl.cleanupBackupExportAndWait(backup); err != nil {
		return err
	}
	ctrl.setFailed(backup, backupv1.ReasonVMDeleted, fmt.Sprintf(offlineVMDeletedMsg, sourceName))
	return nil
}

func (ctrl *VMBackupController) failLauncherPodRace(backup *backupv1.VirtualMachineBackup, sourceName string) error {
	if err := ctrl.cleanupBackupExportAndWait(backup); err != nil {
		return err
	}
	ctrl.setFailed(backup, backupv1.ReasonSourceUnhealthy, fmt.Sprintf(offlineLauncherRaceMsg, sourceName))
	return nil
}

func (ctrl *VMBackupController) startOfflineBackup(backup *backupv1.VirtualMachineBackup, backupTracker *backupv1.VirtualMachineBackupTracker, sourceName string) error {
	reason, err := ctrl.offlinePreflight(backup, sourceName)
	if err != nil {
		return err
	}
	if reason != "" {
		setInitializing(backup, reason)
		return fmt.Errorf("%s: %w", reason, errOfflineWaiting)
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
	if vm.Status.ChangedBlockTracking == nil || vm.Status.ChangedBlockTracking.State != v1.ChangedBlockTrackingEnabled {
		ctrl.setFailed(backup, backupv1.ReasonFailed, fmt.Sprintf(vmNoChangedBlockTrackingMsg, sourceName))
		return nil
	}

	if backup.IsPushMode() {
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
	if backupTracker != nil {
		if err := ctrl.addTrackerFinalizer(backupTracker); err != nil {
			return err
		}
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

	other, err := ctrl.otherOfflineBackupInProgress(backup, sourceName)
	if err != nil {
		return "", err
	}
	if other != "" {
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

// cleanupBackupExport only requests deletion. The pod itself is GC'd asynchronously,
// so callers need this to confirm it's actually gone before releasing the VM-start hold.
func (ctrl *VMBackupController) hasNonTerminalExporterPod(namespace string, backup *backupv1.VirtualMachineBackup) (bool, error) {
	objs, err := ctrl.podIndexer.ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return false, err
	}
	for _, obj := range objs {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			continue
		}
		if pod.Labels[v1.AppLabel] != exportv1.App {
			continue
		}
		ownerRef := metav1.GetControllerOf(pod)
		if ownerRef == nil || ownerRef.Kind != virtualMachineExportKind || ownerRef.Name != backup.Name {
			continue
		}
		if !controller.PodIsDown(pod) {
			return true, nil
		}
	}
	return false, nil
}

func (ctrl *VMBackupController) waitForExporterPodGone(backup *backupv1.VirtualMachineBackup) error {
	exists, err := ctrl.hasNonTerminalExporterPod(backup.Namespace, backup)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("waiting for exporter pod of backup %s to terminate: %w", backup.Name, errCleanupPending)
	}
	return nil
}

func (ctrl *VMBackupController) otherOfflineBackupInProgress(backup *backupv1.VirtualMachineBackup, sourceName string) (string, error) {
	for _, obj := range ctrl.backupInformer.GetStore().List() {
		other, ok := obj.(*backupv1.VirtualMachineBackup)
		if !ok || other.Namespace != backup.Namespace || other.Name == backup.Name {
			continue
		}
		if !other.IsOffline() || IsBackupTerminal(other) || isBackupDeleting(other) {
			continue
		}
		// other.Spec.Source.Name is the tracker's name here, not the VM name.
		otherTracker, err := ctrl.getBackupTracker(other)
		if err != nil {
			return "", err
		}
		if getSourceName(other, otherTracker) == sourceName {
			return other.Name, nil
		}
	}
	return "", nil
}

func (ctrl *VMBackupController) reconcileOfflinePull(backup *backupv1.VirtualMachineBackup, backupTracker *backupv1.VirtualMachineBackupTracker) error {
	if isPullBackupTTLExpired(backup) {
		if err := ctrl.cleanupBackupExportAndWait(backup); err != nil {
			return err
		}
		ctrl.setFailed(backup, backupv1.ReasonFailed, backupTTLExpiredMsg)
		return nil
	}

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

	if done, err := ctrl.failIfExporterPodFailed(backup, vmExport); done || err != nil {
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

	if done, err := ctrl.failIfExporterPodFailed(backup, vmExport); done || err != nil {
		return err
	}
	if vmExport.Status.Phase == exportv1.Terminated {
		return ctrl.completeOfflineBackup(backup, backupTracker)
	}
	return nil
}

func exportReadyReasonIs(vmExport *exportv1.VirtualMachineExport, reason string) bool {
	for _, cond := range vmExport.Status.Conditions {
		if cond.Type == exportv1.ConditionReady {
			return cond.Reason == reason
		}
	}
	return false
}

func (ctrl *VMBackupController) failIfEmptyIncremental(backup *backupv1.VirtualMachineBackup, vmExport *exportv1.VirtualMachineExport) (bool, error) {
	if vmExport.Status == nil || !exportReadyReasonIs(vmExport, exportEmptyIncrementalReason) {
		return false, nil
	}
	if err := ctrl.cleanupBackupExportAndWait(backup); err != nil {
		return false, err
	}
	ctrl.setFailed(backup, backupv1.ReasonEmptyIncremental, offlineEmptyIncrementalMsg)
	return true, nil
}

func (ctrl *VMBackupController) failIfExporterPodFailed(backup *backupv1.VirtualMachineBackup, vmExport *exportv1.VirtualMachineExport) (bool, error) {
	if vmExport.Status == nil || !exportReadyReasonIs(vmExport, exportPodFailedReason) {
		return false, nil
	}
	if err := ctrl.cleanupBackupExportAndWait(backup); err != nil {
		return false, err
	}
	ctrl.setFailed(backup, backupv1.ReasonFailed, offlineExporterPodFailedMsg)
	return true, nil
}

func (ctrl *VMBackupController) reconcileOfflineDeleting(backup *backupv1.VirtualMachineBackup, backupTracker *backupv1.VirtualMachineBackupTracker) error {
	// Only advance if the export actually wrote its checkpoint bitmap. A backup
	// deleted before that point never did, so advancing would corrupt the chain.
	if !IsBackupTerminal(backup) {
		done, err := ctrl.offlineExportFinished(backup)
		if err != nil {
			return err
		}
		if done {
			if err := ctrl.advanceOfflineTracker(backup, backupTracker); err != nil {
				return err
			}
		}
	}
	if err := ctrl.cleanupBackupExportAndWait(backup); err != nil {
		return err
	}
	return ctrl.removeBackupFinalizer(backup)
}

// Push has no ready signal on the backup itself, so check the export's phase instead.
func (ctrl *VMBackupController) offlineExportFinished(backup *backupv1.VirtualMachineBackup) (bool, error) {
	if isPullMode(backup) {
		return isBackupExportReady(backup), nil
	}
	objKey := types.NamespacedName{Namespace: backup.Namespace, Name: backup.Name}.String()
	obj, exists, err := ctrl.vmExportStore.GetByKey(objKey)
	if err != nil {
		return false, fmt.Errorf("error getting VMExport from store: %w", err)
	}
	if !exists {
		return false, nil
	}
	vmExport := obj.(*exportv1.VirtualMachineExport)
	return vmExport.Status != nil && vmExport.Status.Phase == exportv1.Terminated, nil
}

func (ctrl *VMBackupController) completeOfflineBackup(backup *backupv1.VirtualMachineBackup, backupTracker *backupv1.VirtualMachineBackupTracker) error {
	if err := ctrl.advanceOfflineTracker(backup, backupTracker); err != nil {
		return err
	}
	if err := ctrl.cleanupBackupExportAndWait(backup); err != nil {
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
		// HostDisk has no PVC for the offline exporter pod to mount.
		if IsCBTEligibleVolume(&volume) && storagetypes.PVCNameFromVirtVolume(&volume) != "" {
			volumes = append(volumes, backupv1.BackupVolumeInfo{VolumeName: volume.Name})
		}
	}
	return volumes
}

// Includes the creation time so a backup deleted and recreated with the same
// name doesn't collide with its own, already-committed checkpoint bitmap.
func offlineCheckpointName(backup *backupv1.VirtualMachineBackup) string {
	return fmt.Sprintf("backup-%s-%s", backup.Name, backup.CreationTimestamp.UTC().Format(offlineCheckpointTimeFormat))
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
