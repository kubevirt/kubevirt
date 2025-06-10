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
	"encoding/json"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"libvirt.org/go/libvirt"
	"libvirt.org/go/libvirtxml"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	osdisk "kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/pointer"
	virtutil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/migrations"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/sriov"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	convxml "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/libvirtxml"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/statsconv"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

const liveMigrationFailed = "Live migration failed."

const (
	monitorSleepPeriodMS = 400
	monitorLogPeriodMS   = 4000
	monitorLogInterval   = monitorLogPeriodMS / monitorSleepPeriodMS
)

type migrationDisks struct {
	shared         map[string]bool
	generated      map[string]bool
	localToMigrate map[string]bool
}

type migrationMonitor struct {
	l       *LibvirtDomainManager
	vmi     *v1.VirtualMachineInstance
	options *cmdclient.MigrationOptions

	migrationErr chan error

	start              int64
	lastProgressUpdate int64
	progressWatermark  uint64
	remainingData      uint64

	progressTimeout          int64
	acceptableCompletionTime int64
	migrationFailedWithError error
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
func generateDomainForTargetCPUSetAndTopology(vmi *v1.VirtualMachineInstance, domSpec *api.DomainSpec) (*api.Domain, error) {
	var targetTopology cmdv1.Topology
	targetNodeCPUSet := vmi.Status.MigrationState.TargetCPUSet
	err := json.Unmarshal([]byte(vmi.Status.MigrationState.TargetNodeTopology), &targetTopology)
	if err != nil {
		return nil, err
	}

	useIOThreads := false
	for _, diskDevice := range vmi.Spec.Domain.Devices.Disks {
		if diskDevice.DedicatedIOThread != nil && *diskDevice.DedicatedIOThread {
			useIOThreads = true
			break
		}
	}
	domain := api.NewMinimalDomain(vmi.Name)
	domain.Spec = *domSpec
	cpuTopology := vcpu.GetCPUTopology(vmi)
	cpuCount := vcpu.CalculateRequestedVCPUs(cpuTopology)

	// update cpu count to maximum hot plugable CPUs
	vmiCPU := vmi.Spec.Domain.CPU
	if vmiCPU != nil && vmiCPU.MaxSockets != 0 {
		cpuTopology.Sockets = vmiCPU.MaxSockets
		cpuCount = vcpu.CalculateRequestedVCPUs(cpuTopology)
	}
	domain.Spec.CPU.Topology = cpuTopology
	domain.Spec.VCPU = &api.VCPU{
		Placement: "static",
		CPUs:      cpuCount,
	}
	err = vcpu.AdjustDomainForTopologyAndCPUSet(domain, vmi, &targetTopology, targetNodeCPUSet, useIOThreads)
	if err != nil {
		return nil, err
	}

	return domain, err
}

func convertCPUDedicatedFields(domain *api.Domain, domcfg *libvirtxml.Domain) error {
	if domcfg.CPU == nil {
		domcfg.CPU = &libvirtxml.DomainCPU{}
	}
	domcfg.CPU.Topology = convxml.ConvertKubeVirtCPUTopologyToDomainCPUTopology(domain.Spec.CPU.Topology)
	domcfg.VCPU = convxml.ConvertKubeVirtVCPUToDomainVCPU(domain.Spec.VCPU)
	domcfg.CPUTune = convxml.ConvertKubeVirtCPUTuneToDomainCPUTune(domain.Spec.CPUTune)
	domcfg.NUMATune = convxml.ConvertKubeVirtNUMATuneToDomainNUMATune(domain.Spec.NUMATune)
	domcfg.Features = convxml.ConvertKubeVirtFeaturesToDomainFeatureList(domain.Spec.Features)

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
	if vmi.IsCPUDedicated() {
		// If the VMI has dedicated CPUs, we need to replace the old CPUs that were
		// assigned in the source node with the new CPUs assigned in the target node
		err = xml.Unmarshal([]byte(xmlstr), &domain)
		if err != nil {
			return "", err
		}
		domain, err = generateDomainForTargetCPUSetAndTopology(vmi, domSpec)
		if err != nil {
			return "", err
		}
		if err = convertCPUDedicatedFields(domain, domcfg); err != nil {
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
	copyDisks := []string{}
	migrationVols := classifyVolumesForMigration(vmi)
	disks, err := getAllDomainDisks(dom)
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
	migrationUID := vmi.Status.MigrationState.MigrationUID
	if vmi.Status.MigrationState.SourceState != nil {
		migrationUID = vmi.Status.MigrationState.SourceState.MigrationUID
	}
	if exists && migrationMetadata.UID == migrationUID {
		if migrationMetadata.EndTimestamp == nil {
			// don't stop on currently executing migrations
			return true, nil
		} else {
			// Don't allow the same migration UID to be executed twice.
			// Migration attempts are like pods. One shot.
			return false, fmt.Errorf("migration job %v already executed, finished at %v, failed: %t, abortStatus: %s",
				migrationMetadata.UID, *migrationMetadata.EndTimestamp, migrationMetadata.Failed, migrationMetadata.AbortStatus)
		}
	}

	now := metav1.Now()
	m := api.MigrationMetadata{
		UID:            migrationUID,
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
		return fmt.Errorf(migrations.CancelMigrationFailedVmiNotMigratingErr)
	}

	if err := l.setMigrationAbortStatus(v1.MigrationAbortInProgress); err != nil {
		if err == domainerrors.MigrationAbortInProgressError {
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
		progressWatermark:        0,
		remainingData:            0,
		progressTimeout:          options.ProgressTimeout,
		acceptableCompletionTime: options.CompletionTimeoutPerGiB * getVMIMigrationDataSize(vmi, l.ephemeralDiskDir),
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
	return m.shouldTriggerTimeout(elapsed) && m.options.AllowWorkloadDisruption
}

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

func (m *migrationMonitor) determineNonRunningMigrationStatus(dom cli.VirDomain) *libvirt.DomainJobInfo {
	logger := log.Log.Object(m.vmi)
	// check if an ongoing migration has been completed before we could capture the outcome
	if m.lastProgressUpdate > m.start {
		logger.Info("Migration job has probably completed before we could capture the status. Getting latest status.")
		// at this point the migration is over, but we don't know the result.
		// check if we were trying to cancel this job. In this case, finalize the migration.
		migration, _ := m.l.metadataCache.Migration.Load()
		if migration.AbortStatus == string(v1.MigrationAbortInProgress) {
			logger.Info("Migration job was canceled")
			return &libvirt.DomainJobInfo{
				Type:             libvirt.DOMAIN_JOB_CANCELLED,
				DataRemaining:    m.remainingData,
				DataRemainingSet: true,
			}
		}

		// If the domain is active, it means that the migration has failed.
		domainState, _, err := dom.GetState()
		if err != nil {
			logger.Reason(err).Error("failed to get domain state")
			if libvirtError, ok := err.(libvirt.Error); ok &&
				(libvirtError.Code == libvirt.ERR_NO_DOMAIN ||
					libvirtError.Code == libvirt.ERR_OPERATION_INVALID) {
				logger.Info("domain is not running on this node")
				return nil
			}
		}
		if domainState == libvirt.DOMAIN_RUNNING {
			logger.Info("Migration job failed")
			return &libvirt.DomainJobInfo{
				Type:             libvirt.DOMAIN_JOB_FAILED,
				DataRemaining:    m.remainingData,
				DataRemainingSet: true,
			}
		}
	}
	logger.Info("Migration job didn't start yet")
	return nil
}

func (m *migrationMonitor) processInflightMigration(dom cli.VirDomain, stats *libvirt.DomainJobInfo) *inflightMigrationAborted {
	logger := log.Log.Object(m.vmi)

	// Migration is running
	now := time.Now().UTC().UnixNano()
	elapsed := now - m.start

	m.l.migrateInfoStats = statsconv.Convert_libvirt_DomainJobInfo_To_stats_DomainJobInfo(stats)
	if (m.progressWatermark == 0) || (m.remainingData < m.progressWatermark) {
		m.lastProgressUpdate = now
	}
	m.progressWatermark = m.remainingData

	switch {
	case m.isMigrationPostCopy():
		// Currently, there is nothing for us to track when in Post Copy mode.
		// The reasoning here is that post copy migrations transfer the state
		// directly to the target pod in a way that results in the target pod
		// hosting the active workload while the migration completes.

		// If we were to abort the migration due to a timeout while in post copy,
		// then it would result in that active state being lost.

	case m.shouldAssistMigrationToComplete(elapsed) && !m.isPausedMigration():
		if m.options.AllowPostCopy {
			logger.Info("Starting post copy mode for migration")
			// if a migration has stalled too long, post copy will be
			// triggered when allowPostCopy is enabled
			err := dom.MigrateStartPostCopy(0)
			if err != nil {
				logger.Reason(err).Error("failed to start post migration")
				return nil
			}
			m.l.updateVMIMigrationMode(v1.MigrationPostCopy)
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

	case !m.isMigrationProgressing():
		// check if the migration is still progressing
		// a stuck migration will get terminated when post copy
		// isn't enabled
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

func (m *migrationMonitor) startMonitor() {
	var completedJobInfo *libvirt.DomainJobInfo
	vmi := m.vmi

	m.start = time.Now().UTC().UnixNano()
	m.lastProgressUpdate = m.start

	logger := log.Log.Object(vmi)
	defer func() {
		m.l.migrateInfoStats = &stats.DomainJobInfo{}
	}()

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := m.l.virConn.LookupDomainByName(domName)
	if err != nil {
		logger.Reason(err).Error(liveMigrationFailed)
		m.l.setMigrationResult(true, fmt.Sprintf("%v", err), "")
		return
	}
	defer dom.Free()

	logInterval := 0

	for {
		err = nil
		select {
		case err = <-m.migrationErr:
		case <-time.After(monitorSleepPeriodMS * time.Millisecond):
		}

		if err != nil && m.migrationFailedWithError == nil {
			logger.Reason(err).Error("Received a live migration error. Will check the latest migration status.")
			m.migrationFailedWithError = err
		} else if m.migrationFailedWithError != nil {
			logger.Info("Didn't manage to get a job status. Post the received error and finalize.")
			logger.Reason(m.migrationFailedWithError).Error(liveMigrationFailed)
			var abortStatus v1.MigrationAbortStatus
			if strings.Contains(m.migrationFailedWithError.Error(), "canceled by client") {
				abortStatus = v1.MigrationAbortSucceeded
			}
			// Improve the error message when the volume migration fails because the destination size is smaller then the source volume
			if len(vmi.Status.MigratedVolumes) > 0 && strings.Contains(m.migrationFailedWithError.Error(),
				"has to be smaller or equal to the actual size of the containing file") {
				m.l.setMigrationResult(true, fmt.Sprintf("Volume migration cannot be performed because the destination volume is smaller then the source volume: %v",
					m.migrationFailedWithError), abortStatus)
				return
			}
			m.l.setMigrationResult(true, fmt.Sprintf("Live migration failed %v", m.migrationFailedWithError), abortStatus)
			return
		}

		stats := completedJobInfo
		if stats == nil {
			stats, err = dom.GetJobStats(0)
			if err != nil {
				logger.Reason(err).Warning("failed to get domain job info, will retry")
				continue
			}
		}

		if stats.DataRemainingSet {
			m.remainingData = stats.DataRemaining
		}

		migrationUID := vmi.Status.MigrationState.MigrationUID
		if vmi.Status.MigrationState.SourceState != nil {
			migrationUID = vmi.Status.MigrationState.SourceState.MigrationUID
		}
		switch stats.Type {
		case libvirt.DOMAIN_JOB_UNBOUNDED:
			aborted := m.processInflightMigration(dom, stats)
			if aborted != nil {
				logger.Errorf("Live migration abort detected with reason: %s", aborted.message)
				m.l.setMigrationResult(true, aborted.message, aborted.abortStatus)
				return
			}
			logInterval++
			if logInterval%monitorLogInterval == 0 {
				logMigrationInfo(logger, string(migrationUID), stats)
			}
		case libvirt.DOMAIN_JOB_NONE:
			completedJobInfo = m.determineNonRunningMigrationStatus(dom)
		case libvirt.DOMAIN_JOB_CANCELLED:
			logger.Info("Migration was canceled")
			m.l.setMigrationResult(true, "Live migration aborted ", v1.MigrationAbortSucceeded)
			return
		}
	}
}

// logMigrationInfo logs the same migration info as `virsh -r domjobinfo`
func logMigrationInfo(logger *log.FilteredLogger, uid string, info *libvirt.DomainJobInfo) {
	bToMiB := func(bytes uint64) uint64 {
		return bytes / 1024 / 1024
	}

	bpsToMbps := func(bytes uint64) uint64 {
		return bytes * 8 / 1000000
	}

	logger.V(2).Info(fmt.Sprintf(`Migration info for %s: TimeElapsed:%dms DataProcessed:%dMiB DataRemaining:%dMiB DataTotal:%dMiB `+
		`MemoryProcessed:%dMiB MemoryRemaining:%dMiB MemoryTotal:%dMiB MemoryBandwidth:%dMbps DirtyRate:%dMbps `+
		`Iteration:%d PostcopyRequests:%d ConstantPages:%d NormalPages:%d NormalData:%dMiB ExpectedDowntime:%dms `+
		`DiskMbps:%d`,
		uid, info.TimeElapsed, bToMiB(info.DataProcessed), bToMiB(info.DataRemaining), bToMiB(info.DataTotal),
		bToMiB(info.MemProcessed), bToMiB(info.MemRemaining), bToMiB(info.MemTotal), bpsToMbps(info.MemBps), bpsToMbps(info.MemDirtyRate*info.MemPageSize),
		info.MemIteration, info.MemPostcopyReqs, info.MemConstant, info.MemNormal, bToMiB(info.MemNormalBytes), info.Downtime,
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
		stats, err := dom.GetJobInfo()
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("failed to get domain job info")
			return
		}
		if stats.Type == libvirt.DOMAIN_JOB_UNBOUNDED {
			err := dom.AbortJob()
			if err != nil {
				log.Log.Object(vmi).Reason(err).Error("failed to cancel migration")
				l.setMigrationAbortStatus(v1.MigrationAbortFailed)
				return
			}
			l.setMigrationResult(true, "Live migration aborted ", v1.MigrationAbortSucceeded)
			log.Log.Object(vmi).Info("Live migration abort succeeded")
		}
		return
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
	var path string
	if disk.Source == nil {
		return -1, fmt.Errorf("empty source for the disk")
	}
	switch {
	case disk.Source.File != nil:
		path = disk.Source.File.File
	case disk.Source.Block != nil:
		path = disk.Source.Block.Dev
	default:
		return -1, fmt.Errorf("not path set")
	}
	info, err := osdisk.GetDiskInfo(path)
	if err != nil {
		return -1, err
	}
	return info.VirtualSize, nil
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
	if virtutil.IsNonRootVMI(vmi) {
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

	log.Log.Object(vmi).Errorf("migration completed successfully")
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
	if options.AllowPostCopy {
		return
	}
	if options.ParallelMigrationThreads == nil {
		return
	}

	shouldConfigure = true
	threadsCount = int(*options.ParallelMigrationThreads)
	return
}
