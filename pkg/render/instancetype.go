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
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	instancetypeannotations "kubevirt.io/kubevirt/pkg/instancetype/annotations"
	instancetypeapply "kubevirt.io/kubevirt/pkg/instancetype/apply"
	preferenceannotations "kubevirt.io/kubevirt/pkg/instancetype/preference/annotations"
	preferenceapply "kubevirt.io/kubevirt/pkg/instancetype/preference/apply"
)

func rejectUnsupportedVM(vm *virtv1.VirtualMachine, opts Options) error {
	if vm.Spec.Instancetype != nil && opts.Instancetype == nil {
		return fmt.Errorf("offline render does not apply instancetype matchers")
	}
	if vm.Spec.Preference != nil && opts.Preference == nil {
		return fmt.Errorf("offline render does not apply preference matchers")
	}
	return nil
}

func applyPreferenceAutoAttach(vmi *virtv1.VirtualMachineInstance, pref *v1beta1.VirtualMachinePreferenceSpec) {
	if pref == nil {
		return
	}
	preferenceapply.ApplyAutoAttachPreferences(pref, &vmi.Spec)
}

func applyInstancetypeToVMI(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, opts Options) error {
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
