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
 */

package admission

const (
	// SoftTransferLimit rejects new export transfers with HTTP 429 when a pod
	// already has this many active transfers (HPA average target + headroom).
	// Intentionally equal to HPATargetMaxTransfers so HPA scale-out and per-pod
	// 429 shedding start at the same per-pod load.
	SoftTransferLimit int64 = 150

	// SoftCPUUtilizationPercent rejects new export transfers with HTTP 429 when
	// smoothed cgroup CPU utilization exceeds this percentage of the pod CPU limit.
	SoftCPUUtilizationPercent = 70

	// SoftMemoryUtilizationPercent rejects new export transfers with HTTP 429 when
	// cgroup memory usage exceeds this percentage of the pod memory limit.
	SoftMemoryUtilizationPercent = 70

	// HardTransferLimit removes the pod from Service endpoints via /readyz when
	// active transfers reach this count. Backstop only: SoftTransferLimit (429
	// admission) should cap normal traffic well below this; the hard limit fires
	// if soft admission fails or is bypassed.
	HardTransferLimit int64 = 200

	// HardTransferClear is the hysteresis lower bound for the hard-limit backstop
	// above: a pod that failed readiness at HardTransferLimit becomes ready again
	// at this level.
	HardTransferClear int64 = 180

	// RetryAfterSeconds is the Retry-After header value on 429 responses.
	RetryAfterSeconds = 1

	// HPATargetAverageTransfers is the HPA average active transfers per pod target.
	HPATargetAverageTransfers = 130

	// HPATargetMaxTransfers is the HPA gated per-pod max metric target.
	// Intentionally equal to SoftTransferLimit (see above).
	HPATargetMaxTransfers = 150

	// HPAMaxMetricAverageFloor suppresses the gated max HPA metric when fleet
	// average active transfers is below this value (70% of HPATargetAverageTransfers).
	HPAMaxMetricAverageFloor = 91
)
