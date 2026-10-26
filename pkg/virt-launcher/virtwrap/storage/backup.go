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
	"slices"
	"strings"
	"syscall"

	"github.com/opencontainers/selinux/go-selinux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"libvirt.org/go/libvirt"

	backupv1 "kubevirt.io/api/backup/v1beta1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/storage/cbt"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	kutil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

const (
	ChangedBlockTrackingNotEnabledMsg = "Backup failed ChangedBlockTracking is not enabled"
	backupTimeXMLFormat               = "2006-01-02_15-04-05"
	freezeFailedMsg                   = "Failed freezing guest filesystem: %s"
	unfreezeFailedMsg                 = "Failed to unfreeze filesystem after backup completion"
	operationCanceledMsg              = "Operation canceled"
)

func (m *StorageManager) BackupVirtualMachine(vmi *v1.VirtualMachineInstance, backupOptions *backupv1.BackupOptions) error {
	if OverlaysActive(m.metadataCache) {
		return fmt.Errorf("cannot back up the VMI while its disks are on snapshot overlays")
	}

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
		// In case the backup failed to start, reset the metadata cache so retries can proceed with fresh metadata
		m.metadataCache.Backup.Store(api.BackupMetadata{
			CheckpointCleanupPending: len(m.checkpointCandidates) > 0,
		})
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
		Name:                     backupOptions.BackupName,
		StartTimestamp:           backupOptions.BackupStartTime,
		SkipQuiesce:              backupOptions.SkipQuiesce,
		Mode:                     string(backupOptions.Mode),
		CheckpointCleanupPending: len(m.checkpointCandidates) > 0,
	}
	m.metadataCache.Backup.Store(b)
	log.Log.Infof("Initialized backup metadata: %v", b)

	return false, nil
}

// CleanupBackupCheckpoints reclaims bases after a successful backup has been promoted.
// Keep released candidates eligible if a later backup fails.
// Missing checkpoints and recovery after a restart are deliberately left alone.
func (m *StorageManager) CleanupBackupCheckpoints(vmi *v1.VirtualMachineInstance, dom cli.VirDomain) error {
	uid, initialized := m.metadataCache.UID.Load()
	if !initialized || uid != vmi.UID {
		return nil
	}
	if len(m.checkpointCandidates) == 0 || m.MigrationInProgress() {
		return nil
	}
	// Wait until the controller has finished processing the current backup.
	if vmi.Status.ChangedBlockTracking != nil && vmi.Status.ChangedBlockTracking.BackupStatus != nil {
		return nil
	}
	reference, err := storagetypes.GetBackupCheckpointReference(vmi)
	if err != nil || reference == nil {
		return err
	}
	referencedCheckpointNames := reference.ReferencedCheckpointNames
	backup, exists := m.metadataCache.Backup.Load()
	canRelease := exists && backup.Completed && !backup.Failed &&
		backup.Name == reference.BackupName && backup.CheckpointName == reference.CheckpointName &&
		backup.StartTimestamp != nil && backup.StartTimestamp.Equal(reference.StartTimestamp)
	// Record all releases before a job check or deletion can fail.
	if !m.releaseBackupCheckpointCandidates(referencedCheckpointNames, canRelease) {
		return nil
	}
	job, err := dom.GetJobInfo()
	if err != nil {
		return err
	}
	if job == nil || job.Type != libvirt.DOMAIN_JOB_NONE {
		return nil
	}
	err = m.deleteReleasedBackupCheckpoints(api.VMINamespaceKeyFunc(vmi), referencedCheckpointNames)
	m.metadataCache.Backup.WithSafeBlock(func(current *api.BackupMetadata, _ bool) {
		if current.CheckpointName == backup.CheckpointName {
			current.CheckpointCleanupPending = len(m.checkpointCandidates) > 0
		}
	})
	return err
}

func (m *StorageManager) releaseBackupCheckpointCandidates(referencedCheckpointNames []string, canRelease bool) bool {
	ready := false
	for name, candidate := range m.checkpointCandidates {
		if slices.Contains(referencedCheckpointNames, name) {
			continue
		}
		if canRelease && !candidate.released {
			candidate.released = true
			m.checkpointCandidates[name] = candidate
		}
		ready = ready || candidate.released
	}
	return ready
}

func (m *StorageManager) deleteReleasedBackupCheckpoints(domainName string, referencedCheckpointNames []string) error {
	var cleanupErrors []error
	for name, candidate := range m.checkpointCandidates {
		if !candidate.released || slices.Contains(referencedCheckpointNames, name) {
			continue
		}
		if err := m.virConn.DeleteDomainCheckpoint(domainName, name); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("delete released checkpoint %s: %w", name, err))
			continue
		}
		delete(m.checkpointCandidates, name)
	}
	return errors.Join(cleanupErrors...)
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
	var disksWithoutBitmap sets.Set[string]
	if isIncrementalBackup(backupOptions) {
		base := checkpointDefinition(*backupOptions.Incremental, backupOptions.IncrementalCreationTime)
		_, disksWithoutBitmap, err = redefineCheckpointOnUsableDisks(dom, domainDisks, base)
		if err != nil && !errors.Is(err, errNoUsableCheckpointBitmap) {
			logger.Reason(err).Errorf("failed to redefine base checkpoint %s", base.Name)
			return err
		}
	}

	var nbdSocketPath string
	if backupOptions.Mode == backupv1.PullMode {
		if nbdSocketPath, err = preparePullBackupSocket(vmi); err != nil {
			logger.Reason(err).Error("failed to prepare the pull backup socket")
			return err
		}
	}
	domainBackup, domainCheckpoint, backupVolumesInfo := generateDomainBackup(domainDisks, backupOptions, backupPath, nbdSocketPath, disksWithoutBitmap)
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

	if !backupOptions.SkipQuiesce {
		logger.Info("Freezing VMI to capture backup state")
		if err := m.FreezeVMI(vmi, 0); err != nil {
			logger.Warningf(freezeFailedMsg, err)
			m.metadataCache.Backup.WithSafeBlock(func(backupMetadata *api.BackupMetadata, _ bool) {
				backupMetadata.QuiesceStatus = string(backupv1.QuiesceFailed)
				backupMetadata.BackupMsg = err.Error()
			})
		} else {
			m.metadataCache.Backup.WithSafeBlock(func(backupMetadata *api.BackupMetadata, _ bool) {
				backupMetadata.QuiesceStatus = string(backupv1.QuiesceSucceeded)
			})
		}

		defer func() {
			logger.Info("Thawing VMI after backup job started")
			if err := m.UnfreezeVMI(vmi); err != nil {
				logger.Reason(err).Error(unfreezeFailedMsg)
			}
		}()
	} else {
		m.metadataCache.Backup.WithSafeBlock(func(backupMetadata *api.BackupMetadata, _ bool) {
			backupMetadata.QuiesceStatus = string(backupv1.QuiesceSkipped)
		})
	}

	if err := dom.BackupBegin(strings.ToLower(string(backupXML)), strings.ToLower(string(checkpointXML)), 0); err != nil {
		return err
	}
	// Only track the base if at least one disk used it.
	if domainBackup.Incremental != nil {
		if _, exists := m.checkpointCandidates[*domainBackup.Incremental]; !exists {
			m.checkpointCandidates[*domainBackup.Incremental] = checkpointCandidate{}
		}
		m.metadataCache.Backup.WithSafeBlock(func(backup *api.BackupMetadata, _ bool) {
			backup.CheckpointCleanupPending = len(m.checkpointCandidates) > 0
		})
	}
	return nil
}

func generateDomainBackup(disks []api.Disk, backupOptions *backupv1.BackupOptions, backupPath, nbdSocketPath string, disksWithoutBitmap sets.Set[string]) (*api.DomainBackup, *api.DomainCheckpoint, []v1.VirtualMachineInstanceBackupVolumeInfo) {
	domainBackup := &api.DomainBackup{
		Mode: string(backupOptions.Mode),
	}
	incremental := isIncrementalBackup(backupOptions)
	if backupOptions.Mode == backupv1.PullMode {
		domainBackup.Server = &api.DomainBackupServer{
			Transport: api.BackupUnixTransport,
			Socket:    nbdSocketPath,
		}
	}
	backupTime := backupTimeFormatted(backupOptions.BackupStartTime)
	checkpointName := fmt.Sprintf("%s-%s", backupOptions.BackupName, backupTime)
	backupDisks := &api.BackupDisks{}
	checkpointDisks := &api.CheckpointDisks{}
	var backupVolumesInfo []v1.VirtualMachineInstanceBackupVolumeInfo
	anyIncremental := false
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
		volumeName := disk.Alias.GetName()
		if DiskHasDataStore(&disk) {
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

			volumeType := backupv1.Full
			if incremental {
				if disksWithoutBitmap.Has(disk.Target.Device) {
					backupDisk.BackupMode = "full"
				} else {
					volumeType = backupv1.Incremental
					anyIncremental = true
				}
			}

			backupVolumesInfo = append(backupVolumesInfo, v1.VirtualMachineInstanceBackupVolumeInfo{
				VolumeName: volumeName,
				Type:       string(volumeType),
			})
		} else {
			backupDisk.Backup = "no"
			checkpointDisk.Checkpoint = "no"
		}
		backupDisks.Disks = append(backupDisks.Disks, backupDisk)
		checkpointDisks.Disks = append(checkpointDisks.Disks, checkpointDisk)
	}

	// Name the base only when a disk is actually backed up against it.
	if anyIncremental {
		log.Log.Infof("Generating incremental backup %s from checkpoint: %s", backupOptions.BackupName, *backupOptions.Incremental)
		domainBackup.Incremental = backupOptions.Incremental
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
		failed = backupMetadata.Mode == string(backupv1.PushMode)
		if failed {
			message = "backup aborted"
		}
	case libvirt.DOMAIN_JOB_FAILED:
		logger.Info("Backup has failed")
		failed = true
	default:
		message = fmt.Sprintf("unexpected job completion type: %d", event.Info.Type)
		failed = true
	}

	// If we successfully aborted but we didn't fail (i.e., PullMode)
	// avoid propagating an error message (i.e., Operation canceled)
	if event.Info.ErrorMessageSet && !(backupMetadata.Mode == string(backupv1.PullMode) && event.Info.ErrorMessage == operationCanceledMsg) {
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
	if err := checkBackupEligibility(exists, backupMetadata, backupOptions); err != nil {
		return fmt.Errorf("failed to abort backup: %w", err)
	}
	return m.abortBackup(vmi, backupMetadata)
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

var preparePullBackupSocket = preparePullBackupSocketFunc

func preparePullBackupSocketFunc(vmi *v1.VirtualMachineInstance) (string, error) {
	dir := cbt.PathForBackupNBDSocketDir(vmi)
	if err := kutil.MkdirAllWithNosec(dir); err != nil {
		return "", fmt.Errorf("failed to create %s: %w", dir, err)
	}
	if err := relabelWithoutCategories(dir); err != nil {
		return "", fmt.Errorf("failed to relabel %s: %w", dir, err)
	}
	safeDir, err := safepath.JoinAndResolveWithRelativeRoot("/", dir)
	if err != nil {
		return "", err
	}
	if err := diskutils.DefaultOwnershipManager.SetFileOwnership(safeDir); err != nil {
		return "", fmt.Errorf("failed to set the ownership of %s: %w", dir, err)
	}
	return filepath.Join(dir, cbt.NBDSocketName), nil
}

func relabelWithoutCategories(path string) error {
	label, err := selinux.FileLabel(path)
	if errors.Is(err, syscall.ENODATA) || errors.Is(err, syscall.ENOTSUP) {
		return nil
	}
	if err != nil {
		return err
	}
	ctx, err := selinux.NewContext(label)
	if err != nil {
		return err
	}
	ctx["level"] = "s0"
	return selinux.SetFileLabel(path, ctx.Get())
}

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

func checkpointDefinition(name string, creationTime *metav1.Time) *api.DomainCheckpoint {
	domainCheckpoint := &api.DomainCheckpoint{Name: name}
	if creationTime != nil {
		domainCheckpoint.CreationTime = new(uint64(creationTime.Unix()))
	}
	return domainCheckpoint
}

var errNoUsableCheckpointBitmap = errors.New("no disk carries a usable checkpoint bitmap")

func redefineCheckpointOnUsableDisks(dom cli.VirDomain, disks []api.Disk, domainCheckpoint *api.DomainCheckpoint) (*api.CheckpointDisks, sets.Set[string], error) {
	checkpointDisks, disksWithoutBitmap, err := findDisksWithCheckpointBitmap(dom, disks, domainCheckpoint)
	if err != nil {
		return nil, disksWithoutBitmap, err
	}
	if len(checkpointDisks.Disks) == 0 {
		return checkpointDisks, disksWithoutBitmap, errNoUsableCheckpointBitmap
	}

	redefined := domainCheckpoint.DeepCopy()
	redefined.CheckpointDisks = checkpointDisks
	return checkpointDisks, disksWithoutBitmap, redefineCheckpoint(dom, redefined)
}

// findDisksWithCheckpointBitmap returns the domain disks that carry a usable
// bitmap for the checkpoint, and those that don't.
//
// Bitmap state is only exposed by QEMU, and querying it through the QEMU
// monitor taints the domain. Instead, each disk is probed by redefining the
// checkpoint with that disk alone and REDEFINE_VALIDATE, which makes libvirt
// run the same check it runs at BackupBegin: the bitmap must be present,
// persistent, recording and consistent. Each successful probe leaves a
// single-disk definition behind, so callers must redefine the checkpoint
// afterwards, see redefineCheckpointOnUsableDisks.
func findDisksWithCheckpointBitmap(dom cli.VirDomain, disks []api.Disk, domainCheckpoint *api.DomainCheckpoint) (*api.CheckpointDisks, sets.Set[string], error) {
	checkpointDisks := &api.CheckpointDisks{}
	disksWithoutBitmap := sets.New[string]()

	for _, disk := range disks {
		if disk.Target.Device == "" || !DiskHasDataStore(&disk) {
			continue
		}

		checkpointDisk := api.CheckpointDisk{Name: disk.Target.Device, Checkpoint: "bitmap"}
		probe := domainCheckpoint.DeepCopy()
		probe.CheckpointDisks = &api.CheckpointDisks{Disks: []api.CheckpointDisk{checkpointDisk}}
		err := redefineCheckpoint(dom, probe)
		if err == nil {
			checkpointDisks.Disks = append(checkpointDisks.Disks, checkpointDisk)
			continue
		}
		if !isLibvirtCheckpointInvalidError(err) {
			return nil, nil, fmt.Errorf("failed to probe checkpoint %s bitmap on disk %s: %w", domainCheckpoint.Name, disk.Target.Device, err)
		}
		log.Log.V(3).Reason(err).Infof("Disk %s has no usable bitmap for checkpoint %s", disk.Target.Device, domainCheckpoint.Name)
		disksWithoutBitmap.Insert(disk.Target.Device)
	}

	return checkpointDisks, disksWithoutBitmap, nil
}

func redefineCheckpoint(dom cli.VirDomain, domainCheckpoint *api.DomainCheckpoint) error {
	checkpointXML, err := xml.Marshal(domainCheckpoint)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint XML: %v", err)
	}

	log.Log.V(3).Infof("Checkpoint XML for redefinition: %s", string(checkpointXML))

	redefineFlags := libvirt.DOMAIN_CHECKPOINT_CREATE_REDEFINE | libvirt.DOMAIN_CHECKPOINT_CREATE_REDEFINE_VALIDATE
	chk, err := dom.CreateCheckpointXML(string(checkpointXML), redefineFlags)
	if err != nil {
		return err
	}
	if chk != nil {
		if err := chk.Free(); err != nil {
			log.Log.Reason(err).Warningf("failed to free checkpoint %s", domainCheckpoint.Name)
		}
	}
	return nil
}

func checkBackupEligibility(exists bool, backupMetadata api.BackupMetadata, backupOptions *backupv1.BackupOptions) error {
	if !exists || backupMetadata.Name == "" {
		return fmt.Errorf("could not find ongoing backup")
	}
	if backupMetadata.StartTimestamp == nil {
		return fmt.Errorf("backup did not start yet")
	}
	if backupMetadata.Name != backupOptions.BackupName || !backupMetadata.StartTimestamp.Equal(backupOptions.BackupStartTime) {
		return fmt.Errorf("requested backup differs from ongoing one")
	}
	if backupMetadata.Completed {
		return fmt.Errorf("backup already completed")
	}
	return nil
}
