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
