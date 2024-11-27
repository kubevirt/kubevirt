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
 * Copyright The KubeVirt Authors
 *
 */
package apply

import (
	"fmt"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

type specFinder interface {
	Find(*virtv1.VirtualMachine) (*v1beta1.VirtualMachineInstancetypeSpec, error)
}

type preferenceSpecFinder interface {
	Find(*virtv1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error)
}

type vmApplier struct {
	vmiApplier         *vmiApplier
	instancetypeFinder specFinder
	preferenceFinder   preferenceSpecFinder
}

func NewVMApplier(instancetypeFinder specFinder, preferenceFinder preferenceSpecFinder) *vmApplier {
	return &vmApplier{
		vmiApplier:         NewVMIApplier(),
		instancetypeFinder: instancetypeFinder,
		preferenceFinder:   preferenceFinder,
	}
}

func (a *vmApplier) ApplyToVM(vm *virtv1.VirtualMachine) error {
	if vm.Spec.Instancetype == nil && vm.Spec.Preference == nil {
		return nil
	}
	instancetypeSpec, err := a.instancetypeFinder.Find(vm)
	if err != nil {
		return err
	}
	preferenceSpec, err := a.preferenceFinder.Find(vm)
	if err != nil {
		return err
	}
	if conflicts := a.vmiApplier.ApplyToVMI(
		k8sfield.NewPath("spec"),
		instancetypeSpec,
		preferenceSpec,
		&vm.Spec.Template.Spec,
		&vm.Spec.Template.ObjectMeta,
	); len(conflicts) > 0 {
		return fmt.Errorf("VM conflicts with instancetype spec in fields: [%s]", conflicts.String())
	}
	return nil
}
