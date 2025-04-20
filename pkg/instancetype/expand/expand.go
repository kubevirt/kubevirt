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
package expand

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/defaults"
	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	utils "kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type vmiApplier interface {
	ApplyToVMI(
		field *k8sfield.Path,
		instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
		preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
		vmiSpec *virtv1.VirtualMachineInstanceSpec,
		vmiMetadata *metav1.ObjectMeta,
	) (conflicts conflict.Conflicts)
}

type specFinder interface {
	Find(*virtv1.VirtualMachine) (*v1beta1.VirtualMachineInstancetypeSpec, error)
}

type preferenceSpecFinder interface {
	FindPreference(*virtv1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error)
}

type expander struct {
	vmiApplier
	specFinder
	preferenceSpecFinder

	clusterConfig *virtconfig.ClusterConfig
}

func New(
	clusterConfig *virtconfig.ClusterConfig,
	instancetypeFinder specFinder,
	preferenceFinder preferenceSpecFinder,
) *expander {
	return &expander{
		clusterConfig:        clusterConfig,
		vmiApplier:           apply.NewVMIApplier(),
		specFinder:           instancetypeFinder,
		preferenceSpecFinder: preferenceFinder,
	}
}

func (e *expander) Expand(vm *virtv1.VirtualMachine) (*virtv1.VirtualMachine, error) {
	if vm.Spec.Instancetype == nil && vm.Spec.Preference == nil {
		return vm, nil
	}

	instancetypeSpec, err := e.Find(vm)
	if err != nil {
		return nil, err
	}

	preferenceSpec, err := e.FindPreference(vm)
	if err != nil {
		return nil, err
	}

	expandedVM := vm.DeepCopy()

	utils.SetDefaultVolumeDisk(&expandedVM.Spec.Template.Spec)

	if err := vmispec.SetDefaultNetworkInterface(e.clusterConfig, &expandedVM.Spec.Template.Spec); err != nil {
		return nil, err
	}

	// Replace with VMApplier.ApplyToVM once conflict errors are aligned
	conflicts := e.ApplyToVMI(
		k8sfield.NewPath("spec", "template", "spec"),
		instancetypeSpec, preferenceSpec,
		&expandedVM.Spec.Template.Spec,
		&expandedVM.Spec.Template.ObjectMeta,
	)
	if len(conflicts) > 0 {
		return nil, conflicts
	}

	// Apply defaults to VM.Spec.Template.Spec after applying instance types to ensure we don't conflict
	if err := defaults.SetDefaultVirtualMachineInstanceSpec(e.clusterConfig, &expandedVM.Spec.Template.Spec); err != nil {
		return nil, err
	}

	// Remove {Instancetype,Preference}Matcher and {Instancetype,Preference}Ref
	expandedVM.Spec.Instancetype = nil
	expandedVM.Status.InstancetypeRef = nil
	expandedVM.Spec.Preference = nil
	expandedVM.Status.PreferenceRef = nil

	return expandedVM, nil
}
