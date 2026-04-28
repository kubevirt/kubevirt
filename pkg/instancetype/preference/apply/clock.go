/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package apply

import (
	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

func applyClockPreferences(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	if preferenceSpec.Clock == nil {
		return
	}

	if vmiSpec.Domain.Clock == nil {
		vmiSpec.Domain.Clock = &virtv1.Clock{}
	}

	// We don't want to allow a partial overwrite here so only replace when nothing is set
	if preferenceSpec.Clock.PreferredClockOffset != nil &&
		vmiSpec.Domain.Clock.ClockOffset.UTC == nil &&
		vmiSpec.Domain.Clock.ClockOffset.Timezone == nil {
		vmiSpec.Domain.Clock.ClockOffset = *preferenceSpec.Clock.PreferredClockOffset.DeepCopy()
	}

	if preferenceSpec.Clock.PreferredTimer != nil && vmiSpec.Domain.Clock.Timer == nil {
		vmiSpec.Domain.Clock.Timer = preferenceSpec.Clock.PreferredTimer.DeepCopy()
	}
}
