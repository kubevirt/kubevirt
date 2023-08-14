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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package virtwrap

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"kubevirt.io/kubevirt/pkg/util/migrations"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	virtutil "kubevirt.io/kubevirt/pkg/util"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/sriov"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
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
	shared    map[string]bool
	generated map[string]bool
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
	if options.ParallelMigrationThreads != nil {
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

func injectNewSection(encoder *xml.Encoder, domain *api.Domain, section []string, logger *log.FilteredLogger) error {
	// Marshalling the whole domain, even if we just need the cputune section, for indentation purposes
	xmlstr, err := xml.MarshalIndent(domain.Spec, "", "  ")
	if err != nil {
		logger.Reason(err).Error("Live migration failed. Failed to get XML.")
		return err
	}
	decoder := xml.NewDecoder(bytes.NewReader(xmlstr))
	var location = make([]string, 0)
	var newLocation []string = nil
	injecting := false
	for {
		if newLocation != nil {
			// Postpone popping end elements from `location` to ensure their removal
			location = newLocation
			newLocation = nil
		}
		token, err := decoder.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Errorf("error getting token: %v\n", err)
			return err
		}

		switch v := token.(type) {
		case xml.StartElement:
			location = append(location, v.Name.Local)

		case xml.EndElement:
			newLocation = location[:len(location)-1]
		}

		if len(location) >= len(section) &&
			reflect.DeepEqual(location[:len(section)], section) {
			injecting = true
		} else {
			if injecting == true {
				// We just left the section block, we're done
				break
			} else {
				// We're not in the section block yet, skipping elements
				continue
			}
		}

		if injecting {
			if err := encoder.EncodeToken(xml.CopyToken(token)); err != nil {
				logger.Reason(err).Errorf("Failed to encode token %v", token)
				return err
			}
		}
	}

	return nil
}

// This function returns true for every section that should be adjusted with target data when migrating a VMI
//
//	that includes dedicated CPUs
//
// Strict mode only returns true if we just entered the block
func shouldOverrideForDedicatedCPUTarget(section []string, strict bool) bool {
	if (!strict || len(section) == 2) &&
		len(section) >= 2 &&
		section[0] == "domain" &&
		section[1] == "cputune" {
		return true
	}
	if (!strict || len(section) == 2) &&
		len(section) >= 2 &&
		section[0] == "domain" &&
		section[1] == "numatune" {
		return true
	}
	if (!strict || len(section) == 3) &&
		len(section) >= 3 &&
		section[0] == "domain" &&
		section[1] == "cpu" &&
		section[2] == "numa" {
		return true
	}

	return false
}

// This returns domain xml without the metadata section, as it is only relevant to the source domain
// Note: Unfortunately we can't just use UnMarshall + Marshall here, as that leads to unwanted XML alterations
func migratableDomXML(dom cli.VirDomain, vmi *v1.VirtualMachineInstance, domSpec *api.DomainSpec) (string, error) {
	const (
		exactLocation = true
		insideBlock   = false
	)
	var domain *api.Domain
	var err error

	xmlstr, err := dom.GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Live migration failed. Failed to get XML.")
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
	}

	decoder := xml.NewDecoder(bytes.NewReader([]byte(xmlstr)))
	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)

	var location = make([]string, 0)
	var newLocation []string = nil

	for {
		if newLocation != nil {
			// Postpone popping end elements from `location` to ensure their removal
			location = newLocation
			newLocation = nil
		}
		token, err := decoder.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Log.Object(vmi).Errorf("error getting token: %v\n", err)
			return "", err
		}

		switch v := token.(type) {
		case xml.StartElement:
			location = append(location, v.Name.Local)

			// If the VMI requires dedicated CPUs, we need to patch the domain with
			// the new CPU/NUMA info calculated for the target node prior to migration
			if vmi.IsCPUDedicated() && shouldOverrideForDedicatedCPUTarget(location, exactLocation) {
				err = injectNewSection(encoder, domain, location, log.Log.Object(vmi))
				if err != nil {
					return "", err
				}
			}
		case xml.EndElement:
			newLocation = location[:len(location)-1]
		}
		if vmi.IsCPUDedicated() && shouldOverrideForDedicatedCPUTarget(location, insideBlock) {
			continue
		}

		if err := encoder.EncodeToken(xml.CopyToken(token)); err != nil {
			log.Log.Object(vmi).Reason(err).Errorf("Failed to encode token %v", token)
			return "", err
		}
	}

	if err := encoder.Flush(); err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to flush XML encoder")
		return "", err
	}

	return string(buf.Bytes()), nil
}

func (d *migrationDisks) isSharedVolume(name string) bool {
	_, shared := d.shared[name]
	return shared
}

func (d *migrationDisks) isGeneratedVolume(name string) bool {
	_, generated := d.generated[name]
	return generated
}

func classifyVolumesForMigration(vmi *v1.VirtualMachineInstance) *migrationDisks {
	// This method collects all VMI volumes that should not be copied during
	// live migration. It also collects all generated disks suck as cloudinit, secrets, ServiceAccount and ConfigMaps
	// to make sure that these are being copied during migration.
	// Persistent volume claims without ReadWriteMany access mode
	// should be filtered out earlier in the process

	disks := &migrationDisks{
		shared:    make(map[string]bool),
		generated: make(map[string]bool),
	}
	for _, volume := range vmi.Spec.Volumes {
		volSrc := volume.VolumeSource
		if volSrc.PersistentVolumeClaim != nil || volSrc.DataVolume != nil ||
			(volSrc.HostDisk != nil && *volSrc.HostDisk.Shared) {
			disks.shared[volume.Name] = true
		}
		if volSrc.ConfigMap != nil || volSrc.Secret != nil || volSrc.DownwardAPI != nil ||
			volSrc.ServiceAccount != nil || volSrc.CloudInitNoCloud != nil ||
			volSrc.CloudInitConfigDrive != nil || volSrc.ContainerDisk != nil {
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
		return fmt.Errorf("cannot migration VMI until migrationState is ready")
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
	if exists && migrationMetadata.UID == vmi.Status.MigrationState.MigrationUID {
		if migrationMetadata.EndTimestamp == nil {
			// don't stop on currently executing migrations
			return true, nil
		} else {
			// Don't allow the same migration UID to be executed twice.
			// Migration attempts are like pods. One shot.
			return false, fmt.Errorf("migration job %v already executed, finished at %v, completed: %t, failed: %t, abortStatus: %s",
				migrationMetadata.UID, *migrationMetadata.EndTimestamp, migrationMetadata.Completed, migrationMetadata.Failed, migrationMetadata.AbortStatus)
		}
	}

	now := metav1.Now()
	m := api.MigrationMetadata{
		UID:            vmi.Status.MigrationState.MigrationUID,
		StartTimestamp: &now,
		Mode:           migrationMode,
	}
	l.metadataCache.Migration.Store(m)
	log.Log.V(4).Infof("initialize migration metadata: %v", m)
	return false, nil
}

func (l *LibvirtDomainManager) cancelMigration(vmi *v1.VirtualMachineInstance) error {
	migration, _ := l.metadataCache.Migration.Load()
	if migration.Completed || migration.Failed || migration.StartTimestamp == nil {
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

func (l *LibvirtDomainManager) setMigrationResultHelper(failed bool, completed bool, reason string, abortStatus v1.MigrationAbortStatus) error {
	migrationMetadata, exists := l.metadataCache.Migration.Load()
	if !exists {
		// nothing to report if migration metadata is empty
		return nil
	}

	if abortStatus != "" {
		metaAbortStatus := migrationMetadata.AbortStatus
		if metaAbortStatus == string(abortStatus) && metaAbortStatus == string(v1.MigrationAbortInProgress) {
			return domainerrors.MigrationAbortInProgressError
		}
	}

	l.metadataCache.Migration.WithSafeBlock(func(migrationMetadata *api.MigrationMetadata, _ bool) {
		if abortStatus != "" {
			migrationMetadata.AbortStatus = string(abortStatus)
		}

		if failed {
			migrationMetadata.Failed = true
			migrationMetadata.FailureReason = reason
		}
		if completed {
			migrationMetadata.Completed = true
			now := metav1.Now()
			migrationMetadata.EndTimestamp = &now
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
	return l.setMigrationResultHelper(failed, true, reason, abortStatus)
}

func (l *LibvirtDomainManager) setMigrationAbortStatus(abortStatus v1.MigrationAbortStatus) error {
	return l.setMigrationResultHelper(false, false, "", abortStatus)
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

func (m *migrationMonitor) shouldTriggerTimeout(elapsed int64) bool {
	if m.acceptableCompletionTime == 0 {
		return false
	}

	return elapsed/int64(time.Second) > m.acceptableCompletionTime
}

func (m *migrationMonitor) shouldTriggerPostCopy(elapsed int64) bool {
	return m.shouldTriggerTimeout(elapsed) && m.options.AllowPostCopy
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
				DataRemaining:    uint64(m.remainingData),
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
				DataRemaining:    uint64(m.remainingData),
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

	case m.shouldTriggerPostCopy(elapsed):
		logger.Info("Starting post copy mode for migration")
		// if a migration has stalled too long, post copy will be
		// triggered when allowPostCopy is enabled
		err := dom.MigrateStartPostCopy(uint32(0))
		if err != nil {
			logger.Reason(err).Error("failed to start post migration")
			return nil
		}

		m.l.updateVMIMigrationMode(v1.MigrationPostCopy)

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

func (m *migrationMonitor) hasMigrationErr() error {
	select {
	case err := <-m.migrationErr:
		return err
	default:
		return nil
	}
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
		time.Sleep(monitorSleepPeriodMS * time.Millisecond)

		err := m.hasMigrationErr()
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
			m.l.setMigrationResult(true, fmt.Sprintf("Live migration failed %v", m.migrationFailedWithError), abortStatus)
			return
		}

		stats := completedJobInfo
		if stats == nil {
			stats, err = dom.GetJobStats(0)
			if err != nil {
				logger.Reason(err).Error("failed to get domain job info")
				continue
			}
		}

		if stats.DataRemainingSet {
			m.remainingData = stats.DataRemaining
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
				logMigrationInfo(logger, string(vmi.Status.MigrationState.MigrationUID), stats)
			}
		case libvirt.DOMAIN_JOB_NONE:
			completedJobInfo = m.determineNonRunningMigrationStatus(dom)
		case libvirt.DOMAIN_JOB_COMPLETED:
			logger.Info("Migration has been completed")
			m.l.setMigrationResult(false, "", "")
			return
		case libvirt.DOMAIN_JOB_FAILED:
			logger.Info("Migration job failed")
			m.l.setMigrationResult(true, fmt.Sprintf("%v", m.migrationFailedWithError), "")
			return
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

	bToMbps := func(bytes uint64) uint64 {
		return bytes / 8 / 1000000
	}

	logger.V(4).Info(fmt.Sprintf(`Migration info for %s: TimeElapsed:%dms DataProcessed:%dMiB DataRemaining:%dMiB DataTotal:%dMiB `+
		`MemoryProcessed:%dMiB MemoryRemaining:%dMiB MemoryTotal:%dMiB MemoryBandwidth:%dMbps DirtyRate:%dMbps `+
		`Iteration:%d PostcopyRequests:%d ConstantPages:%d NormalPages:%d NormalData:%dMiB ExpectedDowntime:%dms `+
		`DiskMbps:%d`,
		uid, info.TimeElapsed, bToMiB(info.DataProcessed), bToMiB(info.DataRemaining), bToMiB(info.DataTotal),
		bToMiB(info.MemProcessed), bToMiB(info.MemRemaining), bToMiB(info.MemTotal), bToMbps(info.MemBps), bToMbps(info.MemDirtyRate*info.MemPageSize),
		info.MemIteration, info.MemPostcopyReqs, info.MemConstant, info.MemNormal, bToMiB(info.MemNormalBytes), info.Downtime,
		bToMbps(info.DiskBps),
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
			log.Log.Object(vmi).Info("Live migration abort succeeded")
		}
		return
	}(l, vmi)
}

func isBlockMigration(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Status.MigrationMethod == v1.BlockMigration
}

func generateMigrationParams(dom cli.VirDomain, vmi *v1.VirtualMachineInstance, options *cmdclient.MigrationOptions, virtShareDir string, domSpec *api.DomainSpec) (*libvirt.DomainMigrateParameters, error) {
	bandwidth, err := vcpu.QuantityToMebiByte(options.Bandwidth)
	if err != nil {
		return nil, err
	}

	xmlstr, err := migratableDomXML(dom, vmi, domSpec)
	if err != nil {
		return nil, err
	}

	parallelMigrationSet := false
	var parallelMigrationThreads int
	if options.ParallelMigrationThreads != nil {
		parallelMigrationSet = true
		parallelMigrationThreads = int(*options.ParallelMigrationThreads)
	}

	key := migrationproxy.ConstructProxyKey(string(vmi.UID), migrationproxy.LibvirtDirectMigrationPort)
	migrURI := fmt.Sprintf("unix://%s", migrationproxy.SourceUnixFile(virtShareDir, key))
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
	}

	log.Log.Object(vmi).Infof("generated migration parameters: %+v", params)
	return params, nil
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
	migrateFlags := generateMigrationFlags(isBlockMigration(vmi), migratePaused, options)

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
		return fmt.Errorf("error encountered during MigrateToURI3 libvirt api call: %v", err)
	}

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
