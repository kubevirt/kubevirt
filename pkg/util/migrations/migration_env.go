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

package migrations

import (
	"k8s.io/apimachinery/pkg/api/resource"

	envutil "kubevirt.io/kubevirt/pkg/util/env"
)

const (
	EnvAllowPostCopy             = "KUBEVIRT_MIGRATION_ALLOW_POST_COPY"
	EnvAllowWorkloadDisruption   = "KUBEVIRT_MIGRATION_ALLOW_WORKLOAD_DISRUPTION"
	EnvMaxDowntimeMs             = "KUBEVIRT_MIGRATION_MAX_DOWNTIME_MS"
	EnvBandwidthPerMigration     = "KUBEVIRT_MIGRATION_BANDWIDTH"
	EnvCompletionTimeoutPerGiB   = "KUBEVIRT_MIGRATION_COMPLETION_TIMEOUT_PER_GIB"
	EnvParallelMigrationThreads  = "KUBEVIRT_MIGRATION_PARALLEL_THREADS"
	EnvStallMargin               = "KUBEVIRT_MIGRATION_STALL_MARGIN"
	EnvStallProgressTimeout      = "KUBEVIRT_MIGRATION_STALL_PROGRESS_TIMEOUT"
	EnvSwitchoverTimeout         = "KUBEVIRT_MIGRATION_SWITCHOVER_TIMEOUT"
	EnvEwmaAlpha                 = "KUBEVIRT_MIGRATION_EWMA_ALPHA"
	EnvPrecopyPossibleFactor     = "KUBEVIRT_MIGRATION_PRECOPY_POSSIBLE_FACTOR"
	EnvPatienceWindowDecayFactor = "KUBEVIRT_MIGRATION_PATIENCE_WINDOW_DECAY_FACTOR"
	EnvSearchLocalMinima         = "KUBEVIRT_MIGRATION_SEARCH_LOCAL_MINIMA"
	EnvCompletionTimeoutFactor   = "KUBEVIRT_MIGRATION_COMPLETION_TIMEOUT_FACTOR"
)

// OptionsOverrides holds migration options read from the environment.
type OptionsOverrides struct {
	AllowPostCopy                      *bool
	AllowWorkloadDisruption            *bool
	MaxDowntimeMs                      *uint64
	Bandwidth                          *resource.Quantity
	CompletionTimeoutPerGiB            *int64
	ParallelMigrationThreads           *uint
	ParallelMigrationThreadsConfigured bool
}

// LookupOptionsOverrides returns explicitly set migration option env vars.
func LookupOptionsOverrides() OptionsOverrides {
	var overrides OptionsOverrides

	if v, ok := envutil.Bool(EnvAllowPostCopy); ok {
		overrides.AllowPostCopy = &v
	}
	if v, ok := envutil.Bool(EnvAllowWorkloadDisruption); ok {
		overrides.AllowWorkloadDisruption = &v
	}
	if v, ok := envutil.Uint64(EnvMaxDowntimeMs); ok {
		overrides.MaxDowntimeMs = &v
	}
	if v, ok := envutil.Lookup(EnvBandwidthPerMigration); ok {
		if bandwidth, ok := lookupBandwidthOverride(v); ok {
			overrides.Bandwidth = bandwidth
		}
	}
	if v, ok := envutil.Int64(EnvCompletionTimeoutPerGiB); ok {
		overrides.CompletionTimeoutPerGiB = &v
	}
	if v, ok := envutil.Uint64(EnvParallelMigrationThreads); ok {
		overrides.ParallelMigrationThreadsConfigured = true
		if v == 0 {
			overrides.ParallelMigrationThreads = nil
		} else {
			u := uint(v)
			overrides.ParallelMigrationThreads = &u
		}
	}

	return overrides
}

func lookupBandwidthOverride(value string) (*resource.Quantity, bool) {
	bandwidth, err := resource.ParseQuantity(value)
	if err != nil {
		return nil, false
	}
	if bandwidth.Sign() < 0 {
		return nil, false
	}
	return &bandwidth, true
}

// StallDetectorOptions holds resolved stall-detector tunables for virt-launcher.
type StallDetectorOptions struct {
	StallMargin               float64
	StallProgressTimeout      uint64
	SwitchoverTimeout         uint64
	EwmaAlpha                 float64
	PrecopyPossibleFactor     float64
	PatienceWindowDecayFactor float64
	SearchLocalMinima         bool
	CompletionTimeoutFactor   float64
}

// ApplyEnvOverrides overlays explicitly set stall-detector environment variables on top of base.
func ApplyEnvOverrides(base StallDetectorOptions) StallDetectorOptions {
	if v, ok := envutil.Float(EnvStallMargin); ok {
		base.StallMargin = v
	}
	if v, ok := envutil.Uint64(EnvStallProgressTimeout); ok {
		base.StallProgressTimeout = v
	}
	if v, ok := envutil.Uint64(EnvSwitchoverTimeout); ok {
		base.SwitchoverTimeout = v
	}
	if v, ok := envutil.Float(EnvEwmaAlpha); ok {
		base.EwmaAlpha = v
	}
	if v, ok := envutil.Float(EnvPrecopyPossibleFactor); ok {
		base.PrecopyPossibleFactor = v
	}
	if v, ok := envutil.Float(EnvPatienceWindowDecayFactor); ok {
		base.PatienceWindowDecayFactor = v
	}
	if v, ok := envutil.Bool(EnvSearchLocalMinima); ok {
		base.SearchLocalMinima = v
	}
	if v, ok := envutil.Float(EnvCompletionTimeoutFactor); ok {
		base.CompletionTimeoutFactor = v
	}
	return base
}
