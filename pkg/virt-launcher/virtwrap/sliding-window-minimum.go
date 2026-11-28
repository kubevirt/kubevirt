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

type minimumSample struct {
	elapsedMs uint64
	value     uint64
}

// slidingWindowMinimum keeps a monotonic deque of samples in increasing value
// order, with the minimum at the front.
type slidingWindowMinimum struct {
	durationSeconds uint64
	candidates      []minimumSample
}

// add takes samples in nondecreasing elapsed time order. It expires previous
// samples at the window boundary, then adds the current sample.
func (window *slidingWindowMinimum) add(elapsedMs, value uint64) {
	candidates := window.candidates
	for len(candidates) > 0 && (elapsedMs-candidates[0].elapsedMs)/1000 >= window.durationSeconds {
		candidates = candidates[1:]
	}
	// The new sample outlasts older samples with equal or higher values.
	for len(candidates) > 0 && candidates[len(candidates)-1].value >= value {
		candidates = candidates[:len(candidates)-1]
	}
	window.candidates = append(candidates, minimumSample{elapsedMs: elapsedMs, value: value})
}

func (window *slidingWindowMinimum) minimum() (minimumSample, bool) {
	if len(window.candidates) == 0 {
		return minimumSample{}, false
	}
	return window.candidates[0], true
}
