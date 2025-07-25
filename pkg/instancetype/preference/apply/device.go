//nolint:gocyclo
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
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

type field struct {
	preference *bool
	vmi        **bool
}

func ApplyAutoAttachPreferences(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	if preferenceSpec.Devices == nil {
		return
	}
	autoAttachFields := []field{
		{preferenceSpec.Devices.PreferredAutoattachGraphicsDevice, &vmiSpec.Domain.Devices.AutoattachGraphicsDevice},
		{preferenceSpec.Devices.PreferredAutoattachMemBalloon, &vmiSpec.Domain.Devices.AutoattachMemBalloon},
		{preferenceSpec.Devices.PreferredAutoattachPodInterface, &vmiSpec.Domain.Devices.AutoattachPodInterface},
		{preferenceSpec.Devices.PreferredAutoattachSerialConsole, &vmiSpec.Domain.Devices.AutoattachSerialConsole},
		{preferenceSpec.Devices.PreferredAutoattachInputDevice, &vmiSpec.Domain.Devices.AutoattachInputDevice},
	}
	for _, field := range autoAttachFields {
		if field.preference != nil && *field.vmi == nil {
			*field.vmi = pointer.P(*field.preference)
		}
	}
}

func ApplyDevicePreferences(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	if preferenceSpec.Devices == nil {
		return
	}

	// We only want to apply a preference bool when...
	//
	// 1. A preference has actually been provided
	// 2. The user hasn't defined the corresponding attribute already within the VMI
	//
	if preferenceSpec.Devices.PreferredUseVirtioTransitional != nil && vmiSpec.Domain.Devices.UseVirtioTransitional == nil {
		vmiSpec.Domain.Devices.UseVirtioTransitional = pointer.P(*preferenceSpec.Devices.PreferredUseVirtioTransitional)
	}

	if preferenceSpec.Devices.PreferredBlockMultiQueue != nil && vmiSpec.Domain.Devices.BlockMultiQueue == nil {
		vmiSpec.Domain.Devices.BlockMultiQueue = pointer.P(*preferenceSpec.Devices.PreferredBlockMultiQueue)
	}

	if preferenceSpec.Devices.PreferredNetworkInterfaceMultiQueue != nil && vmiSpec.Domain.Devices.NetworkInterfaceMultiQueue == nil {
		vmiSpec.Domain.Devices.NetworkInterfaceMultiQueue = pointer.P(*preferenceSpec.Devices.PreferredNetworkInterfaceMultiQueue)
	}

	// FIXME DisableHotplug isn't a pointer bool so we don't have a way to tell if a user has actually set it, for now override.
	if preferenceSpec.Devices.PreferredDisableHotplug != nil {
		vmiSpec.Domain.Devices.DisableHotplug = *preferenceSpec.Devices.PreferredDisableHotplug
	}

	if preferenceSpec.Devices.PreferredSoundModel != "" && vmiSpec.Domain.Devices.Sound != nil && vmiSpec.Domain.Devices.Sound.Model == "" {
		vmiSpec.Domain.Devices.Sound.Model = preferenceSpec.Devices.PreferredSoundModel
	}

	if preferenceSpec.Devices.PreferredRng != nil && vmiSpec.Domain.Devices.Rng == nil {
		vmiSpec.Domain.Devices.Rng = preferenceSpec.Devices.PreferredRng.DeepCopy()
	}

	if preferenceSpec.Devices.PreferredTPM != nil && vmiSpec.Domain.Devices.TPM == nil {
		vmiSpec.Domain.Devices.TPM = preferenceSpec.Devices.PreferredTPM.DeepCopy()
	}

	ApplyAutoAttachPreferences(preferenceSpec, vmiSpec)
	applyDiskPreferences(preferenceSpec, vmiSpec)
	applyInterfacePreferences(preferenceSpec, vmiSpec)
	applyInputPreferences(preferenceSpec, vmiSpec)
	applyPanicDevicePreferences(preferenceSpec, vmiSpec)
}

func applyInputPreferences(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	for inputIndex := range vmiSpec.Domain.Devices.Inputs {
		vmiInput := &vmiSpec.Domain.Devices.Inputs[inputIndex]
		if preferenceSpec.Devices.PreferredInputBus != "" && vmiInput.Bus == "" {
			vmiInput.Bus = preferenceSpec.Devices.PreferredInputBus
		}

		if preferenceSpec.Devices.PreferredInputType != "" && vmiInput.Type == "" {
			vmiInput.Type = preferenceSpec.Devices.PreferredInputType
		}
	}
}

func applyPanicDevicePreferences(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	if preferenceSpec.Devices.PreferredPanicDeviceModel == nil {
		return
	}

	// Only apply any preferred panic device when the same panic device has not been provided by a user already
	for idx := range vmiSpec.Domain.Devices.PanicDevices {
		panicDevice := &vmiSpec.Domain.Devices.PanicDevices[idx]
		if panicDevice.Model != nil {
			continue
		}
		panicDevice.Model = preferenceSpec.Devices.PreferredPanicDeviceModel
	}
}
