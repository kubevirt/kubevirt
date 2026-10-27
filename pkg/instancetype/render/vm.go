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
package render

import (
	"fmt"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	preferenceapply "kubevirt.io/kubevirt/pkg/instancetype/preference/apply"
	"kubevirt.io/kubevirt/pkg/pointer"
)

// ApplyPreDefaultPreferences applies preferences that must precede VM defaulting
// or DataVolume creation: architecture, machine type and storage class.
func ApplyPreDefaultPreferences(vm *virtv1.VirtualMachine, preferenceSpec *v1beta1.VirtualMachinePreferenceSpec) {
	if preferenceSpec == nil {
		return
	}
	if vm.Spec.Template != nil {
		preferenceapply.ApplyArchitecturePreferences(preferenceSpec, &vm.Spec.Template.Spec)
		preferenceapply.ApplyMachinePreferences(preferenceSpec, &vm.Spec.Template.Spec)
	}
	if preferenceSpec.Volumes == nil || preferenceSpec.Volumes.PreferredStorageClassName == "" {
		return
	}
	for _, dv := range vm.Spec.DataVolumeTemplates {
		if dv.Spec.PVC != nil && dv.Spec.PVC.StorageClassName == nil {
			dv.Spec.PVC.StorageClassName = pointer.P(preferenceSpec.Volumes.PreferredStorageClassName)
		}
		if dv.Spec.Storage != nil && dv.Spec.Storage.StorageClassName == nil {
			dv.Spec.Storage.StorageClassName = pointer.P(preferenceSpec.Volumes.PreferredStorageClassName)
		}
	}
}

func (h *renderer) ApplyToVM(vm *virtv1.VirtualMachine) error {
	if vm.Spec.Instancetype == nil && vm.Spec.Preference == nil {
		return nil
	}
	instancetypeSpec, err := h.instancetypeFinder.Find(vm)
	if err != nil {
		return err
	}
	preferenceSpec, err := h.preferenceFinder.FindPreference(vm)
	if err != nil {
		return err
	}
	if conflicts := apply.NewVMIApplier().ApplyToVMI(
		specField,
		instancetypeSpec,
		preferenceSpec,
		&vm.Spec.Template.Spec,
		&vm.Spec.Template.ObjectMeta,
	); len(conflicts) > 0 {
		return fmt.Errorf("VM conflicts with instancetype spec in fields: [%s]", conflicts.String())
	}
	return nil
}
