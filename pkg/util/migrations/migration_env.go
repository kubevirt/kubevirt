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
	EnvDisableMultifd            = "KUBEVIRT_MIGRATION_DISABLE_MULTIFD"
	EnvStallMargin               = "KUBEVIRT_MIGRATION_STALL_MARGIN"
	EnvStallProgressTimeout      = "KUBEVIRT_MIGRATION_STALL_PROGRESS_TIMEOUT"
	EnvSwitchoverTimeout         = "KUBEVIRT_MIGRATION_SWITCHOVER_TIMEOUT"
	EnvEwmaAlpha                 = "KUBEVIRT_MIGRATION_EWMA_ALPHA"
	EnvPrecopyPossibleFactor     = "KUBEVIRT_MIGRATION_PRECOPY_POSSIBLE_FACTOR"
	EnvPatienceWindowDecayFactor = "KUBEVIRT_MIGRATION_PATIENCE_WINDOW_DECAY_FACTOR"
	EnvSearchLocalMinima         = "KUBEVIRT_MIGRATION_SEARCH_LOCAL_MINIMA"
	EnvCompletionTimeoutFactor   = "KUBEVIRT_MIGRATION_COMPLETION_TIMEOUT_FACTOR"
)

// ShouldDisableMultifd reports whether KUBEVIRT_MIGRATION_DISABLE_MULTIFD is set to true.
func ShouldDisableMultifd() bool {
	disable, ok := envutil.Bool(EnvDisableMultifd)
	return ok && disable
}

// StallDetectorOptions holds resolved stall-detector tunables for virt-launcher.
type StallDetectorOptions struct {
	StallMargin               int64
	StallProgressTimeout      int64
	SwitchoverTimeout         int64
	EwmaAlpha                 resource.Quantity
	PrecopyPossibleFactor     resource.Quantity
	PatienceWindowDecayFactor resource.Quantity
	SearchLocalMinima         bool
	CompletionTimeoutFactor   resource.Quantity
}

// ApplyEnvOverrides overlays explicitly set stall-detector environment variables on top of base.
func ApplyEnvOverrides(base StallDetectorOptions) StallDetectorOptions {
	if v, ok := envutil.Int64(EnvStallMargin); ok {
		base.StallMargin = v
	}
	if v, ok := envutil.Int64(EnvStallProgressTimeout); ok {
		base.StallProgressTimeout = v
	}
	if v, ok := envutil.Int64(EnvSwitchoverTimeout); ok {
		base.SwitchoverTimeout = v
	}
	if v, ok := envutil.Quantity(EnvEwmaAlpha); ok {
		base.EwmaAlpha = v
	}
	if v, ok := envutil.Quantity(EnvPrecopyPossibleFactor); ok {
		base.PrecopyPossibleFactor = v
	}
	if v, ok := envutil.Quantity(EnvPatienceWindowDecayFactor); ok {
		base.PatienceWindowDecayFactor = v
	}
	if v, ok := envutil.Bool(EnvSearchLocalMinima); ok {
		base.SearchLocalMinima = v
	}
	if v, ok := envutil.Quantity(EnvCompletionTimeoutFactor); ok {
		base.CompletionTimeoutFactor = v
	}
	return base
}
