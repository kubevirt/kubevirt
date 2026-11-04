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

	v1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	instancetypeannotations "kubevirt.io/kubevirt/pkg/instancetype/annotations"
	instancetypeapply "kubevirt.io/kubevirt/pkg/instancetype/apply"
	preferenceannotations "kubevirt.io/kubevirt/pkg/instancetype/preference/annotations"
	preferenceapply "kubevirt.io/kubevirt/pkg/instancetype/preference/apply"
)

func rejectUnsupportedVM(vm *v1.VirtualMachine, opts Options) error {
	if vm.Spec.Instancetype != nil && vm.Spec.Instancetype.InferFromVolume != "" {
		return fmt.Errorf("offline render does not infer instancetype from a volume")
	}
	if vm.Spec.Preference != nil && vm.Spec.Preference.InferFromVolume != "" {
		return fmt.Errorf("offline render does not infer preference from a volume")
	}
	if vm.Spec.Instancetype != nil && opts.Instancetype == nil {
		return fmt.Errorf("offline render does not apply instancetype matchers")
	}
	if vm.Spec.Instancetype == nil && opts.Instancetype != nil {
		return fmt.Errorf("instancetype spec was provided but the VM has no instancetype matcher")
	}
	if vm.Spec.Preference != nil && opts.Preference == nil {
		return fmt.Errorf("offline render does not apply preference matchers")
	}
	if vm.Spec.Preference == nil && opts.Preference != nil {
		return fmt.Errorf("preference spec was provided but the VM has no preference matcher")
	}
	return nil
}

// applyAdmissionPreferences matches the VM mutator: architecture and machine
// type preferences are written onto the VM before SetVirtualMachineDefaults,
// which only fills fields that are still empty.
func applyAdmissionPreferences(vm *v1.VirtualMachine, pref *v1beta1.VirtualMachinePreferenceSpec) {
	if pref == nil || vm.Spec.Template == nil {
		return
	}
	preferenceapply.ApplyArchitecturePreferences(pref, &vm.Spec.Template.Spec)
	applyMachinePreference(pref, &vm.Spec.Template.Spec)
}

func applyMachinePreference(pref *v1beta1.VirtualMachinePreferenceSpec, spec *v1.VirtualMachineInstanceSpec) {
	if pref.Machine == nil || pref.Machine.PreferredMachineType == "" {
		return
	}
	if spec.Domain.Machine != nil && spec.Domain.Machine.Type != "" {
		return
	}
	if spec.Domain.Machine == nil {
		spec.Domain.Machine = &v1.Machine{}
	}
	spec.Domain.Machine.Type = pref.Machine.PreferredMachineType
}

func applyPreferenceAutoAttach(vmi *v1.VirtualMachineInstance, pref *v1beta1.VirtualMachinePreferenceSpec) {
	if pref == nil {
		return
	}
	preferenceapply.ApplyAutoAttachPreferences(pref, &vmi.Spec)
}

func applyInstancetypeToVMI(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance, opts Options) error {
	if opts.Instancetype == nil && opts.Preference == nil {
		return nil
	}

	instancetypeannotations.Set(vm, vmi)
	preferenceannotations.Set(vm, vmi)

	if conflicts := instancetypeapply.NewVMIApplier().ApplyToVMI(
		k8sfield.NewPath("spec"),
		opts.Instancetype,
		opts.Preference,
		&vmi.Spec,
		&vmi.ObjectMeta,
	); len(conflicts) > 0 {
		return fmt.Errorf("VMI conflicts with instancetype spec in fields: [%s]", conflicts.String())
	}
	return nil
}
