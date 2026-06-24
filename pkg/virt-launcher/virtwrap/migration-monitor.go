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
	"fmt"
	"time"

	"libvirt.org/go/libvirt"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/statsconv"
	"kubevirt.io/kubevirt/pkg/vmitrait"
)

const (
	monitorSleepPeriodMS = 400
	monitorLogPeriodMS   = 4000
	monitorLogInterval   = monitorLogPeriodMS / monitorSleepPeriodMS
)

type migrationMonitor struct {
	l       *LibvirtDomainManager
	vmi     *v1.VirtualMachineInstance
	options *cmdclient.MigrationOptions

	migrationDone <-chan struct{}

	start              int64
	lastProgressUpdate int64
	progressWatermark  uint64
	remainingData      uint64

	progressTimeout          int64
	acceptableCompletionTime int64
}

func newMigrationMonitor(vmi *v1.VirtualMachineInstance, l *LibvirtDomainManager, options *cmdclient.MigrationOptions, migrationDone <-chan struct{}) *migrationMonitor {
	monitor := &migrationMonitor{
		l:                        l,
		vmi:                      vmi,
		options:                  options,
		migrationDone:            migrationDone,
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
	return m.options.AllowWorkloadDisruption && m.shouldTriggerTimeout(elapsed)
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

func (m *migrationMonitor) processInflightMigration(dom cli.VirDomain, stats *libvirt.DomainJobInfo) {
	logger := log.Log.Object(m.vmi)

	// only send abort if one is not in progress already or if a previous attempt failed
	shouldAbort := func() bool {
		migration, _ := m.l.metadataCache.Migration.Load()
		return migration.AbortStatus == "" || migration.AbortStatus == string(v1.MigrationAbortFailed)
	}

	now := time.Now().UTC().UnixNano()
	elapsed := now - m.start

	if stats != nil && stats.Type == libvirt.DOMAIN_JOB_UNBOUNDED {
		m.l.domainInfoStats = statsconv.Convert_libvirt_DomainJobInfo_To_stats_DomainJobInfo(stats)
		if stats.DataRemainingSet {
			m.remainingData = stats.DataRemaining
		}
		if (m.progressWatermark == 0) || (m.remainingData < m.progressWatermark) {
			m.lastProgressUpdate = now
		}
		m.progressWatermark = m.remainingData
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
		if m.options.AllowPostCopy && !vmitrait.HasVFIO(m.vmi) {
			logger.Info("Starting post copy mode for migration")
			// if a migration has stalled too long, post copy will be
			// triggered when allowPostCopy is enabled (post-copy is not supported with VFIO devices)
			err := dom.MigrateStartPostCopy(0)
			if err != nil {
				logger.Reason(err).Error("failed to start post migration")
				return
			}
			m.l.updateVMIMigrationMode(v1.MigrationPostCopy)
		} else if vmitrait.HasVFIO(m.vmi) {
			logger.Info("Setting large max downtime to trigger migration switchover")
			// TODO: once the VGPULiveMigration featuregate graduates
			//  (and even possibly other VFIO live migration featuregates)
			//  we should consider merging this with the "else" case below.
			// Setting a very high max downtime causes QEMU to trigger its
			// internal switchover, which pauses vCPUs and transitions VFIO
			// devices to _STOP_COPY. This is more correct than dom.Suspend()
			// which only pauses vCPUs but leaves VFIO devices in _RUNNING
			// with perpetual dirty page reporting.
			maxDowntimeSec := m.acceptableCompletionTime * 2
			// qemu doesn't allow max downtime larger than 2000s
			err := dom.MigrateSetMaxDowntime(min(uint64(maxDowntimeSec)*1000, 2_000_000), 0)
			if err != nil {
				logger.Reason(err).Error("Setting max downtime failed.")
				return
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
				return
			}
			logger.Infof("Signaled pause for %s", m.vmi.GetObjectMeta().GetName())

			// update acceptableCompletionTime to prevent premature migration
			// cancellation
			m.acceptableCompletionTime *= 2
			m.l.paused.add(m.vmi.UID)
			m.l.updateVMIMigrationMode(v1.MigrationPaused)
		}

	case !m.isMigrationProgressing():
		// The migration is completely stuck.
		// It usually indicates a problem with the network or qemu's connection handling.
		// In this case, we abort the migration directly without trying to pause/post-copy,
		// since the problem is highly unlikely to be caused by a high dirty rate.
		if shouldAbort() {
			progressDelay := now - m.lastProgressUpdate
			logger.Warningf("Aborting migration: stuck for %d seconds", progressDelay/int64(time.Second))
			m.l.cancelMigration(m.vmi)
		}

	case m.shouldTriggerTimeout(elapsed):
		// check the overall migration time
		// if the total migration time exceeds an acceptable
		// limit, then the migration will get aborted, but
		// only if post copy migration hasn't been enabled
		if shouldAbort() {
			logger.Warningf("Aborting migration: not completed after %d seconds", m.acceptableCompletionTime)
			m.l.cancelMigration(m.vmi)
		}
	}
}

func (m *migrationMonitor) startMonitor(ready chan<- error) {
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
		ready <- fmt.Errorf("migration monitor failed to look up domain: %v", err)
		return
	}
	defer dom.Free()
	close(ready) // signal we're ready to monitor migration

	logInterval := 0

	for {
		select {
		case <-m.migrationDone:
			return
		case <-time.After(monitorSleepPeriodMS * time.Millisecond):
		}

		jobStats, err := dom.GetJobStats(0)
		if err != nil {
			logger.Reason(err).Info("failed to get domain job info")
			jobStats = nil
		}

		m.processInflightMigration(dom, jobStats)

		if jobStats != nil && jobStats.Type == libvirt.DOMAIN_JOB_UNBOUNDED {
			logInterval++
			if logInterval%monitorLogInterval == 0 {
				LogMigrationInfo(logger, MigrationUID(vmi), jobStats)
			}
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
