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
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	backupv1 "kubevirt.io/api/backup/v1beta1"
	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

const (
	offlineNoVolumesMsg         = "vm %s has no volumes eligible for backup"
	offlineStalePodsMsg         = "vm %s still has non-terminal virt-launcher pods, waiting before starting offline backup"
	offlineExporterPodFailedMsg = "offline backup exporter pod failed"

	exportPodFailedReason = "PodFailed"

	virtualMachineExportKind = "VirtualMachineExport"

	offlineCheckpointTimeFormat = "2006-01-02_15-04-05"

	backupTypeProbeContainerName = "backup-type-probe"
)

// reconcileOffline runs once the source VM is held for this backup (see heldSourceVM). The hold keeps
// the VM stopped, so there is no live VMI to race with.
func (ctrl *VMBackupController) reconcileOffline(backup *backupv1.VirtualMachineBackup, vm *v1.VirtualMachine, backupTracker *backupv1.VirtualMachineBackupTracker, backupDeleting bool) error {
	if backupDeleting {
		return ctrl.reconcileOfflineDeleting(backup, backupTracker)
	}
	if backup.Status.StartTimestamp == nil {
		return ctrl.startOfflineBackup(backup, vm, backupTracker)
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

// startOfflineBackup initializes a held VM's offline backup. The VM is already stopped and held, so
// this only validates it can be backed up and stamps the starting status.
func (ctrl *VMBackupController) startOfflineBackup(backup *backupv1.VirtualMachineBackup, vm *v1.VirtualMachine, backupTracker *backupv1.VirtualMachineBackupTracker) error {
	if stale, err := ctrl.hasNonTerminalLauncherPods(backup.Namespace, vm.Name); err != nil {
		return err
	} else if stale {
		setInitializing(backup, fmt.Sprintf(offlineStalePodsMsg, vm.Name))
		// No pod event handler re-drives this backup, so poll until the launcher pods are gone.
		return errOfflineWaiting
	}

	volumes := offlineEligibleVolumes(vm)
	if len(volumes) == 0 {
		ctrl.setFailed(backup, backupv1.ReasonFailed, fmt.Sprintf(offlineNoVolumesMsg, vm.Name))
		return nil
	}
	if vm.Status.ChangedBlockTracking == nil || vm.Status.ChangedBlockTracking.State != v1.ChangedBlockTrackingEnabled {
		ctrl.setFailed(backup, backupv1.ReasonFailed, fmt.Sprintf(vmNoChangedBlockTrackingMsg, vm.Name))
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

	backupType := backupv1.Full
	if isIncrementalBackup(backup, backupTracker) {
		backupType = backupv1.Incremental
	}
	// Offline backups have no live VMI to report a type per volume, so every volume starts with
	// the same, backup-wide type. The exporter's bitmap probe later corrects any volume that has
	// to fall back to full.
	for i := range volumes {
		volumes[i].Type = backupType
	}
	backup.Status.CheckpointName = pointer.P(offlineCheckpointName(backup))
	backup.Status.IncludedVolumes = volumes
	if backup.Status.StartTimestamp == nil {
		backup.Status.StartTimestamp = pointer.P(metav1.Now())
	}
	setProgressing(backup)

	log.Log.Object(backup).Infof("Started offline %s backup for VM %s", backupType, vm.Name)
	return nil
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

// getExporterPod finds the exporter pod owned by this backup's VirtualMachineExport.
func (ctrl *VMBackupController) getExporterPod(namespace string, backup *backupv1.VirtualMachineBackup) (*corev1.Pod, bool, error) {
	objs, err := ctrl.podIndexer.ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, false, err
	}
	for _, obj := range objs {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			continue
		}
		ownerRef := metav1.GetControllerOf(pod)
		if ownerRef == nil || ownerRef.Kind != virtualMachineExportKind || ownerRef.Name != backup.Name {
			continue
		}
		return pod, true, nil
	}
	return nil, false, nil
}

// cleanupBackupExport only requests deletion. The pod itself is GC'd asynchronously,
// so callers need this to confirm it's actually gone before releasing the VM-start hold.
func (ctrl *VMBackupController) hasNonTerminalExporterPod(namespace string, backup *backupv1.VirtualMachineBackup) (bool, error) {
	pod, exists, err := ctrl.getExporterPod(namespace, backup)
	if err != nil || !exists {
		return false, err
	}
	return !controller.PodIsDown(pod), nil
}

type offlineVolumeTypeProbeResult struct {
	VolumeName string `json:"volumeName"`
	Type       string `json:"type"`
}

// applyOfflineVolumeBackupTypes relays the probe init container's per-volume fallback decision
// into the backup's status.
func (ctrl *VMBackupController) applyOfflineVolumeBackupTypes(backup *backupv1.VirtualMachineBackup) error {
	pod, exists, err := ctrl.getExporterPod(backup.Namespace, backup)
	if err != nil || !exists {
		return err
	}
	for _, cs := range pod.Status.InitContainerStatuses {
		if cs.Name != backupTypeProbeContainerName || cs.State.Terminated == nil {
			continue
		}
		var results []offlineVolumeTypeProbeResult
		if err := json.Unmarshal([]byte(cs.State.Terminated.Message), &results); err != nil {
			log.Log.Object(backup).Reason(err).Warning("failed to parse backup type probe result")
			return nil
		}
		for _, r := range results {
			for i := range backup.Status.IncludedVolumes {
				if backup.Status.IncludedVolumes[i].VolumeName == r.VolumeName {
					backup.Status.IncludedVolumes[i].Type = backupv1.BackupType(r.Type)
				}
			}
		}
		return nil
	}
	return nil
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

func (ctrl *VMBackupController) reconcileOfflinePull(backup *backupv1.VirtualMachineBackup, backupTracker *backupv1.VirtualMachineBackupTracker) error {
	if isBackupTTLExpired(backup) {
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

	if err := ctrl.applyOfflineVolumeBackupTypes(backup); err != nil {
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

	if err := ctrl.applyOfflineVolumeBackupTypes(backup); err != nil {
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
	// Release the VM before dropping the finalizer: once the object is gone no reconcile can clear
	// the hold, which would leave the VM permanently unstartable.
	if err := ctrl.releaseSourceVM(backup); err != nil {
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
	obj, exists, err := ctrl.vmStore.GetByKey(types.NamespacedName{Namespace: namespace, Name: name}.String())
	if err != nil || !exists {
		return nil, err
	}
	return obj.(*v1.VirtualMachine), nil
}

// holdSourceVM latches the stopped source VM for this offline backup by stamping
// vm.Status.BackupInProgress. The VM controller refuses to start a held VM, which both prevents a
// concurrent writer and serializes offline backups (a VM held by another backup can't be re-held).
func (ctrl *VMBackupController) holdSourceVM(backup *backupv1.VirtualMachineBackup, sourceName string) (bool, error) {
	vm, err := ctrl.getVM(backup.Namespace, sourceName)
	if err != nil || vm == nil {
		return false, err
	}
	if holder := vm.Status.BackupInProgress; holder != nil {
		return *holder == backup.Name, nil
	}
	if vm.DeletionTimestamp != nil || vm.Status.Created || controller.IsSetToStart(vm, nil) ||
		vm.Status.SnapshotInProgress != nil || vm.Status.RestoreInProgress != nil {
		return false, nil
	}

	vmCopy := vm.DeepCopy()
	vmCopy.Status.BackupInProgress = pointer.P(backup.Name)
	if _, err := ctrl.client.VirtualMachine(vm.Namespace).UpdateStatus(context.Background(), vmCopy, metav1.UpdateOptions{}); err != nil {
		return false, fmt.Errorf("failed to hold VM %s for backup %s: %w", vm.Name, backup.Name, err)
	}
	log.Log.Object(backup).Infof("Holding VM %s for an offline backup", vm.Name)
	return true, nil
}

// releaseSourceVM clears the hold once the offline backup is terminal, letting the VM start again.
func (ctrl *VMBackupController) releaseSourceVM(backup *backupv1.VirtualMachineBackup) error {
	vm, err := ctrl.heldVMForRelease(backup)
	if err != nil || vm == nil {
		return err
	}

	vmCopy := vm.DeepCopy()
	vmCopy.Status.BackupInProgress = nil
	if _, err := ctrl.client.VirtualMachine(vm.Namespace).UpdateStatus(context.Background(), vmCopy, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to release VM %s from backup %s: %w", vm.Name, backup.Name, err)
	}
	log.Log.Object(backup).Infof("Released VM %s from its offline backup", vm.Name)
	return nil
}

// heldVMForRelease returns the VM this backup currently holds, or nil if none does. It resolves the
// source VM by name first, then falls back to scanning for the holder: the tracker that name lookup
// depends on can be garbage-collected once a backup goes terminal, and without the fallback the hold
// would leak and leave the VM permanently unstartable.
func (ctrl *VMBackupController) heldVMForRelease(backup *backupv1.VirtualMachineBackup) (*v1.VirtualMachine, error) {
	sourceName, err := BackupSourceVMName(ctrl.backupTrackerInformer.GetStore(), backup)
	if err != nil {
		return nil, err
	}
	if sourceName != "" {
		vm, err := ctrl.getVM(backup.Namespace, sourceName)
		if err != nil {
			return nil, err
		}
		if vm != nil {
			if backupHoldsVM(vm, backup) {
				return vm, nil
			}
			return nil, nil
		}
	}
	for _, obj := range ctrl.vmStore.List() {
		vm, ok := obj.(*v1.VirtualMachine)
		if ok && vm.Namespace == backup.Namespace && backupHoldsVM(vm, backup) {
			return vm, nil
		}
	}
	return nil, nil
}

func backupHoldsVM(vm *v1.VirtualMachine, backup *backupv1.VirtualMachineBackup) bool {
	holder := vm.Status.BackupInProgress
	return holder != nil && *holder == backup.Name
}

// heldSourceVM reports whether the source VM is currently held by this backup, which is what marks
// a backup as running offline.
func (ctrl *VMBackupController) heldSourceVM(backup *backupv1.VirtualMachineBackup, sourceName string) (*v1.VirtualMachine, bool, error) {
	vm, err := ctrl.getVM(backup.Namespace, sourceName)
	if err != nil || vm == nil {
		return nil, false, err
	}
	return vm, backupHoldsVM(vm, backup), nil
}

func (ctrl *VMBackupController) advanceOfflineTracker(backup *backupv1.VirtualMachineBackup, backupTracker *backupv1.VirtualMachineBackupTracker) error {
	if backupTracker == nil || backup.Status.CheckpointName == nil {
		return nil
	}
	checkpoint := backupv1.BackupCheckpoint{
		Name:         *backup.Status.CheckpointName,
		CreationTime: &backup.CreationTimestamp,
	}
	return ctrl.patchTrackerCheckpoint(backup.Namespace, backupTracker, checkpoint)
}
