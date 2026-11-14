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
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	preferenceApply "kubevirt.io/kubevirt/pkg/instancetype/preference/apply"
)

type preferenceVMApplier interface {
	Apply(*v1beta1.VirtualMachinePreferenceSpec, *virtv1.VirtualMachine)
	ApplyToTemplate(*v1beta1.VirtualMachinePreferenceSpec, *virtv1.VirtualMachineInstanceTemplateSpec)
}

type vmApplier struct {
	specApplier       *specApplier
	preferenceApplier preferenceVMApplier
}

func NewVM() *vmApplier {
	return &vmApplier{
		specApplier:       NewSpec(),
		preferenceApplier: preferenceApply.NewVM(),
	}
}

func (a *vmApplier) Apply(
	field *k8sfield.Path,
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
	vm *virtv1.VirtualMachine,
) conflict.Conflicts {
	if instancetypeSpec == nil && preferenceSpec == nil {
		return nil
	}

	template := vm.Spec.Template
	if conflicts := a.specApplier.Apply(field, instancetypeSpec, preferenceSpec, &template.Spec, &template.ObjectMeta); len(conflicts) > 0 {
		return conflicts
	}

	a.preferenceApplier.Apply(preferenceSpec, vm)

	return nil
}

func (a *vmApplier) ApplyToTemplate(
	field *k8sfield.Path,
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
	template *virtv1.VirtualMachineInstanceTemplateSpec,
) conflict.Conflicts {
	if instancetypeSpec == nil && preferenceSpec == nil {
		return nil
	}

	if conflicts := a.specApplier.Apply(field, instancetypeSpec, preferenceSpec, &template.Spec, &template.ObjectMeta); len(conflicts) > 0 {
		return conflicts
	}

	a.preferenceApplier.ApplyToTemplate(preferenceSpec, template)

	return nil
}
