package controller

import (
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"
)

type mockController struct {
	syncFunc                   func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error)
	applyToVMFunc              func(*virtv1.VirtualMachine) error
	applyToVMIFunc             func(*virtv1.VirtualMachine, *virtv1.VirtualMachineInstance) error
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
		applyToVMIFunc: func(*virtv1.VirtualMachine, *virtv1.VirtualMachineInstance) error {
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

func (m *mockController) ApplyToVMI(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	return m.applyToVMIFunc(vm, vmi)
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
