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
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"libvirt.org/go/libvirt"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

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
)

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
	if backupOptions.PushPath != nil {
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
	domainBackup, domainCheckpoint := generateDomainBackup(domainDisks, backupOptions, backupPath)
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

	m.metadataCache.Backup.WithSafeBlock(func(backupMetadata *api.BackupMetadata, _ bool) {
		backupMetadata.CheckpointName = domainCheckpoint.Name
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

func generateDomainBackup(disks []api.Disk, backupOptions *backupv1.BackupOptions, backupPath string) (*api.DomainBackup, *api.DomainCheckpoint) {
	domainBackup := &api.DomainBackup{
		Mode: string(backupOptions.Mode),
	}
	if isIncrementalBackup(backupOptions) {
		log.Log.Infof("Generating incremental backup %s from checkpoint: %s", backupOptions.BackupName, *backupOptions.Incremental)
		domainBackup.Incremental = backupOptions.Incremental
	}
	backupDisks := &api.BackupDisks{}
	checkpointDisks := &api.CheckpointDisks{}
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
			if backupOptions.PushPath != nil {
				backupDisk.Target = &api.BackupTarget{
					File: targetQCOW2File(backupPath, backupOptions.BackupName, volumeName),
				}
			}
			checkpointDisk.Checkpoint = "bitmap"
		} else {
			backupDisk.Backup = "no"
			checkpointDisk.Checkpoint = "no"
		}
		backupDisks.Disks = append(backupDisks.Disks, backupDisk)
		checkpointDisks.Disks = append(checkpointDisks.Disks, checkpointDisk)
	}

	domainBackup.BackupDisks = backupDisks
	backupTime := backupTimeFormatted(backupOptions.BackupStartTime)
	checkpointName := fmt.Sprintf("%s-%s", backupOptions.BackupName, backupTime)
	domainCheckpoint := &api.DomainCheckpoint{
		Name:            checkpointName,
		CheckpointDisks: checkpointDisks,
	}
	return domainBackup, domainCheckpoint
}

func getBackupPath(backupOptions *backupv1.BackupOptions, vmiName string) string {
	backupTime := backupTimeFormatted(backupOptions.BackupStartTime)
	backupNameWithTime := fmt.Sprintf("%s-%s", backupOptions.BackupName, backupTime)
	return filepath.Join(*backupOptions.PushPath, vmiName, backupNameWithTime)
}

func targetQCOW2File(pushPath, backupName, volumeName string) string {
	fileName := fmt.Sprintf("%s-%s.qcow2", backupName, volumeName)
	return filepath.Join(pushPath, fileName)
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

	// TODO: Handle non-success job completion (DOMAIN_JOB_FAILED, DOMAIN_JOB_CANCELLED, unknown types)
	if event.Info.Type == libvirt.DOMAIN_JOB_COMPLETED {
		logger.Info("Backup has been completed successfully")
	} else {
		logger.Warningf("Unexpected job completion type: %d (only handling success case)", event.Info.Type)
	}

	metadataCache.Backup.WithSafeBlock(func(backupMetadata *api.BackupMetadata, exists bool) {
		// Verify the backup metadata is still for the same backup to avoid race conditions
		if !exists || backupMetadata.Name != backupName {
			logger.Warning("Backup metadata changed or was cleared before update could complete. Backup completion may not be properly recorded.")
			return
		}
		backupMetadata.Completed = true
		now := metav1.Now()
		backupMetadata.EndTimestamp = &now
	})

	log.Log.V(2).Infof("Updated backup result in metadata via Notifier: %s", metadataCache.Backup.String())
}

// TODO: Implement backup abort functionality for graceful shutdown
