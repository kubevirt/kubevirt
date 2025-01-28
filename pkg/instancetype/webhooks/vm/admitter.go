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
package vm

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"kubevirt.io/client-go/kubecli"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	"kubevirt.io/kubevirt/pkg/instancetype/find"
	preferenceFind "kubevirt.io/kubevirt/pkg/instancetype/preference/find"
	"kubevirt.io/kubevirt/pkg/instancetype/preference/requirements"
	"kubevirt.io/kubevirt/pkg/instancetype/preference/validation"
)

type instancetypeFinder interface {
	Find(*virtv1.VirtualMachine) (*v1beta1.VirtualMachineInstancetypeSpec, error)
}

type preferenceFinder interface {
	FindPreference(*virtv1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error)
}

type requirementsChecker interface {
	Check(*v1beta1.VirtualMachineInstancetypeSpec,
		*v1beta1.VirtualMachinePreferenceSpec,
		*virtv1.VirtualMachineInstanceSpec,
	) (conflict.Conflicts, error)
}

type applyVMIHandler interface {
	ApplyToVMI(
		*k8sfield.Path,
		*v1beta1.VirtualMachineInstancetypeSpec,
		*v1beta1.VirtualMachinePreferenceSpec,
		*virtv1.VirtualMachineInstanceSpec,
		*metav1.ObjectMeta,
	) conflict.Conflicts
}

type admitter struct {
	instancetypeFinder
	preferenceFinder
	applyVMIHandler
	requirementsChecker
}

func NewAdmitter(virtClient kubecli.KubevirtClient) *admitter {
	return &admitter{
		instancetypeFinder:  find.NewSpecFinder(nil, nil, nil, virtClient),
		preferenceFinder:    preferenceFind.NewSpecFinder(nil, nil, nil, virtClient),
		requirementsChecker: requirements.New(),
		applyVMIHandler:     apply.NewVMIApplier(),
	}
}

func (a *admitter) ApplyToVM(vm *virtv1.VirtualMachine) (
	*v1beta1.VirtualMachineInstancetypeSpec,
	*v1beta1.VirtualMachinePreferenceSpec,
	[]metav1.StatusCause,
) {
	instancetypeSpec, err := a.Find(vm)
	if err != nil {
		return nil, nil, []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueNotFound,
			Message: fmt.Sprintf("Failure to find instancetype: %v", err),
			Field:   k8sfield.NewPath("spec", "instancetype").String(),
		}}
	}

	preferenceSpec, err := a.FindPreference(vm)
	if err != nil {
		return nil, nil, []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueNotFound,
			Message: fmt.Sprintf("Failure to find preference: %v", err),
			Field:   k8sfield.NewPath("spec", "preference").String(),
		}}
	}

	if spreadConflict := validation.CheckSpreadCPUTopology(instancetypeSpec, preferenceSpec); spreadConflict != nil {
		return nil, nil, spreadConflict.StatusCauses()
	}

	conflicts := a.ApplyToVMI(
		k8sfield.NewPath("spec", "template", "spec"),
		instancetypeSpec,
		preferenceSpec,
		&vm.Spec.Template.Spec,
		&vm.Spec.Template.ObjectMeta,
	)

	if len(conflicts) > 0 {
		return nil, nil, conflicts.StatusCauses()
	}

	return instancetypeSpec, preferenceSpec, nil
}
