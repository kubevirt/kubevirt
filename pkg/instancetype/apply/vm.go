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

	"kubevirt.io/kubevirt/pkg/instancetype/find"
	preferenceFind "kubevirt.io/kubevirt/pkg/instancetype/preference/find"
)

type VMApplier struct {
	vmiApplier         *VMIApplier
	instancetypeFinder *find.SpecFinder
	preferenceFinder   *preferenceFind.SpecFinder
}

func NewVMApplier(instancetypeFinder *find.SpecFinder, preferenceFinder *preferenceFind.SpecFinder) *VMApplier {
	return &VMApplier{
		vmiApplier:         NewVMIApplier(),
		instancetypeFinder: instancetypeFinder,
		preferenceFinder:   preferenceFinder,
	}
}

func (a *VMApplier) ApplyToVM(vm *virtv1.VirtualMachine) error {
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
