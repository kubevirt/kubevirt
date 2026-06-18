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
	"maps"
	"slices"

	v1 "kubevirt.io/api/core/v1"
)

type State string

const (
	Alpha        State = "Alpha"
	Beta         State = "Beta"
	GA           State = "General Availability"
	Deprecated   State = "Deprecated"
	Discontinued State = "Discontinued"

	WarningPattern = "feature gate %s is deprecated (feature state is %q), therefore it can be safely removed and is redundant. " +
		"For more info, please look at: https://github.com/kubevirt/kubevirt/blob/main/docs/deprecation.md"
)

// ConfigReader provides read access to cluster-level feature gate configuration.
type ConfigReader interface {
	GetDeveloperConfiguration() *v1.DeveloperConfiguration
}

type FeatureGate struct {
	Name        string
	State       State
	VmiSpecUsed func(spec *v1.VirtualMachineInstanceSpec) bool
	Message     string
}

var featureGates = map[string]FeatureGate{}

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

func GetRegisteredFeatureGates() map[string]FeatureGate {
	return maps.Clone(featureGates)
}

func IsEnabled(gate string, devConfig *v1.DeveloperConfiguration) bool {
	fg := FeatureGateInfo(gate)
	if fg == nil {
		return false
	}

	if fg.State == GA {
		return true
	}

	if devConfig != nil {
		if slices.Contains(devConfig.FeatureGates, gate) {
			return true
		}

		if slices.Contains(devConfig.DisabledFeatureGates, gate) {
			return false
		}
	}

	return fg.State == Beta
}

func GateEnabled(gate string, reader ConfigReader) bool {
	return IsEnabled(gate, reader.GetDeveloperConfiguration())
}
