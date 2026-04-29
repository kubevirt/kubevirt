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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/certificate"
	"k8s.io/client-go/util/workqueue"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	"kubevirt.io/kubevirt/pkg/controller"
	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/pointer"
	migrations "kubevirt.io/kubevirt/pkg/util/migrations"
	kvtls "kubevirt.io/kubevirt/pkg/util/tls"
)

const (
	vmBackupFinalizer = "backup.kubevirt.io/vmbackup-protection"

	backupAbortingEvent             = "VirtualMachineBackupAborting"
	backupCompletedEvent            = "VirtualMachineBackupCompletedSuccessfully"
	backupCompletedWithWarningEvent = "VirtualMachineBackupCompletedWithWarning"
	backupFailedEvent               = "VirtualMachineBackupFailed"

	backupInProgress                     = "Backup is in progress"
	backupPreparingVMExport              = "Backup export is being initialized"
	backupExportInitiated                = "Backup export has been initiated, waiting for export links"
	backupExportReady                    = "Backup export is ready to pull"
	backupAborting                       = "Backup is aborting"
	backupCompleted                      = "Successfully completed VirtualMachineBackup"
	backupFailed                         = "Backup has failed: %s"
	backupCompletedWithWarningMsg        = "Completed VirtualMachineBackup, warning: %s"
	vmNotFoundMsg                        = "VM %s/%s doesnt exist"
	vmNotRunningMsg                      = "vm %s is not running, cannot do backup"
	vmNoVolumesToBackupMsg               = "vm %s has no volumes to backup"
	vmNoChangedBlockTrackingMsg          = "vm %s has no ChangedBlockTracking, cannot start backup"
	backupTrackerNotFoundMsg             = "BackupTracker %s does not exist"
	trackerCheckpointRedefinitionPending = "Waiting for checkpoint redefinition on tracker %s"
	invalidBackupModeMsg                 = "invalid backup mode: %s"
	vmMigrationInProgressMsg             = "vm %s is currently migrating, waiting for migration to complete before starting backup"

	caDefaultPath = "/etc/virt-controller/backupca"
	caCertFile    = caDefaultPath + "/tls.crt"
	caKeyFile     = caDefaultPath + "/tls.key"
)

var (
	errSourceNameEmpty = fmt.Errorf("source name is empty")
)

type VMBackupController struct {
	client                kubecli.KubevirtClient
	backupInformer        cache.SharedIndexInformer
	backupTrackerInformer cache.SharedIndexInformer
	vmStore               cache.Store
	vmiStore              cache.Store
	pvcStore              cache.Store
	vmExportStore         cache.Store
	recorder              record.EventRecorder
	backupQueue           workqueue.TypedRateLimitingInterface[string]
	trackerQueue          workqueue.TypedRateLimitingInterface[string]
	hasSynced             func() bool
	caCertManager         certificate.Manager
	exportCaManager       kvtls.ClientCAManager
}

func NewVMBackupController(client kubecli.KubevirtClient,
	backupInformer cache.SharedIndexInformer,
	backupTrackerInformer cache.SharedIndexInformer,
	vmInformer cache.SharedIndexInformer,
	vmiInformer cache.SharedIndexInformer,
	pvcInformer cache.SharedIndexInformer,
	vmExportInformer cache.SharedIndexInformer,
	cmInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	kubevirtNamespace string,
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
		recorder:              recorder,
		client:                client,
		exportCaManager:       kvtls.NewCAManager(cmInformer.GetStore(), kubevirtNamespace, "kubevirt-export-ca"),
	}

	initCert(c)

	c.hasSynced = func() bool {
		return backupInformer.HasSynced() && backupTrackerInformer.HasSynced() && vmInformer.HasSynced() && vmiInformer.HasSynced() && pvcInformer.HasSynced() && vmExportInformer.HasSynced()
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

	_, err = backupTrackerInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleBackupTracker,
			UpdateFunc: func(oldObj, newObj interface{}) { c.handleBackupTracker(newObj) },
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

var initCert = func(ctrl *VMBackupController) {
	ctrl.caCertManager = bootstrap.NewFileCertificateManager(caCertFile, caKeyFile)
	go ctrl.caCertManager.Start()
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
	key := types.NamespacedName{Namespace: nvmi.Namespace, Name: nvmi.Name}.String()

	// Find backups directly referencing this VMI
	keys, err := ctrl.backupInformer.GetIndexer().IndexKeys("vmi", key)
	if err != nil {
		return
	}

	for _, key := range keys {
		ctrl.backupQueue.Add(key)
	}

	// Find backups referencing this VMI via BackupTracker
	// First find all trackers that reference this VMI
	trackerKeys, err := ctrl.backupTrackerInformer.GetIndexer().IndexKeys("vmi", key)
	if err != nil {
		return
	}

	// For each tracker, find all backups that reference it
	for _, trackerKey := range trackerKeys {
		backupKeys, err := ctrl.backupInformer.GetIndexer().IndexKeys("backupTracker", trackerKey)
		if err != nil {
			continue
		}
		for _, backupKey := range backupKeys {
			ctrl.backupQueue.Add(backupKey)
		}
	}
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

	// Enqueue tracker for checkpoint redefinition if needed
	if trackerNeedsCheckpointRedefinition(tracker) {
		log.Log.V(3).Infof("enqueued tracker %q for checkpoint redefinition", key)
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
	var key string
	if ownerRef != nil {
		if ownerRef.Kind == backupv1.VirtualMachineBackupGroupVersionKind.Kind && ownerRef.APIVersion == backupv1.VirtualMachineBackupGroupVersionKind.GroupVersion().String() {
			key = types.NamespacedName{Namespace: obj.GetNamespace(), Name: ownerRef.Name}.String()
		}
	}
	return key
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
		log.Log.Reason(err).Infof("reenqueuing VirtualMachineBackup %v", key)
		ctrl.backupQueue.AddRateLimited(key)
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
	if backupCopy.Status == nil {
		backupCopy.Status = &backupv1.VirtualMachineBackupStatus{}
	}

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
	logger := log.Log.With("VirtualMachineBackup", backup.Name)
	backupDeleting := isBackupDeleting(backup)
	// If backup is done and not being deleted, nothing to do
	if IsBackupDone(backup) {
		if !backupDeleting {
			logger.V(4).Info("Backup is already done, skipping reconciliation")
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
		logger.Errorf("source name is empty")
		return errSourceNameEmpty
	}

	sourceExists, err := ctrl.sourceVMExists(backup, sourceName)
	if err != nil {
		return err
	}
	vmi, vmiExists, err := ctrl.vmiFromSource(backup, sourceName)
	if err != nil {
		return err
	}

	if isBackupInitializing(backup) {
		if backupDeleting {
			logger.V(3).Infof("Backup deleting during initialization")
			if !vmiExists {
				return ctrl.removeBackupFinalizer(backup)
			}
			done, err := ctrl.cleanup(backup, vmi)
			if err != nil {
				return err
			}
			if !done {
				return fmt.Errorf("ongoing cleanup for backup deletion")
			}
			ctrl.setFailed(backup, "backup was deleted during initialization")
			return nil
		}

		if !sourceExists {
			setInitializing(backup, fmt.Sprintf(vmNotFoundMsg, backup.Namespace, sourceName))
			return nil
		}
		if !vmiExists {
			setInitializing(backup, fmt.Sprintf(vmNotRunningMsg, sourceName))
			return nil
		}
		if reason := ctrl.verifyVMIEligibleForBackup(vmi); reason != "" {
			setInitializing(backup, reason)
			return nil
		}

		// If the tracker needs checkpoint redefinition, wait for it to complete.
		if trackerNeedsCheckpointRedefinition(backupTracker) {
			reason := fmt.Sprintf(trackerCheckpointRedefinitionPending, backupTracker.Name)
			logger.Info(reason)
			setInitializing(backup, reason)
			return nil
		}

		if migrations.IsMigrating(vmi) {
			reason := fmt.Sprintf(vmMigrationInProgressMsg, vmi.Name)
			logger.Info(reason)
			setInitializing(backup, reason)
			return nil
		}

		return ctrl.handleBackupInitiation(backup, vmi, backupTracker, logger)
	}

	if isBackupProgressing(backup) {
		if !vmiExists {
			ctrl.setFailed(backup, "VMI was deleted during backup")
			return nil
		}

		if !hasVMIBackupStatus(vmi) {
			logger.V(3).Infof("VMI backup status was lost while progressing")
			done, err := ctrl.cleanup(backup, vmi)
			if err != nil {
				return err
			}
			if !done {
				return fmt.Errorf("ongoing cleanup for backup deletion")
			}
			ctrl.setFailed(backup, "VMI backup status was lost")
			return nil
		}

		if err := ctrl.validateVMIHealth(backup, vmi); err != nil {
			return err
		}

		if backupDeleting {
			backupStatus := vmi.Status.ChangedBlockTracking.BackupStatus
			if !backupStatus.Completed && backupStatus.BackupName == backup.Name {
				if err := ctrl.handleAbort(backup, vmi); err != nil {
					return err
				}
			}
		} else if isPullMode(backup) {
			if err := ctrl.handlePullMode(backup, vmi); err != nil {
				return err
			}
		}
	}

	return ctrl.checkBackupCompletion(backup, vmi, backupTracker)
}

func (ctrl *VMBackupController) handleBackupInitiation(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance, backupTracker *backupv1.VirtualMachineBackupTracker, logger *log.FilteredLogger) error {
	if err := ctrl.addBackupFinalizer(backup); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	if err := ctrl.updateSourceBackupInProgress(vmi, backup.Name, backup.CreationTimestamp); err != nil {
		return fmt.Errorf("failed to update source backup in progress: %w", err)
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
		logger.Errorf(invalidBackupModeMsg, *backup.Spec.Mode)
		return fmt.Errorf(invalidBackupModeMsg, *backup.Spec.Mode)
	}

	logger.Infof("Starting backup for VMI %s with mode %s", vmi.Name, backupOptions.Mode)
	backupType := backupv1.Full
	if isIncrementalBackup(backup, backupTracker) {
		backupOptions.Incremental = pointer.P(backupTracker.Status.LatestCheckpoint.Name)
		backupType = backupv1.Incremental
		logger.Infof("Setting incremental backup from checkpoint: %s", backupTracker.Status.LatestCheckpoint.Name)
	}

	err := ctrl.client.VirtualMachineInstance(vmi.Namespace).Backup(context.Background(), vmi.Name, &backupOptions)
	if err != nil {
		return fmt.Errorf("failed to send Start backup command: %w", err)
	}
	logger.Infof("Started backup for VMI %s successfully", vmi.Name)

	setProgressing(backup)
	backup.Status.Type = backupType
	return nil
}

func (ctrl *VMBackupController) handleAbort(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) error {
	if isBackupAborting(backup) {
		return nil
	}

	backupOptions := &backupv1.BackupOptions{
		BackupName:      backup.Name,
		Cmd:             backupv1.Abort,
		BackupStartTime: &backup.CreationTimestamp,
	}

	if err := ctrl.client.VirtualMachineInstance(vmi.Namespace).Backup(context.Background(), vmi.Name, backupOptions); err != nil {
		return err
	}

	ctrl.setAborting(backup)
	return nil
}

func (ctrl *VMBackupController) validateVMIHealth(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) error {
	if !vmi.IsRunning() || vmi.DeletionTimestamp != nil {
		done, err := ctrl.cleanup(backup, vmi)
		if err != nil {
			return err
		}
		if !done {
			return fmt.Errorf("not done cleaning backup for failed VMI: %s", vmi.Name)
		}
		ctrl.setFailed(backup, "VMI is not in a running state")
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
	if controller.HasFinalizer(backup, vmBackupFinalizer) {
		return nil
	}

	cpy := backup.DeepCopy()
	controller.AddFinalizer(cpy, vmBackupFinalizer)

	patchBytes, err := generateFinalizerPatch(backup.Finalizers, cpy.Finalizers)
	if err != nil {
		return err
	}

	_, err = ctrl.client.VirtualMachineBackup(cpy.Namespace).Patch(context.Background(), cpy.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	return err
}

func (ctrl *VMBackupController) removeBackupFinalizer(backup *backupv1.VirtualMachineBackup) error {
	if !controller.HasFinalizer(backup, vmBackupFinalizer) {
		return nil
	}

	cpy := backup.DeepCopy()
	controller.RemoveFinalizer(cpy, vmBackupFinalizer)

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

func getSourceName(backup *backupv1.VirtualMachineBackup, backupTracker *backupv1.VirtualMachineBackupTracker) string {
	if backupTracker != nil {
		return backupTracker.Spec.Source.Name
	}
	return backup.Spec.Source.Name
}

func (ctrl *VMBackupController) getBackupTracker(backup *backupv1.VirtualMachineBackup) (*backupv1.VirtualMachineBackupTracker, error) {
	if backup.Spec.Source.Kind != backupv1.VirtualMachineBackupTrackerGroupVersionKind.Kind {
		return nil, nil
	}

	objKey := types.NamespacedName{Namespace: backup.Namespace, Name: backup.Spec.Source.Name}.String()
	obj, exists, err := ctrl.backupTrackerInformer.GetStore().GetByKey(objKey)
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

func (ctrl *VMBackupController) checkBackupCompletion(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance, backupTracker *backupv1.VirtualMachineBackupTracker) error {
	if vmi == nil {
		ctrl.setFailed(backup, "unexpected state: VMI is nil")
		return nil
	}
	backupStatus := vmi.Status.ChangedBlockTracking.BackupStatus
	if !backupStatus.Completed {
		if len(backupStatus.Volumes) > 0 && len(backup.Status.IncludedVolumes) == 0 {
			backup.Status.IncludedVolumes = backupStatus.Volumes
			backup.Status.CheckpointName = backupStatus.CheckpointName
		}
		return nil
	}

	// Update BackupTracker with the new checkpoint if applicable
	if backupTracker != nil && backupStatus.CheckpointName != nil && !backupStatus.Failed {
		if err := ctrl.updateBackupTracker(backup.Namespace, backupTracker, backupStatus); err != nil {
			log.Log.Object(backup).Reason(err).Error("Failed to update BackupTracker")
			return err
		}
	}

	log.Log.Object(backup).Info("Backup completed, performing cleanup")
	done, err := ctrl.cleanup(backup, vmi)
	if err != nil {
		return err
	}
	if !done {
		return nil
	}

	ctrl.resolveCompletion(backup, backupStatus)

	// We allow tracking checkpoints only if BackupTracker is specified
	if backupTracker != nil && !backupStatus.Failed {
		backup.Status.CheckpointName = backupStatus.CheckpointName
	}
	backup.Status.IncludedVolumes = backupStatus.Volumes

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
		ctrl.setFailed(backup, reason)
		return
	}

	if status.BackupMsg != nil {
		message := fmt.Sprintf(backupCompletedWithWarningMsg, *status.BackupMsg)
		log.Log.Object(backup).Info(message)
		setDoneWithWarning(backup, message)
		ctrl.recorder.Eventf(backup, corev1.EventTypeWarning, backupCompletedWithWarningEvent, message)
		return
	}

	log.Log.Object(backup).Info(backupCompleted)
	setDone(backup)
	ctrl.recorder.Eventf(backup, corev1.EventTypeNormal, backupCompletedEvent, backupCompleted)
}

func (ctrl *VMBackupController) updateBackupTracker(namespace string, tracker *backupv1.VirtualMachineBackupTracker, backupStatus *v1.VirtualMachineInstanceBackupStatus) error {
	if tracker == nil {
		return nil
	}

	newCheckpoint := backupv1.BackupCheckpoint{
		Name:         *backupStatus.CheckpointName,
		CreationTime: backupStatus.StartTimestamp,
		Volumes:      backupStatus.Volumes,
	}

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

	log.Log.Infof("Successfully updated BackupTracker %s/%s with checkpoint %s",
		namespace, tracker.Name, newCheckpoint.Name)
	log.Log.V(3).Infof("Checkpoint details: name=%s, creationTime=%s, volumes=%d",
		newCheckpoint.Name, newCheckpoint.CreationTime, len(newCheckpoint.Volumes))

	return nil
}

func (ctrl *VMBackupController) cleanup(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) (bool, error) {
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

func isBackupInitializing(backup *backupv1.VirtualMachineBackup) bool {
	conds := backupConditions(backup)
	return conds == nil || meta.IsStatusConditionTrue(conds, string(backupv1.ConditionInitializing))
}

func isBackupProgressing(backup *backupv1.VirtualMachineBackup) bool {
	return meta.IsStatusConditionTrue(backupConditions(backup), string(backupv1.ConditionProgressing))
}

func IsBackupDone(backup *backupv1.VirtualMachineBackup) bool {
	return meta.IsStatusConditionTrue(backupConditions(backup), string(backupv1.ConditionDone))
}

func isBackupAborting(backup *backupv1.VirtualMachineBackup) bool {
	return meta.IsStatusConditionTrue(backupConditions(backup), string(backupv1.ConditionAborting))
}

func isBackupDeleting(backup *backupv1.VirtualMachineBackup) bool {
	return backup != nil && backup.DeletionTimestamp != nil
}

func hasVMIBackupStatus(vmi *v1.VirtualMachineInstance) bool {
	return vmi != nil && vmi.Status.ChangedBlockTracking != nil && vmi.Status.ChangedBlockTracking.BackupStatus != nil
}

func setInitializing(backup *backupv1.VirtualMachineBackup, reason string) {
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type: string(backupv1.ConditionInitializing), Status: metav1.ConditionTrue,
		Reason: "Initializing", Message: reason,
	})
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type: string(backupv1.ConditionProgressing), Status: metav1.ConditionFalse,
		Reason: "Initializing", Message: reason,
	})
}

func setProgressing(backup *backupv1.VirtualMachineBackup) {
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type: string(backupv1.ConditionInitializing), Status: metav1.ConditionFalse,
		Reason: "Initiated", Message: backupInProgress,
	})
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type: string(backupv1.ConditionProgressing), Status: metav1.ConditionTrue,
		Reason: "Initiated", Message: backupInProgress,
	})
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type: string(backupv1.ConditionDone), Status: metav1.ConditionFalse,
		Reason: "Initiated", Message: backupInProgress,
	})
}

func (ctrl *VMBackupController) setAborting(backup *backupv1.VirtualMachineBackup) {
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type: string(backupv1.ConditionProgressing), Status: metav1.ConditionTrue,
		Reason: "Aborting", Message: backupAborting,
	})
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type: string(backupv1.ConditionAborting), Status: metav1.ConditionTrue,
		Reason: "Aborting", Message: backupAborting,
	})
	eventSev := corev1.EventTypeNormal
	if isPushMode(backup) {
		eventSev = corev1.EventTypeWarning
	}
	ctrl.recorder.Eventf(backup, eventSev, backupAbortingEvent, backupAborting)
}

func (ctrl *VMBackupController) setFailed(backup *backupv1.VirtualMachineBackup, reason string) {
	markDone(backup, "Failed", fmt.Sprintf(backupFailed, reason))
	ctrl.recorder.Eventf(backup, corev1.EventTypeWarning, backupFailedEvent, reason)
}

func markDone(backup *backupv1.VirtualMachineBackup, reason, message string) {
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type: string(backupv1.ConditionDone), Status: metav1.ConditionTrue,
		Reason: reason, Message: message,
	})
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type: string(backupv1.ConditionProgressing), Status: metav1.ConditionFalse,
		Reason: reason, Message: message,
	})
	if isBackupAborting(backup) {
		meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
			Type: string(backupv1.ConditionAborting), Status: metav1.ConditionFalse,
			Reason: reason, Message: message,
		})
	}
}

func setDone(backup *backupv1.VirtualMachineBackup) {
	markDone(backup, "Completed", backupCompleted)
}

func setDoneWithWarning(backup *backupv1.VirtualMachineBackup, message string) {
	markDone(backup, "CompletedWithWarning", message)
}
