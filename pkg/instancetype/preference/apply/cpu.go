/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package apply

import (
	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

func applyCPUPreferences(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
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

func GetPreferredTopology(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec) v1beta1.PreferredCPUTopology {
	// Default to PreferSockets when a PreferredCPUTopology isn't provided
	preferredTopology := v1beta1.Sockets
	if preferenceSpec != nil && preferenceSpec.CPU != nil && preferenceSpec.CPU.PreferredCPUTopology != nil {
		preferredTopology = *preferenceSpec.CPU.PreferredCPUTopology
	}
	return preferredTopology
}

const defaultSpreadRatio uint32 = 2

func GetSpreadOptions(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec) (uint32, v1beta1.SpreadAcross) {
	ratio := defaultSpreadRatio
	if preferenceSpec.PreferSpreadSocketToCoreRatio != 0 {
		ratio = preferenceSpec.PreferSpreadSocketToCoreRatio
	}
	across := v1beta1.SpreadAcrossSocketsCores
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
