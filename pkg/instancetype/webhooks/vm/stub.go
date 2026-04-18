/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
)

type admitterStub struct {
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

func NewAdmitterStub() *admitterStub {
	return &admitterStub{
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

func (m *admitterStub) ApplyToVM(vm *virtv1.VirtualMachine) (
	*v1beta1.VirtualMachineInstancetypeSpec,
	*v1beta1.VirtualMachinePreferenceSpec,
	[]metav1.StatusCause,
) {
	return m.ApplyToVMFunc(vm)
}

func (m *admitterStub) Check(
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) (conflict.Conflicts, error) {
	return m.CheckFunc(instancetypeSpec, preferenceSpec, vmiSpec)
}
