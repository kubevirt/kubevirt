/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package apply

import (
	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

func ApplyArchitecturePreferences(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	if preferenceSpec == nil || preferenceSpec.PreferredArchitecture == nil || *preferenceSpec.PreferredArchitecture == "" {
		return
	}

	if vmiSpec.Architecture == "" {
		vmiSpec.Architecture = *preferenceSpec.PreferredArchitecture
	}
}
