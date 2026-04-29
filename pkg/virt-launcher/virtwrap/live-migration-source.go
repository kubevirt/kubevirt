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
	"context"
	"encoding/xml"
	"errors"
	"fmt"
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

	virtwait "kubevirt.io/kubevirt/pkg/apimachinery/wait"
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

	stallMargin           float64 = 0.04
	preCopyPossibleFactor float64 = 1.5
	bandwidthEWMAAlpha    float64 = 0.5
)

type migrationDisks struct {
	shared         map[string]bool
	generated      map[string]bool
	localToMigrate map[string]bool
}

type iterationRecord struct {
	elapsedMs      uint64
	remainingBytes uint64
}

type migrationMonitor struct {
	l       *LibvirtDomainManager
	vmi     *v1.VirtualMachineInstance
	options *cmdclient.MigrationOptions

	migrationErr chan error
	iterationCh  chan int

	start int64
	// TODO: remove after MigrationStallDetection feature graduates
	lastProgressUpdate int64
	// TODO: remove after MigrationStallDetection feature graduates
	progressWatermark uint64
	remainingData     uint64

	progressTimeout          int64
	acceptableCompletionTime int64
	maxDowntime              int64
	stallDetectionEnabled    bool
	lastIterTimestamp        time.Time

	minCandidates          []iterationRecord
	minRecordOutsideWindow *iterationRecord
	stallDetected          bool
	remainingBytesHistory  *utilheap.Heap[uint64]
	bestRemainingBytes     uint64
	relaxationDeadlineMs   uint64
	relaxationPatienceMs   uint64
	ewmaBandwidthBps       float64
	switchoverInitiated    bool
	stallDetectedAtMs      uint64
	lastIterElapsedMs      uint64
	lastStallRemaining     uint64
	lastRecoveryLogMs      uint64
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
		if disk.Source.File != "" {
			log.Log.Infof("Updating disk %s source file from %s to %s", disk.Alias.GetName(), domcfg.Devices.Disks[i].Source.File.File, disk.Source.File)
			domcfg.Devices.Disks[i].Source.File.File = disk.Source.File
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
	uid := migrationUID(vmi)
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

func newMigrationMonitor(vmi *v1.VirtualMachineInstance, l *LibvirtDomainManager, options *cmdclient.MigrationOptions, migrationErr chan error) *migrationMonitor {
	monitor := &migrationMonitor{
		l:                        l,
		vmi:                      vmi,
		options:                  options,
		migrationErr:             migrationErr,
		iterationCh:              make(chan int, 16),
		progressWatermark:        0,
		remainingData:            0,
		progressTimeout:          options.ProgressTimeout,
		acceptableCompletionTime: options.CompletionTimeoutPerGiB * getVMIMigrationDataSize(vmi, l.ephemeralDiskDir),
		maxDowntime:              options.MaxDowntime,
		stallDetectionEnabled:    options.StallDetectionEnabled,
	}
	log.Log.Object(vmi).V(3).Infof(
		"Initialized migration monitor: stallDetection=%t progressTimeout=%ds completionTimeoutPerGiB=%d maxDowntimeMs=%d allowPostCopy=%t allowWorkloadDisruption=%t",
		options.StallDetectionEnabled,
		options.ProgressTimeout,
		options.CompletionTimeoutPerGiB,
		options.MaxDowntime,
		options.AllowPostCopy,
		options.AllowWorkloadDisruption,
	)
	if options.StallDetectionEnabled && virtutil.IsVFIOVMI(vmi) {
		log.Log.Object(vmi).V(3).Info("VFIO detected: QEMU remaining-bytes signals may under-report outstanding migration data")
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

func (m *migrationMonitor) shouldTriggerTimeout(elapsed int64) bool {
	if m.acceptableCompletionTime == 0 {
		return false
	}

	return elapsed/int64(time.Second) > m.acceptableCompletionTime
}

func (m *migrationMonitor) shouldAssistMigrationToComplete(elapsed int64) bool {
	return m.options.AllowWorkloadDisruption && m.shouldTriggerTimeout(elapsed)
}

// TODO: remove after MigrationStallDetection feature graduates
func (m *migrationMonitor) isMigrationProgressing() bool {
	logger := log.Log.Object(m.vmi)

	now := time.Now().UTC().UnixNano()

	// check if the migration is progressing
	progressDelay := (now - m.lastProgressUpdate) / int64(time.Second)
	if m.progressTimeout != 0 && progressDelay > m.progressTimeout {
		logger.Warningf("Live migration stuck for %d seconds", progressDelay)
		return false
	}

	return true
}

func (m *migrationMonitor) updateBandwidthEstimate(stats *libvirt.DomainJobInfo) {
	logger := log.Log
	if m.vmi != nil {
		logger = log.Log.Object(m.vmi)
	}
	prev := m.ewmaBandwidthBps
	if m.ewmaBandwidthBps == 0 {
		m.ewmaBandwidthBps = float64(stats.MemBps)
		logger.V(4).Infof("Initialized migration bandwidth EWMA: sampleBps=%d ewmaBps=%.2f", stats.MemBps, m.ewmaBandwidthBps)
		return
	}
	m.ewmaBandwidthBps = bandwidthEWMAAlpha*float64(stats.MemBps) + (1-bandwidthEWMAAlpha)*m.ewmaBandwidthBps
	logger.V(4).Infof("Updated migration bandwidth EWMA: sampleBps=%d previousEwmaBps=%.2f newEwmaBps=%.2f", stats.MemBps, prev, m.ewmaBandwidthBps)
}

func (m *migrationMonitor) updateCandidates(iterElapsedMs uint64, remainingBytes uint64) {
	logger := log.Log
	if m.vmi != nil {
		logger = log.Log.Object(m.vmi)
	}
	if m.progressTimeout <= 0 {
		logger.V(4).Infof("Skipping candidate update: progressTimeout=%d", m.progressTimeout)
		return
	}

	progressTimeoutMs := uint64(m.progressTimeout) * 1000
	agedOut := 0
	for len(m.minCandidates) > 0 {
		oldestCandidate := m.minCandidates[0]
		// iterElapsedMs > oldestCandidate.elapsedMs because iterElapsedMs is monotonically increasing
		ageMs := iterElapsedMs - oldestCandidate.elapsedMs
		if ageMs < progressTimeoutMs {
			break
		}

		m.minCandidates = m.minCandidates[1:]
		if m.minRecordOutsideWindow == nil || oldestCandidate.remainingBytes < m.minRecordOutsideWindow.remainingBytes {
			m.minRecordOutsideWindow = &oldestCandidate
		}
		agedOut++
	}
	if agedOut > 0 {
		outsideWindowMin := uint64(0)
		if m.minRecordOutsideWindow != nil {
			outsideWindowMin = m.minRecordOutsideWindow.remainingBytes
		}
		logger.V(4).Infof("Aged out candidates: count=%d iterElapsedMs=%d outsideWindowMinRemainingBytes=%d remainingCandidates=%d", agedOut, iterElapsedMs, outsideWindowMin, len(m.minCandidates))
	}

	// candidates larger than the current out-of-window min can never become relevant.
	if m.minRecordOutsideWindow != nil && remainingBytes > m.minRecordOutsideWindow.remainingBytes {
		logger.V(4).Infof("Skipping candidate above outside-window min: remainingBytes=%d outsideWindowMin=%d", remainingBytes, m.minRecordOutsideWindow.remainingBytes)
		return
	}

	// keep only monotonic minima in timestamp order.
	if len(m.minCandidates) > 0 && remainingBytes >= m.minCandidates[len(m.minCandidates)-1].remainingBytes {
		logger.V(4).Infof("Skipping candidate that is not a new minimum: remainingBytes=%d lastCandidateRemainingBytes=%d", remainingBytes, m.minCandidates[len(m.minCandidates)-1].remainingBytes)
		return
	}

	record := iterationRecord{
		elapsedMs:      iterElapsedMs,
		remainingBytes: remainingBytes,
	}
	m.minCandidates = append(m.minCandidates, record)
	logger.V(4).Infof("Added candidate minimum: iterElapsedMs=%d remainingBytes=%d candidates=%d", iterElapsedMs, remainingBytes, len(m.minCandidates))
}

func (m *migrationMonitor) checkStallCondition(remainingBytes uint64) bool {
	logger := log.Log
	if m.vmi != nil {
		logger = log.Log.Object(m.vmi)
	}
	if m.minRecordOutsideWindow == nil {
		logger.V(4).Infof("Stall check skipped: no outside-window minimum yet, remainingBytes=%d", remainingBytes)
		return false
	}

	stallThreshold := uint64(float64(m.minRecordOutsideWindow.remainingBytes) * (1 - stallMargin))
	stalled := remainingBytes >= stallThreshold
	logger.V(4).Infof("Stall check result: remainingBytes=%d outsideWindowMin=%d threshold=%d stalled=%t", remainingBytes, m.minRecordOutsideWindow.remainingBytes, stallThreshold, stalled)
	return stalled
}

func (m *migrationMonitor) findBestRemainingBytes() uint64 {
	bestRemainingBytes := m.minRecordOutsideWindow.remainingBytes
	for _, candidate := range m.minCandidates {
		if candidate.remainingBytes < bestRemainingBytes {
			bestRemainingBytes = candidate.remainingBytes
		}
	}
	return bestRemainingBytes
}

func (m *migrationMonitor) initializeRelaxationState(iterElapsedMs uint64) {
	logger := log.Log
	if m.vmi != nil {
		logger = log.Log.Object(m.vmi)
	}
	m.remainingBytesHistory = utilheap.NewMin[uint64]()
	m.relaxationPatienceMs = uint64(m.progressTimeout) * 1000
	if m.relaxationPatienceMs < 1000 {
		m.relaxationPatienceMs = 1000
	}
	m.relaxationDeadlineMs = iterElapsedMs + m.relaxationPatienceMs
	logger.V(3).Infof("Initialized relaxation state: iterElapsedMs=%d patienceMs=%d deadlineMs=%d", iterElapsedMs, m.relaxationPatienceMs, m.relaxationDeadlineMs)
}

func (m *migrationMonitor) relaxBestRemainingBytes(iterElapsedMs uint64) {
	logger := log.Log
	if m.vmi != nil {
		logger = log.Log.Object(m.vmi)
	}
	if iterElapsedMs < m.relaxationDeadlineMs || m.remainingBytesHistory.Len() == 0 {
		logger.V(4).Infof("Relaxation not due: iterElapsedMs=%d deadlineMs=%d historyLen=%d", iterElapsedMs, m.relaxationDeadlineMs, m.remainingBytesHistory.Len())
		return
	}
	oldBest := m.bestRemainingBytes
	nextCandidate, exists := m.remainingBytesHistory.Pop()
	if !exists {
		// should never happen
		log.Log.Error("failed to pop remaining bytes history")
		return
	}
	m.bestRemainingBytes = nextCandidate
	m.relaxationPatienceMs = max(m.relaxationPatienceMs/2, uint64(1000))
	m.relaxationDeadlineMs = iterElapsedMs + m.relaxationPatienceMs
	logger.V(3).Infof("Relaxed best remaining bytes: oldBest=%d newBest=%d iterElapsedMs=%d nextPatienceMs=%d nextDeadlineMs=%d", oldBest, m.bestRemainingBytes, iterElapsedMs, m.relaxationPatienceMs, m.relaxationDeadlineMs)
}

func (m *migrationMonitor) isWithinSwitchMargin(remainingBytes uint64) bool {
	logger := log.Log
	if m.vmi != nil {
		logger = log.Log.Object(m.vmi)
	}
	if m.bestRemainingBytes == 0 {
		return false
	}
	target := uint64(float64(m.bestRemainingBytes) * (1 + stallMargin))
	within := remainingBytes <= target
	logger.V(4).Infof("Switch margin check: remainingBytes=%d bestRemainingBytes=%d margin=%.2f target=%d within=%t", remainingBytes, m.bestRemainingBytes, stallMargin, target, within)
	return within
}

func (m *migrationMonitor) triggerConvergenceAction(dom cli.VirDomain, impliedDowntimeMs int64, iterElapsedMs uint64, remainingBytes uint64) *inflightMigrationAborted {
	logger := log.Log.Object(m.vmi)

	if m.switchoverInitiated {
		logger.V(4).Info("Switchover already initiated, skipping convergence action")
		return nil
	}
	m.switchoverInitiated = true
	stallDurationMs := uint64(0)
	if m.stallDetectedAtMs > 0 && iterElapsedMs >= m.stallDetectedAtMs {
		stallDurationMs = iterElapsedMs - m.stallDetectedAtMs
	}
	logger.V(3).Infof(
		"Evaluating convergence action: iterElapsedMs=%d stallDurationMs=%d remainingBytes=%d bestRemainingBytes=%d impliedDowntimeMs=%d maxDowntimeMs=%d allowPostCopy=%t allowWorkloadDisruption=%t",
		iterElapsedMs,
		stallDurationMs,
		remainingBytes,
		m.bestRemainingBytes,
		impliedDowntimeMs,
		m.maxDowntime,
		m.options.AllowPostCopy,
		m.options.AllowWorkloadDisruption,
	)

	if m.options.AllowPostCopy && !virtutil.IsVFIOVMI(m.vmi) {
		logger.Info("Starting post copy mode for migration")
		if err := dom.MigrateStartPostCopy(0); err != nil {
			m.switchoverInitiated = false
			logger.Reason(err).Error("failed to start post migration")
			return nil
		}
		m.l.updateVMIMigrationMode(v1.MigrationPostCopy)
		return nil
	}

	if m.options.AllowWorkloadDisruption || virtutil.IsVFIOVMI(m.vmi) {
		logger.V(3).Infof("Convergence decision: force switchover with max downtime %dms", migrationutils.QEMUMaxMigrationDowntimeMS)
		logger.Infof("Setting max downtime to %dms to force switchover", migrationutils.QEMUMaxMigrationDowntimeMS)
		if err := dom.MigrateSetMaxDowntime(uint64(migrationutils.QEMUMaxMigrationDowntimeMS), 0); err != nil {
			m.switchoverInitiated = false
			logger.Reason(err).Error("setting max downtime failed")
		}
		return nil
	}

	if impliedDowntimeMs <= m.maxDowntime {
		targetDowntime := max(impliedDowntimeMs, int64(1))
		if targetDowntime > migrationutils.QEMUMaxMigrationDowntimeMS {
			targetDowntime = migrationutils.QEMUMaxMigrationDowntimeMS
		}
		logger.V(3).Infof("Convergence decision: set computed target downtime=%dms (implied=%dms budget=%dms)", targetDowntime, impliedDowntimeMs, m.maxDowntime)
		logger.Infof("Setting max downtime to %dms", targetDowntime)
		if err := dom.MigrateSetMaxDowntime(uint64(targetDowntime), 0); err != nil {
			m.switchoverInitiated = false
			logger.Reason(err).Error("setting max downtime failed")
		}
		return nil
	}

	if float64(m.maxDowntime)*preCopyPossibleFactor <= float64(impliedDowntimeMs) {
		logger.V(3).Infof("Convergence decision: abort migration as implied downtime exceeds pre-copy feasible range (implied=%dms budget=%dms factor=%.2f)", impliedDowntimeMs, m.maxDowntime, preCopyPossibleFactor)
		logger.Info("Aborting migration: convergence target is unlikely to be reached")
		if err := dom.AbortJob(); err != nil {
			m.switchoverInitiated = false
			logger.Reason(err).Error("failed to abort migration")
			return nil
		}
		return &inflightMigrationAborted{
			message:     fmt.Sprintf("Live migration aborted because implied downtime %dms exceeds maxDowntime budget", impliedDowntimeMs),
			abortStatus: v1.MigrationAbortSucceeded,
		}
	}

	logger.V(3).Infof("Convergence decision: set configured maxDowntime budget=%dms (implied=%dms)", m.maxDowntime, impliedDowntimeMs)
	logger.Infof("Setting max downtime to configured budget %dms", m.maxDowntime)
	if err := dom.MigrateSetMaxDowntime(uint64(m.maxDowntime), 0); err != nil {
		m.switchoverInitiated = false
		logger.Reason(err).Error("setting max downtime failed")
	}
	return nil
}

func (m *migrationMonitor) processStallDetectionIteration(dom cli.VirDomain, stats *libvirt.DomainJobInfo) *inflightMigrationAborted {
	logger := log.Log
	if m.vmi != nil {
		logger = log.Log.Object(m.vmi)
	}
	if !stats.DataRemainingSet || !stats.TimeElapsedSet || !stats.MemBpsSet {
		logger.V(4).Infof("Skipping stall-detection iteration due to missing stats: dataRemaining=%t timeElapsed=%t memBps=%t", stats.DataRemainingSet, stats.TimeElapsedSet, stats.MemBpsSet)
		return nil
	}
	if m.isMigrationPostCopy() || m.isPausedMigration() {
		logger.V(4).Info("Skipping stall-detection iteration while migration is post-copy or paused")
		return nil
	}
	if !stats.TimeElapsedSet {
		return nil
	}

	iterElapsedMs := stats.TimeElapsed
	remainingBytes := stats.DataRemaining
	now := time.Now().UTC()
	logger.V(4).Infof("Processing stall-detection iteration: iterElapsedMs=%d remainingBytes=%d memBps=%d currentEwmaBps=%.2f", iterElapsedMs, remainingBytes, stats.MemBps, m.ewmaBandwidthBps)
	if !m.lastIterTimestamp.IsZero() && iterElapsedMs >= m.lastIterElapsedMs {
		wallDeltaMs := now.Sub(m.lastIterTimestamp).Milliseconds()
		iterDeltaMs := int64(iterElapsedMs - m.lastIterElapsedMs)
		delaySkewMs := wallDeltaMs - iterDeltaMs
		if delaySkewMs < 0 {
			delaySkewMs = -delaySkewMs
		}
		if delaySkewMs >= 2000 {
			logger.V(3).Infof("Observed iteration boundary processing skew: wallDeltaMs=%d iterDeltaMs=%d skewMs=%d", wallDeltaMs, iterDeltaMs, delaySkewMs)
		}
	}
	m.lastIterTimestamp = now
	m.lastIterElapsedMs = iterElapsedMs

	m.updateCandidates(iterElapsedMs, remainingBytes)

	// when stall is first detected initialize stall-related state
	if !m.stallDetected && m.checkStallCondition(remainingBytes) {
		m.bestRemainingBytes = m.findBestRemainingBytes()
		m.stallDetected = true
		m.stallDetectedAtMs = iterElapsedMs
		m.lastStallRemaining = remainingBytes
		m.initializeRelaxationState(iterElapsedMs)
		logger.V(3).Infof("Stall detected: bestRemainingBytes=%d outsideWindowMin=%d candidates=%d", m.bestRemainingBytes, m.minRecordOutsideWindow.remainingBytes, len(m.minCandidates))
	}

	if !m.stallDetected {
		logger.V(4).Info("Stall not detected yet; continuing pre-copy monitoring")
		return nil
	}
	if m.lastStallRemaining > 0 && remainingBytes < m.lastStallRemaining {
		recoveredBytes := m.lastStallRemaining - remainingBytes
		recoveryPct := (float64(recoveredBytes) / float64(m.lastStallRemaining)) * 100
		shouldLogRecovery := recoveryPct >= 5
		if m.lastRecoveryLogMs > 0 && iterElapsedMs < m.lastRecoveryLogMs+5000 {
			shouldLogRecovery = false
		}
		if shouldLogRecovery {
			logger.V(3).Infof("Observed post-stall pre-copy recovery: prevRemainingBytes=%d currentRemainingBytes=%d recoveredBytes=%d recoveryPct=%.2f", m.lastStallRemaining, remainingBytes, recoveredBytes, recoveryPct)
			m.lastRecoveryLogMs = iterElapsedMs
		}
	}
	m.lastStallRemaining = remainingBytes

	m.remainingBytesHistory.Push(remainingBytes)
	logger.V(4).Infof("Recorded remaining-bytes history: value=%d historyLen=%d", remainingBytes, m.remainingBytesHistory.Len())
	m.relaxBestRemainingBytes(iterElapsedMs)
	if !m.isWithinSwitchMargin(remainingBytes) {
		return nil
	}

	bandwidthBpms := m.ewmaBandwidthBps / 1000
	if bandwidthBpms <= 0 {
		logger.V(4).Infof("Skipping convergence action due to non-positive EWMA bandwidth: ewmaBps=%.2f bandwidthBpms=%.4f", m.ewmaBandwidthBps, bandwidthBpms)
		return nil
	}
	impliedDowntimeMs := int64(float64(m.bestRemainingBytes) / bandwidthBpms)
	logger.V(3).Infof("Switch margin reached: bestRemainingBytes=%d impliedDowntimeMs=%d bandwidthBpms=%.4f", m.bestRemainingBytes, impliedDowntimeMs, bandwidthBpms)
	return m.triggerConvergenceAction(dom, impliedDowntimeMs, iterElapsedMs, remainingBytes)
}

func (m *migrationMonitor) processInflightMigration(dom cli.VirDomain, stats *libvirt.DomainJobInfo, isIterationBoundary bool) *inflightMigrationAborted {
	logger := log.Log.Object(m.vmi)

	// Migration is running
	now := time.Now().UTC().UnixNano()
	elapsed := now - m.start

	m.l.domainInfoStats = statsconv.Convert_libvirt_DomainJobInfo_To_stats_DomainJobInfo(stats)
	if (m.progressWatermark == 0) || (m.remainingData < m.progressWatermark) {
		m.lastProgressUpdate = now
	}
	m.progressWatermark = m.remainingData

	if m.stallDetectionEnabled {
		if isIterationBoundary {
			logger.V(4).Info("Processing migration iteration boundary for stall detection")
			if aborted := m.processStallDetectionIteration(dom, stats); aborted != nil {
				return aborted
			}
		} else {
			m.updateBandwidthEstimate(stats)
		}
		if m.shouldTriggerTimeout(elapsed) {
			logger.V(3).Infof("Aborting migration due to completion timeout: elapsedSec=%d acceptableCompletionSec=%d", elapsed/int64(time.Second), m.acceptableCompletionTime)
			err := dom.AbortJob()
			if err != nil {
				logger.Reason(err).Error("failed to abort migration")
				return nil
			}

			return &inflightMigrationAborted{
				message:     fmt.Sprintf("Live migration is not completed after %d seconds and has been aborted", m.acceptableCompletionTime),
				abortStatus: v1.MigrationAbortSucceeded,
			}
		}
		return nil
	}

	switch {
	case m.isMigrationPostCopy():
		// Currently, there is nothing for us to track when in Post Copy mode.
		// The reasoning here is that post copy migrations transfer the state
		// directly to the target pod in a way that results in the target pod
		// hosting the active workload while the migration completes.

		// If we were to abort the migration due to a timeout while in post copy,
		// then it would result in that active state being lost.

	case m.shouldAssistMigrationToComplete(elapsed) && !m.isPausedMigration():
		if m.options.AllowPostCopy && !virtutil.IsVFIOVMI(m.vmi) {
			logger.Info("Starting post copy mode for migration")
			// if a migration has stalled too long, post copy will be
			// triggered when allowPostCopy is enabled (post-copy is not supported with VFIO devices)
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
			err := dom.MigrateSetMaxDowntime(min(uint64(maxDowntimeSec)*1000, uint64(migrationutils.QEMUMaxMigrationDowntimeMS)), 0)
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
			// if a migration has stalled too long, the guest will be paused
			// to complete the migration when allowPostCopy is disabled
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
	// TODO: remove after MigrationStallDetection feature graduates
	case !m.isMigrationProgressing():
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
	case m.shouldTriggerTimeout(elapsed):
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

func (m *migrationMonitor) registerIterationCallback(domName string) (int, error) {
	return m.l.virConn.DomainEventMigrationIterationRegister(func(_ *libvirt.Connect, domain *libvirt.Domain, event *libvirt.DomainEventMigrationIteration) {
		logger := log.Log.Object(m.vmi)
		name, err := domain.GetName()
		if err != nil || name != domName {
			return
		}

		select {
		case m.iterationCh <- event.Iteration:
			logger.V(4).Infof("Queued migration iteration event: iteration=%d", event.Iteration)
		default:
			logger.V(3).Infof("Dropped migration iteration event: iteration=%d reason=channel-full", event.Iteration)
		}
	})
}

func (m *migrationMonitor) startMonitor() {
	vmi := m.vmi

	m.start = time.Now().UTC().UnixNano()
	m.lastProgressUpdate = m.start

	logger := log.Log.Object(vmi)
	defer func() {
		m.l.domainInfoStats = &stats.DomainJobInfo{}
	}()

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := m.l.virConn.LookupDomainByName(domName)
	if err != nil {
		logger.Reason(err).Error(liveMigrationFailed)
		m.l.setMigrationResult(true, fmt.Sprintf("%v", err), "")
		return
	}
	defer dom.Free()

	if m.stallDetectionEnabled {
		registrationID, registerErr := m.registerIterationCallback(domName)
		if registerErr != nil {
			logger.Reason(registerErr).Error("failed to register migration iteration callback, falling back to legacy stall handling")
			m.stallDetectionEnabled = false
		} else {
			logger.V(3).Infof("Registered migration iteration callback: registrationID=%d", registrationID)
			defer func() {
				if err := m.l.virConn.DomainEventDeregister(registrationID); err != nil {
					logger.Reason(err).V(3).Info("failed to deregister migration iteration callback")
				}
			}()
		}
	}

	logInterval := 0

	for {
		isIterationBoundary := false
		var ok bool
		err = nil
		select {
		case err, ok = <-m.migrationErr:
			if !ok {
				return
			}
		case <-m.iterationCh:
			isIterationBoundary = true
		case <-time.After(monitorSleepPeriodMS * time.Millisecond):
		}

		if err != nil {
			logger.Reason(err).Error(liveMigrationFailed)
			var abortStatus v1.MigrationAbortStatus
			if strings.Contains(err.Error(), "canceled by client") {
				abortStatus = v1.MigrationAbortSucceeded
			}
			if len(vmi.Status.MigratedVolumes) > 0 && strings.Contains(standardizeSpaces(err.Error()),
				"has to be smaller or equal to the actual size of the containing file") {
				m.l.setMigrationResult(true, fmt.Sprintf("Volume migration cannot be performed because the destination volume is smaller than the source volume: %v",
					err), abortStatus)
				return
			}
			m.l.setMigrationResult(true, fmt.Sprintf("Live migration failed %v", err), abortStatus)
			return
		}

		jobStats, err := dom.GetJobStats(0)
		if err != nil {
			logger.Reason(err).Info("failed to get domain job info, will retry")
			continue
		}

		if jobStats.DataRemainingSet {
			m.remainingData = jobStats.DataRemaining
		}

		uid := migrationUID(vmi)
		switch jobStats.Type {
		case libvirt.DOMAIN_JOB_UNBOUNDED:
			aborted := m.processInflightMigration(dom, jobStats, isIterationBoundary)
			if aborted != nil {
				logger.Errorf("Live migration abort detected with reason: %s", aborted.message)
				m.l.setMigrationResult(true, aborted.message, aborted.abortStatus)
				return
			}
			if !isIterationBoundary {
				logInterval++
			}
			if logInterval%monitorLogInterval == 0 || isIterationBoundary {
				logMigrationInfo(logger, uid, jobStats)
			}
		case libvirt.DOMAIN_JOB_NONE:
			logger.Info("Migration job is not active")
		}
	}
}

// Attempts reading the completed stats for the job that just finished. Needs to be called right after the migration before the source pod is destroyed.
// It retries due to possible contention in libvirt calls but not for too long to avoid blocking the migration completion.
func (l *LibvirtDomainManager) retrieveCompletedStats(vmi *v1.VirtualMachineInstance) {
	logger := log.Log.Object(vmi)

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		logger.Reason(err).Warning("failed to look up domain for completed migration stats")
		return
	}
	defer dom.Free()

	var jobStats *libvirt.DomainJobInfo
	err = virtwait.PollImmediately(200*time.Millisecond, 2*time.Second, func(_ context.Context) (bool, error) {
		var getErr error
		jobStats, getErr = dom.GetJobStats(libvirt.DOMAIN_JOB_STATS_COMPLETED)
		if getErr != nil {
			logger.Reason(getErr).V(3).Info("completed migration stats not yet available, retrying")
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		logger.Warning("timed out retrieving completed migration job stats")
		return
	}

	logMigrationInfo(logger, migrationUID(vmi), jobStats)
}

func migrationUID(vmi *v1.VirtualMachineInstance) types.UID {
	if s := vmi.Status.MigrationState; s != nil {
		if s.SourceState != nil {
			return s.SourceState.MigrationUID
		}
		return s.MigrationUID
	}
	return ""
}

// logMigrationInfo logs the same migration info as `virsh -r domjobinfo`
func logMigrationInfo(logger *log.FilteredLogger, uid types.UID, info *libvirt.DomainJobInfo) {
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
			log.Log.Object(vmi).Info("Live migration abort succeeded")
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
		// Modify the domain XML to update paths to the target volumes to match the new domain
		for i, disk := range domSpec.Devices.Disks {
			if strings.Contains(disk.Source.File, vmi.Namespace) {
				// Need to update the namespace in the path to the new namespace.
				oldPath := disk.Source.File
				domSpec.Devices.Disks[i].Source.File = strings.Replace(disk.Source.File, vmi.Namespace, *vmi.Status.MigrationState.TargetState.DomainNamespace, 1)
				log.Log.Object(vmi).V(4).Infof("Updated disk %s source path to %s", oldPath, domSpec.Devices.Disks[i].Source.File)
			}
		}
		for _, disk := range domSpec.Devices.Disks {
			if disk.Source.Dev != "" {
				log.Log.Object(vmi).V(4).Infof("Paths of disk %s: %s", disk.Alias.GetName(), disk.Source.Dev)
			} else if disk.Source.File != "" {
				log.Log.Object(vmi).V(4).Infof("Paths of disk %s: %s", disk.Alias.GetName(), disk.Source.File)
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
		if dom.Devices.Disks[i].Source.Slices == nil {
			dom.Devices.Disks[i].Source.Slices = &libvirtxml.DomainDiskSlices{
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
			dom.Devices.Disks[i].Source.Block = &libvirtxml.DomainDiskSourceBlock{
				Dev: path,
			}
			dom.Devices.Disks[i].Source.File = nil
		}
		if _, ok := blockSrcFsDstVols[name]; ok {
			log.Log.V(2).Infof("Replace block source with destination for volume %s", name)
			if hotplugVol {
				path = hotplugdisk.GetVolumeMountDir(name) + ".img"
			} else {
				path = filepath.Join(hostdisk.GetMountedHostDiskDir(name), "disk.img")
			}
			dom.Devices.Disks[i].Source.File = &libvirtxml.DomainDiskSourceFile{
				File: path,
			}
			dom.Devices.Disks[i].Source.Block = nil
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
		close(migrationErrorChan)
		return
	}

	close(migrationErrorChan)
	l.retrieveCompletedStats(vmi)
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
