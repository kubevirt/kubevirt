//go:build linux

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

import (
	"fmt"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/hypervisor/common"
)

// Collapse collapses THPeligible guest-RAM VMAs below GuaranteedCoverageThreshold.
// Non-backed regions are collapsed in passes (up to collapseAttempts); each pass
// retries only regions that still failed. Result.Coverage is the lowest
// per-region coverage after collapse attempts (or the initial reading if nothing
// needed collapsing).
func Collapse(pid int) (CollapseResult, error) {
	smaps, err := readProcessSmaps(pid)
	if err != nil {
		return CollapseResult{}, fmt.Errorf("read process smaps: %w", err)
	}

	regions, err := ParseTHPEligibleRegions(smaps)
	if err != nil {
		return CollapseResult{}, fmt.Errorf("parse process smaps: %w", err)
	}
	if len(regions) == 0 {
		// Irrecoverable: no guest-RAM VMA is THPeligible (e.g. misaligned memfd).
		return CollapseResult{}, &CollapseFailedError{Coverage: 0}
	}

	var result CollapseResult
	result.Coverage = 1
	var pending []Region
	for _, region := range regions {
		if cov := region.Coverage(); cov < result.Coverage {
			result.Coverage = cov
		}
		if region.FullyBacked() {
			log.Log.V(3).Infof("THP region pid=%d 0x%x+%d coverage=%.1f%% skip collapse",
				pid, region.Start, region.Size, region.Coverage()*100)
			continue
		}
		result.BytesRequested += region.Size
		pending = append(pending, region)
	}

	if len(pending) == 0 {
		return result, nil
	}

	for attempt := 1; attempt <= collapseAttempts && len(pending) > 0; attempt++ {
		var remaining []Region
		for _, region := range pending {
			log.Log.V(3).Infof("THP collapsing pid=%d region 0x%x+%d coverage=%.1f%% (pass %d/%d)",
				pid, region.Start, region.Size, region.Coverage()*100, attempt, collapseAttempts)
			n, err := common.ProcessMadviseCollapse(pid, region.Start, region.Size)
			if err != nil || uint64(n) < region.Size {
				if err != nil {
					log.Log.Warningf("MADV_COLLAPSE failed for pid %d region 0x%x+%d (pass %d/%d): %v",
						pid, region.Start, region.Size, attempt, collapseAttempts, err)
				} else {
					log.Log.Warningf("MADV_COLLAPSE partial for pid %d region 0x%x+%d (pass %d/%d): %d/%d bytes",
						pid, region.Start, region.Size, attempt, collapseAttempts, n, region.Size)
				}
				remaining = append(remaining, region)
				continue
			}
			result.RegionsCollapsed++
		}
		pending = remaining
	}

	// Always re-measure: partial/soft failures may still have changed coverage.
	smaps, err = readProcessSmaps(pid)
	if err != nil {
		return result, fmt.Errorf("re-read process smaps: %w", err)
	}
	regions, err = ParseTHPEligibleRegions(smaps)
	if err != nil {
		return result, fmt.Errorf("re-parse process smaps: %w", err)
	}
	if len(regions) == 0 {
		return result, &CollapseFailedError{Coverage: 0}
	}
	result.Coverage = 1
	for _, region := range regions {
		if cov := region.Coverage(); cov < result.Coverage {
			result.Coverage = cov
		}
		log.Log.V(3).Infof("THP region pid=%d 0x%x+%d coverage after collapse=%.1f%%",
			pid, region.Start, region.Size, region.Coverage()*100)
	}
	return result, nil
}
