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
	"math"
	"time"

	"kubevirt.io/client-go/log"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

type convergenceAction int

const (
	actionNothing convergenceAction = iota
	actionAbort
	actionPostCopy
	actionHardStopAndCopy
	actionSoftStopAndCopy
)

type iterationRecord struct {
	elapsedMs       uint64
	remainingBytes  uint64
	iterationNumber uint64
}

type stallDetector struct {
	stallDetectorOptions cmdclient.StallDetectorOptions
	maxDowntimeMs        uint64

	// a bool indicating whether initial max downtime has been set (only set when maxDowntimeMs < 300, the default QEMU target downtime)
	initialMaxDowntimeSet bool
	// iteration records with the potential to end up in minRecordOutsideWindow
	minCandidates []iterationRecord
	// smallest iteration record outside the progressTimeout window
	minRecordOutsideWindow iterationRecord
	lastIterationElapsedMs uint64
	localMinimumWindow     slidingWindowMinimum
	// whether migration is currently stalled
	stallDetected bool
	// Current bandwidth smoothed using an exponential weighted moving average
	ewmaBandwidthBps float64
	// Whether we already initiated switchover to post-copy or stop-and-copy
	switchoverInitiated bool
	// migration policy flags (set once at init, immutable during migration)
	allowPostCopy           bool
	allowWorkloadDisruption bool
	hasVFIO                 bool
}

func newStallDetector(options *cmdclient.MigrationOptions, allowPostCopy, hasVFIO bool) *stallDetector {
	detectorOptions := *options.StallDetectorOptions
	return &stallDetector{
		stallDetectorOptions:    detectorOptions,
		maxDowntimeMs:           options.MaxDowntimeMs,
		localMinimumWindow:      slidingWindowMinimum{durationSeconds: uint64(detectorOptions.StallProgressTimeout)},
		allowPostCopy:           allowPostCopy,
		allowWorkloadDisruption: options.AllowWorkloadDisruption,
		hasVFIO:                 hasVFIO,
	}
}

func (sd *stallDetector) updateBandwidthEstimate(bandwidthSample uint64, logger *log.FilteredLogger) {
	prev := sd.ewmaBandwidthBps
	if sd.ewmaBandwidthBps == 0 {
		sd.ewmaBandwidthBps = float64(bandwidthSample)
		logger.V(4).Infof("initialized migration bandwidth EWMA: sampleBps=%dbps ewmaBps=%.2fbps", bandwidthSample, sd.ewmaBandwidthBps)
		return
	}
	bandwidthEWMAAlpha := sd.stallDetectorOptions.EwmaAlpha.AsApproximateFloat64()
	sd.ewmaBandwidthBps = bandwidthEWMAAlpha*float64(bandwidthSample) + (1-bandwidthEWMAAlpha)*sd.ewmaBandwidthBps
	logger.V(4).Infof("updated migration bandwidth EWMA: sampleBps=%dbps previousEwmaBps=%.2fbps newEwmaBps=%.2fbps", bandwidthSample, prev, sd.ewmaBandwidthBps)
}

func bytesToMiB(bytes uint64) float32 {
	return float32(bytes) / float32(1024) / float32(1024)
}

func (sd *stallDetector) updateMinRecordOutsideWindowCandidates(record iterationRecord, logger *log.FilteredLogger) {
	progressTimeoutMs := uint64(sd.stallDetectorOptions.StallProgressTimeout) * 1000
	agedOut := 0
	for len(sd.minCandidates) > 0 {
		oldestCandidate := sd.minCandidates[0]
		// record.elapsedMs > oldestCandidate.elapsedMs because record.elapsedMs is monotonically increasing
		ageMs := record.elapsedMs - oldestCandidate.elapsedMs
		if ageMs < progressTimeoutMs {
			break
		}

		sd.minCandidates = sd.minCandidates[1:]
		if sd.minRecordOutsideWindow == (iterationRecord{}) || oldestCandidate.remainingBytes < sd.minRecordOutsideWindow.remainingBytes {
			sd.minRecordOutsideWindow = oldestCandidate
		}
		agedOut++
	}
	if agedOut > 0 {
		outsideWindowMin := uint64(0)
		if sd.minRecordOutsideWindow != (iterationRecord{}) {
			outsideWindowMin = sd.minRecordOutsideWindow.remainingBytes
		}
		logger.V(4).Infof("aged out candidates: count=%d iterElapsedMs=%dms outsideWindowMinRemainingBytes=%.2fMib remainingCandidates=%d", agedOut, record.elapsedMs, bytesToMiB(outsideWindowMin), len(sd.minCandidates))
	}

	// optimization: candidates larger than the current out-of-window min can never become relevant.
	if sd.minRecordOutsideWindow != (iterationRecord{}) && record.remainingBytes > sd.minRecordOutsideWindow.remainingBytes {
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
	if sd.minRecordOutsideWindow == (iterationRecord{}) {
		logger.V(4).Infof("stall check skipped: no outside-window minimum yet, remainingBytes=%.2fMib", bytesToMiB(remainingBytes))
		return false
	}

	stallMargin := float64(sd.stallDetectorOptions.StallMargin) / 100
	stallThreshold := uint64(float64(sd.minRecordOutsideWindow.remainingBytes) * (1 - stallMargin))
	stalled := remainingBytes >= stallThreshold
	logger.V(4).Infof("stall check result: remainingBytes=%.2fMiB outsideWindowMinRemainingBytes=%.2fMiB threshold=%.2fMiB stalled=%t", bytesToMiB(remainingBytes), bytesToMiB(sd.minRecordOutsideWindow.remainingBytes), bytesToMiB(stallThreshold), stalled)
	return stalled
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

func (sd *stallDetector) isNearLocalMinimum(record iterationRecord, logger *log.FilteredLogger) bool {
	minimum, exists := sd.localMinimumWindow.minimum()
	if !exists {
		logger.Errorf("migration stall detector is in an illegal state")
		return false
	}
	windowMin := minimum.value
	threshold := float64(windowMin) * float64(100+sd.stallDetectorOptions.StallMargin) / 100
	nearMinimum := float64(record.remainingBytes) <= threshold
	logger.V(4).Infof("switchover margin check: remainingBytes=%.2fMiB windowMin=%.2fMiB threshold=%.2fMiB nearMinimum=%t", bytesToMiB(record.remainingBytes), bytesToMiB(windowMin), threshold/(1024*1024), nearMinimum)
	return nearMinimum
}

func (sd *stallDetector) processStallDetectionIteration(record iterationRecord, logger *log.FilteredLogger) bool {
	if sd.ewmaBandwidthBps == 0 {
		logger.V(4).Info("skipping stall-detection iteration because bandwidth data is unavailable")
		return false
	}
	if sd.switchoverInitiated {
		logger.V(4).Info("skipping stall-detection iteration because switchover action was already triggered.")
		return false
	}

	// Libvirt job elapsed time can regress if the host wall clock moves backward.
	if record.elapsedMs < sd.lastIterationElapsedMs {
		logger.V(3).Infof("ignoring regressed job elapsed time for stall detection: previousElapsedMs=%d elapsedMs=%d", sd.lastIterationElapsedMs, record.elapsedMs)
		return false
	}

	logger.V(4).Infof("processing stall-detection iteration: iterElapsedMs=%dms remainingBytes=%.2fMib currentEwmaBps=%.2fbps", record.elapsedMs, bytesToMiB(record.remainingBytes), sd.ewmaBandwidthBps)
	sd.updateMinRecordOutsideWindowCandidates(record, logger)
	if sd.stallDetectorOptions.SearchLocalMinima {
		sd.localMinimumWindow.add(record.elapsedMs, record.remainingBytes)
	}
	sd.lastIterationElapsedMs = record.elapsedMs

	if !sd.stallDetected && sd.checkStallCondition(record.remainingBytes, logger) {
		sd.stallDetected = true
		logger.V(3).Infof("stall detected: remainingBytes=%.2fMiB outsideWindowMin=%.2fMiB", bytesToMiB(record.remainingBytes), bytesToMiB(sd.minRecordOutsideWindow.remainingBytes))
	}
	return sd.stallDetected
}

func (sd *stallDetector) decideAction(record iterationRecord, estimatedDowntimeMs uint32, startTimeNs int64, acceptableCompletionTime int64, logger *log.FilteredLogger) (convergenceAction, string) {
	if sd.switchoverInitiated {
		return actionNothing, "switchover already initiated"
	}

	searchLocalMinima := sd.stallDetectorOptions.SearchLocalMinima
	if searchLocalMinima && !sd.isNearLocalMinimum(record, logger) {
		return actionNothing, "not at a local minima yet"
	}

	var localMinLogMessage string
	localMinimum, exists := sd.localMinimumWindow.minimum()
	if !exists || !searchLocalMinima {
		localMinLogMessage = "local minima search skipped: "
	} else {
		localMinLogMessage = fmt.Sprintf("arrived at a local minima: localMinimumBytes=%.2fMib ", bytesToMiB(localMinimum.value))
	}

	logger.V(4).Infof(localMinLogMessage+"remainingBytes=%.2fMib iterElapsedMs=%dms impliedDowntimeMs=%dms maxDowntimeMs=%dms allowPostCopy=%t allowWorkloadDisruption=%t",
		bytesToMiB(record.remainingBytes),
		record.elapsedMs,
		estimatedDowntimeMs,
		sd.maxDowntimeMs,
		sd.allowPostCopy,
		sd.allowWorkloadDisruption,
	)

	now := time.Now().UTC().UnixNano()
	elapsedSeconds := (now - startTimeNs) / int64(time.Second)

	// usually this case can only be triggered by a sudden network drop unless acceptableCompletionTime is very small
	completionTimeoutFactor := sd.stallDetectorOptions.CompletionTimeoutFactor.AsApproximateFloat64()
	deadlineSeconds := int64(float64(acceptableCompletionTime) * completionTimeoutFactor)
	if !sd.canFinishByDeadline(elapsedSeconds, deadlineSeconds, estimatedDowntimeMs, logger) {
		remainingBudgetMs := (deadlineSeconds - elapsedSeconds) * 1000
		return actionNothing, fmt.Sprintf("estimated transfer time (%dms) exceeds remaining budget (%dms) before completion deadline (%ds)", estimatedDowntimeMs, remainingBudgetMs, deadlineSeconds)
	}

	if sd.allowWorkloadDisruption && sd.allowPostCopy && !sd.hasVFIO {
		return actionPostCopy, fmt.Sprintf("estimated transfer time %dms is a local minima", estimatedDowntimeMs)
	}

	if sd.allowWorkloadDisruption {
		return actionHardStopAndCopy, fmt.Sprintf("estimated downtime %dms is a local minima", estimatedDowntimeMs)
	}

	preCopyPossibleFactor := sd.stallDetectorOptions.PrecopyPossibleFactor.AsApproximateFloat64()
	maxDowntimeMs := sd.maxDowntimeMs
	if float64(estimatedDowntimeMs) <= float64(maxDowntimeMs) {
		return actionSoftStopAndCopy, fmt.Sprintf("estimated downtime %dms within max allowed downtime %dms", estimatedDowntimeMs, maxDowntimeMs)
	} else if float64(estimatedDowntimeMs) <= float64(maxDowntimeMs)*preCopyPossibleFactor {
		return actionSoftStopAndCopy, fmt.Sprintf("estimated downtime %dms within tolerable factor %gx to max allowed downtime %dms", estimatedDowntimeMs, preCopyPossibleFactor, maxDowntimeMs)
	}

	return actionAbort, fmt.Sprintf("estimated downtime %dms exceeds max allowed downtime %dms by a factor of more than %gx", estimatedDowntimeMs, maxDowntimeMs, preCopyPossibleFactor)
}
