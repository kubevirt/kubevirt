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

package vm

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
)

type MockAdmitter struct {
	ApplyToVMFunc func(*virtv1.VirtualMachine) (
		*v1beta1.VirtualMachineInstancetypeSpec,
		*v1beta1.VirtualMachinePreferenceSpec,
		[]metav1.StatusCause,
	)
	CheckFunc func(*v1beta1.VirtualMachineInstancetypeSpec,
		*v1beta1.VirtualMachinePreferenceSpec,
		*virtv1.VirtualMachineInstanceSpec,
	) (conflict.Conflicts, error)
}

func NewMockAdmitter() *MockAdmitter {
	return &MockAdmitter{
		ApplyToVMFunc: func(*virtv1.VirtualMachine) (
			*v1beta1.VirtualMachineInstancetypeSpec,
			*v1beta1.VirtualMachinePreferenceSpec,
			[]metav1.StatusCause,
		) {
			return nil, nil, nil
		},
		CheckFunc: func(*v1beta1.VirtualMachineInstancetypeSpec,
			*v1beta1.VirtualMachinePreferenceSpec,
			*virtv1.VirtualMachineInstanceSpec,
		) (conflict.Conflicts, error) {
			return nil, nil
		},
	}
}

func (m *MockAdmitter) ApplyToVM(vm *virtv1.VirtualMachine) (
	*v1beta1.VirtualMachineInstancetypeSpec,
	*v1beta1.VirtualMachinePreferenceSpec,
	[]metav1.StatusCause,
) {
	return m.ApplyToVMFunc(vm)
}

func (m *MockAdmitter) Check(
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) (conflict.Conflicts, error) {
	return m.CheckFunc(instancetypeSpec, preferenceSpec, vmiSpec)
}
