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
)

const (
	stallMargin           float64 = 0.04
	switchoverTimeout     int64   = 60
	preCopyPossibleFactor float64 = 1.5
	bandwidthEWMAAlpha    float64 = 0.4
	searchLocalMinima             = true
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
	elapsedMs      uint64
	remainingBytes uint64
}

type stallDetector struct {
	// how long in seconds does a migration have to make progress before we take some action
	progressTimeoutSeconds int64
	// the maximum downtime in ms where a stun time up to this long is not considered a "disruption"
	maxDowntimeMs uint64
	// iteration records with the potential to end up in minRecordOutsideWindow
	minCandidates []iterationRecord
	// smallest iteration record outside the progressTimeout window
	minRecordOutsideWindow iterationRecord
	// whether migration is currently stalled
	stallDetected bool
	// best value of "remaining bytes" observed so far
	bestRemainingBytes uint64
	// Current bandwidth smoothed using an exponential weighted moving average
	ewmaBandwidthBps float64
	// Whether we already initiated switchover to post-copy or stop-and-copy
	switchoverInitiated bool
	// migration policy flags (set once at init, immutable during migration)
	allowPostCopy           bool
	allowWorkloadDisruption bool
	hasVFIO                 bool
}

func (sd *stallDetector) updateBandwidthEstimate(bandwidthSample uint64) {
	if sd.ewmaBandwidthBps == 0 {
		sd.ewmaBandwidthBps = float64(bandwidthSample)
		return
	}
	sd.ewmaBandwidthBps = bandwidthEWMAAlpha*float64(bandwidthSample) + (1-bandwidthEWMAAlpha)*sd.ewmaBandwidthBps
}

func (sd *stallDetector) updateCandidates(record iterationRecord) {
	progressTimeoutMs := uint64(sd.progressTimeoutSeconds) * 1000
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
	}

	// optimization: candidates larger than the current out-of-window min can never become relevant.
	if sd.minRecordOutsideWindow != (iterationRecord{}) && record.remainingBytes > sd.minRecordOutsideWindow.remainingBytes {
		return
	}

	// optimization: candidates preceded by a smaller value.
	if len(sd.minCandidates) > 0 && record.remainingBytes >= sd.minCandidates[len(sd.minCandidates)-1].remainingBytes {
		return
	}

	sd.minCandidates = append(sd.minCandidates, record)
}

func (sd *stallDetector) checkStallCondition(remainingBytes uint64) bool {
	if sd.minRecordOutsideWindow == (iterationRecord{}) {
		return false
	}

	stallThreshold := uint64(float64(sd.minRecordOutsideWindow.remainingBytes) * (1 - stallMargin))
	return remainingBytes >= stallThreshold
}

func (sd *stallDetector) findBestRemainingBytes() uint64 {
	bestRemainingBytes := sd.minRecordOutsideWindow.remainingBytes
	for _, candidate := range sd.minCandidates {
		if candidate.remainingBytes < bestRemainingBytes {
			bestRemainingBytes = candidate.remainingBytes
		}
	}
	return bestRemainingBytes
}

func (sd *stallDetector) estimateDowntimeMs(record iterationRecord) uint32 {
	if sd.ewmaBandwidthBps == 0 {
		return 0
	}
	bandwidthBpms := sd.ewmaBandwidthBps / 1000
	estimatedDowntime := float64(record.remainingBytes) / bandwidthBpms
	if estimatedDowntime > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(estimatedDowntime)
}

func (sd *stallDetector) processStallDetectionIteration(record iterationRecord) bool {
	if sd.ewmaBandwidthBps == 0 {
		return false
	}
	if sd.switchoverInitiated {
		return false
	}

	sd.updateCandidates(record)

	if sd.stallDetected {
		return true
	} else if sd.checkStallCondition(record.remainingBytes) {
		// when stall is first detected initialize stall-related state
		sd.bestRemainingBytes = sd.findBestRemainingBytes()
		sd.stallDetected = true
		return true
	} else {
		return false
	}
}

func (sd *stallDetector) decideAction(record iterationRecord, estimatedDowntimeMs uint32) (convergenceAction, string) {

	if sd.switchoverInitiated {
		return actionNothing, "switchover already initiated"
	}

	target := uint64(float64(sd.bestRemainingBytes) * (1 + stallMargin))
	atLocalMinima := record.remainingBytes <= target

	if !atLocalMinima && searchLocalMinima {
		return actionNothing, "not at a local minima yet"
	}

	if sd.allowWorkloadDisruption && sd.allowPostCopy && !sd.hasVFIO {
		return actionPostCopy, fmt.Sprintf("estimated transfer time %dms is a local minima", estimatedDowntimeMs)
	}

	if sd.allowWorkloadDisruption {
		return actionHardStopAndCopy, fmt.Sprintf("estimated downtime %dms is a local minima", estimatedDowntimeMs)
	}

	if float64(estimatedDowntimeMs) <= float64(sd.maxDowntimeMs) {
		return actionSoftStopAndCopy, fmt.Sprintf("estimated downtime %dms within max allowed downtime %dms", estimatedDowntimeMs, sd.maxDowntimeMs)
	} else if float64(estimatedDowntimeMs) <= float64(sd.maxDowntimeMs)*preCopyPossibleFactor {
		return actionSoftStopAndCopy, fmt.Sprintf("estimated downtime %dms within tolerable factor %fx to max allowed downtime %dms", estimatedDowntimeMs, preCopyPossibleFactor, sd.maxDowntimeMs)
	}

	return actionAbort, fmt.Sprintf("estimated downtime %dms far exceeds max allowed downtime %dms", estimatedDowntimeMs, sd.maxDowntimeMs)
}
