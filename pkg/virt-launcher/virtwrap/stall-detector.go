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

	utilheap "kubevirt.io/kubevirt/pkg/util/heap"
	migrationutils "kubevirt.io/kubevirt/pkg/util/migrations"
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
	stallDetectorOptions migrationutils.StallDetectorOptions
	maxDowntimeMs        uint64

	// a bool indicating whether initial max downtime has been set (only set when maxDowntimeMs < 300, the default QEMU target downtime)
	initialMaxDowntimeSet bool
	// iteration records with the potential to end up in minRecordOutsideWindow
	minCandidates []iterationRecord
	// smallest iteration record outside the progressTimeout window
	minRecordOutsideWindow iterationRecord
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
	// migration policy flags (set once at init, immutable during migration)
	allowPostCopy           bool
	allowWorkloadDisruption bool
	hasVFIO                 bool
}

func (sd *stallDetector) updateBandwidthEstimate(bandwidthSample uint64, logger *log.FilteredLogger) {
	prev := sd.ewmaBandwidthBps
	if sd.ewmaBandwidthBps == 0 {
		sd.ewmaBandwidthBps = float64(bandwidthSample)
		logger.V(4).Infof("initialized migration bandwidth EWMA: sampleBps=%dbps ewmaBps=%.2fbps", bandwidthSample, sd.ewmaBandwidthBps)
		return
	}
	bandwidthEWMAAlpha := sd.stallDetectorOptions.EwmaAlpha
	sd.ewmaBandwidthBps = bandwidthEWMAAlpha*float64(bandwidthSample) + (1-bandwidthEWMAAlpha)*sd.ewmaBandwidthBps
	logger.V(4).Infof("updated migration bandwidth EWMA: sampleBps=%dbps previousEwmaBps=%.2fbps newEwmaBps=%.2fbps", bandwidthSample, prev, sd.ewmaBandwidthBps)
}

func bytesToMiB(bytes uint64) float32 {
	return float32(bytes) / float32(1024) / float32(1024)
}

func (sd *stallDetector) updateCandidates(record iterationRecord, logger *log.FilteredLogger) {
	progressTimeoutMs := sd.stallDetectorOptions.StallProgressTimeout * 1000
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

	stallMargin := sd.stallDetectorOptions.StallMargin
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
	sd.remainingBytesHistory = utilheap.NewMin[uint64]()
	sd.relaxationPatienceMs = sd.stallDetectorOptions.StallProgressTimeout * 1000
	sd.relaxationDeadlineMs = record.elapsedMs + sd.relaxationPatienceMs
	logger.V(4).Infof("initialized relaxation state: iterElapsedMs=%dms patienceMs=%dms deadlineMs=%dms", record.elapsedMs, sd.relaxationPatienceMs, sd.relaxationDeadlineMs)
}

func (sd *stallDetector) relaxBestRemainingBytes(record iterationRecord, logger *log.FilteredLogger) {
	sd.remainingBytesHistory.Push(record.remainingBytes)
	if record.elapsedMs < sd.relaxationDeadlineMs {
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
	sd.relaxationPatienceMs = uint64(float64(sd.relaxationPatienceMs) * sd.stallDetectorOptions.PatienceWindowDecayFactor)
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
	stallMargin := sd.stallDetectorOptions.StallMargin
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

func (sd *stallDetector) decideAction(record iterationRecord, estimatedDowntimeMs uint32, startTimeNs int64, acceptableCompletionTime int64, logger *log.FilteredLogger) (convergenceAction, string) {

	if sd.switchoverInitiated {
		return actionNothing, "switchover already initiated"
	}

	searchLocalMinima := sd.stallDetectorOptions.SearchLocalMinima
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
		record.elapsedMs,
		bytesToMiB(record.remainingBytes),
		bytesToMiB(sd.bestRemainingBytes),
		estimatedDowntimeMs,
		sd.maxDowntimeMs,
		sd.allowPostCopy,
		sd.allowWorkloadDisruption,
	)

	now := time.Now().UTC().UnixNano()
	elapsedSeconds := (now - startTimeNs) / int64(time.Second)

	// usually this case can only be triggered by a sudden network drop unless acceptableCompletionTime is very small
	completionTimeoutFactor := sd.stallDetectorOptions.CompletionTimeoutFactor
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

	preCopyPossibleFactor := sd.stallDetectorOptions.PrecopyPossibleFactor
	maxDowntimeMs := sd.maxDowntimeMs
	if float64(estimatedDowntimeMs) <= float64(maxDowntimeMs) {
		return actionSoftStopAndCopy, fmt.Sprintf("estimated downtime %dms within max allowed downtime %dms", estimatedDowntimeMs, maxDowntimeMs)
	} else if float64(estimatedDowntimeMs) <= float64(maxDowntimeMs)*preCopyPossibleFactor {
		return actionSoftStopAndCopy, fmt.Sprintf("estimated downtime %dms within tolerable factor %.2fx to max allowed downtime %dms", estimatedDowntimeMs, preCopyPossibleFactor, maxDowntimeMs)
	}

	return actionAbort, fmt.Sprintf("estimated downtime %dms exceeds max allowed downtime %dms by a factor of more than x%.2f", estimatedDowntimeMs, maxDowntimeMs, preCopyPossibleFactor)
}
