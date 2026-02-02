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
package apply

import (
	virtv1 "kubevirt.io/api/core/v1"
	instancetypev1 "kubevirt.io/api/instancetype/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

// deprecatedTopologyToNew maps deprecated topology values to their new equivalents
var deprecatedTopologyToNew = map[instancetypev1.PreferredCPUTopology]instancetypev1.PreferredCPUTopology{
	instancetypev1.PreferredCPUTopology(instancetypev1beta1.DeprecatedPreferSockets): instancetypev1.Sockets,
	instancetypev1.PreferredCPUTopology(instancetypev1beta1.DeprecatedPreferCores):   instancetypev1.Cores,
	instancetypev1.PreferredCPUTopology(instancetypev1beta1.DeprecatedPreferThreads): instancetypev1.Threads,
	instancetypev1.PreferredCPUTopology(instancetypev1beta1.DeprecatedPreferSpread):  instancetypev1.Spread,
	instancetypev1.PreferredCPUTopology(instancetypev1beta1.DeprecatedPreferAny):     instancetypev1.Any,
}

func applyCPUPreferences(preferenceSpec *instancetypev1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	if preferenceSpec.CPU == nil || len(preferenceSpec.CPU.PreferredCPUFeatures) == 0 {
		return
	}
	// Only apply any preferred CPU features when the same feature has not been provided by a user already
	cpuFeatureNames := make(map[string]struct{})
	for _, cpuFeature := range vmiSpec.Domain.CPU.Features {
		cpuFeatureNames[cpuFeature.Name] = struct{}{}
	}
	for _, preferredCPUFeature := range preferenceSpec.CPU.PreferredCPUFeatures {
		if _, foundCPUFeature := cpuFeatureNames[preferredCPUFeature.Name]; !foundCPUFeature {
			vmiSpec.Domain.CPU.Features = append(vmiSpec.Domain.CPU.Features, preferredCPUFeature)
		}
	}
}

func GetPreferredTopology(preferenceSpec *instancetypev1.VirtualMachinePreferenceSpec) instancetypev1.PreferredCPUTopology {
	// Default to Sockets when a PreferredCPUTopology isn't provided
	preferredTopology := instancetypev1.Sockets
	if preferenceSpec != nil && preferenceSpec.CPU != nil && preferenceSpec.CPU.PreferredCPUTopology != nil {
		preferredTopology = *preferenceSpec.CPU.PreferredCPUTopology
	}
	// Normalize deprecated values to new equivalents
	if newTopology, ok := deprecatedTopologyToNew[preferredTopology]; ok {
		return newTopology
	}
	return preferredTopology
}

const defaultSpreadRatio uint32 = 2

func GetSpreadOptions(preferenceSpec *instancetypev1.VirtualMachinePreferenceSpec) (uint32, instancetypev1.SpreadAcross) {
	ratio := defaultSpreadRatio
	if preferenceSpec.PreferSpreadSocketToCoreRatio != 0 {
		ratio = preferenceSpec.PreferSpreadSocketToCoreRatio
	}
	across := instancetypev1.SpreadAcrossSocketsCores
	if preferenceSpec.CPU != nil && preferenceSpec.CPU.SpreadOptions != nil {
		if preferenceSpec.CPU.SpreadOptions.Across != nil {
			across = *preferenceSpec.CPU.SpreadOptions.Across
		}
		if preferenceSpec.CPU.SpreadOptions.Ratio != nil {
			ratio = *preferenceSpec.CPU.SpreadOptions.Ratio
		}
	}
	return ratio, across
}
