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
	// HPATargetAverageTransfers is the HPA average active transfers per pod target.
	HPATargetAverageTransfers = 130

	// HPATargetMaxTransfers is the HPA gated per-pod max metric target (load-test
	// capacity per pod). The max metric drives scale-out when the hottest pod
	// exceeds this value by more than the cluster default HPA tolerance (10%),
	// i.e. when active transfers > HPATargetMaxTransfers * 1.1. Must stay below
	// SoftTransferLimit so the reported metric can clear tolerance before HTTP 429.
	HPATargetMaxTransfers int64 = 150

	// SoftTransferLimit rejects new export transfers with HTTP 429 when a pod
	// already has this many active transfers. Set above HPATargetMaxTransfers
	// (~13% headroom) so the hottest pod can report a value above the HPA max
	// target plus tolerance while still below the per-pod hard cap. Active
	// transfers are CAS-capped here, so this is also the maximum reachable active
	// count and the readiness shed threshold (HardTransferLimit).
	SoftTransferLimit int64 = 170

	// SoftCPUUtilizationPercent rejects new export transfers with HTTP 429 when
	// smoothed cgroup CPU utilization exceeds this percentage of the pod CPU limit.
	SoftCPUUtilizationPercent = 70

	// SoftMemoryUtilizationPercent rejects new export transfers with HTTP 429 when
	// cgroup memory usage exceeds this percentage of the pod memory limit.
	SoftMemoryUtilizationPercent = 70

	// HardTransferLimit removes the pod from Service endpoints via /readyz when
	// active transfers reach this count. Equal to SoftTransferLimit so readiness
	// shedding is reachable: soft admission CAS-caps the counter at SoftTransferLimit,
	// so a higher hard threshold could never fire. At capacity the pod both returns
	// 429 and leaves endpoints (429 still covers races / direct hits).
	HardTransferLimit = SoftTransferLimit

	// HardTransferClear is the hysteresis lower bound for readiness shedding:
	// a pod that failed readiness at HardTransferLimit becomes ready again at
	// this level (aligned with the HPA average target).
	HardTransferClear = HPATargetAverageTransfers

	// RetryAfterSeconds is the Retry-After header value on 429 responses.
	RetryAfterSeconds = 1

	// HPAMaxMetricAverageFloor suppresses the gated max HPA metric when fleet
	// average active transfers is below this value (70% of HPATargetAverageTransfers).
	HPAMaxMetricAverageFloor = 91
)
