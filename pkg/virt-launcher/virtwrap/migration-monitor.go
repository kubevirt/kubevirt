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

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	migrationutils "kubevirt.io/kubevirt/pkg/util/migrations"
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

const (
	defaultInitialDowntimeMs   int64 = 150
	defaultDowntimeSteps       int32 = 7
	defaultStartAfterIteration int64 = 3
	defaultCooldownSeconds     int32 = 10
)

type downtimeTuningConfig struct {
	MaxDowntimeMs       uint64
	InitialMs           int64
	Steps               int32
	StartAfterIteration int64
	CooldownSeconds     int32
}

type migrationMonitor struct {
	l       *LibvirtDomainManager
	vmi     *v1.VirtualMachineInstance
	options *cmdclient.MigrationOptions

	migrationDone <-chan struct{}
	iterationCh   chan int

	// deadline in seconds for the end-to-end migration to complete
	acceptableCompletionTime int64
	// deadline in seconds for switchover to post-copy or stop-and-copy; initialized as the same value as acceptableCompletionTime
	switchOverDeadline int64
	// timestamp in unix nano migration began
	start int64
	// most recent iteration record (remaining bytes, time elapsed) as reported by QEMU
	iterationRecord iterationRecord
	// whether stall detection is enabled or to use the legacy path
	stallDetectionEnabled bool

	stallDetector *stallDetector
	logger        *log.FilteredLogger

	downtimeTuning    *downtimeTuningConfig
	currentDowntimeMs uint64
	lastTunedAt       time.Time

	// TODO: fields used by legacy stall detector; to be removed
	lastProgressUpdate int64
	progressWatermark  uint64
	remainingData      uint64
	progressTimeout    int64
}

func newMigrationMonitor(vmi *v1.VirtualMachineInstance, l *LibvirtDomainManager, options *cmdclient.MigrationOptions, migrationDone <-chan struct{}) *migrationMonitor {
	stallDetectorEnabled := options.StallDetectorOptions != nil
	monitor := &migrationMonitor{
		l:                        l,
		vmi:                      vmi,
		options:                  options,
		migrationDone:            migrationDone,
		progressWatermark:        0,
		remainingData:            0,
		progressTimeout:          options.ProgressTimeout,
		acceptableCompletionTime: options.CompletionTimeoutPerGiB * getVMIMigrationDataSize(vmi, l.ephemeralDiskDir),
		stallDetectionEnabled:    stallDetectorEnabled,
		logger:                   log.Log.Object(vmi),
	}

	if stallDetectorEnabled {
		monitor.iterationCh = make(chan int, 16)
		monitor.switchOverDeadline = monitor.acceptableCompletionTime
		monitor.stallDetector = &stallDetector{
			stallDetectorOptions:    *options.StallDetectorOptions,
			maxDowntimeMs:           options.MaxDowntimeMs,
			allowPostCopy:           options.AllowPostCopy,
			allowWorkloadDisruption: options.AllowWorkloadDisruption,
			hasVFIO:                 vmitrait.HasVFIO(vmi),
		}
		monitor.logger.V(3).Infof(
			"initialized migration monitor: stallDetection=%t progressTimeout=%ds completionTimeoutPerGiB=%d maxDowntimeMs=%d allowPostCopy=%t allowWorkloadDisruption=%t "+
				"stallMargin=%d%% stallProgressTimeout=%ds switchoverTimeout=%ds preCopyPossibleFactor=%g patienceWindowDecayFactor=%g bandwidthEWMAAlpha=%g searchLocalMinima=%t completionTimeoutFactor=%g",
			stallDetectorEnabled,
			options.ProgressTimeout,
			options.CompletionTimeoutPerGiB,
			options.MaxDowntimeMs,
			options.AllowPostCopy,
			options.AllowWorkloadDisruption,
			options.StallDetectorOptions.StallMargin,
			options.StallDetectorOptions.StallProgressTimeout,
			options.StallDetectorOptions.SwitchoverTimeout,
			options.StallDetectorOptions.PrecopyPossibleFactor.AsApproximateFloat64(),
			options.StallDetectorOptions.PatienceWindowDecayFactor.AsApproximateFloat64(),
			options.StallDetectorOptions.EwmaAlpha.AsApproximateFloat64(),
			options.StallDetectorOptions.SearchLocalMinima,
			options.StallDetectorOptions.CompletionTimeoutFactor.AsApproximateFloat64(),
		)
		// TODO: this limitation is actively being worked on; remove when resolved. ETA: QEMU 11.1
		if vmitrait.HasVFIO(vmi) {
			monitor.logger.Warning("VFIO VM detected: QEMU remaining-bytes signals may under-report outstanding migration data for VFIO devices. This is a known limitation.")
		}
	}

	monitor.downtimeTuning = newDowntimeTuningConfig(options.MaxDowntimeMs, options.DowntimeTuning)
	if monitor.downtimeTuning != nil {
		monitor.logger.Infof("downtime tuning enabled: initial=%dms steps=%d startAfterIteration=%d cooldown=%ds ceiling=%dms",
			monitor.downtimeTuning.InitialMs, monitor.downtimeTuning.Steps,
			monitor.downtimeTuning.StartAfterIteration, monitor.downtimeTuning.CooldownSeconds,
			monitor.downtimeTuning.MaxDowntimeMs)
	}

	return monitor
}

func newDowntimeTuningConfig(maxDowntimeMs uint64, dt *v1.DowntimeTuningOptions) *downtimeTuningConfig {
	if dt == nil {
		return nil
	}

	cfg := &downtimeTuningConfig{
		MaxDowntimeMs:       maxDowntimeMs,
		InitialMs:           defaultInitialDowntimeMs,
		Steps:               defaultDowntimeSteps,
		StartAfterIteration: defaultStartAfterIteration,
		CooldownSeconds:     defaultCooldownSeconds,
	}
	if dt.InitialMs != nil && *dt.InitialMs >= 0 {
		cfg.InitialMs = *dt.InitialMs
	}
	if dt.Steps != nil {
		cfg.Steps = max(*dt.Steps, 1)
	}
	if dt.StartAfterIteration != nil && *dt.StartAfterIteration >= 0 {
		cfg.StartAfterIteration = *dt.StartAfterIteration
	}
	if dt.CooldownSeconds != nil {
		cfg.CooldownSeconds = max(*dt.CooldownSeconds, 1)
	}
	if cfg.InitialMs > int64(cfg.MaxDowntimeMs) {
		cfg.InitialMs = int64(cfg.MaxDowntimeMs)
	}
	return cfg
}

func (m *migrationMonitor) tuneDowntime(dom cli.VirDomain, stats *libvirt.DomainJobInfo, logger *log.FilteredLogger) {
	cfg := m.downtimeTuning
	if cfg == nil {
		return
	}

	var newDowntime uint64
	switch {
	case m.currentDowntimeMs == 0:
		m.currentDowntimeMs = migrationutils.QEMUDefaultTargetDowntimeMS
		newDowntime = uint64(cfg.InitialMs)
	case stats == nil || !stats.MemIterationSet:
		return
	case stats.MemIteration < uint64(cfg.StartAfterIteration):
		return
	case time.Since(m.lastTunedAt) < time.Duration(cfg.CooldownSeconds)*time.Second:
		return
	default:
		step := max((cfg.MaxDowntimeMs-uint64(cfg.InitialMs))/uint64(cfg.Steps), 1)
		newDowntime = min(m.currentDowntimeMs+step, cfg.MaxDowntimeMs)
		if newDowntime <= m.currentDowntimeMs {
			return
		}
	}

	if err := dom.MigrateSetMaxDowntime(newDowntime, 0); err != nil {
		logger.Reason(err).Warningf("downtime tuning: failed to set max_downtime to %dms", newDowntime)
		return
	}
	logger.V(2).Infof("downtime tuning: max_downtime %dms -> %dms (ceiling=%dms)",
		m.currentDowntimeMs, newDowntime, cfg.MaxDowntimeMs)
	m.currentDowntimeMs = newDowntime
	m.lastTunedAt = time.Now()
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
	if !m.stallDetectionEnabled {
		return elapsedSeconds > m.acceptableCompletionTime
	}

	if m.isPausedMigration() {
		logger.V(4).Infof("shouldTriggerTimeout: elapsedSeconds=%ds acceptableCompletionTime=%ds paused=true", elapsedSeconds, m.acceptableCompletionTime)
		return elapsedSeconds > m.acceptableCompletionTime
	}

	logger.V(4).Infof("shouldTriggerTimeout: elapsedSeconds=%ds switchOverDeadline=%ds paused=false", elapsedSeconds, m.switchOverDeadline)
	return elapsedSeconds > m.switchOverDeadline
}

func (m *migrationMonitor) shouldAssistMigrationToComplete(elapsedNs int64, logger *log.FilteredLogger) bool {
	return m.options.AllowWorkloadDisruption && m.shouldTriggerTimeout(elapsedNs, logger) && !m.stallDetectionEnabled
}

func (m *migrationMonitor) scaledCompletionDeadlineSeconds(baseSeconds int64) int64 {
	completionTimeoutFactor := m.options.StallDetectorOptions.CompletionTimeoutFactor.AsApproximateFloat64()
	m.logger.V(4).Infof("scaledCompletionDeadlineSeconds: baseSeconds=%ds, completionTimeoutFactor=%g", baseSeconds, completionTimeoutFactor)
	return int64(float64(baseSeconds) * completionTimeoutFactor)
}

func (m *migrationMonitor) isMigrationProgressing() bool {
	now := time.Now().UTC().UnixNano()

	// check if the migration is progressing
	progressDelay := (now - m.lastProgressUpdate) / int64(time.Second)
	if m.progressTimeout != 0 && progressDelay > m.progressTimeout {
		m.logger.Warningf("live migration stuck for %d seconds", progressDelay)
		return false
	}

	return true
}

func (m *migrationMonitor) isAbortInProgress() bool {
	migration, _ := m.l.metadataCache.Migration.Load()
	return migration.AbortStatus != "" && migration.AbortStatus != string(v1.MigrationAbortFailed)
}

func (m *migrationMonitor) processCompletionTimeouts(dom cli.VirDomain, elapsedNs int64, estimatedDowntimeMs uint32, logger *log.FilteredLogger) {
	sd := m.stallDetector

	if !m.shouldTriggerTimeout(elapsedNs, logger) {
		return
	}

	if m.isMigrationPostCopy() {
		return
	}

	if sd.ewmaBandwidthBps == 0 {
		// In a typical migration, this case should not be possible.
		logger.Error("aborting migration due to illegal state: value of ewmaBandwidthBps not set!")
		m.l.cancelMigration(m.vmi)
		return
	}

	elapsedSeconds := elapsedNs / int64(time.Second)

	if !m.stallDetector.switchoverInitiated {

		// safety guard that protects against triggering a switch-over during a network drop
		completable := sd.canFinishByDeadline(elapsedSeconds, m.scaledCompletionDeadlineSeconds(m.acceptableCompletionTime), estimatedDowntimeMs, logger)

		if m.options.AllowPostCopy && !vmitrait.HasVFIO(m.vmi) && completable {
			logger.Info("completion timeout reached: starting post-copy mode to force convergence")
			if err := dom.MigrateStartPostCopy(0); err != nil {
				logger.Reason(err).Error("failed to start post-copy migration")
				return
			}
			m.l.updateVMIMigrationMode(v1.MigrationPostCopy)
			sd.switchoverInitiated = true
			return
		}
		switchoverTimeout := sd.stallDetectorOptions.SwitchoverTimeout
		if m.options.AllowWorkloadDisruption && completable {
			logger.Infof("completion timeout reached: setting max downtime to %dms to force switchover", migrationutils.QEMUMaxMigrationDowntimeMS)
			if err := dom.MigrateSetMaxDowntime(migrationutils.QEMUMaxMigrationDowntimeMS, 0); err != nil {
				logger.Reason(err).Error("setting max downtime failed")
				return
			}
			m.acceptableCompletionTime = m.scaledCompletionDeadlineSeconds(m.acceptableCompletionTime)
			m.switchOverDeadline = elapsedSeconds + switchoverTimeout
			sd.switchoverInitiated = true
			return
		}

	}

	logger.Infof("aborting migration due to completion timeout: elapsedSec=%d acceptableCompletionSec=%d", elapsedSeconds, m.acceptableCompletionTime)
	m.l.cancelMigration(m.vmi)
}

func (m *migrationMonitor) triggerConvergenceAction(dom cli.VirDomain, action convergenceAction, reason string, logger *log.FilteredLogger) {
	sd := m.stallDetector

	sd.switchoverInitiated = true

	switch action {
	case actionNothing:
		sd.switchoverInitiated = false
		logger.V(3).Infof("convergence action is nothing because: %s", reason)
	case actionAbort:
		logger.Warningf("aborting migration: %s", reason)
		m.l.cancelMigration(m.vmi)
	case actionPostCopy:
		logger.Infof("starting post copy mode for migration: %s", reason)
		if err := dom.MigrateStartPostCopy(0); err != nil {
			sd.switchoverInitiated = false
			logger.Reason(err).Error("failed to start post migration")
			return
		}
		m.l.updateVMIMigrationMode(v1.MigrationPostCopy)
	case actionHardStopAndCopy, actionSoftStopAndCopy:
		now := time.Now().UTC().UnixNano()
		elapsedSeconds := (now - m.start) / int64(time.Second)
		switchoverTimeout := sd.stallDetectorOptions.SwitchoverTimeout

		var downtime uint64
		if action == actionHardStopAndCopy {
			downtime = migrationutils.QEMUMaxMigrationDowntimeMS
			logger.Infof("forcing switchover by setting max downtime to %dms: %s", downtime, reason)
		} else {
			downtime = m.options.MaxDowntimeMs
			logger.Infof("max downtime set to %dms: %s", downtime, reason)
		}

		if err := dom.MigrateSetMaxDowntime(downtime, 0); err != nil {
			sd.switchoverInitiated = false
			logger.Reason(err).Error("setting max downtime failed")
			return
		}

		// since stop-and-copy is not guaranteed to start immediately (or ever), a "switch-over" deadline is needed
		m.switchOverDeadline = elapsedSeconds + switchoverTimeout

	default:
		logger.Error("unknown convergence action")
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

func (m *migrationMonitor) handleStallDetection(dom cli.VirDomain, stats *libvirt.DomainJobInfo, elapsedNs int64, isIterationBoundary bool, logger *log.FilteredLogger) {

	// This stall detection mechanism implements VEP 248. In each iteration, pre-copy tries to transfer VM state data (i.e.
	// memory) from source to target. Multiple iterations are required because as the VM transfers data it is
	// actively dirtying new memory. For high-dirty rate VMs with a large writable working set, we would never
	// converge. Stall detection tracks how many bytes are left and if with in a progress timeout window we make
	// little to no progress we are stalled. Then the goal is to manually force trigger switch-over at a local minima
	// of remaining bytes. See VEP for more details.
	sd := m.stallDetector

	if !sd.initialMaxDowntimeSet {
		initialMaxDowntime := m.options.MaxDowntimeMs
		if initialMaxDowntime > migrationutils.QEMUDefaultTargetDowntimeMS {
			initialMaxDowntime = migrationutils.QEMUDefaultTargetDowntimeMS
		}
		if err := dom.MigrateSetMaxDowntime(initialMaxDowntime, 0); err != nil {
			logger.Reason(err).Warning("failed to set initial max downtime")
		} else {
			sd.initialMaxDowntimeSet = true
		}
	}

	m.reconcilePauseState(dom, logger)

	if !m.isAbortInProgress() {
		if stats != nil && stats.Type == libvirt.DOMAIN_JOB_UNBOUNDED &&
			stats.DataRemainingSet && stats.TimeElapsedSet && stats.MemIterationSet {
			// the value in m.iterationRecord is accurate only when (1) we are the start an iteration or (2) if the
			//  VM is paused or (3) if the VM is in post-copy.
			if isIterationBoundary {
				logger.V(4).Info("processing migration iteration boundary for stall detection")
				m.iterationRecord.remainingBytes = stats.DataRemaining
				m.iterationRecord.elapsedMs = stats.TimeElapsed
				m.iterationRecord.iterationNumber = stats.MemIteration
				if stalled := sd.processStallDetectionIteration(m.iterationRecord, logger); stalled {
					estimatedDowntimeMs := sd.estimateDowntimeMs(m.iterationRecord, logger)
					action, reason := sd.decideAction(m.iterationRecord, estimatedDowntimeMs, m.start, m.acceptableCompletionTime, logger)
					m.triggerConvergenceAction(dom, action, reason, logger)
				}
			} else if m.isPausedMigration() || m.isMigrationPostCopy() {
				m.iterationRecord.remainingBytes = stats.DataRemaining
				m.iterationRecord.elapsedMs = stats.TimeElapsed
				m.iterationRecord.iterationNumber = stats.MemIteration
			} else if stats.MemBpsSet {
				sd.updateBandwidthEstimate(stats.MemBps, logger)
			}
		} else if stats == nil {
			logger.V(3).Info("skipping actions for stall detection due to missing job stats")
		} else {
			logger.V(3).Infof("skipping actions for stall detection due to missing stats data: DataRemainingSet=%t, TimeElapsedSet=%t, MemIterationSet=%t, MemBpsSet=%t", stats.DataRemainingSet, stats.TimeElapsedSet, stats.MemIterationSet, stats.MemBpsSet)
		}

		estimatedDowntimeMs := sd.estimateDowntimeMs(m.iterationRecord, logger)
		m.processCompletionTimeouts(dom, elapsedNs, estimatedDowntimeMs, logger)
	}
}

func (m *migrationMonitor) handleLegacyConvergence(dom cli.VirDomain, elapsedNs int64, logger *log.FilteredLogger) {
	now := m.start + elapsedNs

	switch {
	case m.isMigrationPostCopy():
		// Currently, there is nothing for us to track when in Post Copy mode.
		// The reasoning here is that post copy migrations transfer the state
		// directly to the target pod in a way that results in the target pod
		// hosting the active workload while the migration completes.

		// If we were to abort the migration due to a timeout while in post copy,
		// then it would result in that active state being lost.

	case m.shouldAssistMigrationToComplete(elapsedNs, logger) && !m.isPausedMigration():
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
			err := dom.MigrateSetMaxDowntime(min(uint64(maxDowntimeSec)*1000, uint64(migrationutils.QEMUMaxMigrationDowntimeMS)), 0)
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
		if !m.isAbortInProgress() {
			progressDelay := now - m.lastProgressUpdate
			logger.Warningf("Aborting migration: stuck for %d seconds", progressDelay/int64(time.Second))
			m.l.cancelMigration(m.vmi)
		}

	case m.shouldTriggerTimeout(elapsedNs, logger):
		// check the overall migration time
		// if the total migration time exceeds an acceptable
		// limit, then the migration will get aborted, but
		// only if post copy migration hasn't been enabled
		if !m.isAbortInProgress() {
			logger.Warningf("Aborting migration: not completed after %d seconds", m.acceptableCompletionTime)
			m.l.cancelMigration(m.vmi)
		}
	}
}

func (m *migrationMonitor) processInflightMigration(dom cli.VirDomain, stats *libvirt.DomainJobInfo, isIterationBoundary bool, logger *log.FilteredLogger) {
	// Migration is running
	now := time.Now().UTC().UnixNano()
	elapsedNs := now - m.start

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

	if m.stallDetectionEnabled {
		m.handleStallDetection(dom, stats, elapsedNs, isIterationBoundary, logger)
	} else {
		// TODO: to be removed once stall detection graduates
		m.tuneDowntime(dom, stats, logger)
		m.handleLegacyConvergence(dom, elapsedNs, logger)
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

func (m *migrationMonitor) startMonitor(ready chan<- error) {
	vmi := m.vmi

	m.start = time.Now().UTC().UnixNano()
	m.lastProgressUpdate = m.start

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

	if m.stallDetectionEnabled {
		registrationID, registerErr := m.registerIterationCallback(domName)
		if registerErr != nil {
			m.logger.Reason(registerErr).Error("failed to register migration iteration callback, falling back to legacy stall handling")
			m.stallDetectionEnabled = false
		} else {
			m.logger.V(3).Infof("registered migration iteration callback")
			defer func() {
				if err := m.l.virConn.DomainEventDeregister(registrationID); err != nil {
					m.logger.Reason(err).V(3).Info("failed to deregister migration iteration callback")
				}
			}()
		}
	}

	logInterval := 0
	iterationNumber := 0

	for {
		isIterationBoundary := false
		select {
		case <-m.migrationDone:
			return
		case iterationNumber = <-m.iterationCh:
			isIterationBoundary = true
		case <-time.After(monitorSleepPeriodMS * time.Millisecond):
		}

		var loopLogger *log.FilteredLogger
		if m.stallDetectionEnabled {
			loopSource := "poll"
			if isIterationBoundary {
				loopSource = "event"
			}
			loopLogger = log.Log.Object(vmi).With("source", loopSource).With("iteration", iterationNumber)
		} else {
			loopLogger = log.Log.Object(vmi)
		}

		jobStats, err := dom.GetJobStats(0)
		if err != nil {
			loopLogger.Reason(err).Info("failed to get domain job info")
			jobStats = nil
		}

		m.processInflightMigration(dom, jobStats, isIterationBoundary, loopLogger)

		if jobStats != nil && jobStats.Type == libvirt.DOMAIN_JOB_UNBOUNDED {
			if !isIterationBoundary {
				logInterval++
			}
			if logInterval%monitorLogInterval == 0 || isIterationBoundary {
				LogMigrationInfo(loopLogger, MigrationUID(vmi), jobStats)
			}
		}
	}
}
