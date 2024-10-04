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
 * Copyright 2024 Red Hat, Inc.
 *
 */
package apply

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

type VMIApplier struct{}

func (a *VMIApplier) Apply(
	preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
	vmiMetadata *metav1.ObjectMeta,
) {
	if preferenceSpec == nil {
		return
	}

	applyCPUPreferences(preferenceSpec, vmiSpec)
	ApplyDevicePreferences(preferenceSpec, vmiSpec)
	applyFeaturePreferences(preferenceSpec, vmiSpec)
	applyFirmwarePreferences(preferenceSpec, vmiSpec)
	applyMachinePreferences(preferenceSpec, vmiSpec)
	applyClockPreferences(preferenceSpec, vmiSpec)
	applySubdomain(preferenceSpec, vmiSpec)
	applyTerminationGracePeriodSeconds(preferenceSpec, vmiSpec)
	applyPreferenceAnnotations(preferenceSpec.Annotations, vmiMetadata)
}

func applySubdomain(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	if vmiSpec.Subdomain == "" && preferenceSpec.PreferredSubdomain != nil {
		vmiSpec.Subdomain = *preferenceSpec.PreferredSubdomain
	}
}

func applyTerminationGracePeriodSeconds(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	if preferenceSpec.PreferredTerminationGracePeriodSeconds != nil && vmiSpec.TerminationGracePeriodSeconds == nil {
		vmiSpec.TerminationGracePeriodSeconds = pointer.P(*preferenceSpec.PreferredTerminationGracePeriodSeconds)
	}
}
