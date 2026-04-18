/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package apply

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

type vmiApplier struct{}

func New() *vmiApplier {
	return &vmiApplier{}
}

func (a *vmiApplier) Apply(
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
	ApplyArchitecturePreferences(preferenceSpec, vmiSpec)
	applyPreferenceAnnotations(preferenceSpec.Annotations, vmiMetadata)
}
