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

package thp

import "fmt"

const (
	// GuaranteedCoverageThreshold is the fixed VEP 383 coverage gate for policy=guaranteed.
	GuaranteedCoverageThreshold = 0.95
	// EventReasonCollapseFailed is the Kubernetes event reason for guaranteed-policy failure.
	EventReasonCollapseFailed = "THPCollapseFailed"
	// collapseAttempts is how many passes over remaining non-collapsed regions.
	collapseAttempts = 3
)

// CollapseResult summarizes one collapse attempt and smaps-based coverage.
type CollapseResult struct {
	RegionsCollapsed int
	BytesRequested   uint64
	Coverage         float64
}

// CollapseFailedError marks an irrecoverable guaranteed-policy THP coverage failure.
type CollapseFailedError struct {
	Coverage float64
}

func (e *CollapseFailedError) Error() string {
	return fmt.Sprintf("transparent hugepage coverage %.1f%% is below the required %.0f%%",
		e.Coverage*100, GuaranteedCoverageThreshold*100)
}
