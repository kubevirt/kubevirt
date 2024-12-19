package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
)

type mockController struct {
	syncFunc       func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error)
	applyToVMFunc  func(*virtv1.VirtualMachine) error
	applyToVMIFunc func(
		*k8sfield.Path, *v1beta1.VirtualMachineInstancetypeSpec,
		*v1beta1.VirtualMachinePreferenceSpec,
		*virtv1.VirtualMachineInstanceSpec,
		*metav1.ObjectMeta) (conflicts apply.Conflicts)
	findFunc                   func(*virtv1.VirtualMachine) (*v1beta1.VirtualMachineInstancetypeSpec, error)
	findPreferenceFunc         func(*virtv1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error)
	applyDevicePreferencesFunc func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error
}

func NewMockController() *mockController {
	return &mockController{
		syncFunc: func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error) {
			return vm, nil
		},
		applyToVMFunc: func(*virtv1.VirtualMachine) error {
			return nil
		},
		applyToVMIFunc: func(
			*k8sfield.Path, *v1beta1.VirtualMachineInstancetypeSpec,
			*v1beta1.VirtualMachinePreferenceSpec,
			*virtv1.VirtualMachineInstanceSpec,
			*metav1.ObjectMeta,
		) (conflicts apply.Conflicts) {
			return nil
		},
		findFunc: func(*virtv1.VirtualMachine) (*v1beta1.VirtualMachineInstancetypeSpec, error) {
			return nil, nil
		},
		findPreferenceFunc: func(*virtv1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error) {
			return nil, nil
		},
		applyDevicePreferencesFunc: func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
			return nil
		},
	}
}

func (m *mockController) ApplyToVM(vm *virtv1.VirtualMachine) error {
	return m.applyToVMFunc(vm)
}

func (m *mockController) ApplyToVMI(
	path *k8sfield.Path,
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
	objMeta *metav1.ObjectMeta,
) (conflicts apply.Conflicts) {
	return m.applyToVMIFunc(path, instancetypeSpec, preferenceSpec, vmiSpec, objMeta)
}

func (m *mockController) Find(vm *virtv1.VirtualMachine) (*v1beta1.VirtualMachineInstancetypeSpec, error) {
	return m.findFunc(vm)
}

func (m *mockController) FindPreference(vm *virtv1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error) {
	return m.findPreferenceFunc(vm)
}

func (m *mockController) Sync(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error) {
	return m.syncFunc(vm, vmi)
}

func (m *mockController) ApplyDevicePreferences(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	return m.applyDevicePreferencesFunc(vm, vmi)
}
