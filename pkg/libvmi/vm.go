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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package libvmi

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

type VMOption func(vm *v1.VirtualMachine)

func NewVirtualMachine(vmi *v1.VirtualMachineInstance, opts ...VMOption) *v1.VirtualMachine {
	vm := &v1.VirtualMachine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "VirtualMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmi.Name,
			Namespace: vmi.Namespace,
		},
		Spec: v1.VirtualMachineSpec{
			Running: pointer.P(false),
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: vmi.ObjectMeta.Annotations,
					Labels:      vmi.ObjectMeta.Labels,
				},
				Spec: vmi.Spec,
			},
		},
	}

	for _, f := range opts {
		f(vm)
	}

	return vm
}

func WithRunning() VMOption {
	return func(vm *v1.VirtualMachine) {
		vm.Spec.Running = pointer.P(true)
	}
}

func WithDataVolumeTemplate(datavolume *v1beta1.DataVolume) VMOption {
	return func(vm *v1.VirtualMachine) {
		vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates,
			v1.DataVolumeTemplateSpec{
				ObjectMeta: datavolume.ObjectMeta,
				Spec:       datavolume.Spec,
			},
		)
	}
}

func WithClusterInstancetype(name string) VMOption {
	return func(vm *v1.VirtualMachine) {
		vm.Spec.Instancetype = &v1.InstancetypeMatcher{
			Name: name,
		}
	}
}

func WithClusterPreference(name string) VMOption {
	return func(vm *v1.VirtualMachine) {
		vm.Spec.Preference = &v1.PreferenceMatcher{
			Name: name,
		}
	}
}

func WithInstancetype(name string) VMOption {
	return func(vm *v1.VirtualMachine) {
		vm.Spec.Instancetype = &v1.InstancetypeMatcher{
			Name: name,
			Kind: instancetypeapi.SingularResourceName,
		}
	}
}

func WithPreference(name string) VMOption {
	return func(vm *v1.VirtualMachine) {
		vm.Spec.Preference = &v1.PreferenceMatcher{
			Name: name,
			Kind: instancetypeapi.SingularPreferenceResourceName,
		}
	}
}

func WithInstancetypeInferredFromVolume(name string) VMOption {
	return func(vm *v1.VirtualMachine) {
		vm.Spec.Instancetype = &v1.InstancetypeMatcher{
			InferFromVolume: name,
		}
	}
}

func WithPreferenceInferredFromVolume(name string) VMOption {
	return func(vm *v1.VirtualMachine) {
		vm.Spec.Preference = &v1.PreferenceMatcher{
			InferFromVolume: name,
		}
	}
}
