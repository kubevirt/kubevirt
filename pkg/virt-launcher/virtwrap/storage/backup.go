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
 * Copyright 2025 Red Hat, Inc.
 *
 */

package storage

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"libvirt.org/go/libvirt"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	osdisk "kubevirt.io/kubevirt/pkg/os/disk"
	kutil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

const (
	ChangedBlockTrackingNotEnabledMsg = "Backup failed ChangedBlockTracking is not enabled"
	backupTimeXMLFormat               = "2006-01-02_15-04-05"
	freezeFailedMsg                   = "Failed freezing guest filesystem: %s"
	unfreezeFailedMsg                 = "Failed to unfreeze filesystem after backup completion"

	pullBackupSocketDir  = "/var/run/kubevirt/sockets"
	pullBackupSocketName = "backup-nbd-sock"
)

var getDiskInfoWithForceShare = osdisk.GetDiskInfoWithForceShare

func (m *StorageManager) BackupVirtualMachine(vmi *v1.VirtualMachineInstance, backupOptions *backupv1.BackupOptions) error {
	logger := log.Log.With("backupName", backupOptions.BackupName)
	logger.Info("Backup begin called")
	if m.MigrationInProgress() {
		return fmt.Errorf("failed to do backup, VMI is currently during migration")
	}
	inProgress, err := m.initializeBackupMetadata(backupOptions)
	if err != nil {
		logger.Reason(err).Warning("Failed to initialize backup metadata")
		return err
	}
	if inProgress {
		logger.Info("Backup already in progress")
		return nil
	}

	logger.Info("Initializing backup")
	err = m.backup(vmi, backupOptions)
	if err != nil {
		logger.Reason(err).Error("Backup failed to start")
		// Reset metadata cache so retries can proceed with fresh metadata
		m.metadataCache.Backup.Store(api.BackupMetadata{})
		return err
	}

	logger.Info("Backup started")
	return nil
}

func (m *StorageManager) initializeBackupMetadata(backupOptions *backupv1.BackupOptions) (bool, error) {
	backupMetadata, exists := m.metadataCache.Backup.Load()
	if exists && backupMetadata.Name != "" {
		sameBackup := backupMetadata.Name == backupOptions.BackupName &&
			backupMetadata.StartTimestamp != nil && backupOptions.BackupStartTime != nil &&
			backupMetadata.StartTimestamp.Equal(backupOptions.BackupStartTime)

		if sameBackup {
			if backupMetadata.EndTimestamp == nil {
				// backup is already in progress, ignore
				return true, nil
			} else {
				// backup already completed should not initialize the same backup again
				return false, fmt.Errorf("backup %s that started at %s already executed, finished at %v, completed: %t",
					backupOptions.BackupName, *backupMetadata.StartTimestamp, *backupMetadata.EndTimestamp, backupMetadata.Completed)
			}
		} else {
			if backupMetadata.EndTimestamp == nil {
				// another backup already exists and has not completed yet
				return false, fmt.Errorf("backup %s already in progress, need to wait for completion", backupMetadata.Name)
			}
			// Old backup has completed, allow new backup to overwrite the metadata
			log.Log.Infof("Previous backup %s completed at %v, initializing new backup %s",
				backupMetadata.Name, backupMetadata.EndTimestamp, backupOptions.BackupName)
		}
	}

	b := api.BackupMetadata{
		Name:           backupOptions.BackupName,
		StartTimestamp: backupOptions.BackupStartTime,
		SkipQuiesce:    backupOptions.SkipQuiesce,
		Mode:           string(backupOptions.Mode),
	}
	m.metadataCache.Backup.Store(b)
	log.Log.Infof("Initialized backup metadata: %v", b)

	return false, nil
}

func (m *StorageManager) backup(vmi *v1.VirtualMachineInstance, backupOptions *backupv1.BackupOptions) (failed error) {
	logger := log.Log.With("backupName", backupOptions.BackupName)
	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := m.virConn.LookupDomainByName(domName)
	if dom == nil || err != nil {
		return err
	}
	defer dom.Free()

	domainDisks, err := util.GetAllDomainDisks(dom)
	if err != nil {
		logger.Reason(err).Error("failed to parse domain XML to get disks.")
		return err
	}

	var backupPath string
	if backupOptions.TargetPath != nil {
		backupPath = getBackupPath(backupOptions, vmi.Name)
		if err := kutil.MkdirAllWithNosec(backupPath); err != nil {
			logger.Reason(err).Error("error creating dir for backup")
			return fmt.Errorf("error creating dir for backup: %w", err)
		}
		defer func(path string) {
			if failed != nil {
				logger.Reason(failed).Error("failed to run backup, cleaning up backup directory")
				if err := os.RemoveAll(path); err != nil {
					logger.Reason(err).Error("failed to clean up backup directory")
				}
			}
		}(backupPath)
	}
	domainBackup, domainCheckpoint, backupVolumesInfo := generateDomainBackup(domainDisks, backupOptions, backupPath)
	backupXML, err := xml.Marshal(domainBackup)
	if err != nil {
		logger.Reason(err).Error("marshalling backup xml failed")
		return err
	}
	checkpointXML, err := xml.Marshal(domainCheckpoint)
	if err != nil {
		logger.Reason(err).Error("marshalling checkpoint xml failed")
		return err
	}

	volumesJSON, err := json.Marshal(backupVolumesInfo)
	if err != nil {
		logger.Reason(err).Error("Failed to marshal backup volumes info")
		return err
	}
	m.metadataCache.Backup.WithSafeBlock(func(backupMetadata *api.BackupMetadata, _ bool) {
		backupMetadata.CheckpointName = domainCheckpoint.Name
		backupMetadata.Volumes = string(volumesJSON)
	})

	frozenFS := false
	if !backupOptions.SkipQuiesce {
		logger.Info("Freezing VMI to capture backup state")
		if err := dom.FSFreeze(nil, 0); err != nil {
			logger.Warningf(freezeFailedMsg, err)
			m.metadataCache.Backup.WithSafeBlock(func(backupMetadata *api.BackupMetadata, _ bool) {
				backupMetadata.BackupMsg = fmt.Sprintf(freezeFailedMsg, err)
			})
		} else {
			frozenFS = true
		}
	}

	defer func() {
		if frozenFS {
			logger.Info("Thawing VMI after backup job started")
			if err := dom.FSThaw(nil, 0); err != nil {
				logger.Reason(err).Error(unfreezeFailedMsg)
				m.metadataCache.Backup.WithSafeBlock(func(backupMetadata *api.BackupMetadata, _ bool) {
					backupMetadata.BackupMsg = unfreezeFailedMsg
				})
			}
		}
	}()

	return dom.BackupBegin(strings.ToLower(string(backupXML)), strings.ToLower(string(checkpointXML)), 0)
}

func generateDomainBackup(disks []api.Disk, backupOptions *backupv1.BackupOptions, backupPath string) (*api.DomainBackup, *api.DomainCheckpoint, []backupv1.BackupVolumeInfo) {
	domainBackup := &api.DomainBackup{
		Mode: string(backupOptions.Mode),
	}
	if isIncrementalBackup(backupOptions) {
		log.Log.Infof("Generating incremental backup %s from checkpoint: %s", backupOptions.BackupName, *backupOptions.Incremental)
		domainBackup.Incremental = backupOptions.Incremental
	}
	if backupOptions.Mode == backupv1.PullMode {
		domainBackup.Server = &api.DomainBackupServer{
			Transport: api.BackupUnixTransport,
			Socket:    filepath.Join(pullBackupSocketDir, pullBackupSocketName),
		}
	}
	backupTime := backupTimeFormatted(backupOptions.BackupStartTime)
	checkpointName := fmt.Sprintf("%s-%s", backupOptions.BackupName, backupTime)
	backupDisks := &api.BackupDisks{}
	checkpointDisks := &api.CheckpointDisks{}
	var backupVolumesInfo []backupv1.BackupVolumeInfo
	// the name of the volume should match the alias
	for _, disk := range disks {
		if disk.Target.Device == "" {
			continue
		}
		backupDisk := api.BackupDisk{
			Name: disk.Target.Device,
		}
		checkpointDisk := api.CheckpointDisk{
			Name: disk.Target.Device,
		}
		volumeName := converter.GetVolumeNameByDisk(disk)
		if disk.Source.DataStore != nil {
			backupDisk.Backup = "yes"
			backupDisk.Type = "file"
			if backupOptions.Mode == backupv1.PullMode {
				backupDisk.ExportName = volumeName
				backupDisk.ExportBitmap = checkpointName
			}
			if backupOptions.TargetPath != nil {
				setBackupDiskTargetPath(&backupDisk, backupOptions, volumeName, backupPath)
			}
			checkpointDisk.Checkpoint = "bitmap"
			backupVolumesInfo = append(backupVolumesInfo, backupv1.BackupVolumeInfo{
				VolumeName: volumeName,
				DiskTarget: disk.Target.Device,
			})
		} else {
			backupDisk.Backup = "no"
			checkpointDisk.Checkpoint = "no"
		}
		backupDisks.Disks = append(backupDisks.Disks, backupDisk)
		checkpointDisks.Disks = append(checkpointDisks.Disks, checkpointDisk)
	}

	domainBackup.BackupDisks = backupDisks
	domainCheckpoint := &api.DomainCheckpoint{
		Name:            checkpointName,
		CheckpointDisks: checkpointDisks,
	}
	return domainBackup, domainCheckpoint, backupVolumesInfo
}

func setBackupDiskTargetPath(backupDisk *api.BackupDisk, backupOptions *backupv1.BackupOptions, volumeName string, backupPath string) {
	targetFile := targetQCOW2File(backupPath, backupOptions.BackupName, volumeName)
	switch backupOptions.Mode {
	case backupv1.PushMode:
		backupDisk.Target = &api.BackupTarget{
			File: targetFile,
		}
	case backupv1.PullMode:
		backupDisk.Scratch = &api.BackupScratch{
			File: targetFile,
		}
	}
}

func getBackupPath(backupOptions *backupv1.BackupOptions, vmiName string) string {
	backupTime := backupTimeFormatted(backupOptions.BackupStartTime)
	backupNameWithTime := fmt.Sprintf("%s-%s", backupOptions.BackupName, backupTime)
	return filepath.Join(*backupOptions.TargetPath, vmiName, backupNameWithTime)
}

func targetQCOW2File(targetPath, backupName, volumeName string) string {
	fileName := fmt.Sprintf("%s-%s.qcow2", backupName, volumeName)
	return filepath.Join(targetPath, fileName)
}

func backupTimeFormatted(time *metav1.Time) string {
	return time.UTC().Format(backupTimeXMLFormat)
}

func isIncrementalBackup(backupOptions *backupv1.BackupOptions) bool {
	return backupOptions.Incremental != nil && *backupOptions.Incremental != ""
}

func HandleBackupJobCompletedEvent(domain cli.VirDomain, event *libvirt.DomainEventJobCompleted, metadataCache *metadata.Cache) {
	backupMetadata, exists := metadataCache.Backup.Load()
	if !exists {
		log.Log.Warning("Received backup job completed event, but no active backup metadata found in cache. Ignoring event.")
		return
	}
	backupName := backupMetadata.Name
	logger := log.Log.With("backupName", backupName)

	if domain != nil {
		finalStats, err := domain.GetJobStats(libvirt.DOMAIN_JOB_STATS_COMPLETED)
		if err != nil {
			logger.Reason(err).Error("Failed to get final job stats for completed backup.")
		} else if finalStats != nil {
			event.Info.Type = finalStats.Type
		}
	}

	var failed bool
	var message string
	switch event.Info.Type {
	case libvirt.DOMAIN_JOB_COMPLETED:
		logger.Info("Backup has been completed successfully")
	case libvirt.DOMAIN_JOB_CANCELLED:
		logger.Info("Backup has been aborted")
		message = "backup aborted"
		failed = backupMetadata.Mode == string(backupv1.PushMode)
	case libvirt.DOMAIN_JOB_FAILED:
		logger.Info("Backup has failed")
		failed = true
	default:
		message = fmt.Sprintf("unexpected job completion type: %d", event.Info.Type)
		failed = true
	}
	if event.Info.ErrorMessageSet {
		err := event.Info.ErrorMessage
		if message == "" {
			message = err
		} else {
			message = fmt.Sprintf("%s: %s", message, event.Info.ErrorMessage)
		}
	}
	if failed && message == "" {
		message = "unknown failure reason"
	}

	metadataCache.Backup.WithSafeBlock(func(backupMetadata *api.BackupMetadata, exists bool) {
		if !exists || backupMetadata.Name != backupName {
			logger.Warning("Backup metadata changed or was cleared before update could complete. Backup completion may not be properly recorded.")
			return
		}

		backupMetadata.Failed = failed
		backupMetadata.Completed = true
		backupMetadata.BackupMsg = message
		now := metav1.Now()
		backupMetadata.EndTimestamp = &now
	})

	log.Log.V(2).Infof("Updated backup result in metadata via Notifier: %s", metadataCache.Backup.String())
}

func (m *StorageManager) AbortVirtualMachineBackup(vmi *v1.VirtualMachineInstance, backupOptions *backupv1.BackupOptions) error {
	backupMetadata, exists := m.metadataCache.Backup.Load()
	if err := shouldAbort(exists, backupMetadata, backupOptions); err != nil {
		return err
	}
	return m.abortBackup(vmi, backupMetadata)
}

func shouldAbort(exists bool, backupMetadata api.BackupMetadata, backupOptions *backupv1.BackupOptions) error {
	const failedAbort = "failed to abort backup: %s"
	if !exists || backupMetadata.Name == "" {
		return fmt.Errorf(failedAbort, "could not find ongoing backup")
	}
	if backupMetadata.StartTimestamp == nil {
		return fmt.Errorf(failedAbort, "backup did not start yet")
	}
	if backupMetadata.Name != backupOptions.BackupName || !backupMetadata.StartTimestamp.Equal(backupOptions.BackupStartTime) {
		return fmt.Errorf(failedAbort, "requested backup differs from ongoing one")
	}
	if backupMetadata.Completed {
		return fmt.Errorf(failedAbort, "backup already completed")
	}
	return nil
}

func (m *StorageManager) abortBackup(vmi *v1.VirtualMachineInstance, backupMetadata api.BackupMetadata) error {
	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := m.virConn.LookupDomainByName(domName)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Warning("failed to abort backup, domain not found")
		return err
	}
	defer dom.Free()

	stats, err := dom.GetJobStats(0)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("failed to get domain job stats")
		return err
	}
	if stats.Operation != libvirt.DOMAIN_JOB_OPERATION_BACKUP || stats.Type != libvirt.DOMAIN_JOB_UNBOUNDED {
		return fmt.Errorf("cannot abort backup, wrong operation or type: %d, %d", stats.Operation, stats.Type)
	}

	if err := dom.AbortJob(); err != nil {
		log.Log.Object(vmi).Reason(err).Error("failed to abort backup, error calling abort job on domain")
		return err
	}

	log.Log.Object(vmi).Info("backup job abort initiated successfully")
	return nil
}

// isLibvirtCheckpointInvalidError checks if the libvirt error indicates
// the checkpoint is invalid/corrupt (bitmap corruption, inconsistent state, etc.)
func isLibvirtCheckpointInvalidError(err error) bool {
	var libvirtErr libvirt.Error
	if errors.As(err, &libvirtErr) {
		switch libvirtErr.Code {
		case libvirt.ERR_INVALID_DOMAIN_CHECKPOINT,
			libvirt.ERR_NO_DOMAIN_CHECKPOINT,
			libvirt.ERR_CHECKPOINT_INCONSISTENT:
			return true
		}
	}
	return false
}

// RedefineCheckpoint redefines a checkpoint from a previous backup session.
// This is used after VM restart to restore checkpoint metadata in libvirt.
// It iterates over all domain disks and includes those that have the checkpoint bitmap.
func (m *StorageManager) RedefineCheckpoint(vmi *v1.VirtualMachineInstance, checkpoint *backupv1.BackupCheckpoint) (checkpointInvalid bool, err error) {
	logger := log.Log.With("checkpointName", checkpoint.Name)
	logger.Info("Redefining checkpoint")

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := m.virConn.LookupDomainByName(domName)
	if err != nil {
		return false, fmt.Errorf("failed to lookup domain %s: %v", domName, err)
	}
	defer dom.Free()

	// Get all domain disks and find those with the checkpoint bitmap
	checkpointDisks, disksWithoutBitmap, err := findDisksWithCheckpointBitmap(dom, checkpoint.Name)
	if err != nil {
		return false, err
	}

	if len(disksWithoutBitmap) > 0 {
		logger.V(3).Infof("Disks without checkpoint bitmap: %v", disksWithoutBitmap)
	}

	if len(checkpointDisks.Disks) == 0 {
		logger.Warning("No disks found with checkpoint bitmap")
		return true, fmt.Errorf("no disks found with checkpoint bitmap %s", checkpoint.Name)
	}

	domainCheckpoint := &api.DomainCheckpoint{
		Name:            checkpoint.Name,
		CheckpointDisks: checkpointDisks,
	}

	if checkpoint.CreationTime != nil {
		ct := uint64(checkpoint.CreationTime.Unix())
		domainCheckpoint.CreationTime = &ct
	}

	checkpointXML, err := xml.Marshal(domainCheckpoint)
	if err != nil {
		return false, fmt.Errorf("failed to marshal checkpoint XML: %v", err)
	}

	logger.V(3).Infof("Checkpoint XML for redefinition: %s", string(checkpointXML))

	redefineFlags := libvirt.DOMAIN_CHECKPOINT_CREATE_REDEFINE | libvirt.DOMAIN_CHECKPOINT_CREATE_REDEFINE_VALIDATE
	_, err = dom.CreateCheckpointXML(string(checkpointXML), redefineFlags)
	if err != nil {
		checkpointInvalid = isLibvirtCheckpointInvalidError(err)
		if checkpointInvalid {
			logger.Reason(err).Error("Checkpoint bitmap is invalid/corrupt")
		}
		return checkpointInvalid, fmt.Errorf("failed to redefine checkpoint %s: %v", checkpoint.Name, err)
	}

	logger.Infof("Checkpoint redefined successfully with %d disks", len(checkpointDisks.Disks))
	return false, nil
}

// findDisksWithCheckpointBitmap iterates over all domain disks and returns those
// that have the specified checkpoint bitmap in their qcow2 file.
func findDisksWithCheckpointBitmap(dom cli.VirDomain, checkpointName string) (*api.CheckpointDisks, []string, error) {
	disks, err := util.GetAllDomainDisks(dom)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get domain disks: %v", err)
	}

	checkpointDisks := &api.CheckpointDisks{}
	var disksWithoutBitmap []string

	for _, disk := range disks {
		if disk.Target.Device == "" || disk.Source.DataStore == nil {
			continue
		}
		if disk.Source.File == "" {
			log.Log.Warningf("disk with data store source should have the qcow2 overlay file source, disk %s", disk.Target.Device)
			continue
		}

		diskInfo, err := getDiskInfoWithForceShare(disk.Source.File)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get disk info for %s at %s: %v", disk.Target.Device, disk.Source.File, err)
		}

		if diskInfo.HasBitmap(checkpointName) {
			checkpointDisks.Disks = append(checkpointDisks.Disks, api.CheckpointDisk{
				Name:       disk.Target.Device,
				Checkpoint: "bitmap",
			})
		} else {
			disksWithoutBitmap = append(disksWithoutBitmap, disk.Target.Device)
		}
	}

	return checkpointDisks, disksWithoutBitmap, nil
}
