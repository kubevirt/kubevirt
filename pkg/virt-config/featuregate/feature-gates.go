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

package featuregate

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
)

type State string

const (
	// Alpha represents features that are under experimentation.
	// The feature is disabled by default and can be enabled explicitly through the FG.
	Alpha State = "Alpha"
	// Beta represents features that are under evaluation.
	// The feature is disabled by default and can be enabled explicitly through the FG.
	Beta State = "Beta"
	// GA represents features that reached General Availability.
	// GA features are considered feature-gate enabled, with no option to disable them by an FG.
	GA State = "General Availability"
	// Deprecated represents features that are going to be discontinued in the following release.
	// Warn users about the eminent removal of the feature & FG.
	// The feature is disabled by default and can be enabled explicitly through the FG.
	Deprecated State = "Deprecated"
	// Discontinued represents features that have been removed, with no option to enable them.
	Discontinued State = "Discontinued"

	WarningPattern = "feature gate %s is deprecated (feature state is %q), therefore it can be safely removed and is redundant. " +
		"For more info, please look at: https://github.com/kubevirt/kubevirt/blob/main/docs/deprecation.md"
)

type FeatureGate struct {
	Name        string
	State       State
	VmiSpecUsed func(spec *v1.VirtualMachineInstanceSpec) bool
	Message     string
}

var featureGates = map[string]FeatureGate{}

// RegisterFeatureGate adds a given feature-gate to the FG list
// In case the FG already exists (based on its name), it overrides the
// existing FG.
// If an inactive feature-gate is missing a message, a default one is set.
func RegisterFeatureGate(fg FeatureGate) {
	if fg.State != Alpha && fg.State != Beta && fg.Message == "" {
		fg.Message = fmt.Sprintf(WarningPattern, fg.Name, fg.State)
	}
	featureGates[fg.Name] = fg
}

func UnregisterFeatureGate(fgName string) {
	delete(featureGates, fgName)
}

func FeatureGateInfo(featureGate string) *FeatureGate {
	if fg, exist := featureGates[featureGate]; exist {
		return &fg
	}
	return nil
}
