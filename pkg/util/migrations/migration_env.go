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

var (
	DisableMultifd            = envutil.Var[bool]{Name: EnvDisableMultifd}
	StallMargin               = envutil.Var[int64]{Name: EnvStallMargin}
	StallProgressTimeout      = envutil.Var[int64]{Name: EnvStallProgressTimeout}
	SwitchoverTimeout         = envutil.Var[int64]{Name: EnvSwitchoverTimeout}
	EwmaAlpha                 = envutil.Var[resource.Quantity]{Name: EnvEwmaAlpha}
	PrecopyPossibleFactor     = envutil.Var[resource.Quantity]{Name: EnvPrecopyPossibleFactor}
	PatienceWindowDecayFactor = envutil.Var[resource.Quantity]{Name: EnvPatienceWindowDecayFactor}
	SearchLocalMinima         = envutil.Var[bool]{Name: EnvSearchLocalMinima}
	CompletionTimeoutFactor   = envutil.Var[resource.Quantity]{Name: EnvCompletionTimeoutFactor}
)

func override[T any](v envutil.Var[T], dest *T) {
	parsed, err := v.LoadAndParse()
	if err != nil {
		return
	}
	*dest = parsed
}

// ShouldDisableMultifd reports whether KUBEVIRT_MIGRATION_DISABLE_MULTIFD is set to true.
func ShouldDisableMultifd() bool {
	disable, err := DisableMultifd.LoadAndParse()
	return err == nil && disable
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
	override(StallMargin, &base.StallMargin)
	override(StallProgressTimeout, &base.StallProgressTimeout)
	override(SwitchoverTimeout, &base.SwitchoverTimeout)
	override(EwmaAlpha, &base.EwmaAlpha)
	override(PrecopyPossibleFactor, &base.PrecopyPossibleFactor)
	override(PatienceWindowDecayFactor, &base.PatienceWindowDecayFactor)
	override(SearchLocalMinima, &base.SearchLocalMinima)
	override(CompletionTimeoutFactor, &base.CompletionTimeoutFactor)
	return base
}
