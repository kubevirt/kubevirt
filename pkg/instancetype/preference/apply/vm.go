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
 */

package apply

import (
	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

type vmApplier struct {
	specApplier preferenceApply
}

func NewVM() *vmApplier {
	return &vmApplier{specApplier: NewSpec()}
}

func (a *vmApplier) Apply(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vm *virtv1.VirtualMachine) {
	if preferenceSpec == nil {
		return
	}

	a.ApplyToTemplate(preferenceSpec, vm.Spec.Template)
}

func (a *vmApplier) ApplyToTemplate(
	preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
	template *virtv1.VirtualMachineInstanceTemplateSpec,
) {
	if preferenceSpec == nil {
		return
	}

	a.specApplier.Apply(preferenceSpec, &template.Spec, &template.ObjectMeta)
}
