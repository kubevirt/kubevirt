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
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	backupv1 "kubevirt.io/api/backup/v1beta1"
	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
	hotplugdisk "kubevirt.io/kubevirt/pkg/storage/hotplug-disk"
	migrations "kubevirt.io/kubevirt/pkg/util/migrations"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	backupAbortingEvent             = "VirtualMachineBackupAborting"
	backupCompletedEvent            = "VirtualMachineBackupCompletedSuccessfully"
	backupCompletedWithWarningEvent = "VirtualMachineBackupCompletedWithWarning"
	backupFailedEvent               = "VirtualMachineBackupFailed"

	backupInProgress              = "Backup is in progress"
	backupPreparingVMExport       = "Backup export is being initialized"
	backupExportReady             = "Backup export is ready to pull"
	backupAborting                = "Backup is aborting"
	backupCompleted               = "Successfully completed VirtualMachineBackup"
	backupFailed                  = "Backup has failed: %s"
	backupCompletedWithWarningMsg = "Completed VirtualMachineBackup, warning: %s"
	vmNotFoundMsg                 = "VM %s/%s doesnt exist"
	vmNotRunningMsg               = "vm %s is not running, cannot do backup"
	vmHeldForOfflineBackupMsg     = "vm %s is held for an offline backup"
	vmNoVolumesToBackupMsg        = "vm %s has no volumes to backup"
	vmNoChangedBlockTrackingMsg   = "vm %s has no ChangedBlockTracking, cannot start backup"
	backupTrackerNotFoundMsg      = "BackupTracker %s does not exist"
	trackerDeletingMsg            = "tracker %s is being deleted"
	invalidBackupModeMsg          = "invalid backup mode: %s"
	vmMigrationInProgressMsg      = "vm %s is currently migrating, waiting for migration to complete before starting backup"
)

var (
	errSourceNameEmpty = errors.New("source name is empty")
	errCleanupPending  = errors.New("cleanup pending")
	errActiveBackups   = errors.New("active backups pending")
	errOfflineWaiting  = errors.New("waiting on another offline operation")
)

// retry every 1s instead of backing off, errCleanupPending is not a failure
const cleanupPendingRequeueInterval = time.Second

type VMBackupController struct {
	client                kubecli.KubevirtClient
	backupInformer        cache.SharedIndexInformer
	backupTrackerInformer cache.SharedIndexInformer
	vmStore               cache.Store
	vmiStore              cache.Store
	pvcStore              cache.Store
	vmExportStore         cache.Store
	podIndexer            cache.Indexer
	recorder              record.EventRecorder
	backupQueue           workqueue.TypedRateLimitingInterface[string]
	trackerQueue          workqueue.TypedRateLimitingInterface[string]
	hasSynced             func() bool
	clusterConfig         *virtconfig.ClusterConfig
}

func NewVMBackupController(client kubecli.KubevirtClient,
	backupInformer cache.SharedIndexInformer,
	backupTrackerInformer cache.SharedIndexInformer,
	vmInformer cache.SharedIndexInformer,
	vmiInformer cache.SharedIndexInformer,
	pvcInformer cache.SharedIndexInformer,
	vmExportInformer cache.SharedIndexInformer,
	podInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clusterConfig *virtconfig.ClusterConfig,
) (*VMBackupController, error) {
	c := &VMBackupController{
		backupQueue: workqueue.NewTypedRateLimitingQueueWithConfig(
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "virt-controller-vmbackup"},
		),
		trackerQueue: workqueue.NewTypedRateLimitingQueueWithConfig(
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "virt-controller-vmbackup-tracker"},
		),
		backupInformer:        backupInformer,
		backupTrackerInformer: backupTrackerInformer,
		vmStore:               vmInformer.GetStore(),
		vmiStore:              vmiInformer.GetStore(),
		pvcStore:              pvcInformer.GetStore(),
		vmExportStore:         vmExportInformer.GetStore(),
		podIndexer:            podInformer.GetIndexer(),
		recorder:              recorder,
		client:                client,
		clusterConfig:         clusterConfig,
	}

	c.hasSynced = func() bool {
		return backupInformer.HasSynced() && backupTrackerInformer.HasSynced() && vmInformer.HasSynced() && vmiInformer.HasSynced() && pvcInformer.HasSynced() && vmExportInformer.HasSynced() && podInformer.HasSynced()
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
			DeleteFunc: c.handleDeleteVMI,
		},
	)
	if err != nil {
		return nil, err
	}

	_, err = backupTrackerInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleBackupTracker,
			UpdateFunc: func(oldObj, newObj interface{}) { c.handleBackupTracker(newObj) },
			DeleteFunc: c.handleBackupTracker,
		},
	)
	if err != nil {
		return nil, err
	}

	_, err = vmExportInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleVMExport,
			UpdateFunc: func(oldObj, newObj interface{}) { c.handleUpdateVMExport(oldObj, newObj) },
			DeleteFunc: c.handleVMExport,
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

	backup, ok := obj.(*backupv1.VirtualMachineBackup)
	if !ok {
		return
	}

	objName, err := cache.DeletionHandlingMetaNamespaceKeyFunc(backup)
	if err != nil {
		log.Log.Errorf("failed to get key from object: %v, %v", err, backup)
		return
	}

	log.Log.V(3).Infof("enqueued %q for sync", objName)
	ctrl.backupQueue.Add(objName)

	if backup.Spec.Source.Kind == backupv1.VirtualMachineBackupTrackerGroupVersionKind.Kind {
		trackerKey := types.NamespacedName{Namespace: backup.Namespace, Name: backup.Spec.Source.Name}.String()
		ctrl.trackerQueue.Add(trackerKey)
	}
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
	ctrl.enqueueBackupsOfVMI(nvmi)
}

func (ctrl *VMBackupController) handleDeleteVMI(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	vmi, ok := obj.(*v1.VirtualMachineInstance)
	if !ok {
		return
	}
	ctrl.enqueueBackupsOfVMI(vmi)
}

func (ctrl *VMBackupController) enqueueBackupsOfVMI(vmi *v1.VirtualMachineInstance) {
	key := types.NamespacedName{Namespace: vmi.Namespace, Name: vmi.Name}.String()
	for _, backupKey := range backupKeysOfVM(ctrl.backupInformer.GetIndexer(), ctrl.backupTrackerInformer.GetIndexer(), key) {
		ctrl.backupQueue.Add(backupKey)
	}
}

// backupKeysOfVM returns the backups referencing the VM directly or through a BackupTracker.
func backupKeysOfVM(backupIndexer, trackerIndexer cache.Indexer, vmKey string) []string {
	keys, err := backupIndexer.IndexKeys("vmi", vmKey)
	if err != nil {
		return nil
	}

	trackerKeys, err := trackerIndexer.IndexKeys("vmi", vmKey)
	if err != nil {
		return keys
	}
	for _, trackerKey := range trackerKeys {
		backupKeys, err := backupIndexer.IndexKeys("backupTracker", trackerKey)
		if err != nil {
			continue
		}
		keys = append(keys, backupKeys...)
	}
	return keys
}

func (ctrl *VMBackupController) handleBackupTracker(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	tracker, ok := obj.(*backupv1.VirtualMachineBackupTracker)
	if !ok {
		return
	}

	key := types.NamespacedName{Namespace: tracker.Namespace, Name: tracker.Name}.String()

	if isTrackerDeleting(tracker) && controller.HasFinalizer(tracker, backupv1.VirtualMachineBackupTrackerFinalizer) {
		log.Log.V(3).Infof("enqueued tracker %q for deletion cleanup", key)
		ctrl.trackerQueue.Add(key)
	}

	// Enqueue related backups
	backupKeys, err := ctrl.backupInformer.GetIndexer().IndexKeys("backupTracker", key)
	if err != nil {
		return
	}
	for _, key := range backupKeys {
		ctrl.backupQueue.Add(key)
	}
}

func (ctrl *VMBackupController) handleVMExport(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if vmExport, ok := obj.(*exportv1.VirtualMachineExport); ok {
		key := getOwnerVMBackupKey(vmExport)
		_, exists, err := ctrl.backupInformer.GetStore().GetByKey(key)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}
		if exists {
			log.Log.V(3).Infof("Adding VMBackup due to VMExport creation: %s", key)
			ctrl.backupQueue.Add(key)
		}
	}
}

func (ctrl *VMBackupController) handleUpdateVMExport(oldObj, newObj interface{}) {
	ovmExport, ok := oldObj.(*exportv1.VirtualMachineExport)
	if !ok {
		return
	}

	nvmExport, ok := newObj.(*exportv1.VirtualMachineExport)
	if !ok {
		return
	}

	if equality.Semantic.DeepEqual(ovmExport.Status, nvmExport.Status) {
		return
	}

	key := getOwnerVMBackupKey(nvmExport)
	_, exists, err := ctrl.backupInformer.GetStore().GetByKey(key)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	if exists {
		log.Log.V(3).Infof("Adding VMBackup due to VMExport update: %s", key)
		ctrl.backupQueue.Add(key)
	}
}

func getOwnerVMBackupKey(obj metav1.Object) string {
	ownerRef := metav1.GetControllerOf(obj)
	if ownerRef == nil || ownerRef.Kind != backupv1.VirtualMachineBackupGroupVersionKind.Kind {
		return ""
	}
	gv, err := schema.ParseGroupVersion(ownerRef.APIVersion)
	if err != nil || gv.Group != backupv1.VirtualMachineBackupGroupVersionKind.Group {
		return ""
	}
	return types.NamespacedName{Namespace: obj.GetNamespace(), Name: ownerRef.Name}.String()
}

func (ctrl *VMBackupController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer ctrl.backupQueue.ShutDown()
	defer ctrl.trackerQueue.ShutDown()

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
		go wait.Until(ctrl.runTrackerWorker, time.Second, stopCh)
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
		if errors.Is(err, errCleanupPending) || errors.Is(err, errOfflineWaiting) {
			log.Log.V(4).Infof("reenqueuing VirtualMachineBackup %v: %v", key, err)
			ctrl.backupQueue.AddAfter(key, cleanupPendingRequeueInterval)
		} else {
			log.Log.Reason(err).Infof("reenqueuing VirtualMachineBackup %v", key)
			ctrl.backupQueue.AddRateLimited(key)
		}
	} else {
		log.Log.V(4).Infof("processed VirtualMachineBackup %v", key)
		ctrl.backupQueue.Forget(key)
	}
	return true
}

func isIncrementalBackup(backup *backupv1.VirtualMachineBackup, backupTracker *backupv1.VirtualMachineBackupTracker) bool {
	return !backup.Spec.ForceFullBackup &&
		backupTracker != nil && backupTracker.Status != nil &&
		backupTracker.Status.LatestCheckpoint != nil &&
		backupTracker.Status.LatestCheckpoint.Name != ""
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

	backupCopy := backup.DeepCopy()
	syncErr := ctrl.sync(backupCopy)

	if !equality.Semantic.DeepEqual(backup.Status, backupCopy.Status) {
		if _, err := ctrl.client.VirtualMachineBackup(backupCopy.Namespace).UpdateStatus(
			context.Background(), backupCopy, metav1.UpdateOptions{}); err != nil {
			logger.Reason(err).Errorf("Updating the VirtualMachineBackup status failed")
			return err
		}
	}

	logger.V(4).Infof("Successfully processed backup %s", key)
	return syncErr
}

func (ctrl *VMBackupController) sync(backup *backupv1.VirtualMachineBackup) error {
	if backup.Status == nil {
		backup.Status = &backupv1.VirtualMachineBackupStatus{}
	}
	backupDeleting := isBackupDeleting(backup)

	if IsBackupTerminal(backup) {
		if err := ctrl.releaseSourceVM(backup); err != nil {
			return err
		}
		if !backupDeleting {
			log.Log.Object(backup).V(4).Info("Backup is already done, skipping reconciliation")
			return nil
		}
		return ctrl.removeBackupFinalizer(backup)
	}

	backupTracker, err := ctrl.getBackupTracker(backup)
	if err != nil {
		return err
	}
	if backupTracker == nil && backup.Spec.Source.Kind == backupv1.VirtualMachineBackupTrackerGroupVersionKind.Kind {
		setInitializing(backup, fmt.Sprintf(backupTrackerNotFoundMsg, backup.Spec.Source.Name))
		return nil
	}

	sourceName := getSourceName(backup, backupTracker)
	if sourceName == "" {
		log.Log.Object(backup).Errorf("source name is empty")
		return errSourceNameEmpty
	}

	vmi, vmiExists, err := ctrl.vmiFromSource(backup, sourceName)
	if err != nil {
		return err
	}

	if vm, held, err := ctrl.heldSourceVM(backup, sourceName); err != nil {
		return err
	} else if held {
		return ctrl.reconcileOffline(backup, vm, backupTracker, backupDeleting)
	}

	backupStatus := getBackupStatus(vmi)

	switch {
	case backupStatus != nil && backupStatus.Completed:
		return ctrl.reconcileCompleted(backup, vmi, backupTracker, backupStatus)
	case backupStatus != nil:
		return ctrl.reconcileActive(backup, vmi, backupStatus, backupDeleting)
	default:
		return ctrl.reconcileStart(backup, vmi, vmiExists, backupTracker, backupDeleting, sourceName)
	}
}

func getBackupStatus(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstanceBackupStatus {
	if vmi == nil || vmi.Status.ChangedBlockTracking == nil || vmi.Status.ChangedBlockTracking.BackupStatus == nil {
		return nil
	}
	return vmi.Status.ChangedBlockTracking.BackupStatus
}

func (ctrl *VMBackupController) reconcileStart(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance, vmiExists bool, backupTracker *backupv1.VirtualMachineBackupTracker, backupDeleting bool, sourceName string) error {
	if backupDeleting {
		log.Log.Object(backup).V(3).Infof("Backup deleting during initialization")
		if !vmiExists {
			return ctrl.removeBackupFinalizer(backup)
		}
		done, err := ctrl.cleanupVMIState(backup, vmi)
		if err != nil {
			return err
		}
		if !done {
			return fmt.Errorf("ongoing cleanup for backup deletion: %w", errCleanupPending)
		}
		ctrl.setFailed(backup, backupv1.ReasonDeletedDuringInit, "backup was deleted during initialization")
		return nil
	}

	if backup.Status.StartTimestamp != nil {
		if alreadyTerminal, err := ctrl.isBackupAlreadyTerminalOnServer(backup); err != nil {
			return err
		} else if alreadyTerminal {
			// This reconcile is operating on a stale (pre-completion) local
			// snapshot of backup.Status: cleanupVMIState() clears the VMI's
			// BackupStatus as one of the last steps of a normal completion,
			// which re-triggers a reconcile via the VMI informer before this
			// backup's own informer cache has necessarily observed the
			// terminal status we just wrote (UpdateStatus only updates the
			// API server; the local cache updates asynchronously via watch).
			// The live object already reflects the real outcome -- nothing to
			// do here.
			log.Log.Object(backup).V(3).Infof("Skipping stale reconcile: backup is already terminal on the server")
			return nil
		}

		if !vmiExists {
			if isPullMode(backup) {
				if err := ctrl.cleanupBackupExport(backup); err != nil {
					return err
				}
			}
			ctrl.setFailed(backup, backupv1.ReasonSourceLost, "VMI was deleted during backup")
			return nil
		}
		log.Log.Object(backup).V(3).Infof("VMI backup status was lost while progressing")
		done, err := ctrl.cleanupVMIState(backup, vmi)
		if err != nil {
			return err
		}
		if !done {
			return fmt.Errorf("ongoing cleanup for lost backup status: %w", errCleanupPending)
		}
		ctrl.setFailed(backup, backupv1.ReasonSourceLost, "VMI backup status was lost")
		return nil
	}

	reason, err := ctrl.checkPrerequisites(backup, vmi, vmiExists, backupTracker, sourceName)
	if reason != "" {
		setInitializing(backup, reason)
	}
	if err != nil {
		return err
	}
	if reason != "" {
		return nil
	}

	return ctrl.startBackup(backup, vmi, backupTracker)
}

func (ctrl *VMBackupController) checkPrerequisites(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance, vmiExists bool, backupTracker *backupv1.VirtualMachineBackupTracker, sourceName string) (string, error) {
	sourceExists, err := ctrl.sourceVMExists(backup, sourceName)
	if err != nil {
		return "", err
	}
	if !sourceExists {
		return fmt.Sprintf(vmNotFoundMsg, backup.Namespace, sourceName), nil
	}
	if !vmiExists {
		if !ctrl.clusterConfig.OfflineIncrementalBackupEnabled() {
			return fmt.Sprintf(vmNotRunningMsg, sourceName), nil
		}
		// Add the finalizers before stamping the hold: the backup so a backup deleted in this window
		// still has an object left to release the VM, and the tracker so it can't be garbage-collected
		// while it holds the VM, which would otherwise leave the VM held with no way to resolve it.
		if err := ctrl.addBackupFinalizer(backup); err != nil {
			return "", err
		}
		if backupTracker != nil {
			if err := ctrl.addTrackerFinalizer(backupTracker); err != nil {
				return "", err
			}
		}
		held, err := ctrl.holdSourceVM(backup, sourceName)
		if err != nil {
			return "", err
		}
		if held {
			return fmt.Sprintf(vmHeldForOfflineBackupMsg, sourceName), nil
		}
		// The VM is held by another operation (backup, snapshot or restore) or set to start. No VM
		// event handler re-drives this backup, so poll until the VM is free to hold.
		return fmt.Sprintf(vmNotRunningMsg, sourceName), errOfflineWaiting
	}
	if reason := ctrl.verifyVMIEligibleForBackup(vmi); reason != "" {
		return reason, nil
	}
	if isTrackerDeleting(backupTracker) {
		return fmt.Sprintf(trackerDeletingMsg, backupTracker.Name), nil
	}
	if migrations.IsMigrating(vmi) {
		return fmt.Sprintf(vmMigrationInProgressMsg, vmi.Name), nil
	}
	return "", nil
}

func (ctrl *VMBackupController) reconcileActive(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance, backupStatus *v1.VirtualMachineInstanceBackupStatus, backupDeleting bool) error {
	if !vmi.IsRunning() || vmi.DeletionTimestamp != nil {
		done, err := ctrl.cleanupVMIState(backup, vmi)
		if err != nil {
			return err
		}
		if !done {
			return fmt.Errorf("not done cleaning backup for failed VMI %s: %w", vmi.Name, errCleanupPending)
		}
		ctrl.setFailed(backup, backupv1.ReasonSourceUnhealthy, fmt.Sprintf("VMI is not in a running state: %s", vmi.Status.Phase))
		return nil
	}

	if backupDeleting {
		abortReason := backupv1.BackupAbortCancel
		if shouldFinalizePullBackupOnDelete(backup) {
			// Pull-mode deletion is the client completion signal. Finalize when
			// DeletionTimestamp is before the TTL deadline; otherwise Cancel.
			abortReason = backupv1.BackupAbortFinalize
		}
		if err := ctrl.handleAbort(backup, vmi, abortReason); err != nil {
			return err
		}
		ctrl.setAborting(backup, backupAborting, abortReason)
	} else if isPullMode(backup) {
		if err := ctrl.handlePullMode(backup, vmi); err != nil {
			return err
		}
	}

	if len(backupStatus.Volumes) > 0 && len(backup.Status.IncludedVolumes) == 0 {
		backup.Status.IncludedVolumes = toBackupVolumeInfo(backupStatus.Volumes)
		backup.Status.CheckpointName = backupStatus.CheckpointName
	}

	if backupStatus.QuiesceStatus != "" {
		ctrl.setQuiescedCondition(backup, backupStatus.QuiesceStatus)
	}

	return nil
}

func (ctrl *VMBackupController) reconcileCompleted(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance, backupTracker *backupv1.VirtualMachineBackupTracker, backupStatus *v1.VirtualMachineInstanceBackupStatus) error {
	if backupTracker != nil && backupStatus.CheckpointName != nil && !backupStatus.Failed {
		if err := ctrl.updateBackupTracker(backup.Namespace, backupTracker, backupStatus); err != nil {
			log.Log.Object(backup).Reason(err).Error("Failed to update BackupTracker")
			return err
		}
	}

	log.Log.Object(backup).Info("Backup completed, performing cleanup")
	done, err := ctrl.cleanupVMIState(backup, vmi)
	if err != nil {
		return err
	}
	if !done {
		return fmt.Errorf("cleanup not complete for finished backup: %w", errCleanupPending)
	}

	ctrl.resolveCompletion(backup, backupStatus)

	if backupStatus.Failed {
		// Active reconcile may have copied CheckpointName once volumes appeared;
		// a failed/canceled backup must not retain it as a usable checkpoint.
		backup.Status.CheckpointName = nil
	} else if backupTracker != nil {
		backup.Status.CheckpointName = backupStatus.CheckpointName
	}
	backup.Status.IncludedVolumes = toBackupVolumeInfo(backupStatus.Volumes)

	if backupStatus.QuiesceStatus != "" {
		ctrl.setQuiescedCondition(backup, backupStatus.QuiesceStatus)
	}

	return nil
}

func (ctrl *VMBackupController) startBackup(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance, backupTracker *backupv1.VirtualMachineBackupTracker) error {
	if err := ctrl.addBackupFinalizer(backup); err != nil {
		return err
	}

	if backupTracker != nil {
		if err := ctrl.addTrackerFinalizer(backupTracker); err != nil {
			return err
		}
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
	case backupv1.PushMode, backupv1.PullMode:
		reason, err := ctrl.verifyBackupTargetPVC(backup.Spec.PvcName, backup.Namespace)
		if err != nil {
			return err
		}
		if reason != "" {
			setInitializing(backup, reason)
			return nil
		}

		volumeName := backupTargetVolumeName(backup.Name)
		if !ctrl.backupTargetPVCAttached(vmi, volumeName) {
			return ctrl.attachBackupTargetPVC(vmi, *backup.Spec.PvcName, volumeName)
		}
		backupOptions.Mode = *backup.Spec.Mode
		backupOptions.TargetPath = pointer.P(hotplugdisk.GetVolumeMountDir(volumeName))
	default:
		log.Log.Object(backup).Errorf(invalidBackupModeMsg, *backup.Spec.Mode)
		return fmt.Errorf(invalidBackupModeMsg, *backup.Spec.Mode)
	}

	log.Log.Object(backup).Infof("Starting backup for VMI %s with mode %s", vmi.Name, backupOptions.Mode)
	if isIncrementalBackup(backup, backupTracker) {
		backupOptions.Incremental = pointer.P(backupTracker.Status.LatestCheckpoint.Name)
		backupOptions.IncrementalCreationTime = backupTracker.Status.LatestCheckpoint.CreationTime
		log.Log.Object(backup).Infof("Setting incremental backup from checkpoint: %s", backupTracker.Status.LatestCheckpoint.Name)
	}

	if err := ctrl.client.VirtualMachineInstance(vmi.Namespace).Backup(context.Background(), vmi.Name, &backupOptions); err != nil {
		return fmt.Errorf("failed to send Start backup command: %w", err)
	}
	log.Log.Object(backup).Infof("Started backup for VMI %s successfully", vmi.Name)

	if err := ctrl.updateSourceBackupInProgress(vmi, backup.Name, backup.CreationTimestamp); err != nil {
		return fmt.Errorf("failed to update source backup in progress: %w", err)
	}

	setProgressing(backup)
	backup.Status.StartTimestamp = new(metav1.Now())
	return nil
}

func (ctrl *VMBackupController) handleAbort(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance, abortReason backupv1.BackupAbortReason) error {
	backupOptions := &backupv1.BackupOptions{
		BackupName:      backup.Name,
		Cmd:             backupv1.Abort,
		BackupStartTime: &backup.CreationTimestamp,
		AbortReason:     abortReason,
	}

	if err := ctrl.client.VirtualMachineInstance(vmi.Namespace).Backup(context.Background(), vmi.Name, backupOptions); err != nil {
		return err
	}
	return nil
}

func generateFinalizerPatch(test, replace []string) ([]byte, error) {
	return patch.New(
		patch.WithTest("/metadata/finalizers", test),
		patch.WithReplace("/metadata/finalizers", replace),
	).GeneratePayload()
}

func (ctrl *VMBackupController) addBackupFinalizer(backup *backupv1.VirtualMachineBackup) error {
	if controller.HasFinalizer(backup, backupv1.VirtualMachineBackupFinalizer) {
		return nil
	}

	cpy := backup.DeepCopy()
	controller.AddFinalizer(cpy, backupv1.VirtualMachineBackupFinalizer)

	patchBytes, err := generateFinalizerPatch(backup.Finalizers, cpy.Finalizers)
	if err != nil {
		return err
	}

	patched, err := ctrl.client.VirtualMachineBackup(cpy.Namespace).Patch(context.Background(), cpy.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}
	patched.ObjectMeta.DeepCopyInto(&backup.ObjectMeta)
	return nil
}

func (ctrl *VMBackupController) removeBackupFinalizer(backup *backupv1.VirtualMachineBackup) error {
	if !controller.HasFinalizer(backup, backupv1.VirtualMachineBackupFinalizer) {
		return nil
	}

	cpy := backup.DeepCopy()
	controller.RemoveFinalizer(cpy, backupv1.VirtualMachineBackupFinalizer)

	patchBytes, err := generateFinalizerPatch(backup.Finalizers, cpy.Finalizers)
	if err != nil {
		return fmt.Errorf("failed to generate finalizer patch: %w", err)
	}

	_, err = ctrl.client.VirtualMachineBackup(cpy.Namespace).Patch(context.Background(), cpy.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch backup to remove finalizer: %w", err)
	}
	return nil
}

func (ctrl *VMBackupController) addTrackerFinalizer(tracker *backupv1.VirtualMachineBackupTracker) error {
	if !tracker.DeletionTimestamp.IsZero() {
		return nil
	}
	if controller.HasFinalizer(tracker, backupv1.VirtualMachineBackupTrackerFinalizer) {
		return nil
	}

	cpy := tracker.DeepCopy()
	controller.AddFinalizer(cpy, backupv1.VirtualMachineBackupTrackerFinalizer)

	patchBytes, err := generateFinalizerPatch(tracker.Finalizers, cpy.Finalizers)
	if err != nil {
		return fmt.Errorf("failed to generate tracker finalizer patch: %w", err)
	}

	_, err = ctrl.client.VirtualMachineBackupTracker(cpy.Namespace).Patch(context.Background(), cpy.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to add tracker finalizer: %w", err)
	}
	return nil
}

func (ctrl *VMBackupController) removeTrackerFinalizer(tracker *backupv1.VirtualMachineBackupTracker) error {
	if !controller.HasFinalizer(tracker, backupv1.VirtualMachineBackupTrackerFinalizer) {
		return nil
	}

	cpy := tracker.DeepCopy()
	controller.RemoveFinalizer(cpy, backupv1.VirtualMachineBackupTrackerFinalizer)

	patchBytes, err := generateFinalizerPatch(tracker.Finalizers, cpy.Finalizers)
	if err != nil {
		return fmt.Errorf("failed to generate tracker finalizer patch: %w", err)
	}

	_, err = ctrl.client.VirtualMachineBackupTracker(cpy.Namespace).Patch(context.Background(), cpy.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove tracker finalizer: %w", err)
	}
	return nil
}

func getSourceName(backup *backupv1.VirtualMachineBackup, backupTracker *backupv1.VirtualMachineBackupTracker) string {
	if backupTracker != nil {
		return backupTracker.Spec.Source.Name
	}
	return backup.Spec.Source.Name
}

// BackupSourceVMName returns the name of the VM a backup targets, directly or through its
// BackupTracker, or "" while that tracker is not found.
func BackupSourceVMName(trackerStore cache.Store, backup *backupv1.VirtualMachineBackup) (string, error) {
	tracker, err := backupTrackerFromStore(trackerStore, backup)
	if err != nil {
		return "", err
	}
	if tracker == nil && backup.Spec.Source.Kind == backupv1.VirtualMachineBackupTrackerGroupVersionKind.Kind {
		return "", nil
	}
	return getSourceName(backup, tracker), nil
}

func (ctrl *VMBackupController) getBackupTracker(backup *backupv1.VirtualMachineBackup) (*backupv1.VirtualMachineBackupTracker, error) {
	return backupTrackerFromStore(ctrl.backupTrackerInformer.GetStore(), backup)
}

func backupTrackerFromStore(trackerStore cache.Store, backup *backupv1.VirtualMachineBackup) (*backupv1.VirtualMachineBackupTracker, error) {
	if backup.Spec.Source.Kind != backupv1.VirtualMachineBackupTrackerGroupVersionKind.Kind {
		return nil, nil
	}

	objKey := types.NamespacedName{Namespace: backup.Namespace, Name: backup.Spec.Source.Name}.String()
	obj, exists, err := trackerStore.GetByKey(objKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get BackupTracker from store: %w", err)
	}
	if !exists {
		return nil, nil
	}

	tracker, ok := obj.(*backupv1.VirtualMachineBackupTracker)
	if !ok {
		return nil, fmt.Errorf("unexpected object type in BackupTracker store: %T", obj)
	}

	return tracker, nil
}

func (ctrl *VMBackupController) getVMI(namespace, sourceName string) (*v1.VirtualMachineInstance, bool, error) {
	objKey := types.NamespacedName{Namespace: namespace, Name: sourceName}.String()

	obj, exists, err := ctrl.vmiStore.GetByKey(objKey)
	if err != nil {
		return nil, false, err
	}

	if !exists {
		return nil, false, nil
	}

	return obj.(*v1.VirtualMachineInstance), exists, nil
}

func (ctrl *VMBackupController) sourceVMExists(backup *backupv1.VirtualMachineBackup, sourceName string) (bool, error) {
	objKey := types.NamespacedName{Namespace: backup.Namespace, Name: sourceName}.String()
	_, exists, err := ctrl.vmStore.GetByKey(objKey)
	if err != nil {
		err = fmt.Errorf("failed to get VM from store: %w", err)
		log.Log.With("VirtualMachineBackup", backup.Name).Error(err.Error())
	}
	return exists, err
}

func (ctrl *VMBackupController) vmiFromSource(backup *backupv1.VirtualMachineBackup, sourceName string) (*v1.VirtualMachineInstance, bool, error) {
	vmi, exists, err := ctrl.getVMI(backup.Namespace, sourceName)
	if err != nil {
		err = fmt.Errorf("failed to get VMI from store: %w", err)
		log.Log.With("VirtualMachineBackup", backup.Name).Error(err.Error())
	}

	return vmi, exists, err
}

func (ctrl *VMBackupController) verifyVMIEligibleForBackup(vmi *v1.VirtualMachineInstance) string {
	hasEligibleVolumes := false
	for _, volume := range vmi.Spec.Volumes {
		if IsCBTEligibleVolume(&volume) {
			hasEligibleVolumes = true
			break
		}
	}
	if !hasEligibleVolumes {
		return fmt.Sprintf(vmNoVolumesToBackupMsg, vmi.Name)
	}
	if vmi.Status.ChangedBlockTracking == nil || vmi.Status.ChangedBlockTracking.State != v1.ChangedBlockTrackingEnabled {
		return fmt.Sprintf(vmNoChangedBlockTrackingMsg, vmi.Name)
	}
	return ""
}

func (ctrl *VMBackupController) removeSourceBackupInProgress(vmi *v1.VirtualMachineInstance) error {
	if !hasVMIBackupStatus(vmi) {
		return nil
	}

	patchBytes, err := patch.New(
		patch.WithRemove("/status/changedBlockTracking/backupStatus"),
	).GeneratePayload()
	if err != nil {
		return err
	}

	_, err = ctrl.client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove BackupInProgress from VMI %s/%s: %w", vmi.Namespace, vmi.Name, err)
	}

	return nil
}

func (ctrl *VMBackupController) updateSourceBackupInProgress(vmi *v1.VirtualMachineInstance, backupName string, creationTimestamp metav1.Time) error {
	if hasVMIBackupStatus(vmi) {
		if vmi.Status.ChangedBlockTracking.BackupStatus.BackupName != backupName {
			return fmt.Errorf("another backup %s is already in progress, cannot start backup %s",
				vmi.Status.ChangedBlockTracking.BackupStatus.BackupName, backupName)
		}
		return nil
	}

	var startTimestamp *metav1.Time
	if !creationTimestamp.IsZero() {
		startTimestamp = creationTimestamp.DeepCopy()
	}
	backupStatus := &v1.VirtualMachineInstanceBackupStatus{
		BackupName:     backupName,
		StartTimestamp: startTimestamp,
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

	_, err = ctrl.client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		log.Log.Errorf("Failed to update source backup in progress: %s", err)
		return err
	}

	return nil
}

func (ctrl *VMBackupController) resolveCompletion(backup *backupv1.VirtualMachineBackup, status *v1.VirtualMachineInstanceBackupStatus) {
	msgOrDefault := func(msg *string) string {
		if msg == nil {
			return "unknown, no completion message"
		}
		return *msg
	}

	if status.Failed {
		reason := msgOrDefault(status.BackupMsg)
		log.Log.Object(backup).Infof(backupFailed, reason)
		ctrl.setFailed(backup, backupv1.ReasonFailed, reason)
		return
	}

	if status.BackupMsg != nil {
		message := fmt.Sprintf(backupCompletedWithWarningMsg, *status.BackupMsg)
		log.Log.Object(backup).Info(message)
		setCompleteWithWarning(backup, message)
		ctrl.recorder.Event(backup, corev1.EventTypeWarning, backupCompletedWithWarningEvent, message)
		return
	}

	log.Log.Object(backup).Info(backupCompleted)
	setComplete(backup)
	ctrl.recorder.Eventf(backup, corev1.EventTypeNormal, backupCompletedEvent, backupCompleted)
}

func (ctrl *VMBackupController) updateBackupTracker(namespace string, tracker *backupv1.VirtualMachineBackupTracker, backupStatus *v1.VirtualMachineInstanceBackupStatus) error {
	if tracker == nil {
		return nil
	}

	newCheckpoint := backupv1.BackupCheckpoint{
		Name:         *backupStatus.CheckpointName,
		CreationTime: backupStatus.StartTimestamp,
	}

	return ctrl.patchTrackerCheckpoint(namespace, tracker, newCheckpoint)
}

func (ctrl *VMBackupController) patchTrackerCheckpoint(namespace string, tracker *backupv1.VirtualMachineBackupTracker, newCheckpoint backupv1.BackupCheckpoint) error {
	newStatus := &backupv1.VirtualMachineBackupTrackerStatus{
		LatestCheckpoint: &newCheckpoint,
	}

	patchSet := patch.New()
	if tracker.Status == nil || tracker.Status.LatestCheckpoint == nil || tracker.Status.LatestCheckpoint.Name == "" {
		patchSet.AddOption(patch.WithAdd("/status", newStatus))
	} else {
		patchSet.AddOption(patch.WithReplace("/status/latestCheckpoint", &newCheckpoint))
	}

	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return fmt.Errorf("failed to generate patch payload: %w", err)
	}

	_, err = ctrl.client.VirtualMachineBackupTracker(namespace).Patch(
		context.Background(),
		tracker.Name,
		types.JSONPatchType,
		patchBytes,
		metav1.PatchOptions{},
		"status",
	)
	if err != nil {
		return fmt.Errorf("failed to patch BackupTracker status: %w", err)
	}

	log.Log.Infof("Successfully updated BackupTracker %s/%s with checkpoint %s created at %s",
		namespace, tracker.Name, newCheckpoint.Name, newCheckpoint.CreationTime)

	return nil
}

func (ctrl *VMBackupController) cleanupVMIState(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) (bool, error) {
	if isPullMode(backup) {
		if err := ctrl.cleanupBackupExport(backup); err != nil {
			return false, err
		}
	}

	volumeName := backupTargetVolumeName(backup.Name)
	if !ctrl.backupTargetPVCDetached(vmi, volumeName) {
		if err := ctrl.detachBackupTargetPVC(vmi, volumeName); err != nil {
			return false, err
		}
		return false, nil
	}

	if err := ctrl.removeSourceBackupInProgress(vmi); err != nil {
		return false, err
	}

	return true, nil
}

func backupConditions(backup *backupv1.VirtualMachineBackup) []metav1.Condition {
	if backup != nil && backup.Status != nil {
		return backup.Status.Conditions
	}
	return nil
}

func isBackupComplete(backup *backupv1.VirtualMachineBackup) bool {
	return meta.IsStatusConditionTrue(backupConditions(backup), string(backupv1.ConditionComplete))
}

func isBackupFailed(backup *backupv1.VirtualMachineBackup) bool {
	return meta.IsStatusConditionTrue(backupConditions(backup), string(backupv1.ConditionFailed))
}

func IsBackupTerminal(backup *backupv1.VirtualMachineBackup) bool {
	return isBackupComplete(backup) || isBackupFailed(backup)
}

func isBackupExportReady(backup *backupv1.VirtualMachineBackup) bool {
	cond := meta.FindStatusCondition(backupConditions(backup), string(backupv1.ConditionProgressing))
	return cond != nil && cond.Reason == backupv1.ReasonExportReady
}

// isBackupAlreadyTerminalOnServer re-checks the backup directly against the
// API server (bypassing the informer cache) to guard against acting on a
// stale local snapshot: this backup's own informer cache can lag behind a
// terminal status this same controller just wrote, so a local-only check
// here is not sufficient to rule out a race.
func (ctrl *VMBackupController) isBackupAlreadyTerminalOnServer(backup *backupv1.VirtualMachineBackup) (bool, error) {
	liveBackup, err := ctrl.client.VirtualMachineBackup(backup.Namespace).Get(context.Background(), backup.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return IsBackupTerminal(liveBackup), nil
}

func isBackupDeleting(backup *backupv1.VirtualMachineBackup) bool {
	return backup != nil && backup.DeletionTimestamp != nil
}

func hasVMIBackupStatus(vmi *v1.VirtualMachineInstance) bool {
	return vmi != nil && vmi.Status.ChangedBlockTracking != nil && vmi.Status.ChangedBlockTracking.BackupStatus != nil
}

func setInitializing(backup *backupv1.VirtualMachineBackup, reason string) {
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type: string(backupv1.ConditionProgressing), Status: metav1.ConditionTrue,
		Reason: backupv1.ReasonInitializing, Message: reason,
	})
}

func setProgressing(backup *backupv1.VirtualMachineBackup) {
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type: string(backupv1.ConditionProgressing), Status: metav1.ConditionTrue,
		Reason: backupv1.ReasonInitiated, Message: backupInProgress,
	})
}

func (ctrl *VMBackupController) setAborting(backup *backupv1.VirtualMachineBackup, message string, abortReason backupv1.BackupAbortReason) {
	cond := meta.FindStatusCondition(backup.Status.Conditions, string(backupv1.ConditionProgressing))
	if cond == nil || cond.Reason != backupv1.ReasonAborting {
		eventSev := corev1.EventTypeWarning
		if abortReason == backupv1.BackupAbortFinalize {
			// Pull-mode finalize (delete after successful client pull) is expected.
			eventSev = corev1.EventTypeNormal
		}
		ctrl.recorder.Event(backup, eventSev, backupAbortingEvent, message)
	}
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type: string(backupv1.ConditionProgressing), Status: metav1.ConditionTrue,
		Reason: backupv1.ReasonAborting, Message: message,
	})
}

func (ctrl *VMBackupController) setFailed(backup *backupv1.VirtualMachineBackup, reason, message string) {
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type: string(backupv1.ConditionFailed), Status: metav1.ConditionTrue,
		Reason: reason, Message: fmt.Sprintf(backupFailed, message),
	})
	markTerminal(backup, reason, fmt.Sprintf(backupFailed, message))
	ctrl.recorder.Event(backup, corev1.EventTypeWarning, backupFailedEvent, message)
}

func markTerminal(backup *backupv1.VirtualMachineBackup, reason, message string) {
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type: string(backupv1.ConditionProgressing), Status: metav1.ConditionFalse,
		Reason: reason, Message: message,
	})
}

func setComplete(backup *backupv1.VirtualMachineBackup) {
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type: string(backupv1.ConditionComplete), Status: metav1.ConditionTrue,
		Reason: backupv1.ReasonCompleted, Message: backupCompleted,
	})
	markTerminal(backup, backupv1.ReasonCompleted, backupCompleted)
}

func setCompleteWithWarning(backup *backupv1.VirtualMachineBackup, message string) {
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type: string(backupv1.ConditionComplete), Status: metav1.ConditionTrue,
		Reason: backupv1.ReasonCompletedWithWarning, Message: message,
	})
	markTerminal(backup, backupv1.ReasonCompletedWithWarning, message)
}

func (ctrl *VMBackupController) setQuiescedCondition(backup *backupv1.VirtualMachineBackup, quiesceStatus string) {
	var status metav1.ConditionStatus
	var reason string
	var message string

	switch backupv1.QuiesceStatus(quiesceStatus) {
	case backupv1.QuiesceSucceeded:
		status = metav1.ConditionTrue
		reason = backupv1.ReasonQuiesceSucceeded
		message = "Guest filesystem was successfully quiesced"
	case backupv1.QuiesceFailed:
		status = metav1.ConditionFalse
		reason = backupv1.ReasonQuiesceFailed
		message = "Guest filesystem quiesce failed"
	case backupv1.QuiesceSkipped:
		status = metav1.ConditionFalse
		reason = backupv1.ReasonQuiesceSkipped
		message = "Guest filesystem quiesce was skipped"
	default:
		log.Log.Object(backup).Warningf("Invalid QuiesceStatus value: %q, skipping Quiesced condition", quiesceStatus)
		return
	}

	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type:    string(backupv1.ConditionQuiesced),
		Status:  status,
		Reason:  reason,
		Message: message,
	})
}

func toBackupVolumeInfo(vols []v1.VirtualMachineInstanceBackupVolumeInfo) []backupv1.BackupVolumeInfo {
	out := make([]backupv1.BackupVolumeInfo, len(vols))
	for i, v := range vols {
		out[i] = backupv1.BackupVolumeInfo{
			VolumeName: v.VolumeName,
			Type:       backupv1.BackupType(v.Type),
		}
	}
	return out
}
