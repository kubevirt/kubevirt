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
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	vmBackupFinalizer = "backup.kubevirt.io/vmbackup-protection"

	backupInitializingEvent         = "VirtualMachineBackupInitializing"
	backupInitiatedEvent            = "VirtualMachineBackupInitiated"
	backupCompletedEvent            = "VirtualMachineBackupCompletedSuccessfully"
	backupCompletedWithWarningEvent = "VirtualMachineBackupCompletedWithWarning"

	backupInitializing = "Backup is initializing"
	backupInProgress   = "Backup is in progress"
	backupDeleting     = "Backup is deleting"
	backupCompleted    = "Successfully completed VirtualMachineBackup"

	backupCompletedWithWarningMsg        = "Completed VirtualMachineBackup, warning: %s"
	vmNotFoundMsg                        = "VM %s/%s doesnt exist"
	vmNotRunningMsg                      = "vm %s is not running, can not do backup"
	vmNoVolumesToBackupMsg               = "vm %s has no volumes to backup"
	vmNoChangedBlockTrackingMsg          = "vm %s has no ChangedBlockTracking, cannot start backup"
	invalidBackupModeMsg                 = "invalid backup mode: %s"
	backupSourceNameEmptyMsg             = "Source name is empty"
	backupDeletingMsg                    = "Backup is being deleted"
	backupDeletingBeforeVMICompletionMsg = "Backup is being deleted before VMI completion, waiting for completion"
)

var (
	errSourceNameEmpty = fmt.Errorf("source name is empty")
)

type VMBackupController struct {
	client         kubecli.KubevirtClient
	backupInformer cache.SharedIndexInformer
	vmStore        cache.Store
	vmiStore       cache.Store
	pvcStore       cache.Store
	recorder       record.EventRecorder
	backupQueue    workqueue.TypedRateLimitingInterface[string]
	hasSynced      func() bool
}

func NewVMBackupController(client kubecli.KubevirtClient,
	backupInformer cache.SharedIndexInformer,
	vmInformer cache.SharedIndexInformer,
	vmiInformer cache.SharedIndexInformer,
	pvcInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
) (*VMBackupController, error) {
	c := &VMBackupController{
		backupQueue: workqueue.NewTypedRateLimitingQueueWithConfig(
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "virt-controller-vmbackup"},
		),
		backupInformer: backupInformer,
		vmStore:        vmInformer.GetStore(),
		vmiStore:       vmiInformer.GetStore(),
		pvcStore:       pvcInformer.GetStore(),
		recorder:       recorder,
		client:         client,
	}

	c.hasSynced = func() bool {
		return backupInformer.HasSynced() && vmInformer.HasSynced() && vmiInformer.HasSynced() && pvcInformer.HasSynced()
	}

	_, err := backupInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleBackup,
			UpdateFunc: func(oldObj, newObj interface{}) { c.handleBackup(newObj) },
			DeleteFunc: c.handleBackup,
		},
	)
	if err != nil {
		return nil, err
	}
	_, err = vmiInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: c.handleUpdateVMI,
		},
	)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (ctrl *VMBackupController) handleBackup(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if backup, ok := obj.(*backupv1.VirtualMachineBackup); ok {
		objName, err := cache.DeletionHandlingMetaNamespaceKeyFunc(backup)
		if err != nil {
			log.Log.Errorf("failed to get key from object: %v, %v", err, backup)
			return
		}

		log.Log.V(3).Infof("enqueued %q for sync", objName)
		ctrl.backupQueue.Add(objName)
	}
}

func cacheKeyFunc(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func (ctrl *VMBackupController) handleUpdateVMI(oldObj, newObj interface{}) {
	ovmi, ok := oldObj.(*v1.VirtualMachineInstance)
	if !ok {
		return
	}

	nvmi, ok := newObj.(*v1.VirtualMachineInstance)
	if !ok {
		return
	}

	if equality.Semantic.DeepEqual(ovmi.Status, nvmi.Status) {
		return
	}
	key := cacheKeyFunc(nvmi.Namespace, nvmi.Name)
	keys, err := ctrl.backupInformer.GetIndexer().IndexKeys("vmi", key)
	if err != nil {
		return
	}

	for _, key := range keys {
		ctrl.backupQueue.Add(key)
	}
}

func (ctrl *VMBackupController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer ctrl.backupQueue.ShutDown()

	log.Log.Info("Starting backup controller.")
	defer log.Log.Info("Shutting down backup controller.")

	if !cache.WaitForCacheSync(
		stopCh,
		ctrl.hasSynced,
	) {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for range threadiness {
		go wait.Until(ctrl.runWorker, time.Second, stopCh)
	}

	<-stopCh

	return nil
}

func (ctrl *VMBackupController) runWorker() {
	for ctrl.Execute() {
	}
}

func (ctrl *VMBackupController) Execute() bool {
	key, quit := ctrl.backupQueue.Get()
	if quit {
		return false
	}
	defer ctrl.backupQueue.Done(key)

	err := ctrl.execute(key)
	if err != nil {
		log.Log.Reason(err).Infof("reenqueuing VirtualMachineBackup %v", key)
		ctrl.backupQueue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineBackup %v", key)
		ctrl.backupQueue.Forget(key)
	}
	return true
}

type SyncInfo struct {
	err    error
	reason string
	event  string
}

func syncInfoError(err error) *SyncInfo {
	return &SyncInfo{err: err}
}

func (ctrl *VMBackupController) execute(key string) error {
	logger := log.Log.With("VirtualMachineBackup", key)
	logger.V(3).Infof("Processing VirtualMachineBackup %s", key)
	storeObj, exists, err := ctrl.backupInformer.GetStore().GetByKey(key)
	if err != nil {
		logger.Errorf("Error getting backup from store: %v", err)
		return err
	}
	if !exists {
		logger.V(3).Infof("Backup %s no longer exists in store", key)
		return nil
	}

	backup, ok := storeObj.(*backupv1.VirtualMachineBackup)
	if !ok {
		logger.Errorf("Unexpected resource type: %T", storeObj)
		return fmt.Errorf("unexpected resource %+v", storeObj)
	}

	syncInfo := ctrl.sync(backup)
	if syncInfo != nil && syncInfo.err != nil {
		return syncInfo.err
	}

	err = ctrl.updateStatus(backup, syncInfo, logger)
	if err != nil {
		logger.Reason(err).Errorf("Updating the VirtualMachineBackup status failed")
		return err
	}

	logger.V(4).Infof("Successfully processed backup %s", key)
	return nil
}

func (ctrl *VMBackupController) sync(backup *backupv1.VirtualMachineBackup) *SyncInfo {
	logger := log.Log.With("VirtualMachineBackup", backup.Name)
	// If backup is done and not being deleted, nothing to do
	if IsBackupDone(backup.Status) && !isBackupDeleting(backup) {
		logger.V(4).Info("Backup is already done, skipping reconciliation")
		return nil
	}

	sourceName := getSourceName(backup)
	if sourceName == "" {
		logger.Errorf(backupSourceNameEmptyMsg)
		return syncInfoError(errSourceNameEmpty)
	}

	if isBackupDeleting(backup) {
		logger.V(3).Info(backupDeletingMsg)
		return ctrl.deletionCleanup(backup)
	}

	vmi, syncInfo := ctrl.verifyBackupSource(backup, sourceName)
	if syncInfo != nil {
		return syncInfo
	}

	if !isBackupInitializing(backup.Status) || vmi == nil {
		return ctrl.checkBackupCompletion(backup, vmi)
	}

	backup, err := ctrl.addBackupFinalizer(backup)
	if err != nil {
		err = fmt.Errorf("failed to add finalizer: %w", err)
		logger.Error(err.Error())
		return syncInfoError(err)
	}

	if err = ctrl.updateSourceBackupInProgress(vmi, backup.Name); err != nil {
		err = fmt.Errorf("failed to update source backup in progress: %w", err)
		logger.Error(err.Error())
		return syncInfoError(err)
	}

	backupOptions := backupv1.BackupOptions{
		BackupName:      backup.Name,
		Cmd:             backupv1.Start,
		BackupStartTime: &backup.CreationTimestamp,
		SkipQuiesce:     backup.Spec.SkipQuiesce,
	}
	if backup.Spec.Mode == nil {
		backup.Spec.Mode = pointer.P(backupv1.PushMode)
	}
	switch *backup.Spec.Mode {
	case backupv1.PushMode:
		pvcName := backup.Spec.PvcName
		syncInfo = ctrl.verifyBackupTargetPVC(pvcName, backup.Namespace)
		if syncInfo != nil {
			return syncInfo
		}

		volumeName := backupTargetVolumeName(backup.Name)
		attached := ctrl.backupTargetPVCAttached(vmi, volumeName)
		if !attached {
			return ctrl.attachBackupTargetPVC(vmi, *pvcName, volumeName)
		}
		backupOptions.Mode = backupv1.PushMode
		backupOptions.PushPath = pointer.P(hotplugdisk.GetVolumeMountDir(volumeName))
	default:
		logger.Errorf(invalidBackupModeMsg, *backup.Spec.Mode)
		return syncInfoError(fmt.Errorf(invalidBackupModeMsg, *backup.Spec.Mode))
	}

	err = ctrl.client.VirtualMachineInstance(vmi.Namespace).Backup(context.Background(), vmi.Name, &backupOptions)
	if err != nil {
		err = fmt.Errorf("failed to send Start backup command: %w", err)
		logger.Error(err.Error())
		return syncInfoError(err)
	}
	logger.Infof("Started backup for VMI %s successfully", vmi.Name)

	return &SyncInfo{
		event:  backupInitiatedEvent,
		reason: backupInProgress,
	}
}

func (ctrl *VMBackupController) updateStatus(backup *backupv1.VirtualMachineBackup, syncInfo *SyncInfo, logger *log.FilteredLogger) error {
	backupOut := backup.DeepCopy()

	if backup.Status == nil {
		backupOut.Status = &backupv1.VirtualMachineBackupStatus{}
		updateBackupCondition(backupOut, newInitializingCondition(corev1.ConditionTrue, backupInitializing))
		updateBackupCondition(backupOut, newProgressingCondition(corev1.ConditionFalse, backupInitializing))
	}

	if syncInfo != nil {
		// TODO: Handle failure and abort events (backupFailedEvent, backupAbortedEvent)
		switch syncInfo.event {
		case backupInitializingEvent:
			updateBackupCondition(backupOut, newInitializingCondition(corev1.ConditionTrue, syncInfo.reason))
			updateBackupCondition(backupOut, newProgressingCondition(corev1.ConditionFalse, syncInfo.reason))
		case backupInitiatedEvent:
			removeBackupCondition(backupOut, backupv1.ConditionInitializing)
			updateBackupCondition(backupOut, newProgressingCondition(corev1.ConditionTrue, syncInfo.reason))
			updateBackupCondition(backupOut, newDoneCondition(corev1.ConditionFalse, syncInfo.reason))
		case backupCompletedEvent, backupCompletedWithWarningEvent:
			if syncInfo.event == backupCompletedWithWarningEvent {
				ctrl.recorder.Eventf(backupOut, corev1.EventTypeWarning, backupCompletedWithWarningEvent, syncInfo.reason)
			} else {
				ctrl.recorder.Eventf(backupOut, corev1.EventTypeNormal, backupCompletedEvent, syncInfo.reason)
			}
			updateBackupCondition(backupOut, newProgressingCondition(corev1.ConditionFalse, syncInfo.reason))
			updateBackupCondition(backupOut, newDoneCondition(corev1.ConditionTrue, syncInfo.reason))
			backupOut.Status.Type = backupv1.Full
		}
	}

	if isBackupDeleting(backupOut) && controller.HasFinalizer(backupOut, vmBackupFinalizer) {
		logger.Info("update backup is deleting")
		updateBackupCondition(backupOut, newDeletingCondition(corev1.ConditionTrue, backupDeleting))
	}

	if !equality.Semantic.DeepEqual(backup.Status, backupOut.Status) {
		if _, err := ctrl.client.VirtualMachineBackup(backupOut.Namespace).UpdateStatus(context.Background(), backupOut, metav1.UpdateOptions{}); err != nil {
			logger.Reason(err).Error("failed to update backup status")
			return err
		}
	}
	return nil
}

func generateFinalizerPatch(test, replace []string) ([]byte, error) {
	return patch.New(
		patch.WithTest("/metadata/finalizers", test),
		patch.WithReplace("/metadata/finalizers", replace),
	).GeneratePayload()
}

func (ctrl *VMBackupController) addBackupFinalizer(backup *backupv1.VirtualMachineBackup) (*backupv1.VirtualMachineBackup, error) {
	if controller.HasFinalizer(backup, vmBackupFinalizer) {
		return backup, nil
	}

	cpy := backup.DeepCopy()
	controller.AddFinalizer(cpy, vmBackupFinalizer)

	patchBytes, err := generateFinalizerPatch(backup.Finalizers, cpy.Finalizers)
	if err != nil {
		return backup, err
	}

	return ctrl.client.VirtualMachineBackup(cpy.Namespace).Patch(context.Background(), cpy.Name, k8stypes.JSONPatchType, patchBytes, metav1.PatchOptions{})
}

func (ctrl *VMBackupController) removeBackupFinalizer(backup *backupv1.VirtualMachineBackup) *SyncInfo {
	if !controller.HasFinalizer(backup, vmBackupFinalizer) {
		return nil
	}

	cpy := backup.DeepCopy()
	controller.RemoveFinalizer(cpy, vmBackupFinalizer)

	patchBytes, err := generateFinalizerPatch(backup.Finalizers, cpy.Finalizers)
	if err != nil {
		err = fmt.Errorf("failed to generate finalizer patch: %w", err)
		log.Log.With("VirtualMachineBackup", backup.Name).Error(err.Error())
		return syncInfoError(err)
	}

	_, err = ctrl.client.VirtualMachineBackup(cpy.Namespace).Patch(context.Background(), cpy.Name, k8stypes.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		err = fmt.Errorf("failed to patch backup to remove finalizer: %w", err)
		log.Log.With("VirtualMachineBackup", backup.Name).Error(err.Error())
		return syncInfoError(err)
	}
	return nil
}

func getSourceName(backup *backupv1.VirtualMachineBackup) string {
	// source name is fetched from the source field which is required until VirtualMachineBackupTracker is introduced
	return backup.Spec.Source.Name
}

func (ctrl *VMBackupController) getVMI(backup *backupv1.VirtualMachineBackup) (*v1.VirtualMachineInstance, bool, error) {
	sourceName := getSourceName(backup)
	objKey := cacheKeyFunc(backup.Namespace, sourceName)

	obj, exists, err := ctrl.vmiStore.GetByKey(objKey)
	if err != nil {
		return nil, false, err
	}

	if !exists {
		return nil, false, nil
	}

	return obj.(*v1.VirtualMachineInstance), exists, nil
}

func (ctrl *VMBackupController) verifyBackupSource(backup *backupv1.VirtualMachineBackup, sourceName string) (*v1.VirtualMachineInstance, *SyncInfo) {
	objKey := cacheKeyFunc(backup.Namespace, sourceName)
	_, exists, err := ctrl.vmStore.GetByKey(objKey)
	if err != nil {
		err = fmt.Errorf("failed to get VM from store: %w", err)
		log.Log.With("VirtualMachineBackup", backup.Name).Error(err.Error())
		return nil, syncInfoError(err)
	}

	if !exists {
		return nil, &SyncInfo{
			event:  backupInitializingEvent,
			reason: fmt.Sprintf(vmNotFoundMsg, backup.Namespace, sourceName),
		}
	}
	vmi, exists, err := ctrl.getVMI(backup)
	if err != nil {
		err = fmt.Errorf("failed to get VMI from store: %w", err)
		log.Log.With("VirtualMachineBackup", backup.Name).Error(err.Error())
		return nil, syncInfoError(err)
	}
	if !exists {
		return nil, &SyncInfo{
			event:  backupInitializingEvent,
			reason: fmt.Sprintf(vmNotRunningMsg, sourceName),
		}
	}
	hasEligibleVolumes := false
	for _, volume := range vmi.Spec.Volumes {
		if IsCBTEligibleVolume(&volume) {
			hasEligibleVolumes = true
			break
		}
	}
	if !hasEligibleVolumes {
		return nil, &SyncInfo{
			event:  backupInitializingEvent,
			reason: fmt.Sprintf(vmNoVolumesToBackupMsg, sourceName),
		}
	}
	if vmi.Status.ChangedBlockTracking == nil || vmi.Status.ChangedBlockTracking.State != v1.ChangedBlockTrackingEnabled {
		log.Log.With("VirtualMachineBackup", backup.Name).Errorf(vmNoChangedBlockTrackingMsg, sourceName)
		return nil, &SyncInfo{
			event:  backupInitializingEvent,
			reason: fmt.Sprintf(vmNoChangedBlockTrackingMsg, sourceName),
		}
	}

	return vmi, nil
}

func (ctrl *VMBackupController) removeSourceBackupInProgress(vmi *v1.VirtualMachineInstance) *SyncInfo {
	if !hasVMIBackupStatus(vmi) {
		return nil
	}

	patchBytes, err := patch.New(
		patch.WithRemove("/status/changedBlockTracking/backupStatus"),
	).GeneratePayload()
	if err != nil {
		return syncInfoError(err)
	}

	_, err = ctrl.client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, k8stypes.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		err = fmt.Errorf("failed to remove BackupInProgress from VMI %s/%s: %w", vmi.Namespace, vmi.Name, err)
		log.Log.Error(err.Error())
		return syncInfoError(err)
	}

	return nil
}

func (ctrl *VMBackupController) updateSourceBackupInProgress(vmi *v1.VirtualMachineInstance, backupName string) error {
	if hasVMIBackupStatus(vmi) {
		if vmi.Status.ChangedBlockTracking.BackupStatus.BackupName != backupName {
			return fmt.Errorf("another backup %s is already in progress, cannot start backup %s",
				vmi.Status.ChangedBlockTracking.BackupStatus.BackupName, backupName)
		}
		return nil
	}

	backupStatus := &v1.VirtualMachineInstanceBackupStatus{
		BackupName: backupName,
	}

	patchSet := patch.New(
		patch.WithTest("/status/changedBlockTracking/backupStatus", vmi.Status.ChangedBlockTracking.BackupStatus),
	)
	if vmi.Status.ChangedBlockTracking.BackupStatus == nil {
		patchSet.AddOption(patch.WithAdd("/status/changedBlockTracking/backupStatus", backupStatus))
	} else {
		patchSet.AddOption(patch.WithReplace("/status/changedBlockTracking/backupStatus", backupStatus))
	}
	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return err
	}

	_, err = ctrl.client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, k8stypes.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		log.Log.Errorf("Failed to update source backup in progress: %s", err)
		return err
	}

	return nil
}

func (ctrl *VMBackupController) checkBackupCompletion(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) *SyncInfo {
	// If backup is done or VMI backup status is missing, perform cleanup
	if IsBackupDone(backup.Status) || !hasVMIBackupStatus(vmi) {
		_, syncInfo := ctrl.cleanup(backup, vmi)
		return syncInfo
	}

	backupStatus := vmi.Status.ChangedBlockTracking.BackupStatus
	if !backupStatus.Completed {
		return nil
	}

	log.Log.Object(backup).Info("Backup completed, performing cleanup")
	done, syncInfo := ctrl.cleanup(backup, vmi)
	if syncInfo != nil {
		return syncInfo
	}
	if !done {
		return nil
	}

	// TODO: Handle backup failure (backupStatus.Failed) and abort status (backupStatus.AbortStatus)

	// Check if backup completed with a warning message
	if backupStatus.BackupMsg != nil {
		log.Log.Object(backup).Infof(backupCompletedWithWarningMsg, *backupStatus.BackupMsg)
		syncInfo = &SyncInfo{
			event:  backupCompletedWithWarningEvent,
			reason: fmt.Sprintf(backupCompletedWithWarningMsg, *backupStatus.BackupMsg),
		}
	} else {
		log.Log.Object(backup).Info("Backup completed successfully")
		syncInfo = &SyncInfo{
			event:  backupCompletedEvent,
			reason: backupCompleted,
		}
	}

	return syncInfo
}

func (ctrl *VMBackupController) deletionCleanup(backup *backupv1.VirtualMachineBackup) *SyncInfo {
	vmi, _, err := ctrl.getVMI(backup)
	if err != nil {
		err = fmt.Errorf("failed to get VMI during deletion cleanup: %w", err)
		log.Log.With("VirtualMachineBackup", backup.Name).Error(err.Error())
		return syncInfoError(err)
	}

	vmiBackupInProgress := hasVMIBackupStatus(vmi) &&
		vmi.Status.ChangedBlockTracking.BackupStatus.BackupName == backup.Name &&
		!vmi.Status.ChangedBlockTracking.BackupStatus.Completed

	if vmiBackupInProgress {
		log.Log.With("VirtualMachineBackup", backup.Name).V(3).Info(backupDeletingBeforeVMICompletionMsg)
		// TODO: abort running backup on deletion instead of waiting for completion
		return nil
	}

	done, syncInfo := ctrl.cleanup(backup, vmi)
	if syncInfo != nil {
		return syncInfo
	}
	if !done {
		return syncInfoError(fmt.Errorf("cleanup not yet complete for deleted backup"))
	}
	return nil
}

func isPushMode(backup *backupv1.VirtualMachineBackup) bool {
	return backup.Spec.Mode == nil || *backup.Spec.Mode == backupv1.PushMode
}

func (ctrl *VMBackupController) cleanup(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) (bool, *SyncInfo) {
	if isPushMode(backup) {
		volumeName := backupTargetVolumeName(backup.Name)
		detached := ctrl.backupTargetPVCDetached(vmi, volumeName)
		if !detached {
			return false, ctrl.detachBackupTargetPVC(vmi, volumeName)
		}
	}

	syncInfo := ctrl.removeSourceBackupInProgress(vmi)
	if syncInfo != nil {
		return false, syncInfo
	}

	if isBackupDeleting(backup) {
		if syncInfo := ctrl.removeBackupFinalizer(backup); syncInfo != nil {
			return false, syncInfo
		}
	}

	return true, nil
}

func isBackupInitializing(status *backupv1.VirtualMachineBackupStatus) bool {
	return status == nil || hasCondition(status.Conditions, backupv1.ConditionInitializing)
}

func IsBackupDone(status *backupv1.VirtualMachineBackupStatus) bool {
	return status != nil && hasCondition(status.Conditions, backupv1.ConditionDone)
}

func updateCondition(conditions []backupv1.Condition, c backupv1.Condition) []backupv1.Condition {
	found := false
	for i := range conditions {
		if conditions[i].Type == c.Type {
			if conditions[i].Status != c.Status || conditions[i].Reason != c.Reason || conditions[i].Message != c.Message {
				conditions[i] = c
			}
			found = true
			break
		}
	}

	if !found {
		conditions = append(conditions, c)
	}

	return conditions
}

func newCondition(condType backupv1.ConditionType, status corev1.ConditionStatus, reason string) backupv1.Condition {
	return backupv1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: metav1.Now(),
	}
}

func newInitializingCondition(status corev1.ConditionStatus, reason string) backupv1.Condition {
	return newCondition(backupv1.ConditionInitializing, status, reason)
}

func newDoneCondition(status corev1.ConditionStatus, reason string) backupv1.Condition {
	return newCondition(backupv1.ConditionDone, status, reason)
}

func newProgressingCondition(status corev1.ConditionStatus, reason string) backupv1.Condition {
	return newCondition(backupv1.ConditionProgressing, status, reason)
}

func newDeletingCondition(status corev1.ConditionStatus, reason string) backupv1.Condition {
	return newCondition(backupv1.ConditionDeleting, status, reason)
}

func hasCondition(conditions []backupv1.Condition, condType backupv1.ConditionType) bool {
	for _, cond := range conditions {
		if cond.Type == condType {
			return cond.Status == corev1.ConditionTrue
		}
	}
	return false
}

func updateBackupCondition(b *backupv1.VirtualMachineBackup, c backupv1.Condition) {
	b.Status.Conditions = updateCondition(b.Status.Conditions, c)
}

func removeBackupCondition(b *backupv1.VirtualMachineBackup, cType backupv1.ConditionType) {
	var conds []backupv1.Condition
	for _, c := range b.Status.Conditions {
		if c.Type == cType {
			continue
		}
		conds = append(conds, c)
	}
	b.Status.Conditions = conds
}

func isBackupDeleting(backup *backupv1.VirtualMachineBackup) bool {
	return backup != nil && backup.DeletionTimestamp != nil
}

func hasVMIBackupStatus(vmi *v1.VirtualMachineInstance) bool {
	return vmi != nil && vmi.Status.ChangedBlockTracking != nil && vmi.Status.ChangedBlockTracking.BackupStatus != nil
}
