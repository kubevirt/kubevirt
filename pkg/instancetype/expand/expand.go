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
package expand

import (
	"fmt"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/defaults"
	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/instancetype/find"
	preferenceFind "kubevirt.io/kubevirt/pkg/instancetype/preference/find"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	utils "kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	VMFieldsConflictsErrorFmt = "VM fields %s conflict with selected instance type"
)

type Expander struct {
	clusterConfig      *virtconfig.ClusterConfig
	vmiApplier         *apply.VMIApplier
	instancetypeFinder *find.SpecFinder
	preferenceFinder   *preferenceFind.SpecFinder
}

func New(
	clusterConfig *virtconfig.ClusterConfig,
	instancetypeFinder *find.SpecFinder,
	preferenceFinder *preferenceFind.SpecFinder,
) *Expander {
	return &Expander{
		clusterConfig:      clusterConfig,
		vmiApplier:         apply.NewVMIApplier(),
		instancetypeFinder: instancetypeFinder,
		preferenceFinder:   preferenceFinder,
	}
}

func (e *Expander) Expand(vm *virtv1.VirtualMachine) (*virtv1.VirtualMachine, error) {
	if vm.Spec.Instancetype == nil && vm.Spec.Preference == nil {
		return vm, nil
	}

	instancetypeSpec, err := e.instancetypeFinder.Find(vm)
	if err != nil {
		return nil, err
	}

	preferenceSpec, err := e.preferenceFinder.Find(vm)
	if err != nil {
		return nil, err
	}

	expandedVM := vm.DeepCopy()

	utils.SetDefaultVolumeDisk(&expandedVM.Spec.Template.Spec)

	if err := vmispec.SetDefaultNetworkInterface(e.clusterConfig, &expandedVM.Spec.Template.Spec); err != nil {
		return nil, err
	}

	// Replace with VMApplier.ApplyToVM once conflict errors are aligned
	conflicts := e.vmiApplier.ApplyToVMI(
		k8sfield.NewPath("spec", "template", "spec"),
		instancetypeSpec, preferenceSpec,
		&expandedVM.Spec.Template.Spec,
		&expandedVM.Spec.Template.ObjectMeta,
	)
	if len(conflicts) > 0 {
		return nil, fmt.Errorf(VMFieldsConflictsErrorFmt, conflicts.String())
	}

	// Apply defaults to VM.Spec.Template.Spec after applying instance types to ensure we don't conflict
	if err := defaults.SetDefaultVirtualMachineInstanceSpec(e.clusterConfig, &expandedVM.Spec.Template.Spec); err != nil {
		return nil, err
	}

	// Remove InstancetypeMatcher and PreferenceMatcher, so the returned VM object can be used and not cause a conflict
	expandedVM.Spec.Instancetype = nil
	expandedVM.Spec.Preference = nil

	return expandedVM, nil
}
