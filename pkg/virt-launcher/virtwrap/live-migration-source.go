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
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	virtutil "kubevirt.io/kubevirt/pkg/util"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/sriov"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
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
	progressWatermark  int64
	remainingData      int64

	progressTimeout          int64
	acceptableCompletionTime int64
	migrationFailedWithError error
}

type inflightMigrationAborted struct {
	message     string
	abortStatus v1.MigrationAbortStatus
}

func generateMigrationFlags(isBlockMigration, isUnsafeMigration, allowAutoConverge, allowPostyCopy, migratePaused bool) libvirt.DomainMigrateFlags {
	migrateFlags := libvirt.MIGRATE_LIVE | libvirt.MIGRATE_PEER2PEER | libvirt.MIGRATE_PERSIST_DEST

	if isBlockMigration {
		migrateFlags |= libvirt.MIGRATE_NON_SHARED_INC
	}
	if isUnsafeMigration {
		migrateFlags |= libvirt.MIGRATE_UNSAFE
	}
	if allowAutoConverge {
		migrateFlags |= libvirt.MIGRATE_AUTO_CONVERGE
	}
	if allowPostyCopy {
		migrateFlags |= libvirt.MIGRATE_POSTCOPY
	}
	if migratePaused {
		migrateFlags |= libvirt.MIGRATE_PAUSED
	}

	return migrateFlags

}

func hotUnplugHostDevices(virConn cli.Connection, dom cli.VirDomain) error {
	domainSpec, err := util.GetDomainSpecWithFlags(dom, 0)
	if err != nil {
		return err
	}

	eventChan := make(chan interface{}, sriov.MaxConcurrentHotPlugDevicesEvents)
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

// This returns domain xml without the migration metadata section, as it is only relevant to the source domain
// Note: Unfortunately we can't just use UnMarshall + Marshall here, as that leads to unwanted XML alterations
func migratableDomXML(dom cli.VirDomain, vmi *v1.VirtualMachineInstance) (string, error) {
	xmlstr, err := dom.GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Live migration failed. Failed to get XML.")
		return "", err
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
		case xml.EndElement:
			newLocation = location[:len(location)-1]
		}
		if len(location) >= 4 &&
			location[0] == "domain" &&
			location[1] == "metadata" &&
			location[2] == "kubevirt" &&
			location[3] == "migration" {
			continue // We're inside domain/metadata/kubevirt/migration, continue will skip elements
		}

		if err := encoder.EncodeToken(xml.CopyToken(token)); err != nil {
			log.Log.Object(vmi).Reason(err)
			return "", err
		}
	}

	if err := encoder.Flush(); err != nil {
		log.Log.Object(vmi).Reason(err)
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

	err = l.asyncMigrate(vmi, options)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Live migration failed.")
		l.setMigrationResult(vmi, true, fmt.Sprintf("%v", err), "")
		return err
	}

	return nil
}

func (l *LibvirtDomainManager) initializeMigrationMetadata(vmi *v1.VirtualMachineInstance, migrationMode v1.MigrationMode) (bool, error) {
	l.domainModifyLock.Lock()
	defer l.domainModifyLock.Unlock()

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Getting the domain for migration failed.")
		return false, err
	}

	defer dom.Free()
	domainSpec, err := l.getDomainSpec(dom)
	if err != nil {
		return false, err
	}

	migrationMetadata := domainSpec.Metadata.KubeVirt.Migration
	if migrationMetadata != nil && migrationMetadata.UID == vmi.Status.MigrationState.MigrationUID {
		if migrationMetadata.EndTimestamp == nil {
			// don't stomp on currently executing migrations
			return true, nil

		} else {
			// Don't allow the same migration UID to be executed twice.
			// Migration attempts are like pods. One shot.
			return false, fmt.Errorf("migration job already executed")
		}
	}

	now := metav1.Now()
	domainSpec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
		UID:            vmi.Status.MigrationState.MigrationUID,
		StartTimestamp: &now,
		Mode:           migrationMode,
	}
	d, err := l.setDomainSpecWithHooks(vmi, domainSpec)
	if err != nil {
		return false, err
	}
	defer d.Free()
	return false, nil
}

func (l *LibvirtDomainManager) cancelMigration(vmi *v1.VirtualMachineInstance) error {
	if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.Completed ||
		vmi.Status.MigrationState.Failed || vmi.Status.MigrationState.StartTimestamp == nil {

		return fmt.Errorf("failed to cancel migration - vmi is not migrating")
	}
	err := l.setMigrationAbortStatus(vmi, v1.MigrationAbortInProgress)
	if err != nil {
		if err == domainerrors.MigrationAbortInProgressError {
			return nil
		}
		return err
	}

	l.asyncMigrationAbort(vmi)
	return nil
}

func (l *LibvirtDomainManager) setMigrationResultHelper(vmi *v1.VirtualMachineInstance, failed bool, completed bool, reason string, abortStatus v1.MigrationAbortStatus) error {

	l.domainModifyLock.Lock()
	defer l.domainModifyLock.Unlock()

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		if domainerrors.IsNotFound(err) {
			return nil

		} else {
			log.Log.Object(vmi).Reason(err).Error("Getting the domain for completed migration failed.")
		}
		return err
	}

	defer dom.Free()
	domainSpec, err := l.getDomainSpec(dom)
	if err != nil {
		return err
	}
	migrationMetadata := domainSpec.Metadata.KubeVirt.Migration
	if migrationMetadata == nil {
		// nothing to report if migration metadata is empty
		return nil
	}

	now := metav1.Now()
	if abortStatus != "" {
		metaAbortStatus := domainSpec.Metadata.KubeVirt.Migration.AbortStatus
		if metaAbortStatus == string(abortStatus) && metaAbortStatus == string(v1.MigrationAbortInProgress) {
			return domainerrors.MigrationAbortInProgressError
		}
		domainSpec.Metadata.KubeVirt.Migration.AbortStatus = string(abortStatus)
	}

	if failed {
		domainSpec.Metadata.KubeVirt.Migration.Failed = true
		domainSpec.Metadata.KubeVirt.Migration.FailureReason = reason
	}
	if completed {
		domainSpec.Metadata.KubeVirt.Migration.Completed = true
		domainSpec.Metadata.KubeVirt.Migration.EndTimestamp = &now
	}
	d, err := l.setDomainSpecWithHooks(vmi, domainSpec)
	if err != nil {
		return err
	}
	defer d.Free()
	return nil

}

func (l *LibvirtDomainManager) setMigrationResult(vmi *v1.VirtualMachineInstance, failed bool, reason string, abortStatus v1.MigrationAbortStatus) error {
	connectionInterval := 10 * time.Second
	connectionTimeout := 60 * time.Second

	err := utilwait.PollImmediate(connectionInterval, connectionTimeout, func() (done bool, err error) {
		err = l.setMigrationResultHelper(vmi, failed, true, reason, abortStatus)
		if err != nil {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Unable to post migration results to libvirt after multiple tries")
		return err
	}
	return nil

}

func (l *LibvirtDomainManager) setMigrationAbortStatus(vmi *v1.VirtualMachineInstance, abortStatus v1.MigrationAbortStatus) error {
	connectionInterval := 10 * time.Second
	connectionTimeout := 60 * time.Second

	err := utilwait.PollImmediate(connectionInterval, connectionTimeout, func() (done bool, err error) {
		err = l.setMigrationResultHelper(vmi, false, false, "", abortStatus)
		if err != nil {
			if err == domainerrors.MigrationAbortInProgressError {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Unable to post migration results to libvirt after multiple tries")
		return err
	}
	return nil

}

func newMigrationMonitor(vmi *v1.VirtualMachineInstance, l *LibvirtDomainManager, options *cmdclient.MigrationOptions, migrationErr chan error) *migrationMonitor {

	monitor := &migrationMonitor{
		l:                        l,
		vmi:                      vmi,
		options:                  options,
		migrationErr:             migrationErr,
		progressWatermark:        int64(0),
		remainingData:            int64(0),
		progressTimeout:          options.ProgressTimeout,
		acceptableCompletionTime: options.CompletionTimeoutPerGiB * getVMIMigrationDataSize(vmi),
	}

	return monitor
}

func (m *migrationMonitor) isMigrationPostCopy(domSpec *api.DomainSpec) bool {

	if domSpec.Metadata.KubeVirt.Migration != nil && domSpec.Metadata.KubeVirt.Migration.Mode == v1.MigrationPostCopy {
		return true
	}

	return false
}

func (m *migrationMonitor) shouldTriggerTimeout(elapsed int64, domSpec *api.DomainSpec) bool {
	if m.acceptableCompletionTime == 0 {
		return false
	}

	if elapsed/int64(time.Second) > m.acceptableCompletionTime {
		return true
	}

	return false
}

func (m *migrationMonitor) shouldTriggerPostCopy(elapsed int64, domSpec *api.DomainSpec) bool {
	if m.shouldTriggerTimeout(elapsed, domSpec) && m.options.AllowPostCopy {

		return true
	}
	return false
}

func (m *migrationMonitor) isMigrationProgressing(domainSpec *api.DomainSpec) bool {
	logger := log.Log.Object(m.vmi)

	now := time.Now().UTC().UnixNano()

	// check if the migration is progressing
	progressDelay := now - m.lastProgressUpdate
	if m.progressTimeout != 0 && progressDelay/int64(time.Second) > m.progressTimeout {
		logger.Warningf("Live migration stuck for %d sec", progressDelay)
		return false
	}

	return true
}

func isMigrationAbortInProgress(domSpec *api.DomainSpec) bool {
	return domSpec.Metadata.KubeVirt.Migration != nil &&
		domSpec.Metadata.KubeVirt.Migration.AbortStatus == string(v1.MigrationAbortInProgress)
}

func (m *migrationMonitor) determineNonRunningMigrationStatus(dom cli.VirDomain) *libvirt.DomainJobInfo {
	logger := log.Log.Object(m.vmi)
	// check if an ongoing migration has been completed before we could capture the outcome
	if m.lastProgressUpdate > m.start {
		logger.Info("Migration job has probably completed before we could capture the status. Getting latest status.")
		// at this point the migration is over, but we don't know the result.
		// check if we were trying to cancel this job. In this case, finalize the migration.
		domainSpec, err := m.l.getDomainSpec(dom)
		if err != nil {
			logger.Reason(err).Error("failed to get domain spec info")
			return nil
		}
		if isMigrationAbortInProgress(domainSpec) {
			logger.Info("Migration job was canceled")
			return &libvirt.DomainJobInfo{
				Type:          libvirt.DOMAIN_JOB_CANCELLED,
				DataRemaining: uint64(m.remainingData),
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
				Type:          libvirt.DOMAIN_JOB_FAILED,
				DataRemaining: uint64(m.remainingData),
			}
		}
	}
	logger.Info("Migration job didn't start yet")
	return nil
}

func (m *migrationMonitor) processInflightMigration(dom cli.VirDomain) *inflightMigrationAborted {
	logger := log.Log.Object(m.vmi)

	// Migration is running
	now := time.Now().UTC().UnixNano()
	elapsed := now - m.start

	if (m.progressWatermark == 0) ||
		(m.progressWatermark > m.remainingData) {
		m.progressWatermark = m.remainingData
		m.lastProgressUpdate = now
	}

	domainSpec, err := m.l.getDomainSpec(dom)
	if err != nil {
		logger.Reason(err).Error("failed to get domain spec info")
		return nil
	}

	switch {
	case m.isMigrationPostCopy(domainSpec):
		// Currently, there is nothing for us to track when in Post Copy mode.
		// The reasoning here is that post copy migrations transfer the state
		// directly to the target pod in a way that results in the target pod
		// hosting the active workload while the migration completes.

		// If we were to abort the migration due to a timeout while in post copy,
		// then it would result in that active state being lost.

	case m.shouldTriggerPostCopy(elapsed, domainSpec):
		logger.Info("Starting post copy mode for migration")
		// if a migration has stalled too long, post copy will be
		// triggered when allowPostCopy is enabled
		err = dom.MigrateStartPostCopy(uint32(0))
		if err != nil {
			logger.Reason(err).Error("failed to start post migration")
		}

		err = m.l.updateVMIMigrationMode(dom, m.vmi, v1.MigrationPostCopy)
		if err != nil {
			logger.Reason(err).Error("Unable to update migration mode on domain xml")
			return nil
		}

	case !m.isMigrationProgressing(domainSpec):
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
		aborted.message = fmt.Sprintf("Live migration stuck for %d sec and has been aborted", progressDelay)
		aborted.abortStatus = v1.MigrationAbortSucceeded
		return aborted
	case m.shouldTriggerTimeout(elapsed, domainSpec):
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
		aborted.message = fmt.Sprintf("Live migration is not completed after %d sec and has been aborted", m.acceptableCompletionTime)
		aborted.abortStatus = v1.MigrationAbortSucceeded
		return aborted
	}

	return nil
}

func (m *migrationMonitor) hasMigrationErr() error {

	select {
	case err := <-m.migrationErr:
		if err != nil {
			return err
		}
	default:
	}

	return nil
}

func (m *migrationMonitor) startMonitor() {
	var completedJobInfo *libvirt.DomainJobInfo
	vmi := m.vmi

	m.start = time.Now().UTC().UnixNano()
	m.lastProgressUpdate = m.start

	logger := log.Log.Object(vmi)

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := m.l.virConn.LookupDomainByName(domName)
	if err != nil {
		logger.Reason(err).Error("Live migration failed.")
		m.l.setMigrationResult(vmi, true, fmt.Sprintf("%v", err), "")
		return
	}
	defer dom.Free()

	for {
		time.Sleep(400 * time.Millisecond)

		err := m.hasMigrationErr()
		if err != nil && m.migrationFailedWithError == nil {
			logger.Reason(err).Error("Recevied a live migration error. Will check the latest migration status.")
			m.migrationFailedWithError = err
		} else if m.migrationFailedWithError != nil {
			logger.Info("Didn't manage to get a job status. Post the received error and finilize.")
			logger.Reason(m.migrationFailedWithError).Error("Live migration failed")
			var abortStatus v1.MigrationAbortStatus
			if strings.Contains(m.migrationFailedWithError.Error(), "canceled by client") {
				abortStatus = v1.MigrationAbortSucceeded
			}
			m.l.setMigrationResult(vmi, true, fmt.Sprintf("Live migration failed %v", m.migrationFailedWithError), abortStatus)
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
		m.remainingData = int64(stats.DataRemaining)
		switch stats.Type {
		case libvirt.DOMAIN_JOB_UNBOUNDED:
			aborted := m.processInflightMigration(dom)
			if aborted != nil {
				logger.Errorf("Live migration abort detected with reason: %s", aborted.message)
				m.l.setMigrationResult(vmi, true, aborted.message, aborted.abortStatus)
				return
			}
		case libvirt.DOMAIN_JOB_NONE:
			completedJobInfo = m.determineNonRunningMigrationStatus(dom)
		case libvirt.DOMAIN_JOB_COMPLETED:
			logger.Info("Migration has been completed")
			m.l.setMigrationResult(vmi, false, "", "")
			return
		case libvirt.DOMAIN_JOB_FAILED:
			logger.Info("Migration job failed")
			m.l.setMigrationResult(vmi, true, fmt.Sprintf("%v", m.migrationFailedWithError), "")
			return
		case libvirt.DOMAIN_JOB_CANCELLED:
			logger.Info("Migration was canceled")
			m.l.setMigrationResult(vmi, true, "Live migration aborted ", v1.MigrationAbortSucceeded)
			return
		}
	}
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
				l.setMigrationAbortStatus(vmi, v1.MigrationAbortFailed)
				return
			}
			log.Log.Object(vmi).Info("Live migration abort succeeded")
		}
		return
	}(l, vmi)
}

func isBlockMigration(vmi *v1.VirtualMachineInstance) bool {
	return (vmi.Status.MigrationMethod == v1.BlockMigration)
}

func generateMigrationParams(dom cli.VirDomain, vmi *v1.VirtualMachineInstance, options *cmdclient.MigrationOptions, virtShareDir string) (*libvirt.DomainMigrateParameters, error) {
	bandwidth, err := converter.QuantityToMebiByte(options.Bandwidth)
	if err != nil {
		return nil, err
	}

	xmlstr, err := migratableDomXML(dom, vmi)
	if err != nil {
		return nil, err
	}

	key := migrationproxy.ConstructProxyKey(string(vmi.UID), migrationproxy.LibvirtDirectMigrationPort)
	migrURI := fmt.Sprintf("unix://%s", migrationproxy.SourceUnixFile(virtShareDir, key))
	params := &libvirt.DomainMigrateParameters{
		URI:           migrURI,
		URISet:        true,
		Bandwidth:     bandwidth, // MiB/s
		BandwidthSet:  bandwidth > 0,
		DestXML:       xmlstr,
		DestXMLSet:    true,
		PersistXML:    xmlstr,
		PersistXMLSet: true,
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
	migrateFlags := generateMigrationFlags(isBlockMigration(vmi), options.UnsafeMigration, options.AllowAutoConverge, options.AllowPostCopy, migratePaused)

	// anything that modifies the domain needs to be performed with the domainModifyLock held
	// The domain params and unHotplug need to be performed in a critical section together.
	critSection := func() error {
		l.domainModifyLock.Lock()
		defer l.domainModifyLock.Unlock()

		if err := prepareDomainForMigration(l.virConn, dom); err != nil {
			return fmt.Errorf("error encountered during preparing domain for migration: %v", err)
		}

		params, err = generateMigrationParams(dom, vmi, options, l.virtShareDir)
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

func (l *LibvirtDomainManager) asyncMigrate(vmi *v1.VirtualMachineInstance, options *cmdclient.MigrationOptions) error {

	// get connection proxies for tunnelling migration through virt-handler
	go func() {
		if shouldImmediatelyFailMigration(vmi) {
			log.Log.Object(vmi).Error("Live migration failed. Failure is forced by functional tests suite.")
			l.setMigrationResult(vmi, true, "Failed migration to satisfy functional test condition", "")
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
			log.Log.Object(vmi).Reason(err).Error("Live migration failed.")
			migrationErrorChan <- err
			return
		}

		log.Log.Object(vmi).Infof("Live migration succeeded.")
	}()
	return nil
}

func (l *LibvirtDomainManager) updateVMIMigrationMode(dom cli.VirDomain, vmi *v1.VirtualMachineInstance, mode v1.MigrationMode) error {
	domainSpec, err := l.getDomainSpec(dom)
	if err != nil {
		return err
	}

	if domainSpec.Metadata.KubeVirt.Migration == nil {
		domainSpec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{}
	}

	domainSpec.Metadata.KubeVirt.Migration.Mode = mode

	d, err := l.setDomainSpecWithHooks(vmi, domainSpec)
	if err != nil {
		return err
	}
	defer d.Free()

	return nil
}
