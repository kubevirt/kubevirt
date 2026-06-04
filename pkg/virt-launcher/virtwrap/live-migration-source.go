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

package virtwrap

import (
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"libvirt.org/go/libvirt"
	"libvirt.org/go/libvirtxml"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	osdisk "kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/pointer"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	virtutil "kubevirt.io/kubevirt/pkg/util"
	utilheap "kubevirt.io/kubevirt/pkg/util/heap"
	migrationutils "kubevirt.io/kubevirt/pkg/util/migrations"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cpudedicated"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/sriov"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/disksource"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/statsconv"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
	"kubevirt.io/kubevirt/pkg/vmitrait"
)

const liveMigrationFailed = "Live migration failed."

const (
	monitorSleepPeriodMS = 400
	monitorLogPeriodMS   = 4000
	monitorLogInterval   = monitorLogPeriodMS / monitorSleepPeriodMS
)

type convergenceAction int

const (
	actionNothing convergenceAction = iota
	actionAbort
	actionPostCopy
	actionHardStopAndCopy
	actionSoftStopAndCopy
)

type migrationDisks struct {
	shared         map[string]bool
	generated      map[string]bool
	localToMigrate map[string]bool
}

type iterationRecord struct {
	elapsedMs       uint64
	remainingBytes  uint64
	iterationNumber uint64
}

type stallDetector struct {
	options *cmdclient.MigrationOptions

	// a bool indicating whether initial max downtime has been set (only set when maxDowntimeMs < 300, the default QEMU target downtime)
	initialMaxDowntimeSet bool
	// iteration records with the potential to end up in minRecordOutsideWindow
	minCandidates []iterationRecord
	// smallest iteration record outside the progressTimeout window
	minRecordOutsideWindow *iterationRecord
	// whether migration is currently stalled
	stallDetected bool
	// a sorted history of remaining bytes
	remainingBytesHistory *utilheap.Heap[uint64]
	// best value of "remaining bytes" observed so far
	bestRemainingBytes uint64
	// time which when hit we will relax target downtime further
	relaxationDeadlineMs uint64
	// current time in ms to wait before relaxing target downtime
	relaxationPatienceMs uint64
	// Current bandwidth smoothed using an exponential weighted moving average
	ewmaBandwidthBps float64
	// Whether we already initiated switchover to post-copy or stop-and-copy
	switchoverInitiated bool
	// time indicating when stall was first detected
	stallDetectedAtMs uint64
}

type migrationMonitor struct {
	l       *LibvirtDomainManager
	vmi     *v1.VirtualMachineInstance
	options *cmdclient.MigrationOptions

	migrationErr chan error
	iterationCh  chan int

	// deadline in seconds for the end-to-end migration to complete
	acceptableCompletionTime int64
	// deadline in seconds for switchover to post-copy or stop-and-copy; initialized as the same value as acceptableCompletionTime
	switchOverDeadline int64
	// timestamp in unix nano migration began
	start int64
	// most recent iteration record (remaining bytes, time elapsed) as reported by QEMU
	iterationRecord iterationRecord
	// whether stall detection is enabled or to use legacy the path
	stallDetectionEnabled bool

	stallDetector *stallDetector
	logger        *log.FilteredLogger

	// TODO: fields used by legacy stall detector; to be removed
	lastProgressUpdate int64
	progressWatermark  uint64
	remainingData      uint64
	progressTimeout    int64
}

type inflightMigrationAborted struct {
	message     string
	abortStatus v1.MigrationAbortStatus
}

func generateMigrationFlags(isBlockMigration, migratePaused bool, options *cmdclient.MigrationOptions) libvirt.DomainMigrateFlags {
	migrateFlags := libvirt.MIGRATE_LIVE | libvirt.MIGRATE_PEER2PEER | libvirt.MIGRATE_PERSIST_DEST

	if isBlockMigration {
		migrateFlags |= libvirt.MIGRATE_NON_SHARED_INC
	}
	if options.UnsafeMigration {
		migrateFlags |= libvirt.MIGRATE_UNSAFE
	}
	if options.AllowAutoConverge {
		migrateFlags |= libvirt.MIGRATE_AUTO_CONVERGE
	}
	if options.AllowPostCopy {
		migrateFlags |= libvirt.MIGRATE_POSTCOPY
	}
	if migratePaused {
		migrateFlags |= libvirt.MIGRATE_PAUSED
	}
	if shouldConfigureParallel, _ := shouldConfigureParallelMigration(options); shouldConfigureParallel {
		migrateFlags |= libvirt.MIGRATE_PARALLEL
	}

	return migrateFlags

}

func hotUnplugHostDevices(virConn cli.Connection, dom cli.VirDomain) error {
	domainSpec, err := util.GetDomainSpecWithFlags(dom, 0)
	if err != nil {
		return err
	}

	eventChan := make(chan interface{}, hostdevice.MaxConcurrentHotPlugDevicesEvents)
	var callback libvirt.DomainEventDeviceRemovedCallback = func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventDeviceRemoved) {
		eventChan <- event.DevAlias
	}

	if domainEvent := cli.NewDomainEventDeviceRemoved(virConn, dom, callback, eventChan); domainEvent != nil {
		const waitForDetachTimeout = 30 * time.Second
		err := sriov.SafelyDetachHostDevices(domainSpec, domainEvent, dom, waitForDetachTimeout)
		if err != nil {
			return err
		}
	}
	return nil
}

// This returns domain xml without the metadata section, as it is only relevant to the source domain
// Note: Unfortunately we can't just use UnMarshall + Marshall here, as that leads to unwanted XML alterations
func migratableDomXML(dom cli.VirDomain, vmi *v1.VirtualMachineInstance, domSpec *api.DomainSpec) (string, error) {
	var domain *api.Domain
	var err error

	xmlstr, err := dom.GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Live migration failed. Failed to get XML.")
		return "", err
	}
	domcfg := &libvirtxml.Domain{}
	if err := domcfg.Unmarshal(xmlstr); err != nil {
		return "", err
	}
	if err = convertDisks(domSpec, domcfg); err != nil {
		return "", err
	}
	// TODO: Once the LibvirtHooksServerAndClient feature gate is GA,
	// this logic in the source can be removed, as XML modifications
	// for dedicated CPUs will always be handled on the target side.
	if vmi.IsCPUDedicated() {
		// If the VMI has dedicated CPUs, we need to replace the old CPUs that were
		// assigned in the source node with the new CPUs assigned in the target node
		err = xml.Unmarshal([]byte(xmlstr), &domain)
		if err != nil {
			return "", err
		}
		domain, err := cpudedicated.GenerateDomainForTargetCPUSetAndTopology(vmi, domSpec)
		if err != nil {
			return "", err
		}
		if err = cpudedicated.ConvertCPUDedicatedFields(domain, domcfg); err != nil {
			return "", err
		}
	}
	// set slice size for local disks to migrate
	if err := configureLocalDiskToMigrate(domcfg, vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to set size for local disk.")
		return "", err
	}

	return domcfg.Marshal()
}

func convertDisks(domSpec *api.DomainSpec, domcfg *libvirtxml.Domain) error {
	if domcfg == nil || domcfg.Devices == nil || domSpec == nil {
		return nil
	}
	if len(domSpec.Devices.Disks) != len(domcfg.Devices.Disks) {
		return fmt.Errorf("spec and domain have different disks count")
	}
	for i, disk := range domSpec.Devices.Disks {
		domcfgDisk := (&domcfg.Devices.Disks[i])
		diskName := disk.Alias.GetName()

		if disk.Source.File != "" {
			if domcfgDisk.Source == nil || domcfgDisk.Source.File == nil {
				return fmt.Errorf("disk %s: spec has file source but domain is missing it", diskName)
			}
			log.Log.Infof("Updating disk %s source file from %s to %s", diskName, domcfgDisk.Source.File.File, disk.Source.File)
			domcfgDisk.Source.File.File = disk.Source.File
		}
		if disk.Source.DataStore != nil && disk.Source.DataStore.Source != nil && disk.Source.DataStore.Source.File != "" {
			if domcfgDisk.Source == nil || domcfgDisk.Source.DataStore == nil || domcfgDisk.Source.DataStore.Source == nil || domcfgDisk.Source.DataStore.Source.File == nil {
				return fmt.Errorf("disk %s: spec has DataStore file source but domain is missing it", diskName)
			}
			log.Log.Infof("Updating disk %s datastore backend from %s to %s", diskName, domcfgDisk.Source.DataStore.Source.File.File, disk.Source.DataStore.Source.File)
			domcfgDisk.Source.DataStore.Source.File.File = disk.Source.DataStore.Source.File
		}
	}
	return nil
}

func (d *migrationDisks) isSharedVolume(name string) bool {
	_, shared := d.shared[name]
	return shared
}

func (d *migrationDisks) isGeneratedVolume(name string) bool {
	_, generated := d.generated[name]
	return generated
}

func (d *migrationDisks) isLocalVolumeToMigrate(name string) bool {
	_, migrate := d.localToMigrate[name]
	return migrate
}

func classifyVolumesForMigration(vmi *v1.VirtualMachineInstance) *migrationDisks {
	// This method collects all VMI volumes that should not be copied during
	// live migration. It also collects all generated disks suck as cloudinit, secrets, ServiceAccount and ConfigMaps
	// to make sure that these are being copied during migration.

	disks := &migrationDisks{
		shared:         make(map[string]bool),
		generated:      make(map[string]bool),
		localToMigrate: make(map[string]bool),
	}
	migrateDisks := make(map[string]bool)
	for _, v := range vmi.Status.MigratedVolumes {
		migrateDisks[v.VolumeName] = true
	}
	for _, volume := range vmi.Spec.Volumes {
		volSrc := volume.VolumeSource
		switch {
		case volSrc.PersistentVolumeClaim != nil || volSrc.DataVolume != nil:
			if _, ok := migrateDisks[volume.Name]; ok {
				disks.localToMigrate[volume.Name] = true
			} else {
				disks.shared[volume.Name] = true
			}
		case volSrc.HostDisk != nil:
			if _, ok := migrateDisks[volume.Name]; ok {
				disks.localToMigrate[volume.Name] = true
			} else if volSrc.HostDisk.Shared != nil && *volSrc.HostDisk.Shared {
				disks.shared[volume.Name] = true
			}
		case volSrc.ConfigMap != nil || volSrc.Secret != nil || volSrc.DownwardAPI != nil ||
			volSrc.ServiceAccount != nil || volSrc.CloudInitNoCloud != nil ||
			volSrc.CloudInitConfigDrive != nil || volSrc.ContainerDisk != nil:
			disks.generated[volume.Name] = true
		}
	}

	return disks
}

func getDiskTargetsForMigration(dom cli.VirDomain, vmi *v1.VirtualMachineInstance) []string {
	// This method collects all VMI disks that needs to be copied during live migration
	// and returns a list of its target device names.
	// Shared volues are being excluded.
	var copyDisks []string
	migrationVols := classifyVolumesForMigration(vmi)
	disks, err := util.GetAllDomainDisks(dom)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("failed to parse domain XML to get disks.")
	}
	// the name of the volume should match the alias
	for _, disk := range disks {
		// explicitly skip cd-rom drives
		if disk.Device == "cdrom" {
			continue
		}
		if disk.ReadOnly != nil && !migrationVols.isGeneratedVolume(disk.Alias.GetName()) {
			continue
		}
		if (disk.Type != "file" && disk.Type != "block") || migrationVols.isSharedVolume(disk.Alias.GetName()) {
			continue
		}
		copyDisks = append(copyDisks, disk.Target.Device)
	}
	return copyDisks
}

func (l *LibvirtDomainManager) startMigration(vmi *v1.VirtualMachineInstance, options *cmdclient.MigrationOptions) error {
	if vmi.Status.ChangedBlockTracking != nil && vmi.Status.ChangedBlockTracking.BackupStatus != nil {
		return fmt.Errorf("cannot migrate VMI until backup %s is completed", vmi.Status.ChangedBlockTracking.BackupStatus.BackupName)
	}
	if vmi.Status.MigrationState == nil {
		return fmt.Errorf("cannot migrate VMI until migrationState is ready")
	}

	inProgress, err := l.initializeMigrationMetadata(vmi, v1.MigrationPreCopy)
	if err != nil {
		return err
	}
	if inProgress {
		return nil
	}

	go l.migrate(vmi, options)
	return nil
}

func (l *LibvirtDomainManager) initializeMigrationMetadata(vmi *v1.VirtualMachineInstance, migrationMode v1.MigrationMode) (bool, error) {
	migrationMetadata, exists := l.metadataCache.Migration.Load()
	uid := MigrationUID(vmi)
	if exists && migrationMetadata.UID == uid {
		if migrationMetadata.EndTimestamp == nil {
			// don't stop on currently executing migrations
			return true, nil
		}

		// Don't allow the same migration UID to be executed twice.
		// Migration attempts are like pods. One shot.
		return false, fmt.Errorf("migration job %v already executed, finished at %v, failed: %t, abortStatus: %s",
			migrationMetadata.UID, *migrationMetadata.EndTimestamp, migrationMetadata.Failed, migrationMetadata.AbortStatus)
	}

	now := metav1.Now()
	m := api.MigrationMetadata{
		UID:            uid,
		StartTimestamp: &now,
		Mode:           migrationMode,
	}
	l.metadataCache.Migration.Store(m)
	log.Log.V(4).Infof("initialize migration metadata: %v", m)
	return false, nil
}

func (l *LibvirtDomainManager) cancelMigration(vmi *v1.VirtualMachineInstance) error {
	migration, _ := l.metadataCache.Migration.Load()
	if migration.EndTimestamp != nil || migration.Failed || migration.StartTimestamp == nil {
		return fmt.Errorf(migrationutils.CancelMigrationFailedVmiNotMigratingErr)
	}

	if err := l.setMigrationAbortStatus(v1.MigrationAbortInProgress); err != nil {
		if errors.Is(err, domainerrors.MigrationAbortInProgressError) {
			return nil
		}
		return err
	}

	l.asyncMigrationAbort(vmi)
	return nil
}

func (l *LibvirtDomainManager) setMigrationResultHelper(failed bool, reason string, abortStatus v1.MigrationAbortStatus) error {
	migrationMetadata, exists := l.metadataCache.Migration.Load()
	if !exists {
		// nothing to report if migration metadata is empty
		return nil
	}

	metaAbortStatus := migrationMetadata.AbortStatus
	if abortStatus != "" {
		if metaAbortStatus == string(abortStatus) && metaAbortStatus == string(v1.MigrationAbortInProgress) {
			return domainerrors.MigrationAbortInProgressError
		}
	}

	if metaAbortStatus == string(v1.MigrationAbortInProgress) &&
		abortStatus != v1.MigrationAbortFailed &&
		abortStatus != v1.MigrationAbortSucceeded {
		return domainerrors.MigrationAbortInProgressError
	}

	if migrationMetadata.EndTimestamp != nil {
		// the migration result has already been reported and should not be overwritten
		return nil
	}

	// Improve the error message when the volume migration fails because the destination size is smaller than the source volume
	if failed && strings.Contains(standardizeSpaces(reason), "has to be smaller or equal to the actual size of the containing file") {
		reason = fmt.Sprintf("Volume migration cannot be performed because the destination volume is smaller than the source volume: %v", reason)
	}

	l.metadataCache.Migration.WithSafeBlock(func(migrationMetadata *api.MigrationMetadata, _ bool) {
		if failed {
			migrationMetadata.Failed = true
			migrationMetadata.FailureReason = reason
		}

		migrationMetadata.AbortStatus = string(abortStatus)

		if abortStatus == "" || abortStatus == v1.MigrationAbortSucceeded {
			// only mark the migration as complete if there was no abortion or
			// the abortion succeeded
			migrationMetadata.EndTimestamp = pointer.P(metav1.Now())
		}
	})

	logger := log.Log.V(2)
	if !failed {
		logger = logger.V(4)
	}
	logger.Infof("set migration result in metadata: %s", l.metadataCache.Migration.String())
	return nil
}

func (l *LibvirtDomainManager) setMigrationResult(failed bool, reason string, abortStatus v1.MigrationAbortStatus) error {
	return l.setMigrationResultHelper(failed, reason, abortStatus)
}

func (l *LibvirtDomainManager) setMigrationAbortStatus(abortStatus v1.MigrationAbortStatus) error {
	return l.setMigrationResultHelper(false, "", abortStatus)
}

func (sd *stallDetector) updateBandwidthEstimate(bandwidthSample uint64, logger *log.FilteredLogger) {
	prev := sd.ewmaBandwidthBps
	if sd.ewmaBandwidthBps == 0 {
		sd.ewmaBandwidthBps = float64(bandwidthSample)
		logger.V(4).Infof("initialized migration bandwidth EWMA: sampleBps=%dbps ewmaBps=%.2fbps", bandwidthSample, sd.ewmaBandwidthBps)
		return
	}
	bandwidthEWMAAlpha := sd.options.StallDetectorOptions.EwmaAlpha
	sd.ewmaBandwidthBps = bandwidthEWMAAlpha*float64(bandwidthSample) + (1-bandwidthEWMAAlpha)*sd.ewmaBandwidthBps
	logger.V(4).Infof("updated migration bandwidth EWMA: sampleBps=%dbps previousEwmaBps=%.2fbps newEwmaBps=%.2fbps", bandwidthSample, prev, sd.ewmaBandwidthBps)
}

func bytesToMiB(bytes uint64) float32 {
	return float32(bytes) / float32(1024) / float32(1024)
}

func (sd *stallDetector) updateCandidates(record iterationRecord, logger *log.FilteredLogger) {
	stallProgressTimeout := sd.options.StallDetectorOptions.StallProgressTimeout
	progressTimeoutMs := uint64(stallProgressTimeout) * 1000
	agedOut := 0
	for len(sd.minCandidates) > 0 {
		oldestCandidate := sd.minCandidates[0]
		// record.elapsedMs > oldestCandidate.elapsedMs because record.elapsedMs is monotonically increasing
		ageMs := record.elapsedMs - oldestCandidate.elapsedMs
		if ageMs < progressTimeoutMs {
			break
		}

		sd.minCandidates = sd.minCandidates[1:]
		if sd.minRecordOutsideWindow == nil || oldestCandidate.remainingBytes < sd.minRecordOutsideWindow.remainingBytes {
			sd.minRecordOutsideWindow = &oldestCandidate
		}
		agedOut++
	}
	if agedOut > 0 {
		outsideWindowMin := uint64(0)
		if sd.minRecordOutsideWindow != nil {
			outsideWindowMin = sd.minRecordOutsideWindow.remainingBytes
		}
		logger.V(4).Infof("aged out candidates: count=%d iterElapsedMs=%dms outsideWindowMinRemainingBytes=%.2fMib remainingCandidates=%d", agedOut, record.elapsedMs, bytesToMiB(outsideWindowMin), len(sd.minCandidates))
	}

	// optimization: candidates larger than the current out-of-window min can never become relevant.
	if sd.minRecordOutsideWindow != nil && record.remainingBytes > sd.minRecordOutsideWindow.remainingBytes {
		logger.V(4).Infof("skipping candidate above outside-window min: remainingBytes=%.2fMib outsideWindowMin=%.2fMib ", bytesToMiB(record.remainingBytes), bytesToMiB(sd.minRecordOutsideWindow.remainingBytes))
		return
	}

	// optimization: candidates preceded by a smaller value.
	if len(sd.minCandidates) > 0 && record.remainingBytes >= sd.minCandidates[len(sd.minCandidates)-1].remainingBytes {
		logger.V(4).Infof("skipping candidate that is not a new minimum: remainingBytes=%.2fMib lastCandidateRemainingBytes=%.2fMib", bytesToMiB(record.remainingBytes), bytesToMiB(sd.minCandidates[len(sd.minCandidates)-1].remainingBytes))
		return
	}

	sd.minCandidates = append(sd.minCandidates, record)
	logger.V(4).Infof("added candidate minimum: iterElapsedMs=%dms remainingBytes=%.2fMib candidates=%d", record.elapsedMs, bytesToMiB(record.remainingBytes), len(sd.minCandidates))
}

func (sd *stallDetector) checkStallCondition(remainingBytes uint64, logger *log.FilteredLogger) bool {
	if sd.minRecordOutsideWindow == nil {
		logger.V(4).Infof("stall check skipped: no outside-window minimum yet, remainingBytes=%.2fMib", bytesToMiB(remainingBytes))
		return false
	}

	stallMargin := sd.options.StallDetectorOptions.StallMargin
	stallThreshold := uint64(float64(sd.minRecordOutsideWindow.remainingBytes) * (1 - stallMargin))
	stalled := remainingBytes >= stallThreshold
	logger.V(4).Infof("stall check result: remainingBytes=%.2fMib outsideWindowMinRemainingBytes=%.2fMib threshold=%.2fMib stalled=%t", bytesToMiB(remainingBytes), bytesToMiB(sd.minRecordOutsideWindow.remainingBytes), bytesToMiB(stallThreshold), stalled)
	return stalled
}

func (sd *stallDetector) findBestRemainingBytes(logger *log.FilteredLogger) uint64 {
	candidateValues := make([]float32, 0, len(sd.minCandidates)+1)
	candidateValues = append(candidateValues, bytesToMiB(sd.minRecordOutsideWindow.remainingBytes))
	for _, candidate := range sd.minCandidates {
		candidateValues = append(candidateValues, bytesToMiB(candidate.remainingBytes))
	}
	logger.V(4).Infof("findBestRemainingBytes candidates (Mib): %v", candidateValues)

	bestRemainingBytes := sd.minRecordOutsideWindow.remainingBytes
	for _, candidate := range sd.minCandidates {
		if candidate.remainingBytes < bestRemainingBytes {
			bestRemainingBytes = candidate.remainingBytes
		}
	}
	return bestRemainingBytes
}

func (sd *stallDetector) initializeRelaxationState(record iterationRecord, logger *log.FilteredLogger) {
	stallProgressTimeout := sd.options.StallDetectorOptions.StallProgressTimeout
	sd.remainingBytesHistory = utilheap.NewMin[uint64]()
	sd.relaxationPatienceMs = uint64(stallProgressTimeout) * 1000
	sd.relaxationDeadlineMs = record.elapsedMs + sd.relaxationPatienceMs
	logger.V(4).Infof("initialized relaxation state: iterElapsedMs=%dms patienceMs=%dms deadlineMs=%dms", record.elapsedMs, sd.relaxationPatienceMs, sd.relaxationDeadlineMs)
}

func (sd *stallDetector) relaxBestRemainingBytes(record iterationRecord, logger *log.FilteredLogger) {
	sd.remainingBytesHistory.Push(record.remainingBytes)
	if record.elapsedMs < sd.relaxationDeadlineMs || sd.remainingBytesHistory.Len() == 0 {
		logger.V(4).Infof("relaxation not due: iterElapsedMs=%dms deadlineMs=%dms historyLen=%d", record.elapsedMs, sd.relaxationDeadlineMs, sd.remainingBytesHistory.Len())
		return
	}

	nextCandidate, exists := sd.remainingBytesHistory.Pop()
	if !exists {
		// should never happen
		logger.Error("failed to pop remaining bytes history")
		return
	}

	oldBest := sd.bestRemainingBytes
	sd.bestRemainingBytes = nextCandidate
	sd.relaxationPatienceMs = sd.relaxationPatienceMs / 2
	sd.relaxationDeadlineMs = record.elapsedMs + sd.relaxationPatienceMs
	logger.V(3).Infof("relaxed best remaining bytes: oldBest=%.2fMib newBest=%.2fMib iterElapsedMs=%dms nextPatienceMs=%dms nextDeadlineMs=%dms", bytesToMiB(oldBest), bytesToMiB(sd.bestRemainingBytes), record.elapsedMs, sd.relaxationPatienceMs, sd.relaxationDeadlineMs)
}

func (sd *stallDetector) canFinishByDeadline(elapsedSeconds int64, deadlineSeconds int64, estimatedDowntimeMs uint32, logger *log.FilteredLogger) bool {
	if sd.ewmaBandwidthBps == 0 {
		logger.V(3).Info("bandwidth data unavailable, cannot estimate migration completion")
		return false
	}
	remainingBudgetMs := (deadlineSeconds - elapsedSeconds) * 1000
	logger.V(4).Infof("canFinishByDeadline: elapsedSeconds=%ds deadlineSeconds=%ds estimatedDowntimeMs=%dms remainingBudgetMs=%dms", elapsedSeconds, deadlineSeconds, estimatedDowntimeMs, remainingBudgetMs)
	return int64(estimatedDowntimeMs) <= remainingBudgetMs
}

func (sd *stallDetector) estimateDowntimeMs(record iterationRecord, logger *log.FilteredLogger) uint32 {
	if sd.ewmaBandwidthBps == 0 {
		return 0
	}
	bandwidthBpms := sd.ewmaBandwidthBps / 1000
	// Note: when calculated from the polling loop, this is (probably) an overestimate. This is not
	//  a problem since this estimated downtime value is only used to compare to competition timeouts, which
	//  are typically far larger.
	estimatedDowntime := float64(record.remainingBytes) / bandwidthBpms
	logger.V(4).Infof("estimatedDowntime: %.1fms, remainingBytes: %.2fMib, bandwidthBpms: %.2fbps", estimatedDowntime, bytesToMiB(record.remainingBytes), bandwidthBpms)
	if estimatedDowntime > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(estimatedDowntime)
}

func (sd *stallDetector) isAtLocalMinima(record iterationRecord, logger *log.FilteredLogger) bool {
	stallMargin := sd.options.StallDetectorOptions.StallMargin
	target := uint64(float64(sd.bestRemainingBytes) * (1 + stallMargin))
	atLocalMinima := record.remainingBytes <= target
	logger.V(4).Infof("switch margin check: remainingBytes=%.2fMib bestRemainingBytes=%.2fMib margin=%.2f targetRemainingBytes=%.2f atLocalMinima=%t", bytesToMiB(record.remainingBytes), bytesToMiB(sd.bestRemainingBytes), stallMargin, bytesToMiB(target), atLocalMinima)
	return atLocalMinima
}

func (sd *stallDetector) processStallDetectionIteration(record iterationRecord, logger *log.FilteredLogger) bool {
	if sd.ewmaBandwidthBps == 0 {
		logger.V(4).Infof("skipping stall-detection iteration due to missing stats or improper configuration: ewmaBandwidthBps=%.2fbps", sd.ewmaBandwidthBps)
		return false
	}
	if sd.switchoverInitiated {
		logger.V(4).Info("skipping stall-detection iteration because switchover action was already triggered.")
		return false
	}

	logger.V(4).Infof("processing stall-detection iteration: iterElapsedMs=%dms remainingBytes=%.2fMib currentEwmaBps=%.2fbps", record.elapsedMs, bytesToMiB(record.remainingBytes), sd.ewmaBandwidthBps)

	sd.updateCandidates(record, logger)

	if sd.stallDetected {
		sd.relaxBestRemainingBytes(record, logger)
		return true
	} else if sd.checkStallCondition(record.remainingBytes, logger) {
		// when stall is first detected initialize stall-related state
		sd.bestRemainingBytes = sd.findBestRemainingBytes(logger)
		sd.initializeRelaxationState(record, logger)
		sd.stallDetected = true
		logger.V(3).Infof("stall detected: bestRemainingBytes=%.2fMib outsideWindowMin=%.2fMib candidates=%d", bytesToMiB(sd.bestRemainingBytes), bytesToMiB(sd.minRecordOutsideWindow.remainingBytes), len(sd.minCandidates))
		return true
	} else {
		logger.V(4).Info("stall not detected yet; continuing monitoring")
		return false
	}
}

func newMigrationMonitor(vmi *v1.VirtualMachineInstance, l *LibvirtDomainManager, options *cmdclient.MigrationOptions, migrationErr chan error) *migrationMonitor {
	monitor := &migrationMonitor{
		l:                        l,
		vmi:                      vmi,
		options:                  options,
		migrationErr:             migrationErr,
		iterationCh:              make(chan int, 16),
		logger:                   log.Log.Object(vmi),
		progressWatermark:        0,
		remainingData:            0,
		progressTimeout:          options.ProgressTimeout,
		switchOverDeadline:       options.CompletionTimeoutPerGiB * getVMIMigrationDataSize(vmi, l.ephemeralDiskDir),
		acceptableCompletionTime: options.CompletionTimeoutPerGiB * getVMIMigrationDataSize(vmi, l.ephemeralDiskDir),
		stallDetectionEnabled:    options.StallDetectionEnabled,
		stallDetector: &stallDetector{
			options: options,
		},
	}
	monitor.logger.V(3).Infof(
		"initialized migration monitor: stallDetection=%t progressTimeout=%ds completionTimeoutPerGiB=%d maxDowntimeMs=%d allowPostCopy=%t allowWorkloadDisruption=%t "+
			"stallMargin=%.2f stallProgressTimeout=%ds switchoverTimeout=%ds preCopyPossibleFactor=%.2f bandwidthEWMAAlpha=%.2f searchLocalMinima=%t completionTimeoutFactor=%.2f",
		options.StallDetectionEnabled,
		options.ProgressTimeout,
		options.CompletionTimeoutPerGiB,
		options.MaxDowntime,
		options.AllowPostCopy,
		options.AllowWorkloadDisruption,
		options.StallDetectorOptions.StallMargin,
		options.StallDetectorOptions.StallProgressTimeout,
		options.StallDetectorOptions.SwitchoverTimeout,
		options.StallDetectorOptions.PrecopyPossibleFactor,
		options.StallDetectorOptions.EwmaAlpha,
		options.StallDetectorOptions.SearchLocalMinima,
		options.StallDetectorOptions.CompletionTimeoutFactor,
	)
	// TODO: this limitation is actively being worked on; remove when resolved. ETA: QEMU 11.1
	if options.StallDetectionEnabled && virtutil.IsVFIOVMI(vmi) {
		monitor.logger.Warning("VFIO VM detected: QEMU remaining-bytes signals may under-report outstanding migration data for VFIO devices. This is a known limitation.")
	}

	return monitor
}

func (m *migrationMonitor) isMigrationPostCopy() bool {
	migration, _ := m.l.metadataCache.Migration.Load()
	return migration.Mode == v1.MigrationPostCopy
}

func (m *migrationMonitor) isPausedMigration() bool {
	migration, _ := m.l.metadataCache.Migration.Load()
	return migration.Mode == v1.MigrationPaused
}

func (m *migrationMonitor) shouldTriggerTimeout(elapsedNs int64, logger *log.FilteredLogger) bool {
	if m.acceptableCompletionTime == 0 {
		return false
	}

	elapsedSeconds := elapsedNs / int64(time.Second)
	if m.isPausedMigration() {
		logger.V(4).Infof("shouldTriggerTimeout: elapsedSeconds=%ds acceptableCompletionTime=%ds paused=true", elapsedSeconds, m.acceptableCompletionTime)
		return elapsedSeconds > m.acceptableCompletionTime
	} else {
		logger.V(4).Infof("shouldTriggerTimeout: elapsedSeconds=%ds switchOverDeadline=%ds paused=false", elapsedSeconds, m.switchOverDeadline)
		return elapsedSeconds > m.switchOverDeadline
	}
}

func (m *migrationMonitor) shouldAssistMigrationToComplete(elapsedNs int64, logger *log.FilteredLogger) bool {
	return m.options.AllowWorkloadDisruption && m.shouldTriggerTimeout(elapsedNs, logger) && !m.stallDetectionEnabled
}

func (m *migrationMonitor) scaledCompletionDeadlineSeconds(baseSeconds int64) int64 {
	m.logger.V(4).Infof("scaledCompletionDeadlineSeconds: baseSeconds=%ds, completionTimeoutFactor=%f", baseSeconds, m.options.StallDetectorOptions.CompletionTimeoutFactor)
	return int64(float64(baseSeconds) * m.options.StallDetectorOptions.CompletionTimeoutFactor)
}

func (m *migrationMonitor) isMigrationProgressing(logger *log.FilteredLogger) bool {
	now := time.Now().UTC().UnixNano()

	// check if the migration is progressing
	progressDelay := (now - m.lastProgressUpdate) / int64(time.Second)
	if m.progressTimeout != 0 && progressDelay > m.progressTimeout {
		logger.Warningf("live migration stuck for %d seconds", progressDelay)
		return false
	}

	return true
}

func (m *migrationMonitor) processCompletionTimeouts(dom cli.VirDomain, elapsedNs int64, estimatedDowntimeMs uint32, logger *log.FilteredLogger) *inflightMigrationAborted {
	sd := m.stallDetector

	if !m.shouldTriggerTimeout(elapsedNs, logger) {
		return nil
	}

	if m.isMigrationPostCopy() {
		return nil
	}

	if sd.ewmaBandwidthBps == 0 {
		// In a typical migration, this case should not be possible.
		logger.Error("aborting migration due to illegal state: value of ewmaBandwidthBps not set!")
		if err := dom.AbortJob(); err != nil {
			logger.Reason(err).Error("failed to abort migration")
			return nil
		}
		return &inflightMigrationAborted{
			message:     fmt.Sprintf("Migration entered an illegal state and was aborted."),
			abortStatus: v1.MigrationAbortSucceeded,
		}
	}

	elapsedSeconds := elapsedNs / int64(time.Second)

	if !m.stallDetector.switchoverInitiated {

		// safety guard that protects against triggering a switch-over during a network drop
		completable := sd.canFinishByDeadline(elapsedSeconds, m.scaledCompletionDeadlineSeconds(m.acceptableCompletionTime), estimatedDowntimeMs, logger)

		if m.options.AllowPostCopy && !virtutil.IsVFIOVMI(m.vmi) && completable {
			logger.Info("completion timeout reached: starting post-copy mode to force convergence")
			if err := dom.MigrateStartPostCopy(0); err != nil {
				logger.Reason(err).Error("failed to start post-copy migration")
				return nil
			}
			m.l.updateVMIMigrationMode(v1.MigrationPostCopy)
			sd.switchoverInitiated = true
			return nil
		}
		switchoverTimeout := sd.options.StallDetectorOptions.SwitchoverTimeout
		if m.options.AllowWorkloadDisruption && completable {
			logger.Infof("completion timeout reached: setting max downtime to %dms to force switchover", migrationutils.QEMUMaxMigrationDowntimeMS)
			if err := dom.MigrateSetMaxDowntime(migrationutils.QEMUMaxMigrationDowntimeMS, 0); err != nil {
				logger.Reason(err).Error("setting max downtime failed")
			}
			m.acceptableCompletionTime = m.scaledCompletionDeadlineSeconds(m.acceptableCompletionTime)
			m.switchOverDeadline = elapsedSeconds + int64(switchoverTimeout)
			sd.switchoverInitiated = true
			return nil
		}

	}

	logger.Infof("aborting migration due to completion timeout: elapsedSec=%ds acceptableCompletionSec=%ds", elapsedSeconds, m.acceptableCompletionTime)
	if err := dom.AbortJob(); err != nil {
		logger.Reason(err).Error("failed to abort migration")
		return nil
	}
	return &inflightMigrationAborted{
		message:     fmt.Sprintf("Live migration is not completed after %d seconds and has been aborted", elapsedSeconds),
		abortStatus: v1.MigrationAbortSucceeded,
	}
}

func (m *migrationMonitor) triggerConvergenceAction(dom cli.VirDomain, action convergenceAction, reason string, logger *log.FilteredLogger) *inflightMigrationAborted {
	sd := m.stallDetector

	sd.switchoverInitiated = true

	switch action {
	case actionNothing:
		sd.switchoverInitiated = false
		logger.V(3).Infof("convergence action is nothing because: %s", reason)
		return nil
	case actionAbort:
		logger.Warningf("aborting migration: %s", reason)
		if err := dom.AbortJob(); err != nil {
			sd.switchoverInitiated = false
			logger.Reason(err).Error("failed to abort migration")
			return nil
		}
		return &inflightMigrationAborted{
			message:     fmt.Sprintf("Migration aborted: %s", reason),
			abortStatus: v1.MigrationAbortSucceeded,
		}
	case actionPostCopy:
		logger.Infof("starting post copy mode for migration: %s", reason)
		if err := dom.MigrateStartPostCopy(0); err != nil {
			sd.switchoverInitiated = false
			logger.Reason(err).Error("failed to start post migration")
			return nil
		}
		m.l.updateVMIMigrationMode(v1.MigrationPostCopy)
		return nil
	case actionHardStopAndCopy, actionSoftStopAndCopy:
		now := time.Now().UTC().UnixNano()
		elapsedSeconds := (now - m.start) / int64(time.Second)
		switchoverTimeout := sd.options.StallDetectorOptions.SwitchoverTimeout

		// since stop-and-copy is not guaranteed to start immediately (or ever), a "switch-over" deadline is needed
		m.switchOverDeadline = elapsedSeconds + int64(switchoverTimeout)

		var downtime uint64
		if action == actionHardStopAndCopy {
			downtime = migrationutils.QEMUMaxMigrationDowntimeMS
			logger.Infof("forcing switchover by setting max downtime to %dms: %s", downtime, reason)
		} else {
			downtime = m.options.MaxDowntime
			logger.Infof("max downtime set to %dms: %s", downtime, reason)
		}

		if err := dom.MigrateSetMaxDowntime(downtime, 0); err != nil {
			sd.switchoverInitiated = false
			logger.Reason(err).Error("setting max downtime failed")
		}
		return nil

	default:
		logger.Error("unknown convergence action")
		return nil
	}
}

// reconcile pause state (i.e. when QEMU triggers its internal switchover, update KubeVirt's state
// to reflect that the VM is now paused)
func (m *migrationMonitor) reconcilePauseState(dom cli.VirDomain, logger *log.FilteredLogger) {
	migrationState, stateReason, err := dom.GetState()
	if err != nil {
		logger.Reason(err).Error("failed to get migration state")
		return
	}
	logger.V(4).Infof("current migration state=%d and stateReason=%d", migrationState, stateReason)
	// The "!m.isMigrationPostCopy()" may seem redundant since in theory a post-copy VM should never report paused
	// reason as DOMAIN_PAUSED_MIGRATION. However, since QEMU itself does NOT make the DOMAIN_PAUSED_MIGRATION v.s.
	// DOMAIN_PAUSED_POSTCOPY distinction, LibVirt relies on internal state to determine which reason to use. This
	// internal state, however, can briefly be stale since LibVirt does not internally update it until QEMU itself
	// reports the VM has entered post-copy.
	if !m.isPausedMigration() && !m.isMigrationPostCopy() &&
		migrationState == libvirt.DOMAIN_PAUSED &&
		stateReason == int(libvirt.DOMAIN_PAUSED_MIGRATION) {
		logger.V(3).Infof("reconciling VM pause state")
		m.l.paused.add(m.vmi.UID)
		m.l.updateVMIMigrationMode(v1.MigrationPaused)
	}
}

func (m *migrationMonitor) decideAction(record iterationRecord, estimatedDowntimeMs uint32, logger *log.FilteredLogger) (convergenceAction, string) {

	sd := m.stallDetector
	searchLocalMinima := sd.options.StallDetectorOptions.SearchLocalMinima

	if sd.switchoverInitiated {
		return actionNothing, "switchover already initiated"
	}

	if !sd.isAtLocalMinima(record, logger) && searchLocalMinima {
		return actionNothing, "not at a local minima yet"
	}

	var localMinLogMessage string
	if !searchLocalMinima {
		localMinLogMessage = "local minima search skipped: "
	} else {
		localMinLogMessage = "arrived at a local minima: "
	}
	logger.V(4).Infof(localMinLogMessage+"iterElapsedMs=%dms remainingBytes=%.2fMib bestRemainingBytes=%.2fMib impliedDowntimeMs=%dms maxDowntimeMs=%dms allowPostCopy=%t allowWorkloadDisruption=%t",
		m.iterationRecord.elapsedMs,
		bytesToMiB(record.remainingBytes),
		bytesToMiB(sd.bestRemainingBytes),
		estimatedDowntimeMs,
		m.options.MaxDowntime,
		m.options.AllowPostCopy,
		m.options.AllowWorkloadDisruption,
	)

	now := time.Now().UTC().UnixNano()
	elapsedSeconds := (now - m.start) / int64(time.Second)

	// usually this case can only be triggered by a sudden network drop unless acceptableCompletitionTime is very small
	if !sd.canFinishByDeadline(elapsedSeconds, m.scaledCompletionDeadlineSeconds(m.acceptableCompletionTime), estimatedDowntimeMs, logger) {
		return actionNothing, fmt.Sprintf("current estimated downtime (%dms) exceeds timeout budget by over x%.2f times", estimatedDowntimeMs, m.options.StallDetectorOptions.CompletionTimeoutFactor)
	}

	if m.options.AllowWorkloadDisruption && m.options.AllowPostCopy && !virtutil.IsVFIOVMI(m.vmi) {
		return actionPostCopy, fmt.Sprintf("estimated downtime %dms is a local minima", estimatedDowntimeMs)
	}

	if m.options.AllowWorkloadDisruption {
		return actionHardStopAndCopy, fmt.Sprintf("estimated downtime %dms is a local minima", estimatedDowntimeMs)
	}

	preCopyPossibleFactor := m.options.StallDetectorOptions.PrecopyPossibleFactor
	maxDowntimeMs := m.options.MaxDowntime
	if float64(estimatedDowntimeMs) <= float64(maxDowntimeMs) {
		return actionSoftStopAndCopy, fmt.Sprintf("estimated downtime %dms within max allowed downtime %dms", estimatedDowntimeMs, maxDowntimeMs)
	} else if float64(estimatedDowntimeMs) <= float64(maxDowntimeMs)*preCopyPossibleFactor {
		return actionSoftStopAndCopy, fmt.Sprintf("estimated downtime %dms within tolerable factor %.2fx to max allowed downtime %dms", estimatedDowntimeMs, preCopyPossibleFactor, maxDowntimeMs)
	}

	return actionAbort, fmt.Sprintf("estimated downtime %dms exceeds max allowed downtime %dms by a factor of more than x%.2f", estimatedDowntimeMs, maxDowntimeMs, preCopyPossibleFactor)
}

func (m *migrationMonitor) handleStallDetection(dom cli.VirDomain, stats *libvirt.DomainJobInfo, elapsedNs int64, isIterationBoundary bool, logger *log.FilteredLogger) *inflightMigrationAborted {

	// This stall detection mechanism implements VEP 248. In each iteration, pre-copy tries to transfer VM state data (i.e.
	// memory) from source to target. Multiple iterations are required because as the VM transfers data it is
	// actively dirtying new memory. For high-dirty rate VMs with a large writable working set, we would never
	// converge. Stall detection tracks how many bytes are left and if with in a progress timeout window we make
	// little to no progress we are stalled. Then the goal is to manually force trigger switch-over at a local minima
	// of remaining bytes. See VEP for more details.

	sd := m.stallDetector

	if !sd.initialMaxDowntimeSet {
		initialMaxDowntime := uint64(m.options.MaxDowntime)
		if initialMaxDowntime > migrationutils.QEMUDefaultTargetDowntimeMS {
			initialMaxDowntime = migrationutils.QEMUDefaultTargetDowntimeMS
		}
		if err := dom.MigrateSetMaxDowntime(initialMaxDowntime, 0); err != nil {
			logger.Reason(err).Warning("failed to set initial max downtime")
		}
		sd.initialMaxDowntimeSet = true
	}

	m.reconcilePauseState(dom, logger)

	if stats.DataRemainingSet && stats.TimeElapsedSet && stats.MemIterationSet {
		// the value in m.iterationRecord is accurate only when (1) we are the start an iteration or (2) if the
		//  VM is paused or (3) if the VM is in post-copy.
		if isIterationBoundary {
			logger.V(4).Info("processing migration iteration boundary for stall detection")
			m.iterationRecord.remainingBytes = stats.DataRemaining
			m.iterationRecord.elapsedMs = stats.TimeElapsed
			m.iterationRecord.iterationNumber = stats.MemIteration
			if stalled := sd.processStallDetectionIteration(m.iterationRecord, logger); stalled {
				estimatedDowntimeMs := sd.estimateDowntimeMs(m.iterationRecord, logger)
				action, reason := m.decideAction(m.iterationRecord, estimatedDowntimeMs, logger)
				if aborted := m.triggerConvergenceAction(dom, action, reason, logger); aborted != nil {
					return aborted
				}
			}
		} else if m.isPausedMigration() || m.isMigrationPostCopy() {
			m.iterationRecord.remainingBytes = stats.DataRemaining
			m.iterationRecord.elapsedMs = stats.TimeElapsed
			m.iterationRecord.iterationNumber = stats.MemIteration
		} else if stats.MemBpsSet {
			sd.updateBandwidthEstimate(stats.MemBps, logger)
		}
	} else {
		logger.V(3).Infof("skipping actions for stall detection due to missing stats data: DataRemainingSet=%t, TimeElapsedSet=%t, MemBpsSet=%t", stats.DataRemainingSet, stats.TimeElapsedSet, stats.MemBpsSet)
	}

	estimatedDowntimeMs := sd.estimateDowntimeMs(m.iterationRecord, logger)
	if aborted := m.processCompletionTimeouts(dom, elapsedNs, estimatedDowntimeMs, logger); aborted != nil {
		return aborted
	}
	return nil
}

func (m *migrationMonitor) handleLegacyConvergence(dom cli.VirDomain, stats *libvirt.DomainJobInfo, elapsedNs int64, logger *log.FilteredLogger) *inflightMigrationAborted {
	now := m.start + elapsedNs

	// TODO: to be removed once MigrationStallDetection graduates
	switch {
	case m.isMigrationPostCopy():
		// Currently, there is nothing for us to track when in Post Copy mode.
		// The reasoning here is that post copy migrations transfer the state
		// directly to the target pod in a way that results in the target pod
		// hosting the active workload while the migration completes.

		// If we were to abort the migration due to a timeout while in post copy,
		// then it would result in that active state being lost.

	case m.shouldAssistMigrationToComplete(elapsedNs, logger) && !m.isPausedMigration():
		if m.options.AllowPostCopy && !virtutil.IsVFIOVMI(m.vmi) {
			logger.Info("Starting post copy mode for migration")
			err := dom.MigrateStartPostCopy(0)
			if err != nil {
				logger.Reason(err).Error("failed to start post migration")
				return nil
			}
			m.l.updateVMIMigrationMode(v1.MigrationPostCopy)
		} else if virtutil.IsVFIOVMI(m.vmi) {
			logger.Info("Setting large max downtime to trigger migration switchover")
			// TODO: once the VGPULiveMigration featuregate graduates
			//  (and even possibly other VFIO live migration featuregates)
			//  we should consider merging this with the "else" case below.
			// Setting a very high max downtime causes QEMU to
			//  trigger its internal switchover, which pauses vCPUs and
			//  transitions VFIO devices to _STOP_COPY. This is more
			//  correct than dom.Suspend() which only pauses vCPUs but
			//  leaves VFIO devices in _RUNNING with perpetual dirty
			//  page reporting.
			maxDowntimeSec := m.acceptableCompletionTime * 2
			// qemu doesn't allow max downtime larger than 2000s
			err := dom.MigrateSetMaxDowntime(min(uint64(maxDowntimeSec)*1000, migrationutils.QEMUMaxMigrationDowntimeMS), 0)
			if err != nil {
				logger.Reason(err).Error("Setting max downtime failed.")
				return nil
			}
			logger.Infof("Set max downtime to %ds for %s", maxDowntimeSec, m.vmi.GetObjectMeta().GetName())

			m.acceptableCompletionTime = maxDowntimeSec
			m.l.paused.add(m.vmi.UID)
			m.l.updateVMIMigrationMode(v1.MigrationPaused)
		} else {
			logger.Info("Pausing the guest to allow migration to complete")
			err := dom.Suspend()
			if err != nil {
				logger.Reason(err).Error("Signalling suspension failed.")
				return nil
			}
			logger.Infof("Signaled pause for %s", m.vmi.GetObjectMeta().GetName())

			// update acceptableCompletionTime to prevent premature migration
			// cancellation
			m.acceptableCompletionTime *= 2
			m.l.paused.add(m.vmi.UID)
			m.l.updateVMIMigrationMode(v1.MigrationPaused)
		}
	case !m.isMigrationProgressing(logger):
		// The migration is completely stuck.
		// It usually indicates a problem with the network or qemu's connection handling.
		// In this case, we abort the migration directly without trying to pause/post-copy,
		// since the problem is highly unlikely to be caused by a high dirty rate.
		err := dom.AbortJob()
		if err != nil {
			logger.Reason(err).Error("failed to abort migration")
			return nil
		}

		progressDelay := now - m.lastProgressUpdate
		aborted := &inflightMigrationAborted{}
		aborted.message = fmt.Sprintf("Live migration stuck for %d seconds and has been aborted", progressDelay/int64(time.Second))
		aborted.abortStatus = v1.MigrationAbortSucceeded
		return aborted
	case m.shouldTriggerTimeout(elapsedNs, logger):
		// check the overall migration time
		// if the total migration time exceeds an acceptable
		// limit, then the migration will get aborted, but
		// only if post copy migration hasn't been enabled
		err := dom.AbortJob()
		if err != nil {
			logger.Reason(err).Error("failed to abort migration")
			return nil
		}

		aborted := &inflightMigrationAborted{}
		aborted.message = fmt.Sprintf("Live migration is not completed after %d seconds and has been aborted", m.acceptableCompletionTime)
		aborted.abortStatus = v1.MigrationAbortSucceeded
		return aborted
	}

	return nil
}

func (m *migrationMonitor) processInflightMigration(dom cli.VirDomain, stats *libvirt.DomainJobInfo, isIterationBoundary bool, logger *log.FilteredLogger) *inflightMigrationAborted {
	// Migration is running
	now := time.Now().UTC().UnixNano()
	elapsedNs := now - m.start

	m.l.domainInfoStats = statsconv.Convert_libvirt_DomainJobInfo_To_stats_DomainJobInfo(stats)
	if (m.progressWatermark == 0) || (m.remainingData < m.progressWatermark) {
		m.lastProgressUpdate = now
	}
	m.progressWatermark = m.remainingData

	if m.stallDetectionEnabled {
		return m.handleStallDetection(dom, stats, elapsedNs, isIterationBoundary, logger)
	} else {
		// TODO: to be removed once stall detection graduates
		return m.handleLegacyConvergence(dom, stats, elapsedNs, logger)
	}
}

func (m *migrationMonitor) registerIterationCallback(domName string) (int, error) {
	return m.l.virConn.DomainEventMigrationIterationRegister(func(_ *libvirt.Connect, domain *libvirt.Domain, event *libvirt.DomainEventMigrationIteration) {
		name, err := domain.GetName()
		if err != nil || name != domName {
			return
		}

		select {
		case m.iterationCh <- event.Iteration:
			m.logger.V(4).Infof("queued migration iteration event for iteration #%d", event.Iteration)
		default:
			m.logger.V(3).Infof("dropped migration iteration event for iteration #%d: reason=channel-full", event.Iteration)
		}
	})
}

type monitorEvent struct {
	err              error
	iteration        int
	isIterationBound bool
	source           string
}

func (m *migrationMonitor) nextMonitorEvent() monitorEvent {
	ev := monitorEvent{iteration: int(m.iterationRecord.iterationNumber)}
	select {
	case err, ok := <-m.migrationErr:
		if !ok {
			ev.err = fmt.Errorf("migration channel closed")
		} else {
			ev.err = err
		}
		ev.source = "err"
	case iter := <-m.iterationCh:
		ev.iteration = iter
		ev.isIterationBound = true
		ev.source = "event"
	case <-time.After(monitorSleepPeriodMS * time.Millisecond):
		ev.source = "poll"
	}
	return ev
}

func (m *migrationMonitor) startMonitor() {
	vmi := m.vmi

	m.start = time.Now().UTC().UnixNano()
	m.lastProgressUpdate = m.start

	defer func() {
		m.l.domainInfoStats = &stats.DomainJobInfo{}
	}()

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := m.l.virConn.LookupDomainByName(domName)

	if err != nil {
		m.logger.Reason(err).Error(liveMigrationFailed)
		m.l.setMigrationResult(true, fmt.Sprintf("%v", err), "")
		return
	}
	defer dom.Free()

	if m.stallDetectionEnabled {
		registrationID, registerErr := m.registerIterationCallback(domName)
		if registerErr != nil {
			m.logger.Reason(registerErr).Error("failed to register migration iteration callback, falling back to legacy stall handling")
			m.stallDetectionEnabled = false
		} else {
			m.logger.V(3).Infof("registered migration iteration callback: registrationID=%d", registrationID)
			defer func() {
				if err := m.l.virConn.DomainEventDeregister(registrationID); err != nil {
					m.logger.Reason(err).V(3).Info("failed to deregister migration iteration callback")
				}
			}()
		}
	}

	logInterval := 0

	for {
		ev := m.nextMonitorEvent()

		loopLogger := log.Log.Object(vmi).With("source", ev.source).With("iteration", ev.iteration)

		if ev.err != nil {
			loopLogger.Reason(ev.err).Error(liveMigrationFailed)
			var abortStatus v1.MigrationAbortStatus
			if strings.Contains(ev.err.Error(), "canceled by client") {
				abortStatus = v1.MigrationAbortSucceeded
			}
			m.l.setMigrationResult(true, fmt.Sprintf("Live migration failed %v", ev.err), abortStatus)
			if ev.err.Error() == "migration channel closed" {
				return
			}
			return
		}

		jobStats, err := dom.GetJobStats(0)
		if err != nil {
			loopLogger.Reason(err).Info("failed to get domain job info, will retry")
			continue
		}

		if jobStats.DataRemainingSet {
			m.remainingData = jobStats.DataRemaining
		}

		uid := MigrationUID(vmi)
		switch jobStats.Type {
		case libvirt.DOMAIN_JOB_UNBOUNDED:
			aborted := m.processInflightMigration(dom, jobStats, ev.isIterationBound, loopLogger)
			if aborted != nil {
				loopLogger.Errorf("Live migration abort detected with reason: %s", aborted.message)
				m.l.setMigrationResult(true, aborted.message, aborted.abortStatus)
				return
			}
			if !ev.isIterationBound {
				logInterval++
			}
			if logInterval%monitorLogInterval == 0 || ev.isIterationBound {
				LogMigrationInfo(loopLogger, uid, jobStats)
			}
		case libvirt.DOMAIN_JOB_NONE:
			loopLogger.Info("Migration job is not active")
		}
	}
}

func MigrationUID(vmi *v1.VirtualMachineInstance) types.UID {
	if s := vmi.Status.MigrationState; s != nil {
		if s.SourceState != nil {
			return s.SourceState.MigrationUID
		}
		return s.MigrationUID
	}
	return ""
}

// LogMigrationInfo logs the same migration info as `virsh -r domjobinfo`
func LogMigrationInfo(logger *log.FilteredLogger, uid types.UID, info *libvirt.DomainJobInfo) {
	bToMiB := func(bytes uint64) uint64 {
		return bytes / 1024 / 1024
	}

	bpsToMbps := func(bytes uint64) uint64 {
		return bytes * 8 / 1000000
	}

	// For completed jobs, Downtime is the final downtime, DowntimeNet contains the actual network overhead during the cutover
	downtimeInfo := fmt.Sprintf("ExpectedDowntime:%dms", info.Downtime)
	if info.DowntimeNetSet && info.DowntimeNet > 0 {
		downtimeInfo = fmt.Sprintf("Downtime:%dms DowntimeNet:%dms", info.Downtime, info.DowntimeNet)
	}

	logger.V(2).Info(fmt.Sprintf(`Migration info for %s: TimeElapsed:%dms DataProcessed:%dMiB DataRemaining:%dMiB DataTotal:%dMiB `+
		`MemoryProcessed:%dMiB MemoryRemaining:%dMiB MemoryTotal:%dMiB MemoryBandwidth:%dMbps DirtyRate:%dMbps `+
		`Iteration:%d PostcopyRequests:%d ConstantPages:%d NormalPages:%d NormalData:%dMiB %s `+
		`DiskMbps:%d`,
		uid, info.TimeElapsed, bToMiB(info.DataProcessed), bToMiB(info.DataRemaining), bToMiB(info.DataTotal),
		bToMiB(info.MemProcessed), bToMiB(info.MemRemaining), bToMiB(info.MemTotal), bpsToMbps(info.MemBps), bpsToMbps(info.MemDirtyRate*info.MemPageSize),
		info.MemIteration, info.MemPostcopyReqs, info.MemConstant, info.MemNormal, bToMiB(info.MemNormalBytes), downtimeInfo,
		bpsToMbps(info.DiskBps),
	))
}

func (l *LibvirtDomainManager) asyncMigrationAbort(vmi *v1.VirtualMachineInstance) {
	go func(l *LibvirtDomainManager, vmi *v1.VirtualMachineInstance) {

		domName := api.VMINamespaceKeyFunc(vmi)
		dom, err := l.virConn.LookupDomainByName(domName)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Warning("failed to cancel migration, domain not found ")
			return
		}
		defer dom.Free()
		jobInfo, err := dom.GetJobInfo()
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("failed to get domain job info")
			return
		}
		if jobInfo.Type == libvirt.DOMAIN_JOB_UNBOUNDED {
			err := dom.AbortJob()
			if err != nil {
				log.Log.Object(vmi).Reason(err).Error("failed to cancel migration")
				l.setMigrationAbortStatus(v1.MigrationAbortFailed)
				return
			}
			l.setMigrationResult(true, "Live migration aborted ", v1.MigrationAbortSucceeded)
			log.Log.Object(vmi).Info("live migration abort succeeded")
		}
	}(l, vmi)
}

func generateDomainName(vmi *v1.VirtualMachineInstance) string {
	namespace := vmi.Namespace
	name := vmi.Name
	if vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.TargetState != nil &&
		vmi.Status.MigrationState.TargetState.DomainName != nil &&
		*vmi.Status.MigrationState.TargetState.DomainName != "" &&
		vmi.Status.MigrationState.TargetState.DomainNamespace != nil &&
		*vmi.Status.MigrationState.TargetState.DomainNamespace != "" {
		name = *vmi.Status.MigrationState.TargetState.DomainName
		namespace = *vmi.Status.MigrationState.TargetState.DomainNamespace
	}
	domainName := util.DomainFromNamespaceName(namespace, name)
	log.Log.Object(vmi).Infof("generated target domain name %s", domainName)
	return domainName
}

func updateFilePathsToNewDomain(vmi *v1.VirtualMachineInstance, domSpec *api.DomainSpec) {
	if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.TargetState != nil && vmi.Status.MigrationState.TargetState.DomainNamespace != nil {
		targetNS := *vmi.Status.MigrationState.TargetState.DomainNamespace
		// Modify the domain XML to update paths to the target volumes to match the new domain
		for i, disk := range domSpec.Devices.Disks {
			if disk.Source.DataStore != nil &&
				disk.Source.DataStore.Source != nil &&
				strings.Contains(disk.Source.DataStore.Source.File, vmi.Namespace) {
				oldPath := disk.Source.DataStore.Source.File
				domSpec.Devices.Disks[i].Source.DataStore.Source.File = strings.Replace(disk.Source.DataStore.Source.File, vmi.Namespace, targetNS, 1)
				log.Log.Object(vmi).V(4).Infof("Updated disk %s datastore backend path from %s to %s", disk.Alias.GetName(), oldPath, domSpec.Devices.Disks[i].Source.DataStore.Source.File)
			}
			if disk.Source.File != "" && strings.Contains(disk.Source.File, vmi.Namespace) {
				oldPath := disk.Source.File
				domSpec.Devices.Disks[i].Source.File = strings.Replace(disk.Source.File, vmi.Namespace, targetNS, 1)
				log.Log.Object(vmi).V(4).Infof("Updated disk %s source path from %s to %s", disk.Alias.GetName(), oldPath, domSpec.Devices.Disks[i].Source.File)
			}
			if bp := disksource.Resolve(domSpec.Devices.Disks[i]).BackendPath(); bp != "" {
				log.Log.Object(vmi).V(4).Infof("Paths of disk %s: %s", disk.Alias.GetName(), bp)
			}
		}
	}
}

func generateMigrationParams(dom cli.VirDomain, vmi *v1.VirtualMachineInstance, options *cmdclient.MigrationOptions, virtShareDir string, domSpec *api.DomainSpec) (*libvirt.DomainMigrateParameters, error) {
	bandwidth, err := vcpu.QuantityToMebiByte(options.Bandwidth)
	if err != nil {
		return nil, err
	}

	updateFilePathsToNewDomain(vmi, domSpec)
	xmlstr, err := migratableDomXML(dom, vmi, domSpec)
	if err != nil {
		return nil, err
	}
	if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.TargetState != nil &&
		vmi.Status.MigrationState.TargetState.VirtualMachineInstanceUID != nil {
		log.Log.Object(vmi).Infof("Replacing VMI UID %s with target VMI UID %s in the XML", vmi.UID, *vmi.Status.MigrationState.TargetState.VirtualMachineInstanceUID)
		// Replace all occurences of the VMI UID in the XML with the target UID.
		xmlstr = strings.ReplaceAll(xmlstr, string(vmi.UID), string(*vmi.Status.MigrationState.TargetState.VirtualMachineInstanceUID))
	}

	parallelMigrationSet, parallelMigrationThreads := shouldConfigureParallelMigration(options)

	key := migrationproxy.ConstructProxyKey(string(vmi.UID), migrationproxy.LibvirtDirectMigrationPort)
	migrURI := fmt.Sprintf("unix://%s", migrationproxy.SourceUnixFile(virtShareDir, key))
	log.Log.Object(vmi).V(5).Infof("migration URI: %s", migrURI)
	params := &libvirt.DomainMigrateParameters{
		URI:                    migrURI,
		URISet:                 true,
		Bandwidth:              bandwidth, // MiB/s
		BandwidthSet:           bandwidth > 0,
		DestXML:                xmlstr,
		DestXMLSet:             true,
		PersistXML:             xmlstr,
		PersistXMLSet:          true,
		ParallelConnectionsSet: parallelMigrationSet,
		ParallelConnections:    parallelMigrationThreads,
		DestName:               generateDomainName(vmi),
		DestNameSet:            true,
	}

	copyDisks := getDiskTargetsForMigration(dom, vmi)
	if len(copyDisks) != 0 {
		params.MigrateDisks = copyDisks
		params.MigrateDisksSet = true
		// add a socket for live block migration
		key := migrationproxy.ConstructProxyKey(string(vmi.UID), migrationproxy.LibvirtBlockMigrationPort)
		disksURI := fmt.Sprintf("unix://%s", migrationproxy.SourceUnixFile(virtShareDir, key))
		params.DisksURI = disksURI
		params.DisksURISet = true
		params.MigrateDisksDetectZeroesList = copyDisks
		params.MigrateDisksDetectZeroesSet = true
	}

	log.Log.Object(vmi).Infof("generated migration parameters: %+v", params)
	return params, nil
}

type getDiskVirtualSizeFuncType func(disk *libvirtxml.DomainDisk) (int64, error)

var getDiskVirtualSizeFunc getDiskVirtualSizeFuncType

func init() {
	getDiskVirtualSizeFunc = getDiskVirtualSize
}

// getDiskVirtualSize return the size of a local volume to migrate.
// See suggestion in: https://issues.redhat.com/browse/RHEL-4607
func getDiskVirtualSize(disk *libvirtxml.DomainDisk) (int64, error) {
	path, err := getDiskPathFromSource(disk.Source)
	if err != nil {
		return -1, err
	}
	info, err := osdisk.GetDiskInfo(path)
	if err != nil {
		return -1, err
	}
	return info.VirtualSize, nil
}

func getDiskPathFromSource(source *libvirtxml.DomainDiskSource) (string, error) {
	var path string
	if source == nil {
		return "", fmt.Errorf("empty source for the disk")
	}

	if source.DataStore != nil {
		if source.DataStore.Source == nil {
			return "", fmt.Errorf("disk has initialized datastore with no source")
		}
		source = source.DataStore.Source
	}

	switch {
	case source.File != nil:
		path = source.File.File
	case source.Block != nil:
		path = source.Block.Dev
	default:
		return "", fmt.Errorf("no path set")
	}

	return path, nil
}

func getDiskName(disk *libvirtxml.DomainDisk) string {
	if disk == nil {
		return ""
	}
	n := disk.Alias.Name
	if len(n) < 3 {
		return n
	}
	// Trim the ua- prefix
	if strings.HasPrefix(n, "ua-") {
		return n[3:]
	}
	return n
}

func getMigrateVolumeForCondition(vmi *v1.VirtualMachineInstance, condition func(info *v1.StorageMigratedVolumeInfo) bool) map[string]bool {
	res := make(map[string]bool)

	for _, v := range vmi.Status.MigratedVolumes {
		if v.SourcePVCInfo == nil || v.DestinationPVCInfo == nil {
			continue
		}
		if v.SourcePVCInfo.VolumeMode == nil || v.DestinationPVCInfo.VolumeMode == nil {
			continue
		}
		if condition(&v) {
			res[v.VolumeName] = true
		}
	}
	return res
}

func getFsSrcBlockDstVols(vmi *v1.VirtualMachineInstance) map[string]bool {
	return getMigrateVolumeForCondition(vmi, func(v *v1.StorageMigratedVolumeInfo) bool {
		if v == nil {
			return false
		}
		if *v.SourcePVCInfo.VolumeMode == k8sv1.PersistentVolumeFilesystem &&
			*v.DestinationPVCInfo.VolumeMode == k8sv1.PersistentVolumeBlock {
			return true
		}
		return false
	})
}

func getBlockSrcFsDstVols(vmi *v1.VirtualMachineInstance) map[string]bool {
	return getMigrateVolumeForCondition(vmi, func(v *v1.StorageMigratedVolumeInfo) bool {
		if v == nil {
			return false
		}
		if *v.SourcePVCInfo.VolumeMode == k8sv1.PersistentVolumeBlock &&
			*v.DestinationPVCInfo.VolumeMode == k8sv1.PersistentVolumeFilesystem {
			return true
		}
		return false
	})
}

// configureLocalDiskToMigrate modifies the domain XML for the volume migration. For example, it sets the slice to allow the migration to a destination
// volume with different size then the source, or it adjust the XML configuration when it migrates from a filesystem source to a block destination or
// vice versa.
func configureLocalDiskToMigrate(dom *libvirtxml.Domain, vmi *v1.VirtualMachineInstance) error {
	if dom.Devices == nil {
		return nil
	}

	migDisks := classifyVolumesForMigration(vmi)
	fsSrcBlockDstVols := getFsSrcBlockDstVols(vmi)
	blockSrcFsDstVols := getBlockSrcFsDstVols(vmi)
	hotplugVols := make(map[string]bool)

	for _, v := range vmi.Spec.Volumes {
		if storagetypes.IsHotplugVolume(&v) {
			hotplugVols[v.Name] = true
		}
	}

	for i, d := range dom.Devices.Disks {
		if d.Alias == nil {
			return fmt.Errorf("empty alias")
		}
		name := getDiskName(&d)
		if !migDisks.isLocalVolumeToMigrate(name) {
			continue
		}
		// Calculate the size of the volume to migrate
		size, err := getDiskVirtualSizeFunc(&d)
		if err != nil {
			return err
		}
		// Configure the slice to enable to migrate the volume to a destination with different size
		// See suggestion in: https://issues.redhat.com/browse/RHEL-4607
		var source *libvirtxml.DomainDiskSource
		if dom.Devices.Disks[i].Source.DataStore != nil && dom.Devices.Disks[i].Source.DataStore.Source != nil {
			source = dom.Devices.Disks[i].Source.DataStore.Source
		} else {
			source = dom.Devices.Disks[i].Source
		}

		if source.Slices == nil {
			source.Slices = &libvirtxml.DomainDiskSlices{
				Slices: []libvirtxml.DomainDiskSlice{
					{
						Type:   "storage",
						Offset: 0,
						Size:   uint(size),
					},
				}}
		}
		var path string
		_, hotplugVol := hotplugVols[name]
		// Adjust the XML configuration when it migrates from a filesystem source to a block destination or vice versa
		if _, ok := fsSrcBlockDstVols[name]; ok {
			log.Log.V(2).Infof("Replace filesystem source with block destination for volume %s", name)
			if hotplugVol {
				path = hotplugdisk.GetVolumeMountDir(name)
			} else {
				path = filepath.Join(string(filepath.Separator), "dev", name)
			}
			source.Block = &libvirtxml.DomainDiskSourceBlock{Dev: path}
			source.File = nil
		}
		if _, ok := blockSrcFsDstVols[name]; ok {
			log.Log.V(2).Infof("Replace block source with destination for volume %s", name)
			if hotplugVol {
				path = hotplugdisk.GetVolumeMountDir(name) + ".img"
			} else {
				path = filepath.Join(hostdisk.GetMountedHostDiskDir(name), "disk.img")
			}
			source.File = &libvirtxml.DomainDiskSourceFile{File: path}
			source.Block = nil
		}
	}

	return nil
}

func (l *LibvirtDomainManager) migrateHelper(vmi *v1.VirtualMachineInstance, options *cmdclient.MigrationOptions) error {

	var err error
	var params *libvirt.DomainMigrateParameters

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		return err
	}
	defer dom.Free()

	migratePaused, err := isDomainPaused(dom)
	if err != nil {
		return fmt.Errorf("failed to retrive domain state")
	}
	migrateFlags := generateMigrationFlags(vmi.IsBlockMigration(), migratePaused, options)

	// anything that modifies the domain needs to be performed with the domainModifyLock held
	// The domain params and unHotplug need to be performed in a critical section together.
	critSection := func() error {
		l.domainModifyLock.Lock()
		defer l.domainModifyLock.Unlock()

		if err := prepareDomainForMigration(l.virConn, dom); err != nil {
			return fmt.Errorf("error encountered during preparing domain for migration: %v", err)
		}
		domSpec, err := l.getDomainSpec(dom)
		if err != nil {
			return fmt.Errorf("failed to get domain spec: %v", err)
		}
		params, err = generateMigrationParams(dom, vmi, options, l.virtShareDir, domSpec)
		if err != nil {
			return fmt.Errorf("error encountered while generating migration parameters: %v", err)
		}

		return nil
	}
	err = critSection()
	if err != nil {
		return err
	}

	// initiate the live migration
	var dstURI string
	if vmitrait.IsNonRoot(vmi) {
		dstURI = fmt.Sprintf("qemu+unix:///session?socket=%s", migrationproxy.SourceUnixFile(l.virtShareDir, string(vmi.UID)))
	} else {
		dstURI = fmt.Sprintf("qemu+unix:///system?socket=%s", migrationproxy.SourceUnixFile(l.virtShareDir, string(vmi.UID)))
	}

	err = dom.MigrateToURI3(dstURI, params, migrateFlags)
	if err != nil {
		l.setMigrationResult(true, err.Error(), "")
		log.Log.Object(vmi).Errorf("migration failed with error: %v", err)
		return fmt.Errorf("error encountered during MigrateToURI3 libvirt api call: %v", err)
	}

	log.Log.Object(vmi).Info("migration completed successfully")
	l.setMigrationResult(false, "", "")

	return nil
}

// prepareDomainForMigration perform necessary operation
// on the source domain just before migration
func prepareDomainForMigration(virtConn cli.Connection, domain cli.VirDomain) error {
	return hotUnplugHostDevices(virtConn, domain)
}

func shouldImmediatelyFailMigration(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Annotations == nil {
		return false
	}

	_, shouldFail := vmi.Annotations[v1.FuncTestForceLauncherMigrationFailureAnnotation]
	return shouldFail
}

func (l *LibvirtDomainManager) migrate(vmi *v1.VirtualMachineInstance, options *cmdclient.MigrationOptions) {
	if shouldImmediatelyFailMigration(vmi) {
		log.Log.Object(vmi).Error("Live migration failed. Failure is forced by functional tests suite.")
		l.setMigrationResult(true, "Failed migration to satisfy functional test condition", "")
		return
	}

	migrationErrorChan := make(chan error, 1)
	defer close(migrationErrorChan)

	log.Log.Object(vmi).Infof("Initiating live migration.")
	if options.UnsafeMigration {
		log.Log.Object(vmi).Info("UNSAFE_MIGRATION flag is set, libvirt's migration checks will be disabled!")
	}

	// From here on out, any error encountered must be sent to the
	// migrationError channel which is processed by the liveMigrationMonitor
	// go routine.
	monitor := newMigrationMonitor(vmi, l, options, migrationErrorChan)
	go monitor.startMonitor()

	err := l.migrateHelper(vmi, options)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error(liveMigrationFailed)
		migrationErrorChan <- err
		return
	}

	log.Log.Object(vmi).Infof("Live migration succeeded.")
}

func (l *LibvirtDomainManager) updateVMIMigrationMode(mode v1.MigrationMode) {
	l.metadataCache.Migration.WithSafeBlock(func(migrationMetadata *api.MigrationMetadata, _ bool) {
		migrationMetadata.Mode = mode
	})
	log.Log.V(4).Infof("Migration mode set in metadata: %s", l.metadataCache.Migration.String())
}

func shouldConfigureParallelMigration(options *cmdclient.MigrationOptions) (shouldConfigure bool, threadsCount int) {
	if options == nil {
		return
	}
	if options.ParallelMigrationThreads == nil {
		return
	}

	shouldConfigure = true
	threadsCount = int(*options.ParallelMigrationThreads)
	return
}

func standardizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
