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

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/instancetype/annotations"
	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	preferenceannotations "kubevirt.io/kubevirt/pkg/instancetype/preference/annotations"
	preferenceapply "kubevirt.io/kubevirt/pkg/instancetype/preference/apply"
)

func (h *renderer) ApplyAutoAttachPreferences(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	if vm.Spec.Preference == nil {
		return nil
	}
	preferenceSpec, err := h.preferenceFinder.FindPreference(vm)
	if err != nil {
		return err
	}
	if preferenceSpec != nil {
		preferenceapply.ApplyAutoAttachPreferences(preferenceSpec, &vmi.Spec)
	}
	return nil
}

func (h *renderer) ApplyToVMI(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	instancetypeSpec, err := h.instancetypeFinder.Find(vm)
	if err != nil {
		return err
	}
	preferenceSpec, err := h.preferenceFinder.FindPreference(vm)
	if err != nil {
		return err
	}
	if instancetypeSpec == nil && preferenceSpec == nil {
		return nil
	}

	annotations.Set(vm, vmi)
	preferenceannotations.Set(vm, vmi)

	if conflicts := apply.NewVMIApplier().ApplyToVMI(
		k8sfield.NewPath("spec"), instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta,
	); len(conflicts) > 0 {
		return fmt.Errorf("VMI conflicts with instancetype spec in fields: [%s]", conflicts.String())
	}
	return nil
}
