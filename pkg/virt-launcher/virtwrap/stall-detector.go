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

const (
	stallMargin           float64 = 0.04
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
	minRecordOutsideWindow *iterationRecord
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
	// TODO: use EWMA to update the bandwidth estimator
}

func (sd *stallDetector) checkStallCondition(remainingBytes uint64) bool {
	// TODO: check whether the stall condition is satisfied (i.e. are we stalled or not)
	return false
}

func (sd *stallDetector) findBestRemainingBytes() uint64 {
	// TODO: find the best remaining bytes we are observed so far
	return 0
}

func (sd *stallDetector) estimateDowntimeMs(record iterationRecord) uint32 {
	// TODO: estimate the downtime based on bandwidth and remaining bytes as supplied
	return 0
}

func (sd *stallDetector) processStallDetectionIteration(record iterationRecord) bool {
	// TODO: updates necessary state and returns whether we are currently stalled
	return false
}

func (sd *stallDetector) decideAction(record iterationRecord, estimatedDowntimeMs uint32) (convergenceAction, string) {
	// TODO: decides which convergence action to take based on the iteration record and estimated downtime
	return actionNothing, "no action decided"
}
