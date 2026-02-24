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
	"crypto/ecdsa"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/certificate"
	"k8s.io/client-go/util/workqueue"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/controller"
	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/pointer"
	kvtls "kubevirt.io/kubevirt/pkg/util/tls"
)

const (
	vmBackupFinalizer = "backup.kubevirt.io/vmbackup-protection"

	backupInitializingEvent         = "VirtualMachineBackupInitializing"
	backupInitiatedEvent            = "VirtualMachineBackupInitiated"
	backupPreparingVMExportEvent    = "VirtualMachineBackupPreparingExport"
	backupExportInitiatedEvent      = "VirtualMachineBackupExportInitiated"
	backupExportReadyEvent          = "VirtualMachineBackupExportReady"
	backupAbortingEvent             = "VirtualMachineBackupAborting"
	backupCompletedEvent            = "VirtualMachineBackupCompletedSuccessfully"
	backupCompletedWithWarningEvent = "VirtualMachineBackupCompletedWithWarning"
	backupFailedEvent               = "VirtualMachineBackupFailed"

	backupInitializing                   = "Backup is initializing"
	backupInProgress                     = "Backup is in progress"
	backupPreparingVMExport              = "Backup export is being initialized"
	backupExportInitiated                = "Backup export has been initiated, waiting for export links"
	backupExportReady                    = "Backup export is ready to pull"
	backupAborting                       = "Backup is aborting"
	backupDeleting                       = "Backup is being deleted"
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
	backupSourceNameEmptyMsg             = "Source name is empty"

	failedExportCreate             = "failed to create backup export: %w"
	backupTTLExpiredMsg            = "pull mode backup TTL has expired"
	exportExistsWithDifferentOwner = "VMExport %s already exists but is not owned by backup %s"
	defaultPullModeDurationTTL     = 2 * time.Hour

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

	key := cacheKeyFunc(tracker.Namespace, tracker.Name)

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
			key = controller.NamespacedKey(obj.GetNamespace(), ownerRef.Name)
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

type SyncInfo struct {
	err             error
	reason          string
	event           string
	checkpointName  *string
	backupType      backupv1.BackupType
	includedVolumes []backupv1.BackupVolumeInfo
	caCert          *string
}

func syncInfoError(err error) *SyncInfo {
	return &SyncInfo{err: err}
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
	backupDeleting := isBackupDeleting(backup)
	// If backup is done and not being deleted, nothing to do
	if IsBackupDone(backup.Status) {
		if !backupDeleting {
			logger.V(4).Info("Backup is already done, skipping reconciliation")
			return nil
		}
		return ctrl.removeBackupFinalizer(backup)
	}

	backupTracker, syncInfo := ctrl.getBackupTracker(backup)
	if syncInfo != nil {
		return syncInfo
	}

	sourceName := getSourceName(backup, backupTracker)
	if sourceName == "" {
		logger.Errorf(backupSourceNameEmptyMsg)
		return syncInfoError(errSourceNameEmpty)
	}

	sourceExists, err := ctrl.sourceVMExists(backup, sourceName)
	if err != nil {
		return syncInfoError(err)
	}
	vmi, vmiExists, err := ctrl.vmiFromSource(backup, sourceName)
	if err != nil {
		return syncInfoError(err)
	}

	if isBackupInitializing(backup.Status) {
		if backupDeleting {
			logger.V(3).Infof("Backup deleting during initialization")
			if !vmiExists {
				return ctrl.removeBackupFinalizer(backup)
			}
			done, syncInfo := ctrl.cleanup(backup, vmi)
			if syncInfo != nil {
				return syncInfo
			}
			if !done {
				return syncInfoError(fmt.Errorf("ongoing cleanup for backup deletion"))
			}
			return &SyncInfo{
				event:  backupFailedEvent,
				reason: fmt.Sprintf(backupFailed, "backup was deleted during initialization"),
			}
		}

		if !sourceExists {
			return &SyncInfo{
				event:  backupInitializingEvent,
				reason: fmt.Sprintf(vmNotFoundMsg, backup.Namespace, sourceName),
			}
		}
		if !vmiExists {
			return &SyncInfo{
				event:  backupInitializingEvent,
				reason: fmt.Sprintf(vmNotRunningMsg, sourceName),
			}
		}
		if syncInfo := ctrl.verifyVMIEligibleForBackup(vmi, backup.Name); syncInfo != nil {
			return syncInfo
		}

		// If the tracker needs checkpoint redefinition, wait for it to complete.
		if trackerNeedsCheckpointRedefinition(backupTracker) {
			logger.Infof(trackerCheckpointRedefinitionPending, backupTracker.Name)
			return &SyncInfo{
				event:  backupInitializingEvent,
				reason: fmt.Sprintf(trackerCheckpointRedefinitionPending, backupTracker.Name),
			}
		}

		return ctrl.handleBackupInitiation(backup, vmi, backupTracker, logger)
	}

	if isBackupProgressing(backup.Status) {
		if !vmiExists {
			return &SyncInfo{
				event:  backupFailedEvent,
				reason: fmt.Sprintf(backupFailed, "VMI was deleted during backup"),
			}
		}

		if !hasVMIBackupStatus(vmi) {
			logger.V(3).Infof("VMI backup status was lost while progressing")
			done, syncInfo := ctrl.cleanup(backup, vmi)
			if syncInfo != nil {
				return syncInfo
			}
			if !done {
				return syncInfoError(fmt.Errorf("ongoing cleanup for backup deletion"))
			}
			return &SyncInfo{
				event:  backupFailedEvent,
				reason: fmt.Sprintf(backupFailed, "VMI backup status was lost"),
			}
		}

		if syncInfo := ctrl.validateVMIHealth(backup, vmi); syncInfo != nil {
			return syncInfo
		}

		if backupDeleting {
			backupStatus := vmi.Status.ChangedBlockTracking.BackupStatus
			if !backupStatus.Completed && backupStatus.BackupName == backup.Name {
				if syncInfo := ctrl.handleAbort(backup, vmi); syncInfo != nil {
					return syncInfo
				}
			}
		} else {
			if isPullMode(backup) {
				if isPullBackupTTLExpired(backup) {
					syncInfo := ctrl.handlePullModeTTLExpiry(backup, vmi)
					if syncInfo != nil {
						return syncInfo
					}
				} else {
					if !isBackupExportInitialized(backup.Status) {
						syncInfo := ctrl.handlePrepareBackupExport(backup, vmi)
						if syncInfo != nil {
							return syncInfo
						}
					}
					if !isBackupExportReady(backup.Status) {
						syncInfo := ctrl.waitForBackupExportReady(backup, vmi)
						if syncInfo != nil {
							return syncInfo
						}
					}
					if isBackupExportReady(backup.Status) {
						syncInfo := ctrl.validateExportHealth(backup)
						if syncInfo != nil {
							return syncInfo
						}
					}
				}
			}
		}
	}

	return ctrl.checkBackupCompletion(backup, vmi, backupTracker)
}

func (ctrl *VMBackupController) handleBackupInitiation(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance, backupTracker *backupv1.VirtualMachineBackupTracker, logger *log.FilteredLogger) *SyncInfo {
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
	case backupv1.PushMode, backupv1.PullMode:
		pvcName := backup.Spec.PvcName
		syncInfo := ctrl.verifyBackupTargetPVC(pvcName, backup.Namespace)
		if syncInfo != nil {
			return syncInfo
		}

		volumeName := backupTargetVolumeName(backup.Name)
		attached := ctrl.backupTargetPVCAttached(vmi, volumeName)
		if !attached {
			return ctrl.attachBackupTargetPVC(vmi, *pvcName, volumeName)
		}
		backupOptions.Mode = *backup.Spec.Mode
		backupOptions.TargetPath = pointer.P(hotplugdisk.GetVolumeMountDir(volumeName))
	default:
		logger.Errorf(invalidBackupModeMsg, *backup.Spec.Mode)
		return syncInfoError(fmt.Errorf(invalidBackupModeMsg, *backup.Spec.Mode))
	}

	logger.Infof("Starting backup for VMI %s with mode %s", vmi.Name, backupOptions.Mode)
	backupType := backupv1.Full
	if isIncrementalBackup(backup, backupTracker) {
		backupOptions.Incremental = pointer.P(backupTracker.Status.LatestCheckpoint.Name)
		backupType = backupv1.Incremental
		logger.Infof("Setting incremental backup from checkpoint: %s", backupTracker.Status.LatestCheckpoint.Name)
	}

	err = ctrl.client.VirtualMachineInstance(vmi.Namespace).Backup(context.Background(), vmi.Name, &backupOptions)
	if err != nil {
		err = fmt.Errorf("failed to send Start backup command: %w", err)
		logger.Error(err.Error())
		return syncInfoError(err)
	}
	logger.Infof("Started backup for VMI %s successfully", vmi.Name)

	return &SyncInfo{
		event:      backupInitiatedEvent,
		reason:     backupInProgress,
		backupType: backupType,
	}
}

func (ctrl *VMBackupController) handleAbort(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) *SyncInfo {
	if isBackupAborting(backup.Status) {
		return nil
	}

	backupOptions := &backupv1.BackupOptions{
		BackupName:      backup.Name,
		Cmd:             backupv1.Abort,
		BackupStartTime: &backup.CreationTimestamp,
	}

	if err := ctrl.client.VirtualMachineInstance(vmi.Namespace).Backup(context.Background(), vmi.Name, backupOptions); err != nil {
		return syncInfoError(err)
	}

	return &SyncInfo{
		event:  backupAbortingEvent,
		reason: backupAborting,
	}
}

func (ctrl *VMBackupController) validateVMIHealth(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) *SyncInfo {
	if !vmi.IsRunning() || vmi.DeletionTimestamp != nil {
		done, syncInfo := ctrl.cleanup(backup, vmi)
		if syncInfo != nil {
			return syncInfo
		}
		if !done {
			return syncInfoError(fmt.Errorf("not done cleaning backup for failed VMI: %s", vmi.Name))
		}
		return &SyncInfo{
			event:  backupFailedEvent,
			reason: fmt.Sprintf(backupFailed, "VMI is not in a running state"),
		}
	}
	return nil
}

func (ctrl *VMBackupController) handlePrepareBackupExport(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) *SyncInfo {
	syncInfo, vmExport := ctrl.getOrCreateBackupExport(vmi, backup)
	if syncInfo != nil {
		return syncInfo
	}
	ca, err := ctrl.exportCaManager.GetCurrentRaw()
	if err != nil {
		return syncInfoError(err)
	}
	keyPair, err := ctrl.generateBackupTunnelCert(backup)
	if err != nil {
		return syncInfoError(err)
	}
	exportAddr := fmt.Sprintf("virt-export-%s.%s.svc", vmExport.Name, vmExport.Namespace)
	serverName := fmt.Sprintf("%s.cluster.local", exportAddr)
	backupOptions := &backupv1.BackupOptions{
		BackupName:       backup.Name,
		Cmd:              backupv1.Export,
		BackupStartTime:  &backup.CreationTimestamp,
		Mode:             *backup.Spec.Mode,
		ExportServerAddr: &exportAddr,
		ExportServerName: &serverName,
		BackupKey:        cert.EncodePrivateKeyPEM(keyPair.Key),
		BackupCert:       cert.EncodeCertPEM(keyPair.Cert),
		CACert:           ca,
	}
	if err := ctrl.client.VirtualMachineInstance(vmi.Namespace).Backup(context.Background(), vmi.Name, backupOptions); err != nil {
		return syncInfoError(err)
	}
	return &SyncInfo{
		event:  backupExportInitiatedEvent,
		reason: backupExportInitiated,
	}
}

func (ctrl *VMBackupController) getOrCreateBackupExport(vmi *v1.VirtualMachineInstance, backup *backupv1.VirtualMachineBackup) (*SyncInfo, *exportv1.VirtualMachineExport) {
	objKey := cacheKeyFunc(backup.Namespace, backup.Name)
	obj, exists, err := ctrl.vmExportStore.GetByKey(objKey)
	if err != nil {
		err = fmt.Errorf("error getting VMExport from store: %w", err)
		log.Log.Error(err.Error())
		return syncInfoError(err), nil
	}
	if exists {
		vmExport := obj.(*exportv1.VirtualMachineExport)
		if !metav1.IsControlledBy(vmExport, backup) {
			return syncInfoError(fmt.Errorf(exportExistsWithDifferentOwner, vmExport.Name, backup.Name)), nil
		}
		return nil, vmExport
	}

	return ctrl.createBackupExport(backup, vmi), nil
}

func (ctrl *VMBackupController) createBackupExport(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) *SyncInfo {
	vmExport := &exportv1.VirtualMachineExport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backup.Name,
			Namespace: backup.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(backup, backupv1.SchemeGroupVersion.WithKind(backupv1.VirtualMachineBackupGroupVersionKind.Kind)),
			},
		},
		Spec: exportv1.VirtualMachineExportSpec{
			TokenSecretRef: &backup.Spec.TokenSecretRef,
			TTLDuration:    getPullBackupRemainingTTL(backup),
			Source: corev1.TypedLocalObjectReference{
				APIGroup: pointer.P(backupv1.VirtualMachineBackupGroupVersionKind.Group),
				Kind:     backupv1.VirtualMachineBackupGroupVersionKind.Kind,
				Name:     backup.Name,
			},
		},
	}

	_, err := ctrl.client.VirtualMachineExport(backup.Namespace).Create(context.Background(), vmExport, metav1.CreateOptions{})
	if err != nil {
		return syncInfoError(fmt.Errorf(failedExportCreate, err))
	}

	return &SyncInfo{
		event:  backupPreparingVMExportEvent,
		reason: backupPreparingVMExport,
	}
}

func (ctrl *VMBackupController) waitForBackupExportReady(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) *SyncInfo {
	objKey := cacheKeyFunc(backup.Namespace, backup.Name)
	obj, exists, err := ctrl.vmExportStore.GetByKey(objKey)
	if err != nil {
		err = fmt.Errorf("error getting VMExport from store: %w", err)
		log.Log.Error(err.Error())
		return syncInfoError(err)
	}
	if !exists {
		return syncInfoError(fmt.Errorf("associated export does not exist"))
	}
	vmExport := obj.(*exportv1.VirtualMachineExport)

	if vmExport.Status == nil || vmExport.Status.Phase != exportv1.Ready {
		return nil
	}

	links := vmExport.Status.Links
	hasInternalLinks := links != nil && links.Internal != nil && len(links.Internal.Backups) > 0
	hasExternalLinks := links != nil && links.External != nil && len(links.External.Backups) > 0

	if !hasInternalLinks && !hasExternalLinks {
		return syncInfoError(fmt.Errorf("associated export ready but has no backup links"))
	}

	iterableLinks := links.External
	if !hasExternalLinks {
		iterableLinks = links.Internal
	}

	if iterableLinks.Cert == "" {
		return syncInfoError(fmt.Errorf("associated export ready but has no cert exposed"))
	}

	syncInfo := &SyncInfo{
		event:  backupExportReadyEvent,
		reason: backupExportReady,
		caCert: &iterableLinks.Cert,
	}
	endpointMap := make(map[string][]exportv1.VirtualMachineExportBackupEndpoint, 0)
	for _, backupEndpoint := range iterableLinks.Backups {
		endpointMap[backupEndpoint.Name] = backupEndpoint.Endpoints
	}
	for _, volume := range backup.Status.IncludedVolumes {
		if endpoints, ok := endpointMap[volume.VolumeName]; ok {
			for _, link := range endpoints {
				switch link.Endpoint {
				case exportv1.Data:
					volume.DataEndpoint = link.Url
				case exportv1.Map:
					volume.MapEndpoint = link.Url
				}
			}
		}
		syncInfo.includedVolumes = append(syncInfo.includedVolumes, volume)
	}
	return syncInfo
}

func (ctrl *VMBackupController) validateExportHealth(backup *backupv1.VirtualMachineBackup) *SyncInfo {
	objKey := cacheKeyFunc(backup.Namespace, backup.Name)
	_, exists, err := ctrl.vmExportStore.GetByKey(objKey)
	if err != nil {
		return syncInfoError(fmt.Errorf("error getting VMExport from store: %w", err))
	}

	if exists {
		return nil
	}

	return &SyncInfo{
		event:  backupPreparingVMExportEvent,
		reason: backupPreparingVMExport,
	}
}

func (ctrl *VMBackupController) generateBackupTunnelCert(backup *backupv1.VirtualMachineBackup) (*triple.KeyPair, error) {
	caCert := ctrl.caCertManager.Current()
	caKeyPair := &triple.KeyPair{
		Key:  caCert.PrivateKey.(*ecdsa.PrivateKey),
		Cert: caCert.Leaf,
	}
	keyPair, err := triple.NewClientKeyPair(
		caKeyPair,
		fmt.Sprintf("kubevirt.io:system:client:%s", backup.UID),
		nil,
		getPullBackupRemainingTTL(backup).Duration,
	)
	return keyPair, err
}

func (ctrl *VMBackupController) handlePullModeTTLExpiry(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) *SyncInfo {
	if hasVMIBackupStatus(vmi) && !vmi.Status.ChangedBlockTracking.BackupStatus.Completed {
		if syncInfo := ctrl.handleAbort(backup, vmi); syncInfo != nil {
			if syncInfo.reason != "" {
				syncInfo.reason = fmt.Sprintf("%s: %s", backupTTLExpiredMsg, syncInfo.reason)
			}
			return syncInfo
		}
		return nil
	}
	return nil
}

func (ctrl *VMBackupController) cleanupBackupExport(backup *backupv1.VirtualMachineBackup) *SyncInfo {
	objKey := cacheKeyFunc(backup.Namespace, backup.Name)
	_, exists, err := ctrl.vmExportStore.GetByKey(objKey)
	if err != nil {
		return syncInfoError(fmt.Errorf("error getting VMExport from store during TTL expiry: %w", err))
	}
	if exists {
		if err := ctrl.client.VirtualMachineExport(backup.Namespace).Delete(context.Background(), backup.Name, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			return syncInfoError(fmt.Errorf("failed to delete VMExport during TTL expiry: %w", err))
		}
		return nil
	}
	return nil
}

func (ctrl *VMBackupController) updateStatus(backup *backupv1.VirtualMachineBackup, syncInfo *SyncInfo, logger *log.FilteredLogger) error {
	backupOut := backup.DeepCopy()

	if backup.Status == nil {
		backupOut.Status = &backupv1.VirtualMachineBackupStatus{}
		updateBackupCondition(backupOut, newInitializingCondition(corev1.ConditionTrue, backupInitializing))
		updateBackupCondition(backupOut, newProgressingCondition(corev1.ConditionFalse, backupInitializing))
	}

	if syncInfo != nil {
		switch syncInfo.event {
		case backupInitializingEvent:
			updateBackupCondition(backupOut, newInitializingCondition(corev1.ConditionTrue, syncInfo.reason))
			updateBackupCondition(backupOut, newProgressingCondition(corev1.ConditionFalse, syncInfo.reason))
		case backupInitiatedEvent:
			removeBackupCondition(backupOut, backupv1.ConditionInitializing)
			updateBackupCondition(backupOut, newProgressingCondition(corev1.ConditionTrue, syncInfo.reason))
			updateBackupCondition(backupOut, newDoneCondition(corev1.ConditionFalse, syncInfo.reason))
			if syncInfo.backupType != "" {
				backupOut.Status.Type = syncInfo.backupType
			}
		case backupPreparingVMExportEvent:
			updateBackupCondition(backupOut, newProgressingCondition(corev1.ConditionTrue, syncInfo.reason))
			updateBackupCondition(backupOut, newExportInitiatedCondition(corev1.ConditionFalse, syncInfo.reason))
			updateBackupCondition(backupOut, newExportReadyCondition(corev1.ConditionFalse, syncInfo.reason))
		case backupExportInitiatedEvent:
			updateBackupCondition(backupOut, newProgressingCondition(corev1.ConditionTrue, syncInfo.reason))
			updateBackupCondition(backupOut, newExportInitiatedCondition(corev1.ConditionTrue, syncInfo.reason))
			updateBackupCondition(backupOut, newExportReadyCondition(corev1.ConditionFalse, syncInfo.reason))
			updateBackupCondition(backupOut, newDoneCondition(corev1.ConditionFalse, syncInfo.reason))
		case backupExportReadyEvent:
			updateBackupCondition(backupOut, newProgressingCondition(corev1.ConditionTrue, syncInfo.reason))
			updateBackupCondition(backupOut, newExportInitiatedCondition(corev1.ConditionTrue, syncInfo.reason))
			updateBackupCondition(backupOut, newExportReadyCondition(corev1.ConditionTrue, syncInfo.reason))
			updateBackupCondition(backupOut, newDoneCondition(corev1.ConditionFalse, syncInfo.reason))
			if syncInfo.caCert != nil {
				backupOut.Status.EndpointCert = syncInfo.caCert
			}
		case backupAbortingEvent:
			updateBackupCondition(backupOut, newProgressingCondition(corev1.ConditionTrue, syncInfo.reason))
			updateBackupCondition(backupOut, newAbortingCondition(corev1.ConditionTrue, syncInfo.reason))
			eventSev := corev1.EventTypeNormal
			if isPushMode(backup) {
				eventSev = corev1.EventTypeWarning
			}
			ctrl.recorder.Eventf(backupOut, eventSev, backupAbortingEvent, syncInfo.reason)
		case backupCompletedEvent, backupCompletedWithWarningEvent, backupFailedEvent:
			switch syncInfo.event {
			case backupFailedEvent:
				ctrl.recorder.Eventf(backupOut, corev1.EventTypeWarning, backupFailedEvent, syncInfo.reason)
			case backupCompletedWithWarningEvent:
				ctrl.recorder.Eventf(backupOut, corev1.EventTypeWarning, backupCompletedWithWarningEvent, syncInfo.reason)
			case backupCompletedEvent:
				ctrl.recorder.Eventf(backupOut, corev1.EventTypeNormal, backupCompletedEvent, syncInfo.reason)
			}
			updateBackupCondition(backupOut, newProgressingCondition(corev1.ConditionFalse, syncInfo.reason))
			updateBackupCondition(backupOut, newDoneCondition(corev1.ConditionTrue, syncInfo.reason))
			if isBackupAborting(backup.Status) {
				updateBackupCondition(backupOut, newAbortingCondition(corev1.ConditionFalse, syncInfo.reason))
			}
		}
		if len(syncInfo.includedVolumes) > 0 {
			backupOut.Status.IncludedVolumes = syncInfo.includedVolumes
		}
		if syncInfo.checkpointName != nil {
			backupOut.Status.CheckpointName = syncInfo.checkpointName
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

func getSourceName(backup *backupv1.VirtualMachineBackup, backupTracker *backupv1.VirtualMachineBackupTracker) string {
	if backupTracker != nil {
		return backupTracker.Spec.Source.Name
	}
	return backup.Spec.Source.Name
}

func (ctrl *VMBackupController) getBackupTracker(backup *backupv1.VirtualMachineBackup) (*backupv1.VirtualMachineBackupTracker, *SyncInfo) {
	if backup.Spec.Source.Kind != backupv1.VirtualMachineBackupTrackerGroupVersionKind.Kind {
		return nil, nil
	}

	objKey := cacheKeyFunc(backup.Namespace, backup.Spec.Source.Name)
	obj, exists, err := ctrl.backupTrackerInformer.GetStore().GetByKey(objKey)
	if err != nil {
		log.Log.With("VirtualMachineBackup", backup.Name).Errorf("Failed to get BackupTracker from store: %v", err)
		return nil, syncInfoError(fmt.Errorf("failed to get BackupTracker from store: %w", err))
	}
	if !exists {
		trackerName := backup.Spec.Source.Name
		log.Log.With("VirtualMachineBackup", backup.Name).Infof(backupTrackerNotFoundMsg, trackerName)
		return nil, &SyncInfo{
			event:  backupInitializingEvent,
			reason: fmt.Sprintf(backupTrackerNotFoundMsg, trackerName),
		}
	}

	tracker, ok := obj.(*backupv1.VirtualMachineBackupTracker)
	if !ok {
		log.Log.With("VirtualMachineBackup", backup.Name).Errorf("Unexpected object type in BackupTracker store: %T", obj)
		return nil, syncInfoError(fmt.Errorf("unexpected object type in BackupTracker store: %T", obj))
	}

	return tracker, nil
}

func (ctrl *VMBackupController) getVMI(namespace, sourceName string) (*v1.VirtualMachineInstance, bool, error) {
	objKey := cacheKeyFunc(namespace, sourceName)

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
	objKey := cacheKeyFunc(backup.Namespace, sourceName)
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

func (ctrl *VMBackupController) verifyVMIEligibleForBackup(vmi *v1.VirtualMachineInstance, backupName string) *SyncInfo {
	hasEligibleVolumes := false
	for _, volume := range vmi.Spec.Volumes {
		if IsCBTEligibleVolume(&volume) {
			hasEligibleVolumes = true
			break
		}
	}
	if !hasEligibleVolumes {
		return &SyncInfo{
			event:  backupInitializingEvent,
			reason: fmt.Sprintf(vmNoVolumesToBackupMsg, vmi.Name),
		}
	}
	if vmi.Status.ChangedBlockTracking == nil || vmi.Status.ChangedBlockTracking.State != v1.ChangedBlockTrackingEnabled {
		log.Log.With("VirtualMachineBackup", backupName).Errorf(vmNoChangedBlockTrackingMsg, vmi.Name)
		return &SyncInfo{
			event:  backupInitializingEvent,
			reason: fmt.Sprintf(vmNoChangedBlockTrackingMsg, vmi.Name),
		}
	}
	return nil
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

func (ctrl *VMBackupController) checkBackupCompletion(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance, backupTracker *backupv1.VirtualMachineBackupTracker) *SyncInfo {
	if vmi == nil {
		return &SyncInfo{
			event:  backupFailedEvent,
			reason: fmt.Sprintf(backupFailed, "unexpected state: VMI is nil"),
		}
	}
	backupStatus := vmi.Status.ChangedBlockTracking.BackupStatus
	if !backupStatus.Completed {
		if len(backupStatus.Volumes) > 0 && len(backup.Status.IncludedVolumes) == 0 {
			return &SyncInfo{
				includedVolumes: backupStatus.Volumes,
				checkpointName:  backupStatus.CheckpointName,
			}
		}
		return nil
	}

	// Update BackupTracker with the new checkpoint if applicable
	if backupTracker != nil && backupStatus.CheckpointName != nil && !backupStatus.Failed {
		if err := ctrl.updateBackupTracker(backup.Namespace, backupTracker, backupStatus); err != nil {
			log.Log.Object(backup).Reason(err).Error("Failed to update BackupTracker")
			return syncInfoError(err)
		}
	}

	log.Log.Object(backup).Info("Backup completed, performing cleanup")
	done, syncInfo := ctrl.cleanup(backup, vmi)
	if syncInfo != nil {
		return syncInfo
	}
	if !done {
		return nil
	}

	syncInfo = resolveCompletion(backup, backupStatus)

	// We allow tracking checkpoints only if BackupTracker is specified
	if backupTracker != nil && !backupStatus.Failed {
		syncInfo.checkpointName = backupStatus.CheckpointName
	}
	syncInfo.includedVolumes = backupStatus.Volumes

	return syncInfo
}

func resolveCompletion(backup *backupv1.VirtualMachineBackup, status *v1.VirtualMachineInstanceBackupStatus) *SyncInfo {
	fmtReason := func(base string, msg *string) string {
		if msg == nil {
			return fmt.Sprintf(base, "unknown, no completion message")
		}
		return fmt.Sprintf(base, *msg)
	}

	if status.Failed {
		log.Log.Object(backup).Info(fmtReason(backupFailed, status.BackupMsg))
		return &SyncInfo{
			event:  backupFailedEvent,
			reason: fmtReason(backupFailed, status.BackupMsg),
		}
	}

	if status.BackupMsg != nil && isPushMode(backup) {
		log.Log.Object(backup).Infof(backupCompletedWithWarningMsg, *status.BackupMsg)
		return &SyncInfo{
			event:  backupCompletedWithWarningEvent,
			reason: fmtReason(backupCompletedWithWarningMsg, status.BackupMsg),
		}
	}

	log.Log.Object(backup).Info(backupCompleted)
	return &SyncInfo{
		event:  backupCompletedEvent,
		reason: backupCompleted,
	}
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
		k8stypes.JSONPatchType,
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

func isPushMode(backup *backupv1.VirtualMachineBackup) bool {
	return backup.Spec.Mode == nil || *backup.Spec.Mode == backupv1.PushMode
}

func isPullMode(backup *backupv1.VirtualMachineBackup) bool {
	return backup.Spec.Mode != nil && *backup.Spec.Mode == backupv1.PullMode
}

func (ctrl *VMBackupController) cleanup(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) (bool, *SyncInfo) {
	if isPullMode(backup) {
		syncInfo := ctrl.cleanupBackupExport(backup)
		if syncInfo != nil {
			return false, syncInfo
		}
	}

	volumeName := backupTargetVolumeName(backup.Name)
	detached := ctrl.backupTargetPVCDetached(vmi, volumeName)
	if !detached {
		event := backupInitializingEvent
		if isBackupProgressing(backup.Status) {
			event = backupInitiatedEvent
		}
		return false, ctrl.detachBackupTargetPVC(vmi, volumeName, event)
	}

	syncInfo := ctrl.removeSourceBackupInProgress(vmi)
	if syncInfo != nil {
		return false, syncInfo
	}

	return true, nil
}

func isBackupInitializing(status *backupv1.VirtualMachineBackupStatus) bool {
	return status == nil || hasCondition(status.Conditions, backupv1.ConditionInitializing)
}

func isBackupProgressing(status *backupv1.VirtualMachineBackupStatus) bool {
	return status != nil && hasCondition(status.Conditions, backupv1.ConditionProgressing)
}

func isBackupExportInitialized(status *backupv1.VirtualMachineBackupStatus) bool {
	return status != nil && hasCondition(status.Conditions, backupv1.ConditionExportInitiated)
}

func isBackupExportReady(status *backupv1.VirtualMachineBackupStatus) bool {
	return status != nil && hasCondition(status.Conditions, backupv1.ConditionExportReady)
}

func isBackupAborting(status *backupv1.VirtualMachineBackupStatus) bool {
	return status != nil && hasCondition(status.Conditions, backupv1.ConditionAborting)
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

func newExportInitiatedCondition(status corev1.ConditionStatus, reason string) backupv1.Condition {
	return newCondition(backupv1.ConditionExportInitiated, status, reason)
}

func newExportReadyCondition(status corev1.ConditionStatus, reason string) backupv1.Condition {
	return newCondition(backupv1.ConditionExportReady, status, reason)
}

func newAbortingCondition(status corev1.ConditionStatus, reason string) backupv1.Condition {
	return newCondition(backupv1.ConditionAborting, status, reason)
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

func getPullBackupTTL(backup *backupv1.VirtualMachineBackup) *metav1.Duration {
	ttl := &metav1.Duration{Duration: defaultPullModeDurationTTL}
	if backup.Spec.TTLDuration != nil {
		ttl = backup.Spec.TTLDuration
	}
	return ttl
}

func getPullBackupRemainingTTL(backup *backupv1.VirtualMachineBackup) *metav1.Duration {
	totalTTL := getPullBackupTTL(backup)
	creationTime := backup.CreationTimestamp.Time

	if creationTime.IsZero() {
		return totalTTL
	}

	elapsed := time.Since(creationTime)
	remaining := totalTTL.Duration - elapsed

	if remaining <= 0 {
		return &metav1.Duration{Duration: 0}
	}

	return &metav1.Duration{Duration: remaining}
}

func isPullBackupTTLExpired(backup *backupv1.VirtualMachineBackup) bool {
	ttl := getPullBackupTTL(backup)
	return time.Since(backup.CreationTimestamp.Time) >= ttl.Duration
}
